package gui

import (
	"context"
	"encoding/json"
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

// A plugin account's editor saves PassHeaders/ReplyHeaders as a custom
// provider's does (#5359): set, cleared with [], refused when one is never
// passed on, and /api/providers keeps the vendor's secrets out.
func TestPluginAccountHeadersSaveReload(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("MAGPIE_BUN", testenv.Bun(t))
	t.Cleanup(plugin.Settle)
	// the vendor only exists so a stray fetch fails loudly; the save path
	// asks nothing of it
	testenv.FakeVendor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	// a catalog already cached: the host's sync asks models.dev for nothing
	path := catalog.CachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
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
	id := provider.PluginID("fakeco")
	if p, err := provider.Find(id); err != nil || !p.IsPlugin() {
		t.Fatalf("Find(%s) = %+v, %v", id, p, err)
	}
	mux := http.NewServeMux()
	providerRoutes(mux, nil)
	post := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/provider/save", strings.NewReader(body)))
		return w
	}
	if w := post(`{"id":"` + id + `","passHeaders":["X-Route"],"replyHeaders":["X-Trace"]}`); w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body)
	}
	p, err := provider.Find(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.PassHeaders) != 1 || p.PassHeaders[0] != "X-Route" || len(p.ReplyHeaders) != 1 || p.ReplyHeaders[0] != "X-Trace" {
		t.Fatalf("saved: %v %v", p.PassHeaders, p.ReplyHeaders)
	}
	if w := post(`{"id":"` + id + `","passHeaders":[],"replyHeaders":[]}`); w.Code != 200 {
		t.Fatalf("clear: %d %s", w.Code, w.Body)
	}
	p, _ = provider.Find(id)
	if len(p.PassHeaders) != 0 || len(p.ReplyHeaders) != 0 {
		t.Fatalf("cleared: %v %v", p.PassHeaders, p.ReplyHeaders)
	}
	if w := post(`{"id":"` + id + `","passHeaders":["Authorization"]}`); w.Code == 200 {
		t.Fatal("a forbidden pass header was saved")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/providers", nil))
	var out struct {
		Providers []struct {
			ID           string   `json:"id"`
			PassHeaders  []string `json:"passHeaders"`
			ReplyHeaders []string `json:"replyHeaders"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	for _, pr := range out.Providers {
		if pr.ID == id && (len(pr.PassHeaders) != 0 || len(pr.ReplyHeaders) != 0) {
			t.Fatalf("exposed: %v %v", pr.PassHeaders, pr.ReplyHeaders)
		}
	}
}
