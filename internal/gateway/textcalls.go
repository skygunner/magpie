package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
)

// A model deep into a long conversation may write its tool calls into its
// text instead of making them (#823, Dazzle-sys: DeepSeek through a group,
// in Pi, ~700k tokens in): Hermes' <tool_call>{"name": …, "arguments": …}
// </tool_call>, GLM's <tool_call>name<arg_key>…</arg_key><arg_value>…
// </arg_value></tool_call> (#906), or DeepSeek's own <｜DSML｜invoke
// name="…"> with its
// parameters, often with the other's closing tags strewn after it. The
// agent got it as text, ran nothing and ended the turn; told to go on, the
// model read its own text-written call back and wrote the next one the
// same way, so the session was lost. When the request offered tools, text
// from the first such tag on is held; at the reply's end each block in it
// that names an offered tool and whose arguments can be read becomes a
// real tool call, and the reply ends as one that called tools. Held text
// that isn't all such blocks goes on as the text it was.

// textCallMarks are what a call written into the text opens with, beside
// DeepSeek's DSML tags, which dsmlMark reads.
var textCallMarks = []string{"<tool_call>"}

// dsmlMark tells whether s begins with a DSML tag's mark, read as the
// parse regexes read one (dsmlTag): <, spaces, / and spaces for a close
// when closing is true, a pipe, then a run of pipes and spaces, DSML and
// another run, which may be empty. A pipe is ASCII | or full-width ｜, in
// any number and any mix of widths: DeepSeek writes one or two
// (#2623, OKURBITO: <｜｜DSML｜｜invoke went out as text), and its widths
// don't always agree within a tag, so no spelling is listed. n is the
// mark's length when s begins with one; more is whether s, too short for
// one, could be one's beginning still being written, down to a full-width
// pipe's first bytes.
func dsmlMark(s string, closing bool) (n int, more bool) {
	if !strings.HasPrefix(s, "<") {
		return 0, false
	}
	i := 1
	space := func() {
		for i < len(s) && strings.ContainsRune(" \t\r\n", rune(s[i])) {
			i++
		}
	}
	// pipe steps over one pipe at i; part is whether s ends in the first
	// bytes of a full-width one
	pipe := func() (ok, part bool) {
		switch {
		case i < len(s) && s[i] == '|':
			i++
			return true, false
		case strings.HasPrefix(s[i:], "｜"):
			i += len("｜")
			return true, false
		}
		return false, i < len(s) && strings.HasPrefix("｜", s[i:])
	}
	// run reads pipes and spaces from i; part as for pipe
	run := func() (part bool) {
		for {
			space()
			if ok, part := pipe(); !ok {
				return part
			}
		}
	}
	space()
	if closing && i < len(s) && s[i] == '/' {
		i++
		space()
	}
	if i == len(s) {
		return 0, true
	}
	if ok, part := pipe(); !ok {
		return 0, part
	}
	if run() || i == len(s) {
		return 0, true
	}
	if rest := s[i:]; !strings.HasPrefix(rest, "DSML") {
		return 0, strings.HasPrefix("DSML", rest)
	}
	i += len("DSML")
	if run() || i == len(s) {
		// the run after DSML may go on in the next chunk
		return 0, true
	}
	return i, false
}

const (
	// DeepSeek's own special-token markup (#1848, yetone: over DeepSeek's
	// own API, deepseek-flash, a call came as text in it): a calls
	// wrapper, then each call's open, its name, its separation and its
	// arguments, then the call's close and the wrapper's close. The
	// token's space is ▁, which a model also writes as _.
	nativeCallsBegin = "tool▁calls▁begin"
	nativeCallBegin  = "tool▁call▁begin"
	nativeCallEnd    = "tool▁call▁end"
	nativeCallsEnd   = "tool▁calls▁end"
	nativeSep        = "tool▁sep"
	// the invoke the issue's second shape writes without its DSML word
	pipedInvoke = "invoke"
)

// deepSeekTag is the tag naming word as that markup spells it: <, a run of
// pipes and spaces, the word (▁ either way), another run, then >.
func deepSeekTag(word string) *regexp.Regexp {
	return regexp.MustCompile(`<\s*[|｜][\s|｜]*` + strings.ReplaceAll(word, "▁", "[▁_]") + `[\s|｜]*>`)
}

var (
	nativeBegin  = deepSeekTag(nativeCallsBegin)
	nativeOpen   = deepSeekTag(nativeCallBegin)
	nativeEnd    = deepSeekTag(nativeCallEnd)
	nativeClose  = deepSeekTag(nativeCallsEnd)
	nativeSepTag = deepSeekTag(nativeSep)
	// nativeTags are those, nativeWords what a tag cut short may be the
	// start of
	nativeTags  = []*regexp.Regexp{nativeBegin, nativeOpen, nativeEnd, nativeClose, nativeSepTag}
	nativeWords = []string{nativeCallsBegin, nativeCallBegin, nativeCallEnd, nativeCallsEnd, nativeSep}
)

// nativeAt is where in s the first of those tags is, -1 for nowhere.
func nativeAt(s string) int {
	at := -1
	for _, re := range nativeTags {
		if m := re.FindStringIndex(s); m != nil && (at < 0 || m[0] < at) {
			at = m[0]
		}
	}
	return at
}

