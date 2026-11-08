package gui

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/library"
	"github.com/yetone/magpie/internal/mcpauth"
	"github.com/yetone/magpie/internal/mcpauth/mcpauthtest"
)

// browserDown is a desktop whose browser doesn't start, as Windows'
// rundll32 under a PATHEXT of ".CPL" (#2043, xiaobaifly7).
type browserDown struct{ webHost }

func (browserDown) OpenURL(string) error {
	return errors.New(`exec: "rundll32": executable file not found in %PATH%`)
}

// A start returns its URL without opening it, so a canceled pending start
// cannot open a stale authorization page (#3984).
func TestMCPSignInStartDoesNotOpenBrowser(t *testing.T) {
	sandboxHome(t)
	f := mcpauthtest.New(t)
	if _, err := library.SaveServer("", library.Server{Name: "cloudflare-api", Transport: "http", URL: f.URL}); err != nil {
		t.Fatal(err)
	}
	start := func(w Windows) (st struct{ ID, URL, State, OpenError string }) {
		t.Helper()
		rec := httptest.NewRecorder()
		Handler(w, nil).ServeHTTP(rec, httptest.NewRequest("POST", "/api/library/mcp-signin", strings.NewReader(`{"name":"cloudflare-api"}`)))
		if rec.Code != http.StatusOK {
			t.Fatalf("sign-in: %d %s", rec.Code, rec.Body)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mcpauth.Cancel(st.ID) })
		return st
	}
	st := start(browserDown{})
	if st.State != "waiting" || !strings.HasPrefix(st.URL, f.Server.URL) {
		t.Fatalf("not waiting on its page: %+v", st)
	}
	if st.OpenError != "" {
		t.Fatalf("start tried to open the browser: %+v", st)
	}
	if st := start(webHost{}); st.OpenError != "" || st.URL == "" {
		t.Fatalf("start did not return its URL: %+v", st)
	}
}

// Open again on a sign-in says why the browser didn't open, where it said
// nothing (#2043).
func TestOpenSaysTheBrowserDidntOpen(t *testing.T) {
	sandboxHome(t)
	rec := httptest.NewRecorder()
	Handler(browserDown{}, nil).ServeHTTP(rec, httptest.NewRequest("POST", "/api/open", strings.NewReader(`{"url":"https://example.invalid/authorize"}`)))
	if rec.Code == http.StatusNoContent || !strings.Contains(rec.Body.String(), "rundll32") {
		t.Fatalf("open: %d %s", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	webPageHandler(t, webHost{}, nil).ServeHTTP(rec, httptest.NewRequest("POST", "/api/open", strings.NewReader(`{"url":"https://example.invalid/authorize"}`)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("an open that worked: %d %s", rec.Code, rec.Body)
	}
}
