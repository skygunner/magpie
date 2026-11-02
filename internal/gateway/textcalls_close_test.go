package gateway

import (
	"encoding/json"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/yetone/magpie/internal/provider"
)

// 喜欢一个人 on Discord (v0.1.1299, a Responses client, deepseek-v4.1-flash
// through a relay): the client got the text "\n</｜｜DSML｜｜ calls>",
// DeepSeek's calls wrapper closed as it writes it (no tool_, a space after
// the pipes), and nothing went on. These are the reporter's bytes: the
// close as it came, and the invokes and parameters with two pipes as their
// client's history has them.
const reportedClose = "\n</｜｜DSML｜｜ calls>"

const reportedBlock = "<｜｜DSML｜｜tool_calls>\n<｜｜DSML｜｜invoke name=\"exec_command\">\n<｜｜DSML｜｜parameter name=\"cmd\" string=\"true\">ls -la</｜｜DSML｜｜parameter>\n</｜｜DSML｜｜invoke>" + reportedClose

const execChatTool = `"tools":[{"type":"function","function":{"name":"exec_command","parameters":{"type":"object","properties":{"cmd":{"type":"string"}}}}}]`

const execResponsesTool = `"tools":[{"type":"function","name":"exec_command","parameters":{"type":"object","properties":{"cmd":{"type":"string"}}}}]`

// nativeExec is a chunk in which the upstream made the call itself.
const nativeExec = `data: {"id":"c1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_up","type":"function","function":{"name":"exec_command","arguments":"{\"cmd\":\"ls -la\"}"}}]},"finish_reason":null}]}`

// cuts are s sent as two chunks at every place a chunk's JSON can cut it.
func cuts(s string) [][]string {
	var out [][]string
	for i := 1; i < len(s); i++ {
		if utf8.ValidString(s[:i]) {
			out = append(out, []string{chunkOf(s[:i]), chunkOf(s[i:])})
		}
	}
	return out
}

// A relay that made the call of DeepSeek's invokes itself left the
// wrapper's close in the text after it: the close goes, with the line
// break before it, on every route and wherever the stream cuts it.
func TestDSMLCloseLeftAfterCalls(t *testing.T) {
	chat := `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"go"}],` + execChatTool + `}`
	for _, before := range []string{"", "<｜｜DSML｜｜ calls>\n", "<｜｜DSML｜｜tool_calls>"} {
		for _, parts := range cuts(reportedClose) {
			up := []string{nativeExec}
			if before != "" {
				up = []string{chunkOf(before), nativeExec}
			}
			text, calls, _, raw := relayedChat(t, chat, append(append(up, parts...), stopChunk)...)
			if text != "" || calls["exec_command"] != `{"cmd":"ls -la"}` || strings.Contains(raw, "DSML") {
				t.Fatalf("chat, %q before, %q: text %q calls %v\n%s", before, parts, text, calls, raw)
			}
		}
	}

	whole := `{"id":"c1","object":"chat.completion","created":1791370000,"model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"Running it.` + strings.ReplaceAll(reportedClose, "\n", `\n`) + `","tool_calls":[{"id":"call_up","type":"function","function":{"name":"exec_command","arguments":"{\"cmd\":\"ls -la\"}"}}]},"finish_reason":"tool_calls"}]}`
	text, wcalls, finish, raw := wholeChat(t, `{"model":"a/m","messages":[{"role":"user","content":"go"}],`+execChatTool+`}`, "application/json", whole)
	if strOr(text) != "Running it." || len(wcalls) != 1 || finish != "tool_calls" || strings.Contains(raw, "DSML") {
		t.Fatalf("chat whole: text %q calls %+v finish %q\n%s", strOr(text), wcalls, finish, raw)
	}

	for _, stream := range []string{"true", "false"} {
		items, raw := responsesItems(t, `{"model":"a/m","stream":`+stream+`,"input":"go",`+execResponsesTool+`}`,
			nativeExec, chunkOf(reportedClose[:6]), chunkOf(reportedClose[6:]), stopChunk)
		if len(items) != 1 || items[0]["type"] != "function_call" || items[0]["arguments"] != `{"cmd":"ls -la"}` || strings.Contains(raw, "DSML") {
			t.Fatalf("responses stream=%s: %v\n%s", stream, items, raw)
		}
	}

	fresh(t)
	a := &scripted{}
	scriptedOn(t, "a", provider.Chat, a)
	for _, stream := range []string{"true", "false"} {
		a.replies = []reply{{200, "text/event-stream", sse(nativeExec, chunkOf(reportedClose), stopChunk, `data: [DONE]`)}}
		rec := httptest.NewRecorder()
		New().Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/messages", strings.NewReader(
			`{"model":"a/m","max_tokens":100,"stream":`+stream+`,"messages":[{"role":"user","content":"go"}],"tools":[{"name":"exec_command","input_schema":{"type":"object"}}]}`)))
		body := rec.Body.String()
		if rec.Code != 200 || strings.Contains(body, "DSML") || strings.Contains(body, `"type":"text"`) ||
			!strings.Contains(body, `"tool_use"`) || !strings.Contains(body, `"stop_reason":"tool_use"`) {
			t.Fatalf("anthropic stream=%s: %d %s", stream, rec.Code, body)
		}
	}
}

