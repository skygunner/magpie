package gateway

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func TestLastStreamOverflow(t *testing.T) {
	for _, proto := range []provider.Protocol{provider.Chat, provider.Anthropic, provider.Responses} {
		for _, stream := range []bool{false, true} {
			for _, group := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/stream=%t/group=%t", proto, stream, group), func(t *testing.T) {
					fresh(t)
					f := &fake{reply: sse(`data: {"error":{"message":"Context window is full. Reduce conversation history, system prompt, or tools.","type":"invalid_request_error"}}`)}
					up := setup(t, proto, f)
					model := "fake/m1"
					if group {
						p := provider.Provider{ID: "first", Name: "First", Key: "k", Models: []string{"m1"}}
						switch proto {
						case provider.Chat:
							p.Chat = up.URL + "/v1"
						case provider.Anthropic:
							p.Anthropic = up.URL
						case provider.Responses:
							p.Responses = up.URL + "/v1"
						}
						if err := provider.Save(p); err != nil {
							t.Fatal(err)
						}
						if err := provider.SaveGroup(provider.Group{ID: "g", Members: []string{"first/m1", model}, Routing: "order"}); err != nil {
							t.Fatal(err)
						}
						model = "group/g"
					}
					path, body := "/v1/chat/completions", fmt.Sprintf(`{"model":%q,"stream":%t,"messages":[{"role":"user","content":"hi"}]}`, model, stream)
					switch proto {
					case provider.Anthropic:
						path, body = "/v1/messages", fmt.Sprintf(`{"model":%q,"stream":%t,"max_tokens":128,"messages":[{"role":"user","content":"hi"}]}`, model, stream)
					case provider.Responses:
						path, body = "/v1/responses", fmt.Sprintf(`{"model":%q,"stream":%t,"input":"hi"}`, model, stream)
					}
					server := httptest.NewServer(New().Handler())
					defer server.Close()
					res, err := http.Post(server.URL+path, "application/json", bytes.NewBufferString(body))
					if err != nil {
						t.Fatal(err)
					}
					defer res.Body.Close()
					b, err := io.ReadAll(res.Body)
					if err != nil {
						t.Fatal(err)
					}
					if res.StatusCode != http.StatusBadRequest || !strings.Contains(string(b), "context_length_exceeded") && !strings.Contains(string(b), "prompt is too long") {
						t.Errorf("status=%d body=%s", res.StatusCode, b)
					}
					wantCalls := 1
					if group {
						wantCalls = 2
					}
					if f.calls != wantCalls {
						t.Errorf("asked %d times, want %d", f.calls, wantCalls)
					}
					restingUntil.Lock()
					rests := len(restingUntil.m)
					restingUntil.Unlock()
					if rests != 0 {
						t.Errorf("overflow rested %d accounts", rests)
					}
				})
			}
		}
	}
}