// nativeMarkAt is the length of the tag that begins s, 0 when none does.
func nativeMarkAt(s string) int {
	for _, re := range nativeTags {
		if m := re.FindStringIndex(s); m != nil && m[0] == 0 {
			return m[1]
		}
	}
	return 0
}

// nativeShort tells whether s is one of those tags' beginning: its <, its
// pipes and its word there, not their end.
func nativeShort(s string) bool {
	if !strings.HasPrefix(s, "<") {
		return false
	}
	rest, more := tagRest(s)
	if more || rest == "" {
		return true // through the pipes; the word may still come
	}
	for _, w := range nativeWords {
		for _, word := range []string{w, strings.ReplaceAll(w, "▁", "_")} {
			if strings.HasPrefix(word, rest) {
				return true
			}
			if r, ok := strings.CutPrefix(rest, word); ok {
				return strings.HasPrefix(">", strings.TrimLeft(r, " \t|｜"))
			}
		}
	}
	return false
}

// tagRest steps over a tag's <, its spaces and its first pipe, and gives
// what follows; more is whether s ends inside those, a full-width pipe's
// first bytes too.
func tagRest(s string) (rest string, more bool) {
	if !strings.HasPrefix(s, "<") {
		return "", false
	}
	s = strings.TrimLeft(s[1:], " \t")
	switch {
	case strings.HasPrefix(s, "|"):
		return s[1:], false
	case strings.HasPrefix(s, "｜"):
		return s[len("｜"):], false
	}
	return "", strings.HasPrefix("｜", s)
}

// writtenCall is one call read out of the text.
type writtenCall struct {
	Name string
	Args json.RawMessage
}

var (
	dsmlTag    = `<\s*/?\s*[|｜][\s|｜]*DSML[\s|｜]*`
	dsmlInvoke = regexp.MustCompile(`<\s*[|｜][\s|｜]*DSML[\s|｜]*invoke\s+name\s*=\s*"([^"]+)"\s*>`)
	dsmlParam  = regexp.MustCompile(`(?s)<\s*[|｜][\s|｜]*DSML[\s|｜]*parameter\s+name\s*=\s*"([^"]+)"((?:\s+\w+\s*=\s*"[^"]*")*)\s*>(.*?)` + dsmlTag + `parameter\s*>`)
	dsmlClose  = regexp.MustCompile(`<\s*/\s*[|｜][\s|｜]*DSML[\s|｜]*invoke\s*>`)
	dsmlString = regexp.MustCompile(`string\s*=\s*"true"`)
	// what is left of the tags once the calls are read out, and the
	// braces, quotes or > GLM leaves before a close (#1945)
	textCallJunk = regexp.MustCompile(dsmlTag + `[^<>]*>?|<\s*/?\s*[|｜][\s|｜]*[A-Za-z_▁][^<>]*>|[}\]"'>]*\s*</tool_calls?>|<tool_calls?>`)
)

// textCallNames are the tools a request offers, by name.
func textCallNames(tools []Tool) map[string]bool {
	if len(tools) == 0 {
		return nil
	}
	m := map[string]bool{}
	for _, t := range tools {
		if t.Name != "" {
			m[t.Name] = true
		}
	}
	return m
}

// chatToolNames are the function tools of a Chat Completions request.
func chatToolNames(body []byte) map[string]bool {
	var q struct {
		Tools []struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
			Name string `json:"name"`
		} `json:"tools"`
	}
	if decodeJSON(body, &q) != nil || len(q.Tools) == 0 {
		return nil
	}
	m := map[string]bool{}
	for _, t := range q.Tools {
		if n := t.Function.Name; n != "" {
			m[n] = true
		} else if t.Name != "" {
			m[t.Name] = true
		}
	}
	return m
}

// textCallAt is where in s a call written into the text begins, -1 for
// nowhere; and, when it doesn't, how much of s's end could be one's
// beginning still being written. A DSML close begins one too: DeepSeek
// writes its calls wrapper's close however it likes (</｜｜DSML｜｜ calls>,
// 喜欢一个人 on Discord), and one left behind with nothing before it, after
// calls an upstream made of the invokes, is held to be dropped, not shown.
func textCallAt(s string) (at, tail int) {
	at = -1
	for _, m := range textCallMarks {
		if i := strings.Index(s, m); i >= 0 && (at < 0 || i < at) {
			at = i
		}
	}
	for i := strings.IndexByte(s, '<'); i >= 0 && (at < 0 || i < at); {
		n, more := dsmlMark(s[i:], true)
		if n == 0 {
			if n = nativeMarkAt(s[i:]); n == 0 {
				n = pipedMarkAt(s[i:])
			}
		}
		if n > 0 {
			at = i
			break
		}
		if more = more || nativeShort(s[i:]) || pipedShort(s[i:]); more && tail == 0 {
			tail = len(s) - i
		}
		j := strings.IndexByte(s[i+1:], '<')
		if j < 0 {
			break
		}
		i += 1 + j
	}
	if at >= 0 {
		return at, 0
	}
	for _, m := range textCallMarks {
		for n := min(len(m)-1, len(s)); n > tail; n-- {
			if strings.HasSuffix(s, m[:n]) {
				tail = n
				break
			}
		}
	}
	return -1, tail
}

