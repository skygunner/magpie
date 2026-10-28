package sessions

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// MiniMax Code's sessions, on the Sessions page (#2301, LukeZ1988: its
// sessions were missing there while its calls already reached Usage). Each
// session is a folder under v2/sessions, as calls_minimax.go reads its
// messages from for Usage: a user message is what was typed, an assistant
// message names the model that answered and its usage. The folder holds
// neither the folder the session ran in nor a resume command, so a row is
// listed with what it spent and nothing is guessed in their place.
//
// minimaxSessionFiles is in allFiles, and parseMiniMax in parserFor.

// minimaxSessionFiles are each session's messages.jsonl, the generation
// MiniMax Code writes now. The other generations minimaxCallFiles reads are
// not counted here: a reply copied into one of them would be counted twice.
func minimaxSessionFiles() []file { return minimaxMessagesFiles(false) }

// minimaxMessagesFiles are the sessions' messages files: every generation of
// them (all), or only messages.jsonl.
func minimaxMessagesFiles(all bool) []file {
	var out []file
	walkLinked(minimaxSessions(), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "messages") || !strings.HasSuffix(name, ".jsonl") || (!all && name != "messages.jsonl") {
			return nil
		}
		f := file{agent: "minimax-code", key: "minimax-code:" + minimaxSessionID(filepath.Base(filepath.Dir(p))), path: p, main: true}
		if stat(&f) {
			out = append(out, f)
		}
		return nil
	})
	return out
}

// mcodeMessage is a line of a session's messages file, as its listing reads
// it: the call reader's mcodeLine leaves out the content it would copy.
type mcodeMessage struct {
	Message *struct {
		Role      string          `json:"role"`
		Model     string          `json:"model"`
		Content   json.RawMessage `json:"content"`
		Usage     *piUsage        `json:"usage"`
		Timestamp int64           `json:"timestamp"`
	} `json:"message"`
}

// parseMiniMax reads a session's messages whole: a later generation of its
// history replaces the file, so there is no tail to read on from.
func parseMiniMax(f file) *state {
	s := &state{}
	scanAt(f.path, 0, nil, func(b []byte, _, _ int64) bool {
		mcodeCount(s, b, f.main)
		return true
	})
	s.Size, s.Mod = f.size, f.mod.UnixNano()
	return s
}

// mcodeCount counts one line of a session's messages file: what was typed, a
// reply and its usage, and the times they were written.
func mcodeCount(s *state, b []byte, main bool) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || len(b) > maxLine {
		return
	}
	var l mcodeMessage
	if json.Unmarshal(b, &l) != nil || l.Message == nil {
		return
	}
	m := l.Message
	at := ms(m.Timestamp)
	s.saw(at, main)
	switch m.Role {
	case "user":
		prompt := mcodePrompt(ccText(m.Content))
		if prompt == "" {
			return // a tool's result, not something typed
		}
		if main {
			s.day(dateOf(at)).Prompts++
		}
		if main && s.First == "" {
			s.First = untagged(prompt)
		}
	case "assistant":
		if main {
			s.day(dateOf(at)).Replies++
		}
		if m.Model != "" {
			s.use(dateOf(at), m.Model, m.Usage.tokens())
		}
	}
}

// mcodePrompt is what someone typed: MiniMax Code puts its own reminders
// ahead of it in the same text, inside <system-reminder>.
func mcodePrompt(text string) string {
	if i := strings.LastIndex(text, "</system-reminder>"); i >= 0 {
		return strings.TrimSpace(text[i+len("</system-reminder>"):])
	}
	return strings.TrimSpace(text)
}
