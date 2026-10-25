package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// #3475 (mdwsk88): Claude Desktop's gateway mode runs Claude Code with
// experimental betas off, so its "Enable tool search" asks tool search as
// Bedrock's and Vertex's tool-search-tool-2025-10-19. Copilot's Messages
// API turns that name away — 400 unsupported beta header(s) — on every
// session's first request, and Claude Code turns the setting off for the
// session; Copilot serves the same deferred tools under Anthropic's own
// name. A Copilot account is asked advanced-tool-use-2025-11-20 instead,
// and another provider still gets the name the agent asked.
func TestCopilotToolSearchBeta(t *testing.T) {
	var mu sync.Mutex
	var seen []string // "<path> <anthropic-beta>" of each request
	copilotFake(t, "gho_desktop3475", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			io.WriteString(w, `{"data":[{"id":"claude-opus-5.5","name":"Claude Opus 5.5","model_picker_enabled":true,"policy":{"state":"enabled"},"supported_endpoints":["/messages"],"capabilities":{"type":"chat"}}]}`)
		case "/v1/messages", "/v1/messages/count_tokens":
			betas := strings.Join(r.Header.Values("Anthropic-Beta"), ",")
			mu.Lock()
			seen = append(seen, r.URL.Path+" "+betas)
			mu.Unlock()
			for _, b := range strings.Split(betas, ",") {
				if strings.TrimSpace(b) == "tool-search-tool-2025-10-19" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(400)
					io.WriteString(w, `{"error":{"message":"Copilot: unsupported beta header(s): tool-search-tool-2025-10-19","type":"invalid_request_error"}}`)
					return
				}
			}
			w.Header().Set("Content-Type", "application/json")
			if strings.HasSuffix(r.URL.Path, "count_tokens") {
				io.WriteString(w, `{"input_tokens":77}`)
				return
			}
			io.WriteString(w, `{"id":"m1","type":"message","role":"assistant","model":"claude-opus-5.5","content":[{"type":"text","text":"pong"}],"stop_reason":"end_turn","usage":{"input_tokens":77,"output_tokens":1}}`)
		default:
			w.WriteHeader(404)
		}
	})
	s := New()
	ask := func(model, path, betas string) (int, string) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", path, strings.NewReader(`{"model":"`+model+`","max_tokens":20,"messages":[{"role":"user","content":"ping"}]}`))
		req.Header.Set("anthropic-beta", betas)
		s.Handler().ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}
	// what Desktop asks, and both names in either order: one name goes, the
	// one Copilot takes
	for _, c := range []struct {
		asked string
		want  string
	}{
		{"tool-search-tool-2025-10-19", "advanced-tool-use-2025-11-20"},
		{"tool-search-tool-2025-10-19,advanced-tool-use-2025-11-20", "advanced-tool-use-2025-11-20"},
		{"advanced-tool-use-2025-11-20,tool-search-tool-2025-10-19", "advanced-tool-use-2025-11-20"},
	} {
		mu.Lock()
		seen = nil
		mu.Unlock()
		code, body := ask("copilot/claude-opus-5.5", "/v1/messages", c.asked)
		if code != 200 {
			t.Fatalf("Messages, betas %q: %d %s", c.asked, code, body)
		}
		mu.Lock()
		got := slices.Clone(seen)
		mu.Unlock()
		if want := []string{"/v1/messages " + c.want}; !slices.Equal(got, want) {
			t.Errorf("betas %q: Copilot was asked %q, want %q", c.asked, got, want)
		}
	}
	// count_tokens asks Copilot too, and refused the same name
	mu.Lock()
	seen = nil
	mu.Unlock()
	code, body := ask("copilot/claude-opus-5.5", "/v1/messages/count_tokens", "tool-search-tool-2025-10-19")
	if code != 200 || !strings.Contains(body, `"input_tokens":77`) {
		t.Fatalf("count_tokens: %d %s", code, body)
	}
	mu.Lock()
	got := slices.Clone(seen)
	mu.Unlock()
	if want := []string{"/v1/messages/count_tokens advanced-tool-use-2025-11-20"}; !slices.Equal(got, want) {
		t.Errorf("count_tokens: Copilot was asked %q, want %q", got, want)
	}
}

// The name is Copilot's alone: any other provider is asked as the agent
// asked it.
func TestCopilotToolSearchBetaOtherProvidersUnchanged(t *testing.T) {
	fresh(t)
	up := &betaRelay{}
	srv := httptest.NewServer(up)
	t.Cleanup(srv.Close)
	if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "k", Anthropic: srv.URL,
		Models: []string{"claude-opus-5-5"}}); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(
		`{"model":"relay/claude-opus-5-5","max_tokens":20,"messages":[{"role":"user","content":"ping"}]}`))
	req.Header.Set("anthropic-beta", "tool-search-tool-2025-10-19")
	New().Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if got := up.last(); !slices.Equal(got, []string{"tool-search-tool-2025-10-19"}) {
		t.Fatalf("relay was asked %q, want the name the agent asked", got)
	}
}
