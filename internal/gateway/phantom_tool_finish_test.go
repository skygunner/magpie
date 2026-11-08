package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// A Chat relay may end a text-only reply as tool_calls (#5376). Messages and
// Responses must finish the turn without asking the client to run a tool.
func TestTranslatedPhantomToolFinish(t *testing.T) {
	fresh(t)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Stream bool `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		if req.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"id\":\"x\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hello\"}}]}\n\ndata: {\"id\":\"x\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\ndata: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"x","object":"chat.completion","model":"m1","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"tool_calls"}]}`)
	}))
	t.Cleanup(up.Close)
	if err := provider.Save(provider.Provider{ID: "up", Name: "Up", Key: "k", Chat: up.URL + "/v1", Models: []string{"m1"}}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New().Handler())
	t.Cleanup(srv.Close)
	for _, path := range []string{"/v1/messages", "/v1/responses"} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stream=%v", path, stream), func(t *testing.T) {
				body := fmt.Sprintf(`{"model":"up/m1","max_tokens":100,"stream":%t,"messages":[{"role":"user","content":"hi"}]}`, stream)
				if path == "/v1/responses" {
					body = fmt.Sprintf(`{"model":"up/m1","stream":%t,"input":"hi"}`, stream)
				}
				resp, err := http.Post(srv.URL+path, "application/json", strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				data, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				out := string(data)
				if resp.StatusCode != 200 || !strings.Contains(out, "hello") {
					t.Fatalf("status %d: %s", resp.StatusCode, out)
				}
				if strings.Contains(out, `"tool_use"`) || strings.Contains(out, `"function_call"`) {
					t.Errorf("phantom tool call: %s", out)
				}
				if path == "/v1/messages" && !strings.Contains(out, `"stop_reason":"end_turn"`) {
					t.Errorf("missing end_turn: %s", out)
				}
				if path == "/v1/responses" && !strings.Contains(out, `"status":"completed"`) {
					t.Errorf("missing completed: %s", out)
				}
			})
		}
	}
}

// Non-tool stop reasons remain authoritative even if calls were present.
func TestCollectorKeepsOtherStopReasons(t *testing.T) {
	for _, stop := range []string{"length", "filter"} {
		for _, call := range []bool{false, true} {
			var c collector
			if call {
				c.res.Parts = append(c.res.Parts, Part{Kind: ToolCall})
			}
			c.res.Stop = stop
			if got := c.finish().Stop; got != stop {
				t.Errorf("stop=%s call=%v: got %s", stop, call, got)
			}
		}
	}
}
