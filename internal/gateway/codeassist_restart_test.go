package gateway

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// codeAssistRestart is Antigravity restarting its answer inside one SSE
// response, as it did for #3248: the reply says some of it, ends, and a
// fresh generation of the same answer begins in the stream after it. Each
// generation's text arrived as deltas of the one completion, so the agent
// was given two to four partial answers glued end to end.
func TestCodeAssistRestartIsNotSpliced(t *testing.T) {
	for _, c := range []struct {
		name   string
		want   string
		chunks []string
	}{
		{"content after the finish", "Provider is kota-local.", []string{
			`data: {"response":{"responseId":"r1","candidates":[{"content":{"parts":[{"text":"Provider is "}]}}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":3}}}`,
			`data: {"response":{"candidates":[{"content":{"parts":[{"text":"kota-local."}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":8}}}`,
			`data: {"response":{"candidates":[{"content":{"parts":[{"text":"Provider is "}]}}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":3}}}`,
			`data: {"response":{"candidates":[{"content":{"parts":[{"text":"kota-local, and the model is argon."}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":14}}}`,
		}},
		{"another responseId", "Provider is ", []string{
			`data: {"response":{"responseId":"r1","candidates":[{"content":{"parts":[{"text":"Provider is "}]}}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":3}}}`,
			`data: {"response":{"responseId":"r2","candidates":[{"content":{"parts":[{"text":"Provider is "}]}}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":3}}}`,
			`data: {"response":{"responseId":"r2","candidates":[{"content":{"parts":[{"text":"kota-local, and the model is argon."}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":14}}}`,
		}},
		{"a count gone back", "Provider is ", []string{
			`data: {"response":{"responseId":"r1","candidates":[{"content":{"parts":[{"text":"Provider is "}]}}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":3}}}`,
			`data: {"response":{"responseId":"r1","candidates":[{"content":{"parts":[{"text":"Provider is "}]}}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":2}}}`,
			`data: {"response":{"responseId":"r1","candidates":[{"content":{"parts":[{"text":"kota-local, and the model is argon."}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":14}}}`,
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, _ := antigravityGroupFor(t, []string{"u@example.com"}, "argon", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				io.WriteString(w, sse(c.chunks...))
			})
			code, out := postTo(t, s, "/v1/chat/completions",
				`{"model":"antigravity/argon","stream":true,"messages":[{"role":"user","content":"which provider serves you?"}]}`)
			if code != 200 {
				t.Fatalf("the turn: %d %s", code, out)
			}
			if text := chatTextOf(out); text != c.want {
				t.Fatalf("one answer has both generations' text: %q", text)
			}
			if !strings.Contains(out, `"error"`) {
				t.Fatalf("the restart was not told, and the reply looks whole: %s", out)
			}
		})
	}
}

// A generation that restarted before any of it reached the client says
// nothing the agent can see: the abandoned prefix goes and the generation
// that restarted is the answer, with no error told for it.
func TestCodeAssistRestartBeforeAnythingSaid(t *testing.T) {
	s, _ := antigravityGroupFor(t, []string{"u@example.com"}, "argon", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse(
			`data: {"response":{"responseId":"r1","candidates":[{"content":{"parts":[{"text":""}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":0}}}`,
			`data: {"response":{"responseId":"r2","candidates":[{"content":{"parts":[{"text":"Provider is "}]}}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":2}}}`,
			`data: {"response":{"responseId":"r2","candidates":[{"content":{"parts":[{"text":"kota-local."}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2049,"candidatesTokenCount":8}}}`))
	})
	code, out := postTo(t, s, "/v1/chat/completions",
		`{"model":"antigravity/argon","stream":true,"messages":[{"role":"user","content":"which provider serves you?"}]}`)
	if code != 200 {
		t.Fatalf("the turn: %d %s", code, out)
	}
	if text := chatTextOf(out); text != "Provider is kota-local." {
		t.Fatalf("the answer: %q", text)
	}
	if strings.Contains(out, `"error"`) {
		t.Fatalf("a restart nothing reached the client for was told as an error: %s", out)
	}
}