// parseTextCalls reads the calls written in s, which begins with one.
// ok is false unless every block in it is a call of one of names whose
// arguments can be read; rest is the text in it that is no call or tag.
func parseTextCalls(s string, names map[string]bool) (calls []writtenCall, rest string, ok bool) {
	var left strings.Builder
	for s != "" {
		// at is where the first of the four marks a call opens with is,
		// kind which it is: 1 Hermes', 2 DSML's, 3 DeepSeek's own (#1848),
		// 4 its invoke without the DSML word
		at, kind := -1, 0
		take := func(i, k int) {
			if i >= 0 && (at < 0 || i < at) {
				at, kind = i, k
			}
		}
		take(strings.Index(s, "<tool_call>"), 1)
		if d := dsmlInvoke.FindStringSubmatchIndex(s); d != nil {
			take(d[0], 2)
		}
		take(nativeAt(s), 3)
		if p := pipedInvokeTag.FindStringIndex(s); p != nil {
			take(p[0], 4)
		}
		switch kind {
		case 1: // Hermes' block, or DeepSeek's half of it
			left.WriteString(s[:at])
			if t := strings.TrimLeft(s[at+len("<tool_call>"):], " \t\r\n"); t == "" || strings.HasPrefix(t, "<") {
				// no JSON after it: DeepSeek's own tags, an invoke or the
				// closes of one (#823), or nothing; the opener is dropped
				// and what follows read as it is
				s = t
				continue
			}
			c, n, good := paramCall(s[at+len("<tool_call>"):], names)
			if !good {
				c, n, good = hermesCall(s[at+len("<tool_call>"):], names)
			}
			if !good && !strings.HasPrefix(strings.TrimLeft(s[at+len("<tool_call>"):], " \t\r\n"), "{") {
				// GLM's own: the name, then its arguments in pairs (#906)
				c, n, good = glmCall(s[at+len("<tool_call>"):], names)
			}
			if !good {
				return nil, "", false
			}
			calls = append(calls, c)
			s = s[at+len("<tool_call>")+n:]
		case 2: // DeepSeek's DSML
			d := dsmlInvoke.FindStringSubmatchIndex(s)
			left.WriteString(s[:at])
			c, n, good := dsmlCall(s[at:], s[d[2]:d[3]], d[1]-d[0], names)
			if !good {
				return nil, "", false
			}
			calls = append(calls, c)
			s = s[at+n:]
		case 3: // DeepSeek's own markup (#1848)
			left.WriteString(s[:at])
			c, n, good := deepSeekCalls(s[at:], names)
			if !good {
				return nil, "", false
			}
			calls = append(calls, c...)
			s = s[at+n:]
		case 4: // its invoke without the DSML word (#1848)
			left.WriteString(s[:at])
			c, n, good := pipedCall(s[at:], names)
			if !good {
				return nil, "", false
			}
			calls = append(calls, c)
			s = s[at+n:]
		default:
			left.WriteString(s)
			s = ""
		}
	}
	rest = strings.TrimSpace(textCallJunk.ReplaceAllString(left.String(), ""))
	rest = strings.TrimSpace(strings.TrimSuffix(rest, cutMark(rest)))
	// tags and nothing else (#823, Dazzle-sys: <tool_call></｜DSML｜parameter>
	// </｜DSML｜invoke></ was all a reply said) are dropped too: shown to the
	// agent and read back, they had the model write its next calls the same
	// way
	return calls, rest, len(calls) > 0 || rest == ""
}

// tagsOnly reports whether held text is the tags of calls and nothing
// else: no call read out of it, and no text left once they go.
func tagsOnly(s string) bool {
	calls, rest, ok := parseTextCalls(s, nil)
	return ok && len(calls) == 0 && rest == ""
}

// spaceBack keeps s, the text about to be handed on, back with what is
// held or pending when it is only spaces after a call the reply made
// itself (native).
func spaceBack(native, holding bool, pending, held, s string) (string, string, string) {
	if !native || s == "" || strings.TrimSpace(s) != "" {
		return pending, held, s
	}
	if holding {
		return pending, s + held, ""
	}
	return s + pending, held, ""
}

// settled is what is held and pending once the reply ends or makes a
// call: held tags and nothing else go, and so does what is only spaces
// after a call the reply made itself.
func settled(native, holding bool, pending, held string) (string, string, bool) {
	if holding && tagsOnly(held) {
		held, holding = "", false
	}
	if native && strings.TrimSpace(pending+held) == "" {
		pending, held, holding = "", "", false
	}
	return pending, held, holding
}

// cutMark is the end of s that is a mark's beginning cut short (</ of
// </tool_call>), "" when there's none; a lone < may be text.
func cutMark(s string) string {
	for _, m := range append([]string{"</tool_call>", "</parameter>"}, textCallMarks...) {
		for n := len(m) - 1; n >= 2; n-- {
			if strings.HasSuffix(s, m[:n]) {
				return m[:n]
			}
		}
	}
	// a DSML tag's, open or close, however many its pipes; a run of
	// them holds no <, so only the last can be one cut short. It has
	// begun its pipes (a full-width one's first bytes too): "x < " is text
	if i := strings.LastIndexByte(s, '<'); i >= 0 && strings.IndexAny(s[i:], "|｜\xef") >= 0 {
		if _, more := dsmlMark(s[i:], true); more {
			return s[i:]
		}
		if nativeShort(s[i:]) {
			return s[i:]
		}
		if pipedShort(s[i:]) {
			return s[i:]
		}
	}
	return ""
}