// The whole block, closed as the reporter's was, is the call on every
// route, streamed at every cut and whole, opened with tool_calls, with
// " calls" or not at all.
func TestDSMLBlockClosedWithCalls(t *testing.T) {
	chat := `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"go"}],` + execChatTool + `}`
	for _, block := range []string{
		reportedBlock,
		strings.Replace(reportedBlock, "tool_calls>", " calls>", 1),
		reportedBlock[strings.Index(reportedBlock, "\n")+1:],
	} {
		s := "Let me look.\n\n" + block
		for _, parts := range cuts(s) {
			text, calls, finish, raw := relayedChat(t, chat, append(parts, stopChunk)...)
			if text != "Let me look.\n\n" || calls["exec_command"] != `{"cmd":"ls -la"}` || finish != "tool_calls" || strings.Contains(raw, "DSML") {
				t.Fatalf("chat %q: text %q calls %v finish %q\n%s", parts, text, calls, finish, raw)
			}
		}
		text, wcalls, finish, raw := wholeChat(t, `{"model":"a/m","messages":[{"role":"user","content":"go"}],`+execChatTool+`}`, "application/json", wholeChatUp(s))
		if strOr(text) != "Let me look.\n\n" || len(wcalls) != 1 || finish != "tool_calls" || strings.Contains(raw, "DSML") {
			t.Fatalf("chat whole: text %q calls %+v finish %q\n%s", strOr(text), wcalls, finish, raw)
		}
		for _, stream := range []string{"true", "false"} {
			for i := 1; i < len(s); i += 5 {
				if !utf8.ValidString(s[:i]) {
					continue
				}
				items, raw := responsesItems(t, `{"model":"a/m","stream":`+stream+`,"input":"go",`+execResponsesTool+`}`, chunkOf(s[:i]), chunkOf(s[i:]), stopChunk)
				var call map[string]any
				for _, it := range items {
					if it["type"] == "function_call" {
						call = it
					}
				}
				if call == nil || call["arguments"] != `{"cmd":"ls -la"}` || strings.Contains(raw, "DSML") {
					t.Fatalf("responses stream=%s cut %d: %v\n%s", stream, i, items, raw)
				}
			}
		}
	}
}

// A DSML close with nothing written before it and no call is dropped as
// other tags and nothing else are; text after it keeps it, as written.
func TestDSMLCloseAlone(t *testing.T) {
	chat := `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"go"}],` + execChatTool + `}`
	for _, parts := range cuts("Done." + reportedClose) {
		text, calls, finish, raw := relayedChat(t, chat, append(parts, stopChunk)...)
		if text != "Done.\n" || len(calls) != 0 || finish != "stop" || strings.Contains(raw, "DSML") {
			t.Fatalf("%q: text %q calls %v finish %q\n%s", parts, text, calls, finish, raw)
		}
	}
	text, _, finish, raw := wholeChat(t, `{"model":"a/m","messages":[{"role":"user","content":"go"}],`+execChatTool+`}`, "application/json", wholeChatUp("Done."+reportedClose))
	if strOr(text) != "Done.\n" || finish != "stop" || strings.Contains(raw, "DSML") {
		t.Fatalf("whole: text %q finish %q\n%s", strOr(text), finish, raw)
	}
	for _, stream := range []string{"true", "false"} {
		items, raw := responsesItems(t, `{"model":"a/m","stream":`+stream+`,"input":"go",`+execResponsesTool+`}`, chunkOf("Done."+reportedClose), stopChunk)
		if len(items) != 1 || items[0]["type"] != "message" || strings.Contains(raw, "DSML") {
			t.Fatalf("responses stream=%s: %v\n%s", stream, items, raw)
		}
	}
	// prose about a close: the text is kept whole
	prose := "The close is </｜DSML｜invoke> and it ends a call."
	text2, calls, finish, _ := relayedChat(t, chat, chunkOf(prose), stopChunk)
	if text2 != prose || len(calls) != 0 || finish != "stop" {
		t.Fatalf("prose: text %q calls %v finish %q", text2, calls, finish)
	}
	// and after a call the upstream made, too
	text2, _, _, _ = relayedChat(t, chat, nativeExec, chunkOf(prose), stopChunk)
	if text2 != prose {
		t.Fatalf("prose after a call: text %q", text2)
	}
}

