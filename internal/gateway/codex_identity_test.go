package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func TestCodexClientIdentity(t *testing.T) {
	for _, c := range []struct {
		ua, orig, want string
	}{
		{"Codex Desktop/0.162.3", "codex-tui", "Codex Desktop"},
		{"Codex Desktop/0.162.3", "codex_cli_rs", "Codex Desktop"},
		{"codex_cli_rs/0.159.2", "codex-tui", "codex-tui"},
		{"codex_cli_rs/0.159.2", "codex_cli_rs", "codex_cli_rs"},
		{"Codex Desktop/0.162.3", "", "Codex Desktop"},
		{"", "codex-tui", "codex-tui"},
		{"Other/1.0", "codex-tui", "codex-tui"},
	} {
		h := http.Header{}
		if c.ua != "" {
			h.Set("User-Agent", c.ua)
		}
		if c.orig != "" {
			h.Set("originator", c.orig)
		}
		got := codexClientIdentity(h)
		if got != c.want {
			t.Errorf("ua=%q orig=%q: got %q, want %q", c.ua, c.orig, got, c.want)
		}
	}
}

// #3851: a Codex Desktop request resuming a TUI thread should have its
// originator normalized to "Codex Desktop" instead of forwarding the
// TUI originator.
func TestCodexDesktopResumedTUIThreadOriginator(t *testing.T) {
	fresh(t)
	provider.Save(provider.Provider{
		ID: "echo", Name: "Echo", Key: "k", Chat: "https://echo.test/v1", Models: []string{"m1"},
	})

	s := New()
	// request with Desktop UA but TUI originator
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{"model":"echo/m1","input":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Codex Desktop/0.162.3")
	req.Header.Set("originator", "codex-tui")
	req.Header.Set("session-id", "sess-1")
	req.Header.Set("thread-id", "thread-1")

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	// the request should not 403
	if rec.Code == 403 {
		t.Fatal("got 403: originator not normalized")
	}
	if rec.Code != 200 {
		t.Logf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}
