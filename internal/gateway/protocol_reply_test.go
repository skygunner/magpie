package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func TestProtocolFailureKeepsHTTPStatus(t *testing.T) {
	for _, body := range []string{`{"id":"bad","choices":[]}`, `{"id":"bad","usage":{"completion_tokens":1}}`} {
		t.Run(body, func(t *testing.T) {
			fresh(t)
			serveOn(t, "fake", "k", []string{"m"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, body)
			}))
			s := New()
			front := httptest.NewServer(s.Front())
			defer front.Close()
			resp, err := http.Post(front.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"fake/m","messages":[{"role":"user","content":"hi"}]}`))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			got, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != 200 {
				t.Fatalf("wire status %d: %s", resp.StatusCode, got)
			}
			rec := lastRecord(t)
			if rec.Status != 200 || rec.ErrType != "protocol_error" || !strings.Contains(rec.Error, "choices") || !rec.Failed() {
				t.Fatalf("want HTTP 200 marked protocol failure, got status=%d type=%q error=%q", rec.Status, rec.ErrType, rec.Error)
			}
			if calls := s.Recent(); len(calls) == 0 || !strings.Contains(calls[0].Error, "choices") {
				t.Fatalf("recent calls: %+v", calls)
			}
		})
	}
}

func TestProtocolReplyEvidence(t *testing.T) {
	for _, c := range []struct {
		name                     string
		proto                    provider.Protocol
		stream, cut              bool
		ct, encoding, body, want string
	}{
		{name: "nonstream SSE", proto: provider.Chat, ct: "text/event-stream; charset=utf-8", body: ": keepalive\n", want: "SSE"},
		{name: "SSE with wrong header", proto: provider.Chat, ct: "application/json", body: "data: {}\n\n", want: "SSE"},
		{name: "stream usage-only choices", proto: provider.Chat, stream: true, ct: "text/event-stream", body: "data: {\"choices\":[],\"usage\":{}}\n\n"},
		{name: "invalid JSON", proto: provider.Chat, body: `{"choices":`, want: "JSON"},
		{name: "truncated JSON", proto: provider.Chat, cut: true, body: `{"choices":`},
		{name: "compressed", proto: provider.Chat, encoding: "gzip", body: "encoded bytes"},
		{name: "null", proto: provider.Chat, body: `null`, want: "JSON"},
		{name: "tools", proto: provider.Chat, body: `{"choices":[{"message":{"content":null,"tool_calls":[{"id":"call_a","type":"function","function":{"name":"recall","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`},
		{name: "refusal", proto: provider.Chat, body: `{"choices":[{"message":{"content":null,"refusal":"cannot comply"},"finish_reason":"stop"}]}`},
		{name: "empty valid content", proto: provider.Chat, body: `{"choices":[{"message":{"content":""},"finish_reason":"length"}]}`},
		{name: "error words are content", proto: provider.Chat, body: `{"choices":[{"message":{"content":"ProviderResponseError Traceback ERROR missing choices"}}]}`},
		{name: "background response", proto: provider.Responses, body: `{"object":"response","status":"queued","output":[]}`},
		{name: "missing response output", proto: provider.Responses, body: `{"id":"r"}`, want: "output"},
		{name: "message", proto: provider.Anthropic, body: `{"type":"message","content":[]}`},
		{name: "missing message content", proto: provider.Anthropic, body: `{"type":"message"}`, want: "content"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := &captureResponseWriter{ResponseWriter: httptest.NewRecorder()}
			w.Header().Set("Content-Type", c.ct)
			w.Header().Set("Content-Encoding", c.encoding)
			io.WriteString(w, c.body)
			w.body.truncated = c.cut
			got := w.protocolError(c.proto, c.stream)
			if c.want == "" && got != "" || c.want != "" && !strings.Contains(got, c.want) {
				t.Fatalf("error=%q want %q", got, c.want)
			}
		})
	}
}

func TestProtocolCheckUsesCommittedHeaders(t *testing.T) {
	r := httptest.NewRecorder()
	w := &captureResponseWriter{ResponseWriter: r}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Flush()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(503)
	io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	if w.status != 200 || r.Code != 200 || !strings.Contains(w.protocolError(provider.Chat, false), "SSE") {
		t.Fatal("lost original HTTP status or Content-Type")
	}
}
