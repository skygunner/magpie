package gui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// The Providers page answers at once while an account's vendor is slow to
// list its models (#541: ten seconds and more of placeholders at start-up,
// the page waiting on each account's list in turn). Here Codex's account is
// asked over HTTP and the vendor answers only once the test lets it: the
// page is told the lists are on their way, and has them once they are in.
func TestProvidersAnswerWhileListsComeIn(t *testing.T) {
	home := sandboxHome(t)
	jwt := func(claims any) string {
		b, _ := json.Marshal(claims)
		return "e30." + base64.RawURLEncoding.EncodeToString(b) + ".sig"
	}
	auth := filepath.Join(home, ".codex", "auth.json")
	signedIn, _ := json.Marshal(map[string]any{"auth_mode": "chatgpt", "tokens": map[string]any{
		"id_token":     jwt(map[string]any{"email": "me@example.com", "https://api.openai.com/auth": map[string]any{"chatgpt_plan_type": "pro", "chatgpt_account_id": "acct-1"}}),
		"access_token": jwt(map[string]any{"exp": time.Now().Add(24 * time.Hour).Unix()}), "refresh_token": "r", "account_id": "acct-1",
	}})
	if err := os.MkdirAll(filepath.Dir(auth), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(auth, signedIn, 0600); err != nil {
		t.Fatal(err)
	}

	inFlight := atomic.Int32{}
	release := make(chan struct{})
	var once sync.Once
	answered := func() { once.Do(func() { close(release) }) }
	t.Cleanup(answered)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/backend-api/codex/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		inFlight.Add(1)
		defer inFlight.Add(-1)
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Write([]byte(`{"models":[{"slug":"slow-1","display_name":"Slow One","visibility":"list","priority":1}]}`))
	}))
	t.Cleanup(srv.Close)
	old := provider.CodexBase
	provider.CodexBase = srv.URL + "/backend-api/codex"
	t.Cleanup(func() { provider.CodexBase = old })
	provider.ForgetAccounts()
	t.Cleanup(provider.ForgetAccounts)

	mux := http.NewServeMux()
	providerRoutes(mux, nil)
	get := func() (providersJSON, error) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/providers", nil))
		var s providersJSON
		if w.Code != 200 {
			return s, fmt.Errorf("GET /api/providers: %d %s", w.Code, w.Body)
		}
		return s, json.Unmarshal(w.Body.Bytes(), &s)
	}
	slow := func(s providersJSON) (listed, found bool) {
		for _, p := range s.Providers {
			if p.ID == "codex" {
				for _, m := range p.Models {
					if m.ID == "slow-1" {
						return true, true
					}
				}
				return false, true
			}
		}
		return false, false
	}

	// the vendor holds its answer: the list is still being asked for when
	// the page answered is the state that says it didn't wait (one that
	// waited could only return once the fetch's own 8s deadline had passed)
	page := make(chan providersJSON, 1)
	failed := make(chan error, 1)
	go func() {
		s, err := get()
		if err != nil {
			failed <- err
			return
		}
		page <- s
	}()
	for deadline := time.Now().Add(10 * time.Second); inFlight.Load() == 0; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("no account's list was asked for")
		}
	}
	var s providersJSON
	select {
	case s = <-page:
	case err := <-failed:
		t.Fatal(err)
	case <-time.After(30 * time.Second):
		t.Fatal("the page waited for an account's list")
	}
	if n := inFlight.Load(); n != 1 {
		t.Fatalf("%d lists in flight when the page answered, want the one still to come", n)
	}
	if listed, found := slow(s); !found || listed {
		t.Fatalf("codex found %v, its list in %v before the vendor answered", found, listed)
	}
	if !s.Fetching {
		t.Fatal("not told the lists are on their way")
	}

	// the vendor answers: the list is in on the next look
	answered()
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		s, err := get()
		if err != nil {
			t.Fatal(err)
		}
		if listed, _ := slow(s); listed && !s.Fetching {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Codex's list never came in: fetching %v", s.Fetching)
		}
	}
	for deadline := time.Now().Add(10 * time.Second); provider.FetchingNew(); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("a fetch still runs after 10s")
		}
	}
}
