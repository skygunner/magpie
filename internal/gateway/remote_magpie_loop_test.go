package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// hopCountHeader is the header a magpie counts a request's hops in. It is
// written out here, not taken from the package: it is what another magpie
// sends, and a test that followed the constant could not tell it changed.
const hopCountHeader = "X-Magpie-Hop"

// nickfan's incident (#3834): configuration sync put a Remote magpie
// provider on the computer it pointed at, so every request went back to
// the same gateway, which picked the provider again, until the computer
// ran out of memory and its web page timed out. A request names the
// magpies it has passed through, each adding one, and a magpie that is
// handed a request already through far too many refuses it with a 508: a
// loop ends in an answer, not in requests nesting without end.
func TestRemoteMagpieLoopIsRefused(t *testing.T) {
	fresh(t)
	var mu sync.Mutex
	var asked []http.Header
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		mu.Lock()
		asked = append(asked, r.Header.Clone())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"c1","choices":[{"index":0,"message":{"role":"assistant","content":"from the remote"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`)
	}))
	t.Cleanup(remote.Close)
	if err := provider.Save(provider.Provider{ID: "office", Name: "Office", Preset: provider.RemoteMagpiePreset, Key: "remote-key",
		Models: []string{"m1"}, Chat: remote.URL + "/v1"}); err != nil {
		t.Fatal(err)
	}
	h := New().Handler()
	const ask = `{"model":"office/m1","messages":[{"role":"user","content":"hi"}]}`

	// a request passed on to a remote magpie names the hop it is on, so
	// the magpie it reaches can tell a chain from a loop
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(ask)))
	if rec.Code != 200 {
		t.Fatalf("a request to a remote magpie: %d %s", rec.Code, rec.Body.String())
	}
	mu.Lock()
	if len(asked) == 0 {
		mu.Unlock()
		t.Fatal("the remote magpie was not asked")
	}
	hop := asked[len(asked)-1].Get(hopCountHeader)
	mu.Unlock()
	if hop != "1" {
		t.Errorf("the request went on with %s %q, want %q", hopCountHeader, hop, "1")
	}

	// the same request, back at a magpie with the chain already at its
	// bound: refused, and no provider asked with it
	mu.Lock()
	sent := len(asked)
	mu.Unlock()
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(ask))
	req.Header.Set("User-Agent", magpieUA())
	req.Header.Set(hopCountHeader, strconv.Itoa(8))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusLoopDetected {
		t.Fatalf("a request through 8 magpies: %d %s, want 508", rec.Code, rec.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(asked) != sent {
		t.Fatal("the provider was asked with a request through 8 magpies")
	}
}
