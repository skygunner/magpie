package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/testenv"
)

func TestPluginConfiguredHeaders5359(t *testing.T) {
	seen := make(chan http.Header, 1)
	fresh(t)
	t.Setenv("MAGPIE_BUN", testenv.Bun(t))
	t.Setenv("FAKE_ID", "fakeco")
	t.Setenv("FAKE_RESPONSES", "1")
	t.Cleanup(plugin.Settle)
	path := catalog.CachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	testenv.FakeVendor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header.Clone()
		w.Header().Set("X-Vendor-Trace", "trace-5359")
		w.Header().Set("X-Unlisted-Reply", "no")
		w.Header().Set("Set-Cookie", "secret=never")
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, `data: {"id":"c","object":"chat.completion.chunk","model":"fake-1","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":null}]}`+"\n\n")
		io.WriteString(w, `data: {"id":"c","object":"chat.completion.chunk","model":"fake-1","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`+"\n\ndata: [DONE]\n\n")
	}))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	abs, err := filepath.Abs("../plugin/testdata/fake/index.js")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plugin.Add(ctx, abs); err != nil {
		t.Fatal(err)
	}
	if _, err := plugin.Providers(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := plugin.APIKey(ctx, "fakeco", 0, nil, "k1", plugin.NewAccount); err != nil {
		t.Fatal(err)
	}
	pid := provider.PluginID("fakeco")
	p, err := provider.Find(pid)
	if err != nil {
		t.Fatal(err)
	}
	p.PassHeaders = []string{"X-Custom-Route", "Authorization"}
	p.ReplyHeaders = []string{"X-Vendor-Trace", "Set-Cookie"}
	if err := provider.Save(*p); err != nil {
		t.Fatal(err)
	}
	p, err = provider.Find(pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.PassHeaders) != 1 || len(p.ReplyHeaders) != 1 {
		t.Fatalf("saved lists: %v %v", p.PassHeaders, p.ReplyHeaders)
	}
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"`+pid+`/fake-1","stream":false,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Route", "route-5359")
	req.Header.Set("X-Unlisted-Request", "never")
	req.Header.Set("Authorization", "Bearer should-not-forward")
	rec := httptest.NewRecorder()
	New().Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	h := <-seen
	if h.Get("X-Custom-Route") != "route-5359" || h.Get("X-Unlisted-Request") != "" || h.Get("Authorization") == "Bearer should-not-forward" {
		t.Errorf("request header filtering failed")
	}
	if rec.Header().Get("X-Vendor-Trace") != "trace-5359" || rec.Header().Get("X-Unlisted-Reply") != "" || rec.Header().Get("Set-Cookie") != "" {
		t.Errorf("reply header filtering failed: %v", rec.Header())
	}
}
