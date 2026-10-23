package gateway

import (
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// An ordinary Chat upstream that answers 200, says nothing and stops is a
// failure, not a turn's end: through the Responses translation it completed
// a Codex turn with no message at all, and Codex recorded the task done —
// twice in a row through affinity, with no stop reason, no first token and
// no error in the usage (#3358, Latias94 — Krill and AIGO, both with usage
// counted on the way, so counting is no test of an answer).
func TestChatEmptyReplyIsAnError(t *testing.T) {
	empty := sse(
		`data: {"id":"c1","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":31871,"completion_tokens":0}}`,
		`data: [DONE]`)
	f := &fake{t: t, reply: empty}
	setup(t, provider.Chat, f)
	// the Codex route the incidents took, non-streaming and streamed:
	// both must fail, not complete a turn with no message
	code, out := post(t, "/v1/responses", `{"model":"fake/m1","stream":false,"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"continue"}]}]}`)
	if code != 502 || !strings.Contains(out, "an empty reply") {
		t.Errorf("non-streaming: an empty 200 answered as done: %d %s", code, out)
	}
	f.calls = 0
	code, out = post(t, "/v1/responses", `{"model":"fake/m1","stream":true,"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"continue"}]}]}`)
	// the stream's own failure event ends it (response.failed), which a
	// Codex reads as the try's failure and answers by failing over
	if !strings.Contains(out, `"status":"failed"`) || !strings.Contains(out, "an empty reply") {
		t.Errorf("streaming: an empty 200 answered as done: %d %s", code, out)
	}
	if f.calls < 2 {
		t.Errorf("asked %d times, not again", f.calls)
	}
}

// What says something, or ended for a reason of its own, still goes as it
// came: a text answer, and a length end of reasoning only.
func TestChatEmptyReplyKeepsRealAnswers(t *testing.T) {
	for name, reply := range map[string]string{
		"text": sse(
			`data: {"id":"c1","choices":[{"index":0,"delta":{"content":"OK"}}]}`,
			`data: {"id":"c1","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`),
		"a length end of reasoning": sse(
			`data: {"id":"c1","choices":[{"index":0,"delta":{"reasoning_content":"Let me see."}}]}`,
			`data: {"id":"c1","choices":[{"index":0,"delta":{},"finish_reason":"length"}],"usage":{"prompt_tokens":10,"completion_tokens":100}}`,
			`data: [DONE]`),
	} {
		f := &fake{t: t, reply: reply}
		setup(t, provider.Chat, f)
		code, out := post(t, "/v1/chat/completions", `{"model":"fake/m1","messages":[{"role":"user","content":"continue"}]}`)
		if code != 200 || strings.Contains(out, "an empty reply") {
			t.Errorf("%s: %d %s", name, code, out)
		}
	}
}

// The controls on the translated route itself: a normal text answer, a
// tool call, a reasoning-only reply ended at its length and one ended by
// a filter all keep their own semantics through Chat→Responses — the
// empty-reply failure is for a reply that says nothing for no reason of
// its own, not for these (#3358 review).
func TestChatEndingsKeepTheirSemanticsThroughResponses(t *testing.T) {
	for name, reply := range map[string]string{
		"text": sse(
			`data: {"id":"c1","choices":[{"index":0,"delta":{"content":"Done."}}]}`,
			`data: {"id":"c1","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":2}}`,
			`data: [DONE]`),
		"a tool call": sse(
			`data: {"id":"c1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.go\"}"}}]}}]}`,
			`data: {"id":"c1","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
			`data: [DONE]`),
		`reasoning to its length`: sse(
			`data: {"id":"c1","choices":[{"index":0,"delta":{"reasoning_content":"thinking"}}]}`,
			`data: {"id":"c1","choices":[{"index":0,"delta":{},"finish_reason":"length"}],"usage":{"prompt_tokens":10,"completion_tokens":100}}`,
			`data: [DONE]`),
		"ended by a filter": sse(
			`data: {"id":"c1","choices":[{"index":0,"delta":{"content":"I can't help with that."}}]}`,
			`data: {"id":"c1","choices":[{"index":0,"delta":{},"finish_reason":"content_filter"}],"usage":{"prompt_tokens":10,"completion_tokens":6}}`,
			`data: [DONE]`),
	} {
		f := &fake{t: t, reply: reply}
		setup(t, provider.Chat, f)
		for _, stream := range []bool{false, true} {
			f.calls = 0
			st := "false"
			if stream {
				st = "true"
			}
			code, out := post(t, "/v1/responses", `{"model":"fake/m1","stream":`+st+`,"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"continue"}]}]}`)
			if code != 200 || strings.Contains(out, "an empty reply") {
				t.Errorf("%s streamed=%v: %d %s", name, stream, code, out)
				continue
			}
			switch name {
			case "text":
				if !strings.Contains(out, "Done.") {
					t.Errorf("%s streamed=%v: no text: %s", name, stream, out)
				}
			case "a tool call":
				if !strings.Contains(out, `"name":"read_file"`) || !strings.Contains(out, `"call_id":"call_1"`) {
					t.Errorf("%s streamed=%v: no call: %s", name, stream, out)
				}
			case "reasoning to its length":
				if !strings.Contains(out, `"status":"incomplete"`) && !strings.Contains(out, "length") {
					t.Errorf("%s streamed=%v: the length end was lost: %s", name, stream, out)
				}
			case "ended by a filter":
				if !strings.Contains(out, "I can't help with that.") && !strings.Contains(out, "refusal") {
					t.Errorf("%s streamed=%v: the filter answer was lost: %s", name, stream, out)
				}
			}
		}
	}
}