// hermesCall reads the JSON object a <tool_call> opens, and says how much
// of s it took.
func hermesCall(s string, names map[string]bool) (writtenCall, int, bool) {
	lead := len(s) - len(strings.TrimLeft(s, " \t\r\n"))
	body := s[lead:]
	if !strings.HasPrefix(body, "{") {
		return writtenCall{}, 0, false
	}
	// a file's content written raw puts real newlines and tabs inside
	// the JSON's strings, which JSON doesn't allow: they are read as the
	// escapes they stand for
	fixed, back := escapeRawControls(body)
	dec := json.NewDecoder(strings.NewReader(fixed))
	dec.UseNumber()
	var v struct {
		Name       string          `json:"name"`
		Arguments  json.RawMessage `json:"arguments"`
		Parameters json.RawMessage `json:"parameters"`
		Input      json.RawMessage `json:"input"`
	}
	if dec.Decode(&v) != nil || !names[v.Name] {
		return writtenCall{}, 0, false
	}
	args := v.Arguments
	if len(args) == 0 {
		args = v.Parameters
	}
	if len(args) == 0 {
		args = v.Input
	}
	args, ok := callArgs(args)
	if !ok {
		return writtenCall{}, 0, false
	}
	n := lead + back(int(dec.InputOffset()))
	// what follows the object up to its close: GLM leaves braces,
	// brackets, quotes or a > of its own there (}}</tool_call>,
	// "></tool_call>, #1945), taken with the close. Other text right
	// after the object, before a close, means the object ended early, at
	// a quote a string's content left unescaped (a heredoc's command cut
	// short, #1945): no call, rather than one cut wrong
	junk := len(s[n:]) - len(strings.TrimLeft(s[n:], " \t\r\n}]\"'>"))
	next := s[n+junk:]
	if strings.HasPrefix(next, "</tool_call>") {
		return writtenCall{Name: v.Name, Args: args}, n + junk + len("</tool_call>"), true
	}
	if strings.TrimLeft(s[n:n+junk], " \t\r\n") != "" || (junk == 0 && next != "" && next[0] != '<') {
		if strings.Contains(next, "</tool_call>") {
			return writtenCall{}, 0, false
		}
	}
	return writtenCall{Name: v.Name, Args: args}, n, true
}

var (
	paramHead = regexp.MustCompile(`^\s*\{\s*"name"\s*:\s*"([^"]+)"\s*\}?\s*>?`)
	paramOpen = regexp.MustCompile(`^\s*<parameter(?:\s+name\s*=\s*"([^"]+)"|=([^\s>]+))\s*>`)
)

// paramCall reads a call DeepSeek wrote half as Hermes' block and half as
// its own template (#917, Moody-Sin: DeepSeek through a group, in Pi): the
// name as JSON, closed with } or > or not at all, then each argument as
// <parameter name="k">v</parameter> (or Qwen's <parameter=k>), the closes
// often missing, as DeepSeek's are tokens of its own the API drops. A
// value runs to its close, the next parameter, </tool_call> or the end;
// one that is JSON is taken as it, else it is the text it says. It says
// how much of s it took.
func paramCall(s string, names map[string]bool) (writtenCall, int, bool) {
	m := paramHead.FindStringSubmatchIndex(s)
	if m == nil || !names[s[m[2]:m[3]]] {
		return writtenCall{}, 0, false
	}
	name, n := s[m[2]:m[3]], m[1]
	args := map[string]any{}
	for {
		p := paramOpen.FindStringSubmatchIndex(s[n:])
		if p == nil {
			break
		}
		var key string
		if p[2] >= 0 {
			key = s[n+p[2] : n+p[3]]
		} else {
			key = s[n+p[4] : n+p[5]]
		}
		n += p[1]
		end, next := len(s), len(s)
		for _, stop := range []string{"</parameter>", "<parameter", "</tool_call>", "<tool_call>"} {
			if i := strings.Index(s[n:], stop); i >= 0 && n+i < end {
				end, next = n+i, n+i
				if stop == "</parameter>" {
					next += len(stop)
				}
			}
		}
		val := strings.Trim(s[n:end], "\r\n")
		val = strings.TrimSuffix(val, cutMark(val))
		var v any
		if decodeJSON([]byte(strings.TrimSpace(val)), &v) == nil {
			args[key] = v
		} else {
			args[key] = val
		}
		n = next
	}
	if len(args) == 0 {
		return writtenCall{}, 0, false
	}
	if t := strings.TrimLeft(s[n:], " \t\r\n"); strings.HasPrefix(t, "</tool_call>") {
		n = len(s) - len(t) + len("</tool_call>")
	}
	b, err := json.Marshal(args)
	if err != nil {
		return writtenCall{}, 0, false
	}
	return writtenCall{Name: name, Args: b}, n, true
}

var (
	glmPair = regexp.MustCompile(`(?s)^\s*<arg_key>(.*?)</arg_key>\s*<arg_value>(.*?)</arg_value>`)
	glmName = regexp.MustCompile(`^\s*([^\s<>{}"]+)\s*`)
)

