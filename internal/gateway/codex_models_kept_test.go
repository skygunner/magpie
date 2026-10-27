package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/provider"
)

// The 500K window the backend gave Codex must survive the next model-list
// timeout, magpie's restart included (#1955): with the backend away the
// list falls back to Codex's own cache, which says 500000, and the window
// must not be restored from the provider's saved list, a day old, whose
// value is still 272000 — Codex's conversation fell from 475000 usable to
// 258400 and compacted at ~277K input tokens.
func TestCodexModelListKeepsBackendsWindowOnTimeout(t *testing.T) {
	codexSignedIn(t)
	codexHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	// what Codex cached from the official 500K list magpie handed it
	if err := os.WriteFile(filepath.Join(codexHome, "models_cache.json"), []byte(`{"etag":"\"v1\"","models":[
		{"slug":"gpt-6.1-sol","priority":1,"context_window":500000,"max_context_window":500000,"base_instructions":"sol prompt"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// the backend answers once, with the official 500K list, and then
	// never again inside codexModelsWait
	forgetCodexLists(t)
	var once sync.Once
	release := make(chan struct{})
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answered := false
		once.Do(func() { answered = true })
		if answered {
			w.Write([]byte(`{"models":[{"slug":"gpt-6.1-sol","priority":1,"context_window":500000,"max_context_window":500000,"base_instructions":"sol prompt"}]}`))
			return
		}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() { close(release); up.Close() })
	wasBase := provider.CodexBase
	provider.CodexBase = up.URL + "/backend-api/codex"
	t.Cleanup(func() { provider.CodexBase = wasBase })
	// the provider's own list, kept from the day before, still says 272K
	live := catalog.LivePath("codex")
	if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
		t.Fatal(err)
	}
	kept := `{"Fetched":"` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `","Base":"` + provider.CodexBase + `","Models":[{"ID":"gpt-6.1-sol","Context":272000}]}`
	if err := os.WriteFile(live, []byte(kept), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog.Reset()
	t.Cleanup(catalog.Reset)
	was := codexModelsWait
	codexModelsWait = 200 * time.Millisecond
	t.Cleanup(func() { codexModelsWait = was })
	h := New().Handler()

	contextWindow := func() (int, bool) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", CodexPath+"/models?client_version=0.162.0", nil)
		req.Header.Set("Authorization", "Bearer chatgpt-token")
		req.Header.Set("chatgpt-account-id", "acct-1")
		h.ServeHTTP(rec, req)
		var got struct {
			Models []struct {
				Slug    string `json:"slug"`
				Context int    `json:"context_window"`
			} `json:"models"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		for _, m := range got.Models {
			if m.Slug == "gpt-6.1-sol" {
				return m.Context, true
			}
		}
		return 0, false
	}

	// the backend's list comes through, and Codex takes its 500K
	if n, ok := contextWindow(); !ok || n != 500000 {
		t.Fatalf("gpt-6.1-sol context_window %d (%v) from the backend's own list, want 500000", n, ok)
	}
	// magpie restarts: the lists it kept in memory are gone, the one it
	// kept on disk is not; the backend, asked again, doesn't answer
	codexLists.asking.Wait()
	codexLists.Lock()
	codexLists.at = map[string]*codexListed{}
	codexLists.Unlock()
	if n, ok := contextWindow(); !ok || n != 500000 {
		t.Errorf("gpt-6.1-sol context_window %d (%v) after the timeout, want the backend's 500000, not the provider's 272000", n, ok)
	}
}
