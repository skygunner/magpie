package gateway

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

// #1848's marks, as the issue writes them: over DeepSeek's own API
// (deepseek-flash), the model wrote its call into the text in its native
// special-token markup, <｜tool▁calls▁begin｜> … or <｜invoke name="Bash"｜>
// …, and neither rescue knew them, so the client got the markup as text
// with finish_reason stop and the turn ran nothing.
const (
	// DeepSeek V3.1's call, as its template writes one and vLLM's
	// deepseekv31 parser reads it: the wrapper, then the name, then
	// <｜tool▁sep｜> and the arguments as JSON.
	issueNative = "<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>Bash<｜tool▁sep｜>{\"command\":\"ls /tmp\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>"
	// V3's, which wrote function<｜tool▁sep｜> before the name and fenced
	// the JSON.
	issueNativeV3 = "<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>function<｜tool▁sep｜>Bash\n```json\n{\"command\":\"ls /tmp\"}\n```<｜tool▁call▁end｜><｜tool▁calls▁end｜>"
	// The issue's second shape: the invoke without the DSML word in it.
	// The issue records its opening tag alone; the parameters are the
	// ones DSML's invoke carries, which is the template the model
	// dropped the word from.
	issuePipedInvoke = "<｜invoke name=\"Bash\"｜>\n<｜parameter name=\"command\" string=\"true\">ls /tmp</｜parameter>\n</｜invoke｜>"
)

// #1848: a call written into the text in DeepSeek's own markup, in either
// of the two shapes the issue names, is made the call it is in a whole
// (stream:false) reply, as the DSML it also writes already is.
func TestDeepSeekNativeCallWrittenWhole(t *testing.T) {
	ask := `{"model":"a/m","stream":false,"messages":[{"role":"user","content":"ls /tmp"}],` + issueTools + `}`
	for _, c := range []struct {
		name, up string
		text     string // "<null>" for none
		args     string
	}{
		{"V3.1's native tags", wholeChatUp(issueNative), "<null>", `{"command":"ls /tmp"}`},
		{"V3's", wholeChatUp(issueNativeV3), "<null>", `{"command":"ls /tmp"}`},
		{"after text", wholeChatUp("看一下。\n\n" + issueNative), "看一下。\n\n", `{"command":"ls /tmp"}`},
		{"the invoke without DSML", wholeChatUp(issuePipedInvoke), "<null>", `{"command":"ls /tmp"}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			text, calls, finish, raw := wholeChat(t, ask, "application/json", c.up)
			var want, got any
			json.Unmarshal([]byte(c.args), &want)
			if len(calls) == 1 {
				json.Unmarshal([]byte(calls[0].Function.Arguments), &got)
			}
			if strOr(text) != c.text || len(calls) != 1 || calls[0].Function.Name != "Bash" ||
				got == nil || !jsonEqual(got, want) || finish != "tool_calls" {
				t.Fatalf("text %q calls %+v finish %q\n%s", strOr(text), calls, finish, raw)
			}
			if strings.Contains(raw, "tool▁") || strings.Contains(raw, "invoke") {
				t.Fatalf("markup left in the reply: %s", raw)
			}
		})
	}
}

// A streamed #1848 reply: the tags arrive split across chunks, mid-tag,
// and the call is still the call it is, after the text before it.
func TestDeepSeekNativeCallStreamed(t *testing.T) {
	body := `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"hi"}],` + textCallTools + `}`
	text, calls, finish, raw := relayedChat(t, body,
		chunkOf("我先看一下。\n\n<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>bash<｜tool▁sep｜>{\"comm"),
		chunkOf("and\":\"ls -la\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>"),
		stopChunk)
	if text != "我先看一下。\n\n" || calls["bash"] != `{"command":"ls -la"}` || finish != "tool_calls" || strings.Contains(raw, "tool▁") {
		t.Fatalf("text %q calls %v finish %q\n%s", text, calls, finish, raw)
	}
	// cut anywhere, its parts as chunks of their own, it is the same call
	issueBody := `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"hi"}],` + issueTools + `}`
	s := "Sure.\n" + issueNative
	for i := 1; i < len(s); i++ {
		if !utf8.ValidString(s[:i]) {
			continue // a chunk's JSON carries whole characters only
		}
		text, calls, finish, raw := relayedChat(t, issueBody, chunkOf(s[:i]), chunkOf(s[i:]), stopChunk)
		if text != "Sure.\n" || calls["Bash"] != `{"command":"ls /tmp"}` || finish != "tool_calls" {
			t.Fatalf("cut at %d: text %q calls %v finish %q\n%s", i, text, calls, finish, raw)
		}
	}
	// and the issue's second shape, the same way
	s = "Sure.\n" + issuePipedInvoke
	for i := 1; i < len(s); i++ {
		if !utf8.ValidString(s[:i]) {
			continue
		}
		text, calls, finish, raw := relayedChat(t, issueBody, chunkOf(s[:i]), chunkOf(s[i:]), stopChunk)
		if text != "Sure.\n" || calls["Bash"] != `{"command":"ls /tmp"}` || finish != "tool_calls" {
			t.Fatalf("invoke cut at %d: text %q calls %v finish %q\n%s", i, text, calls, finish, raw)
		}
	}
}

// A reply cut short after the tags began: what it wrote is read, and what
// can't be read leaves the text as it came.
func TestDeepSeekNativeCallCutShort(t *testing.T) {
	// the wrapper and the call's opener, then arguments only partly
	// written: the JSON doesn't parse, so the text goes on as it came
	if text, calls, finish, raw := relayedChat(t, `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"hi"}],`+textCallTools+`}`,
		chunkOf("<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>bash<｜tool▁sep｜>{\"command\":\"ls -l"),
		stopChunk); text != "<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>bash<｜tool▁sep｜>{\"command\":\"ls -l" || len(calls) != 0 || finish != "stop" {
		t.Fatalf("text %q calls %v finish %q\n%s", text, calls, finish, raw)
	}
	// names a tool the request didn't offer: text, as it came
	if text, calls, finish, raw := relayedChat(t, `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"hi"}],`+textCallTools+`}`,
		chunkOf("<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>WebSearch<｜tool▁sep｜>{\"q\":\"x\"}<｜tool▁call▁end｜><｜tool▁calls▁end｜>"),
		stopChunk); !strings.Contains(text, "WebSearch") || len(calls) != 0 || finish != "stop" {
		t.Fatalf("text %q calls %v finish %q\n%s", text, calls, finish, raw)
	}
	// the wrapper's tags alone, with no call in them, go as DSML's do
	if text, calls, finish, raw := relayedChat(t, `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"hi"}],`+textCallTools+`}`,
		chunkOf("看一下。\n\n<｜tool▁calls▁begin｜><｜tool▁calls▁end｜>"),
		stopChunk); text != "看一下。\n\n" || len(calls) != 0 || finish != "stop" {
		t.Fatalf("text %q calls %v finish %q\n%s", text, calls, finish, raw)
	}
	// the second shape with arguments said in a way neither reads: text,
	// not a call with none
	if text, calls, finish, raw := relayedChat(t, `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"hi"}],`+textCallTools+`}`,
		chunkOf("<｜invoke name=\"bash\"｜>ls -la</｜invoke｜>"),
		stopChunk); !strings.Contains(text, "ls -la") || len(calls) != 0 || finish != "stop" {
		t.Fatalf("text %q calls %v finish %q\n%s", text, calls, finish, raw)
	}
}