// glmCall reads GLM's own call (#906, nullburn: GLM-5.3-Flash through
// vLLM wrote them into its text in Codex): the tool's name right after
// <tool_call>, then <arg_key>…</arg_key><arg_value>…</arg_value> for
// each argument, then </tool_call>. A value that is JSON (a number, an
// object, a list, true) is taken as it, as GLM's template writes them;
// else it is the string it says. It says how much of s it took.
func glmCall(s string, names map[string]bool) (writtenCall, int, bool) {
	m := glmName.FindStringSubmatchIndex(s)
	if m == nil || !names[s[m[2]:m[3]]] {
		return writtenCall{}, 0, false
	}
	name, n := s[m[2]:m[3]], m[1]
	args := map[string]any{}
	for {
		p := glmPair.FindStringSubmatchIndex(s[n:])
		if p == nil {
			break
		}
		key, val := strings.TrimSpace(s[n+p[2]:n+p[3]]), s[n+p[4]:n+p[5]]
		var v any
		if decodeJSON([]byte(strings.TrimSpace(val)), &v) == nil {
			args[key] = v
		} else {
			args[key] = val
		}
		n += p[1]
	}
	t := strings.TrimLeft(s[n:], " \t\r\n")
	switch {
	case strings.HasPrefix(t, "</tool_call>"):
		n = len(s) - len(t) + len("</tool_call>")
	case t != "" && !strings.HasPrefix(t, "<tool_call>"):
		// something else follows the name: not a call of GLM's
		return writtenCall{}, 0, false
	}
	b, err := json.Marshal(args)
	if err != nil {
		return writtenCall{}, 0, false
	}
	return writtenCall{Name: name, Args: b}, n, true
}

// dsmlCall reads DeepSeek's <｜DSML｜invoke name="…"> and its parameters,
// up to its close, the next invoke or the end of s.
func dsmlCall(s, name string, open int, names map[string]bool) (writtenCall, int, bool) {
	if !names[name] {
		return writtenCall{}, 0, false
	}
	end := len(s)
	if m := dsmlClose.FindStringIndex(s[open:]); m != nil {
		end = open + m[1]
	}
	if m := dsmlInvoke.FindStringIndex(s[open:]); m != nil && open+m[0] < end {
		end = open + m[0]
	}
	args := map[string]any{}
	for _, p := range dsmlParam.FindAllStringSubmatch(s[open:end], -1) {
		val := p[3]
		if dsmlString.MatchString(p[2]) {
			args[p[1]] = val
			continue
		}
		var v any
		if decodeJSON([]byte(strings.TrimSpace(val)), &v) == nil {
			args[p[1]] = v
		} else {
			args[p[1]] = val
		}
	}
	b, err := json.Marshal(args)
	if err != nil {
		return writtenCall{}, 0, false
	}
	return writtenCall{Name: name, Args: b}, end, true
}

// deepSeekCalls reads the calls DeepSeek wrote in its own markup (#1848),
// s beginning with the calls wrapper or a call's open: each call's name,
// its separation and its arguments, the wrapper's close and each call's
// close taken with them. n is how much of s they took. A call of a name
// the request didn't offer or with arguments that can't be read, and a
// wrapper with no call that can be read, mean the text goes on as it came.
func deepSeekCalls(s string, names map[string]bool) (calls []writtenCall, n int, ok bool) {
	// a close left over with nothing before it goes, as DSML's does
	for _, re := range []*regexp.Regexp{nativeClose, nativeEnd} {
		if m := re.FindStringIndex(s); m != nil && m[0] == 0 {
			return nil, m[1], true
		}
	}
	if m := nativeBegin.FindStringIndex(s); m != nil && m[0] == 0 {
		n = m[1]
	}
	for {
		t := strings.TrimLeft(s[n:], " \t\r\n")
		n += len(s[n:]) - len(t)
		if m := nativeClose.FindStringIndex(s[n:]); m != nil && m[0] == 0 {
			n += m[1]
			break
		}
		m := nativeOpen.FindStringIndex(s[n:])
		if m == nil || m[0] != 0 {
			break
		}
		rest := s[n+m[1]:]
		end := len(rest)
		for _, re := range []*regexp.Regexp{nativeEnd, nativeOpen, nativeClose} {
			if e := re.FindStringIndex(rest); e != nil && e[0] < end {
				end = e[0]
			}
		}
		c, good := nativeCall(rest[:end], names)
		if !good {
			return nil, 0, false
		}
		calls = append(calls, c)
		n += m[1] + end
		if e := nativeEnd.FindStringIndex(s[n:]); e != nil && e[0] == 0 {
			n += e[1]
			continue
		}
		if e := nativeClose.FindStringIndex(s[n:]); e != nil && e[0] == 0 {
			n += e[1]
			break
		}
		if e := nativeOpen.FindStringIndex(s[n:]); e == nil || e[0] != 0 {
			break // cut short, or text the call's close never came before
		}
	}
	if len(calls) == 0 && n == 0 {
		return nil, 0, false
	}
	return calls, n, true
}

