package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// A recovered account takes the next user turn while its stand-in finishes the tool loop (#5178).
func TestProviderAccountRecoveryNextTurn(t *testing.T) {
	for _, routing := range []string{provider.Ordered, ""} {
		for _, mode := range []string{"", provider.AffinitySession, provider.AffinityStrict} {
			t.Run(routing+"/"+mode, func(t *testing.T) {
				codexSignedIn(t, "spare@example.com")
				var mu sync.Mutex
				down := true
				var seen []string
				up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, _ := io.ReadAll(r.Body)
					account := r.Header.Get("chatgpt-account-id")
					mu.Lock()
					seen = append(seen, account)
					fail := down && account == "acct-1"
					mu.Unlock()
					if fail {
						w.WriteHeader(429)
						io.WriteString(w, `{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached","plan_type":"plus","resets_in_seconds":7200}}`)
						return
					}
					if strings.Contains(string(body), "function_call_output") && (!strings.Contains(string(body), "gAAAA-sealed") || account != "acct-2") {
						t.Errorf("sealed tool continuation sent to %s: %s", account, body)
					}
					w.Header().Set("Content-Type", "text/event-stream")
					io.WriteString(w, sse(`data: {"type":"response.created","response":{"id":"r1","model":"gpt-5.5"}}`, `data: {"type":"response.output_text.delta","delta":"`+account+`"}`, `data: {"type":"response.completed","response":{"id":"r1","usage":{"input_tokens":7000,"output_tokens":1,"input_tokens_details":{"cached_tokens":6000}}}}`))
				}))
				t.Cleanup(up.Close)
				old := provider.CodexBase
				provider.CodexBase = up.URL + "/backend-api/codex"
				t.Cleanup(func() { provider.CodexBase = old })
				if err := provider.Save(provider.Provider{ID: "codex", Routing: routing, Affinity: mode}); err != nil {
					t.Fatal(err)
				}
				s := New()
				t.Cleanup(s.subscription.abortAll)
				ask := func(body string, want string) {
					t.Helper()
					req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body))
					req.Header.Set("Authorization", "Bearer chatgpt-token")
					req.Header.Set("session_id", "session-5178")
					rec := httptest.NewRecorder()
					s.Handler().ServeHTTP(rec, req)
					if rec.Code != 200 || !strings.Contains(rec.Body.String(), want) {
						t.Fatalf("want %s: %d %s; attempts %v", want, rec.Code, rec.Body, seen)
					}
				}
				first := `{"model":"codex/gpt-5.5","stream":true,"input":[{"role":"user","content":"hi"}]}`
				tool := `{"model":"codex/gpt-5.5","stream":true,"input":[{"role":"user","content":"hi"},{"type":"reasoning","id":"rs_sealed","summary":[],"encrypted_content":"gAAAA-sealed"},{"type":"function_call","name":"read","call_id":"c1","arguments":"{}"},{"type":"function_call_output","call_id":"c1","output":"a.go"}]}`
				next := `{"model":"codex/gpt-5.5","stream":true,"input":[{"role":"user","content":"hi"},{"role":"assistant","content":"done"},{"role":"user","content":"more"}]}`
				ask(first, "acct-2")
				mu.Lock()
				if len(seen) != 2 || seen[0] != "acct-1" || seen[1] != "acct-2" {
					t.Fatalf("first turn attempts = %v, want acct-1 then acct-2", seen)
				}
				down = false
				mu.Unlock()
				restingUntil.Lock()
				for k := range restingUntil.m {
					restingUntil.m[k] = time.Now().Add(-time.Second)
				}
				restingUntil.Unlock()
				ask(tool, "acct-2")
				want := "acct-1"
				if mode != "" {
					want = "acct-2"
				}
				ask(next, want)
			})
		}
	}
}

func TestProviderHealthyAccountKeepsItsCache(t *testing.T) {
	codexSignedIn(t, "spare@example.com")
	used := map[string]float64{"me@example.com": 92, "spare@example.com": 10}
	capUsage(t, used)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		account := r.Header.Get("chatgpt-account-id")
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse(`data: {"type":"response.created","response":{"id":"r1","model":"gpt-5.5"}}`, `data: {"type":"response.output_text.delta","delta":"`+account+`"}`, `data: {"type":"response.completed","response":{"id":"r1","usage":{"input_tokens":7000,"output_tokens":1,"input_tokens_details":{"cached_tokens":6000}}}}`))
	}))
	t.Cleanup(up.Close)
	old := provider.CodexBase
	provider.CodexBase = up.URL + "/backend-api/codex"
	t.Cleanup(func() { provider.CodexBase = old })
	s := New()
	t.Cleanup(s.subscription.abortAll)
	ask := func(session string, turn int, want string) {
		t.Helper()
		body := strings.Replace(responsesTurn(turn), "group/g", "codex/gpt-5.5", 1)
		req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer chatgpt-token")
		req.Header.Set("session_id", session)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("session %s turn %d: %d %s, want %s", session, turn, rec.Code, rec.Body, want)
		}
	}
	ask("kept", 1, "acct-2")
	used["me@example.com"] = 10
	provider.ForgetAccounts()
	ask("new", 1, "acct-1")
	ask("kept", 2, "acct-2")
}
