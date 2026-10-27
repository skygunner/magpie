package gateway

// #3310: Copilot answered every request on a session magpie had traded
// "403 Forbidden: forbidden", while a session traded afresh served the
// same model from the same account, and magpie went on sending the
// refused session until its expires_at (the reporter restarted magpie,
// and only then did requests go through again). A 403 now drops the
// session it names, a fresh one is traded and the request sent again once;
// a 403 on that one too is the account's, and is passed on rather than
// traded again.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// reporterAsks is the request the reporter sent magpie's /v1/messages
// while Copilot refused it.
const reporterAsks = `{"model":"copilot/claude-opus-5.5","max_tokens":1024,"messages":[{"role":"user","content":"Respond exactly OK."}],"thinking":{"type":"adaptive"},"output_config":{"effort":"low"},"stream":false}`

// forbiddenCopilot is a machine with the editors signed in to octocat:
// Copilot's API refuses the session it is sent while refusing is on and
// serves the others, and the trade gives out a session numbered each time
// it is asked.
func forbiddenCopilot(t *testing.T) (ask func() (trades int, bearers []string), refuse func(bool)) {
	t.Helper()
	fresh(t)
	cfg := os.Getenv("XDG_CONFIG_HOME")
	os.MkdirAll(filepath.Join(cfg, "github-copilot"), 0o755)
	os.WriteFile(filepath.Join(cfg, "github-copilot", "apps.json"), mustJSON(map[string]any{
		"github.com:Iv1.x": map[string]any{"user": "octocat", "oauth_token": "gho_octocat"},
	}), 0o600)

	var mu sync.Mutex
	var n int
	var bearer []string // each model request's session token, in order
	refusing := true
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/models" {
			io.WriteString(w, `{"data":[
			  {"id":"claude-opus-5.5","name":"Claude Opus 5.5","model_picker_enabled":true,"policy":{"state":"enabled"},"supported_endpoints":["/chat/completions","/v1/messages"],"capabilities":{"type":"chat"}}]}`)
			return
		}
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		bearer = append(bearer, tok)
		if refusing || tok == "sess1" {
			w.WriteHeader(403)
			io.WriteString(w, `{"error":{"message":"forbidden","type":"forbidden"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5.5",`+
			`"content":[{"type":"text","text":"OK"}],"stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":1}}`)
	}))
	t.Cleanup(api.Close)
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/token") {
			mu.Lock()
			n++
			tok := fmt.Sprintf("sess%d", n)
			mu.Unlock()
			json.NewEncoder(w).Encode(map[string]any{"token": tok, "expires_at": time.Now().Add(time.Hour).Unix(),
				"endpoints": map[string]string{"api": api.URL}})
			return
		}
		io.WriteString(w, `{"copilot_plan":"individual"}`)
	}))
	t.Cleanup(gh.Close)
	oldTok, oldUser := provider.CopilotTokenURL, provider.CopilotUserURL
	provider.CopilotTokenURL, provider.CopilotUserURL = gh.URL+"/token", gh.URL+"/user"
	t.Cleanup(func() { provider.CopilotTokenURL, provider.CopilotUserURL = oldTok, oldUser })

	return func() (int, []string) {
			mu.Lock()
			defer mu.Unlock()
			return n, append([]string(nil), bearer...)
		}, func(refuseAll bool) {
			mu.Lock()
			defer mu.Unlock()
			refusing = refuseAll
		}
}

func askMessages(t *testing.T, s *Server, body string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body)))
	return rec.Code, rec.Body.String()
}

// The reporter's request: Copilot's answer to it is a 403 on the session
// magpie holds, and the same model is served on a session traded after it.
// The request goes through on the second one, and the next request sends
// that one rather than trading again.
func TestCopilotForbiddenTradesAFreshSession(t *testing.T) {
	ask, refuse := forbiddenCopilot(t)
	refuse(false) // the first session alone is refused
	s := New()
	for range 2 {
		code, body := askMessages(t, s, reporterAsks)
		if code != 200 || !strings.Contains(body, `"text":"OK"`) {
			t.Fatalf("copilot/claude-opus-5.5: %d %s", code, body)
		}
	}
	trades, bearer := ask()
	if trades != 2 || strings.Join(bearer, "|") != "sess1|sess2|sess2" {
		t.Fatalf("GitHub traded %d sessions, Copilot was asked %q", trades, bearer)
	}
}

// A session refused too: the 403 is the account's, so it is passed on, and
// the request is not traded and sent again until the gateway's bound.
func TestCopilotForbiddenPassesOnWhatAFreshSessionDoesNotClear(t *testing.T) {
	ask, _ := forbiddenCopilot(t) // every session refused
	s := New()
	code, body := askMessages(t, s, reporterAsks)
	if code != 403 || !strings.Contains(body, "forbidden") {
		t.Fatalf("copilot/claude-opus-5.5: %d %s", code, body)
	}
	trades, bearer := ask()
	if trades != 2 || strings.Join(bearer, "|") != "sess1|sess2" {
		t.Fatalf("GitHub traded %d sessions, Copilot was asked %q", trades, bearer)
	}
}