// nativeCall reads one call of that markup (#1848): its name, nativeSep,
// then its arguments as JSON. V3 wrote function before the name and fenced
// the JSON, V3.1 the name alone, and both are read.
func nativeCall(body string, names map[string]bool) (writtenCall, bool) {
	name, args := strings.TrimSpace(body), ""
	if m := nativeSepTag.FindStringIndex(body); m != nil {
		name, args = strings.TrimSpace(body[:m[0]]), body[m[1]:]
		if name == "function" {
			// V3 wrote function, its separation, then the name and the
			// arguments: the name ends at the line's end or at the JSON
			cut := len(args)
			for _, s := range []string{"\n", "\r", "{", "```"} {
				if i := strings.Index(args, s); i >= 0 && i < cut {
					cut = i
				}
			}
			name, args = strings.TrimSpace(args[:cut]), args[cut:]
		}
	}
	if !names[name] {
		return writtenCall{}, false
	}
	args = strings.TrimSpace(args)
	if rest, fenced := strings.CutPrefix(args, "```"); fenced {
		if i := strings.IndexByte(rest, '\n'); i >= 0 && strings.TrimSpace(rest[:i]) == "json" {
			rest = rest[i+1:]
		}
		args = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(rest), "```"))
	}
	b, ok := callArgs([]byte(args))
	if !ok {
		return writtenCall{}, false
	}
	return writtenCall{Name: name, Args: b}, true
}

// The issue's second shape (#1848): the invoke of DeepSeek's own DSML
// template with its DSML word gone, and its parameters. The report holds
// the opening tag alone, so they are read as that template writes them,
// and a body that is the JSON its other calls carry is read too.
var (
	pipedInvokeTag = regexp.MustCompile(`<\s*[|｜][\s|｜]*invoke\s+name\s*=\s*"([^"]+)"[\s|｜]*>`)
	pipedParamTag  = regexp.MustCompile(`(?s)<\s*[|｜][\s|｜]*parameter\s+name\s*=\s*"([^"]+)"([^<>]*)[\s|｜]*>(.*?)<\s*/\s*[|｜][\s|｜]*parameter[\s|｜]*>`)
	pipedCloseTag  = regexp.MustCompile(`<\s*/\s*[|｜][\s|｜]*invoke[\s|｜]*>`)
)

// pipedMarkAt is the length of that invoke's open at the start of s, 0 when
// it isn't there; pipedShort tells whether s is one's beginning.
func pipedMarkAt(s string) int {
	if m := pipedInvokeTag.FindStringIndex(s); m != nil && m[0] == 0 {
		return m[1]
	}
	return 0
}

func pipedShort(s string) bool {
	if !strings.HasPrefix(s, "<") {
		return false
	}
	rest, more := tagRest(s)
	if more || rest == "" {
		return true
	}
	if r, ok := strings.CutPrefix(rest, pipedInvoke); ok {
		// the word is there: its name's attribute and its close may still
		// be on the way
		r = strings.TrimLeft(r, " \t")
		return strings.HasPrefix(r, `name="`) || strings.HasPrefix(`name="`, r)
	}
	return strings.HasPrefix(pipedInvoke, rest)
}

// pipedCall reads a call of the issue's second shape: its name, its
// parameters and its close, up to that close or the end of s.
func pipedCall(s string, names map[string]bool) (writtenCall, int, bool) {
	m := pipedInvokeTag.FindStringSubmatchIndex(s)
	if m == nil || m[0] != 0 || !names[s[m[2]:m[3]]] {
		return writtenCall{}, 0, false
	}
	name, open := s[m[2]:m[3]], m[1]
	end := len(s)
	if i := pipedCloseTag.FindStringIndex(s[open:]); i != nil {
		end = open + i[1]
	}
	if i := pipedInvokeTag.FindStringIndex(s[open:]); i != nil && open+i[0] < end {
		end = open + i[0]
	}
	args := map[string]any{}
	for _, p := range pipedParamTag.FindAllStringSubmatch(s[open:end], -1) {
		val := p[3]
		if dsmlString.MatchString(p[2]) {
			args[p[1]] = val
			continue
		}
		var v any
		if decodeJSON([]byte(strings.TrimSpace(val)), &v) == nil {
			args[p[1]] = v
		} else {
			args[p[1]] = val
		}
	}
	if body := strings.TrimSpace(s[open:end]); len(args) == 0 && body != "" {
		b, ok := callArgs([]byte(body))
		if !ok {
			return writtenCall{}, 0, false // arguments that can't be read
		}
		return writtenCall{Name: name, Args: b}, end, true
	}
	b, err := json.Marshal(args)
	if err != nil {
		return writtenCall{}, 0, false
	}
	return writtenCall{Name: name, Args: b}, end, true
}

// callArgs is a call's arguments as a JSON object: one given as a string
// of JSON is read, none is {}.
func callArgs(raw json.RawMessage) (json.RawMessage, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("{}"), true
	}
	if raw[0] == '"' {
		var s string
		if decodeJSON(raw, &s) != nil {
			return nil, false
		}
		fixed, _ := escapeRawControls(strings.TrimSpace(s))
		raw = json.RawMessage(fixed)
	}
	var m map[string]any
	if decodeJSON(raw, &m) != nil {
		return nil, false
	}
	b, err := json.Marshal(m)
	return b, err == nil
}

