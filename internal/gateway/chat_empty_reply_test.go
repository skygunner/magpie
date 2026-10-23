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
