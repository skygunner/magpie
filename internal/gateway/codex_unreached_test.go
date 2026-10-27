package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// One of Codex's own models, when OpenAI closes on the request before its
// response headers, ends as the connection that failed: a status magpie
// makes up is an answer Codex retries as a wrong one, and never the
// connection failure it waits out (#3135). What went wrong stays in the
// routing view, saying which model went where (#322).
func TestCodexOwnModelUnreachedEndsAsTheFailedConnection(t *testing.T) {
	setup(t, provider.Chat, &fake{t: t})
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// nothing written, the connection closed: the EOF of the report
		panic(http.ErrAbortHandler)
	}))
	t.Cleanup(up.Close)
	was := codexAPIBase
	codexAPIBase = up.URL
	t.Cleanup(func() { codexAPIBase = was })
	s := New()
	gw := httptest.NewServer(s.Handler())
	t.Cleanup(gw.Close)
	req, err := http.NewRequest("POST", gw.URL+CodexPath+"/responses", strings.NewReader(`{"model":"gpt-5.5","input":"hi","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-relay-key")
	res, err := gw.Client().Do(req)
	if err == nil {
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Fatalf("answered %d %q, want the connection Codex can wait out", res.StatusCode, body)
	}
	s.trace.mu.Lock()
	last := *s.trace.routes[len(s.trace.routes)-1]
	s.trace.mu.Unlock()
	if !strings.Contains(last.Error, "OpenAI can't be reached") || !strings.Contains(last.Error, "gpt-5.5 is one of Codex's own models") ||
		!strings.Contains(last.Error, "pick one of magpie's models") {
		t.Fatalf("routing view says %q", last.Error)
	}
}