// escapeRawControls escapes the newlines, carriage returns and tabs inside
// the strings of JSON text; back maps an offset in what it gives to one in
// s.
func escapeRawControls(s string) (string, func(int) int) {
	var b strings.Builder
	var grew []int // offsets in the output after which it is one byte longer
	in, esc := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case esc:
			esc = false
		case in && c == '\\':
			esc = true
		case c == '"':
			in = !in
		case in && (c == '\n' || c == '\r' || c == '\t'):
			b.WriteString(map[byte]string{'\n': `\n`, '\r': `\r`, '\t': `\t`}[c])
			grew = append(grew, b.Len())
			continue
		}
		b.WriteByte(c)
	}
	return b.String(), func(n int) int {
		for _, g := range grew {
			if g <= n {
				n--
			}
		}
		return n
	}
}

var textCallSeq atomic.Int64

func textCallID() string { return fmt.Sprintf("call_text_%d", textCallSeq.Add(1)) }

// textCallSee holds, in a translated reply, the text from where a call
// written into it begins, and at the reply's end hands see the calls it
// held, or the text when it held no calls.
type textCallSee struct {
	names   map[string]bool
	see     func(Event)
	pending string // the text's end, which may be a mark begun
	held    string
	holding bool
	// the reply made a call itself: no call is read out of its text, but
	// tags left in it (an upstream that made calls of DeepSeek's invokes
	// left the wrapper's </｜｜DSML｜｜ calls>) are still dropped
	native bool
}

func (t *textCallSee) event(ev Event) {
	switch ev.Kind {
	case KText:
		s := t.pending + ev.Text
		t.pending = ""
		if t.holding {
			t.held += s
			return
		}
		at, tail := textCallAt(s)
		if at >= 0 {
			t.holding, t.held, s = true, s[at:], s[:at]
		} else {
			t.pending, s = s[len(s)-tail:], s[:len(s)-tail]
		}
		s = t.keepSpace(s)
		if s != "" {
			t.see(Event{Kind: KText, Text: s})
		}
		return
	case KToolStart:
		t.settle()
		t.native = true
	case KStop:
		if t.native {
			t.settle()
		}
		if t.holding && !t.native {
			if calls, rest, ok := parseTextCalls(t.held, t.names); ok {
				t.holding, t.held = false, ""
				if rest != "" {
					t.see(Event{Kind: KText, Text: rest})
				}
				for _, c := range calls {
					t.see(Event{Kind: KToolStart, ID: textCallID(), Name: c.Name})
					t.see(Event{Kind: KToolArgs, Text: string(c.Args)})
				}
				if len(calls) > 0 {
					ev.Stop = "tool"
				}
			}
		}
		t.release()
	case KError:
		t.release()
	}
	t.see(ev)
}

// keepSpace keeps back text that is only spaces after a call the reply
// made itself: the line break before tags left after the calls goes with
// them. What it gives is the text to hand on now.
func (t *textCallSee) keepSpace(s string) string {
	t.pending, t.held, s = spaceBack(t.native, t.holding, t.pending, t.held, s)
	return s
}

// settle drops what is held when it is tags and nothing else, or after a
// call the reply made only spaces, and hands it on as text when it isn't.
func (t *textCallSee) settle() {
	t.pending, t.held, t.holding = settled(t.native, t.holding, t.pending, t.held)
	t.release()
}

// release hands on as text what is held.
func (t *textCallSee) release() {
	if s := t.pending + t.held; s != "" {
		t.see(Event{Kind: KText, Text: s})
	}
	t.pending, t.held, t.holding = "", "", false
}

// textCallTidy does the same in a relayed Chat Completions stream.
type textCallTidy struct {
	names   map[string]bool
	buf     []byte
	pending string
	held    string
	holding bool
	native  bool
	last    map[string]any // the last chunk read, for the ones made
}

func (t *textCallTidy) write(b []byte) []byte {
	t.buf = append(t.buf, b...)
	i := bytes.LastIndexByte(t.buf, '\n')
	if i < 0 {
		return nil
	}
	var out []byte
	for _, line := range bytes.SplitAfter(t.buf[:i+1], []byte("\n")) {
		out = append(out, t.line(line)...)
	}
	t.buf = append(t.buf[:0], t.buf[i+1:]...)
	return out
}

func (t *textCallTidy) flush() []byte {
	var out []byte
	if len(t.buf) > 0 {
		out = t.line(t.buf)
		t.buf = nil
	}
	return append(out, t.release()...)
}

// chunk is a Chat chunk like the last one read, with delta and finish.
func (t *textCallTidy) chunk(delta map[string]any, finish any) []byte {
	c := map[string]any{"object": "chat.completion.chunk"}
	for _, k := range []string{"id", "object", "created", "model"} {
		if v, ok := t.last[k]; ok {
			c[k] = v
		}
	}
	c["choices"] = []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}
	b, _ := marshalPlain(c)
	return append(append([]byte("data: "), b...), '\n', '\n')
}

// settle is release, but what is held goes as textCallSee's settle has
// it go.
func (t *textCallTidy) settle() []byte {
	t.pending, t.held, t.holding = settled(t.native, t.holding, t.pending, t.held)
	return t.release()
}

func (t *textCallTidy) release() []byte {
	s := t.pending + t.held
	t.pending, t.held, t.holding = "", "", false
	if s == "" {
		return nil
	}
	return t.chunk(map[string]any{"content": s}, nil)
}