// blockMarks is the (event, index) of every content-block start and stop
// a Messages client read, as #4874's fixture reads them.
func blockMarks(t *testing.T, raw string) [][2]string {
	t.Helper()
	var seen [][2]string
	for _, line := range strings.Split(raw, "\n") {
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		var ev struct {
			Type  string `json:"type"`
			Index int    `json:"index"`
		}
		if json.Unmarshal([]byte(data), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "content_block_start", "content_block_stop":
			seen = append(seen, [2]string{ev.Type, strconv.Itoa(ev.Index)})
		}
	}
	return seen
}

// A Messages reply whose empty text block is overlapped by a tool block —
// the tool starts before the text block stops — keeps the text block's
// held start, so the client never gets a stop for a block it never saw
// start (#4874: an isolated Claude Code replay lost the first tool call of
// the reply for it). The fixture sends whole frames, and an ordered
// control keeps both events too.
func TestOverlapKeepsTheHeldTextStart(t *testing.T) {
	fresh(t)
	a := &scripted{}
	scriptedOn(t, "a", provider.Anthropic, a)
	ev := func(typ string, v any) string {
		b, _ := json.Marshal(v)
		return "event: " + typ + "\ndata: " + string(b)
	}
	head := []string{
		ev("message_start", map[string]any{"type": "message_start", "message": map[string]any{"id": "msg_fixture", "type": "message", "role": "assistant", "model": "m", "content": []any{}, "usage": map[string]any{"input_tokens": 1, "output_tokens": 0}}}),
		ev("content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "thinking", "thinking": "", "signature": ""}}),
		ev("content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "thinking_delta", "thinking": "fixture"}}),
		ev("content_block_stop", map[string]any{"type": "content_block_stop", "index": 0}),
		ev("content_block_start", map[string]any{"type": "content_block_start", "index": 1, "content_block": map[string]any{"type": "text", "text": ""}}),
	}
	stop1 := ev("content_block_stop", map[string]any{"type": "content_block_stop", "index": 1})
	tail := []string{
		ev("content_block_start", map[string]any{"type": "content_block_start", "index": 2, "content_block": map[string]any{"type": "tool_use", "id": "toolu_fixture", "name": "echo", "input": map[string]any{}}}),
		ev("content_block_delta", map[string]any{"type": "content_block_delta", "index": 2, "delta": map[string]any{"type": "input_json_delta", "partial_json": `{"value":"ping"}`}}),
		ev("content_block_stop", map[string]any{"type": "content_block_stop", "index": 2}),
		ev("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "tool_use", "stop_sequence": nil}, "usage": map[string]any{"output_tokens": 5}}),
		ev("message_stop", map[string]any{"type": "message_stop"}),
	}
	for _, c := range []struct {
		name   string
		frames []string
	}{
		{"overlap", append(append(append([]string{}, head...), tail[:2]...), append([]string{stop1}, tail[2:]...)...)},
		{"ordered", append(append([]string{}, head...), append([]string{stop1}, tail...)...)},
	} {
		a.replies, a.n = []reply{{200, "text/event-stream", sse(c.frames...)}}, 0
		rec := httptest.NewRecorder()
		New().Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/messages", strings.NewReader(
			`{"model":"a/m","max_tokens":128,"stream":true,"messages":[{"role":"user","content":"Call echo with ping."}],"tools":[{"name":"echo","description":"Echo a value","input_schema":{"type":"object","properties":{"value":{"type":"string"}}}}]}`)))
		if rec.Code != 200 {
			t.Fatalf("%s: %d %s", c.name, rec.Code, rec.Body.String())
		}
		seen := blockMarks(t, rec.Body.String())
		has := func(mark [2]string) bool { return slices.Contains(seen, mark) }
		for _, mark := range [][2]string{{"content_block_start", "0"}, {"content_block_stop", "0"},
			{"content_block_start", "1"}, {"content_block_stop", "1"},
			{"content_block_start", "2"}, {"content_block_stop", "2"}} {
			if !has(mark) {
				t.Fatalf("%s: %s %s never reached the client\n%s", c.name, mark[0], mark[1], rec.Body.String())
			}
		}
	}
}
