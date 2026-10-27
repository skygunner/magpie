package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// flushSpy is a writer that says when a response hook's cleanup flushed an
// exchange, and whether anything of it was written first.
type flushSpy struct{ status, flushes int }

func (w *flushSpy) Header() http.Header { return http.Header{} }

func (w *flushSpy) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *flushSpy) Write(p []byte) (int, error) { return len(p), nil }

func (w *flushSpy) Flush() { w.flushes++ }

// A reply the handler never began stays unwritten. Flushing it sends a 200
// for an exchange that failed before any answer, which Codex takes as an
// answer to retry rather than a connection to wait out (#3135). A reply
// that was begun is still flushed out to the client.
func TestGatewayHookFlushLeavesAnUnansweredReplyUnwritten(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","input":"hi","stream":true}`)
	spy := &flushSpy{}
	_, done := gatewayResponse(spy, httptest.NewRequest("POST", CodexPath+"/responses", nil), provider.Responses, body, nil)
	done()
	if spy.status != 0 || spy.flushes != 0 {
		t.Fatalf("an unanswered reply went out: status %d, %d flushes", spy.status, spy.flushes)
	}

	spy = &flushSpy{}
	w, done := gatewayResponse(spy, httptest.NewRequest("POST", CodexPath+"/responses", nil), provider.Responses, body, nil)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, sse(`data: {"type":"response.completed","response":{"id":"r1","status":"completed"}}`))
	done()
	if spy.flushes != 1 {
		t.Fatalf("an answered reply was not flushed: status %d, %d flushes", spy.status, spy.flushes)
	}
}