func (t *textCallTidy) line(line []byte) []byte {
	body := bytes.TrimRight(line, "\r\n")
	data, ok := bytes.CutPrefix(body, []byte("data:"))
	if !ok {
		return line
	}
	data = bytes.TrimSpace(data)
	if string(data) == "[DONE]" {
		return append(t.release(), line...)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var ev map[string]any
	if dec.Decode(&ev) != nil || ev == nil {
		return line
	}
	t.last = ev
	choices, _ := ev["choices"].([]any)
	if len(choices) != 1 {
		return line
	}
	ch, _ := choices[0].(map[string]any)
	delta, _ := ch["delta"].(map[string]any)
	if ch == nil {
		return line
	}
	var pre []byte
	changed := false
	if calls, _ := delta["tool_calls"].([]any); len(calls) > 0 && !t.native {
		pre, t.native = t.settle(), true
	}
	if s, isStr := delta["content"].(string); isStr && s != "" {
		s = t.pending + s
		t.pending = ""
		if t.holding {
			t.held += s
			s = ""
		} else if at, tail := textCallAt(s); at >= 0 {
			t.holding, t.held, s = true, s[at:], s[:at]
		} else {
			t.pending, s = s[len(s)-tail:], s[:len(s)-tail]
		}
		t.pending, t.held, s = spaceBack(t.native, t.holding, t.pending, t.held, s)
		if s == "" {
			delete(delta, "content")
		} else {
			delta["content"] = s
		}
		changed = true
	}
	if f, _ := ch["finish_reason"].(string); f != "" && (t.holding || t.pending != "") {
		if calls, rest, ok := parseTextCalls(t.held, t.names); t.holding && !t.native && ok {
			t.holding, t.held = false, ""
			if rest != "" {
				pre = append(pre, t.chunk(map[string]any{"content": rest}, nil)...)
			}
			tcs := make([]any, len(calls))
			for i, c := range calls {
				tcs[i] = map[string]any{"index": i, "id": textCallID(), "type": "function",
					"function": map[string]any{"name": c.Name, "arguments": string(c.Args)}}
			}
			pre = append(pre, t.release()...)
			if len(tcs) > 0 {
				pre = append(pre, t.chunk(map[string]any{"tool_calls": tcs}, nil)...)
				ch["finish_reason"] = "tool_calls"
				changed = true
			}
		} else {
			pre = append(pre, t.settle()...)
		}
	}
	if !changed {
		return append(pre, line...)
	}
	if len(delta) == 0 && ch["finish_reason"] == nil && ev["usage"] == nil {
		return pre // nothing left to say
	}
	nb, err := marshalPlain(ev)
	if err != nil {
		return append(pre, line...)
	}
	return append(append(append(pre, "data: "...), nb...), line[len(body):]...)
}

// mayWriteCalls tells whether a whole Chat reply b could carry a call
// written into its text, for a request that offered names: one of
// textCallMarks' words is in it, however its JSON escapes the brackets,
// or DeepSeek's own markup (#1848), whose ▁ a relay may escape as \u2581,
// or the invoke it writes without its DSML word.
func mayWriteCalls(b []byte, names map[string]bool) bool {
	return names != nil && (bytes.Contains(b, []byte("tool_call")) || bytes.Contains(b, []byte("DSML")) ||
		bytes.Contains(b, []byte("▁")) || bytes.Contains(b, []byte(`\u2581`)) || bytes.Contains(b, []byte("invoke")))
}

// wholeTextCalls reads the calls written into the content of a whole Chat
// reply's message, as textCallTidy reads a stream's: the text before the
// first is kept, the blocks become tool_calls and what is left of their
// tags goes. wrote is whether msg changed, called whether it now calls
// tools. A message that made a call itself, or whose written blocks
// aren't all calls of names with arguments that can be read, is left as
// it came.
func wholeTextCalls(msg map[string]json.RawMessage, names map[string]bool) (wrote, called bool) {
	if names == nil {
		return false, false
	}
	var native []json.RawMessage
	made := decodeJSON(msg["tool_calls"], &native) == nil && len(native) > 0
	var s string
	if decodeJSON(msg["content"], &s) != nil {
		return false, false
	}
	at, _ := textCallAt(s)
	if at < 0 {
		return false, false
	}
	if made {
		// the upstream made the calls itself: only tags left in its
		// text go (</｜｜DSML｜｜ calls>, 喜欢一个人 on Discord)
		if !tagsOnly(s[at:]) {
			return false, false
		}
		if text := strings.TrimRight(s[:at], " \t\r\n"); text != "" {
			msg["content"], _ = marshalPlain(text)
		} else {
			msg["content"] = json.RawMessage("null")
		}
		return true, true
	}
	calls, rest, ok := parseTextCalls(s[at:], names)
	if !ok {
		return false, false
	}
	text := s[:at] + rest
	if len(calls) == 0 {
		// tags and nothing else, dropped as a stream drops them
		msg["content"], _ = marshalPlain(text)
		return true, false
	}
	tcs := make([]map[string]any, len(calls))
	for i, c := range calls {
		tcs[i] = map[string]any{"id": textCallID(), "type": "function",
			"function": map[string]any{"name": c.Name, "arguments": string(c.Args)}}
	}
	if text != "" {
		msg["content"], _ = marshalPlain(text)
	} else {
		msg["content"] = json.RawMessage("null")
	}
	msg["tool_calls"], _ = marshalPlain(tcs)
	return true, true
}
