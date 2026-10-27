package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// A relay that names the Responses API in its model list's
// supported_endpoint_types while the user set one Chat URL (…/v1) serves
// Responses at that same base: Codex's tools request goes there as it is,
// not to Chat, which turns tools with reasoning away (#2822).
func TestDeclaredResponsesAtTheChatBase(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	paths := map[string]int{}
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		paths[r.Method+" "+r.URL.Path]++
		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			// the relay's own list, as the report has it
			io.WriteString(w, `{"object":"list","data":[`+
				`{"id":"gpt-6.1-sol","supported_endpoint_types":["openai","chat","responses"]},`+
				`{"id":"gpt-6-astra","supported_endpoint_types":["openai","openai-chat","openai-response"]}]}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		switch r.URL.Path {
		case "/v1/responses":
			io.WriteString(w, sse(
				`event: response.created`+"\n"+`data: {"type":"response.created","response":{"id":"r1","model":"gpt-6.1-sol"}}`,
				`event: response.completed`+"\n"+`data: {"type":"response.completed","response":{"id":"r1","model":"gpt-6.1-sol","status":"completed","usage":{"input_tokens":1,"output_tokens":1}}}`))
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error":{"message":"tools with reasoning_effort are not supported with this model, use /v1/responses"}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(up.Close)
	p := provider.Provider{ID: "fake", Name: "Fake", Key: "k",
		Models: []string{"gpt-6.1-sol", "gpt-6-astra"}, Chat: up.URL + "/v1"}
	if err := provider.Save(p); err != nil {
		t.Fatal(err)
	}
	// what its /v1/models said, read as it reads it
	if _, err := p.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{"gpt-6.1-sol", "gpt-6-astra"} {
		clear(paths)
		body := `{"model":"` + model + `","stream":true,"reasoning":{"effort":"high"},` +
			`"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}],` +
			`"tools":[{"type":"function","name":"sh","parameters":{"type":"object","properties":{}}}]}`
		rec := httptest.NewRecorder()
		New().Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body)))
		if rec.Code != 200 || paths["POST /v1/responses"] != 1 || paths["POST /v1/chat/completions"] != 0 {
			t.Errorf("%s: status %d, upstream %v: %s", model, rec.Code, paths, rec.Body.String())
		}
	}
}
