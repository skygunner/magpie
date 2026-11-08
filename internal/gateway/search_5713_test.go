package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

func TestResponsesSearchHistoryReachesChat5713(t *testing.T) {
	fresh(t)
	bodies := make(chan []byte, 4)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		bodies <- b
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse(`data: {"id":"chat1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"answer"}}]}`, `data: {"id":"chat1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`, `data: [DONE]`))
	}))
	defer up.Close()
	if err := provider.Save(provider.Provider{ID: "relay5713", Name: "Relay", Key: "dummy", Chat: up.URL + "/v1", Responses: up.URL + "/v1", Models: []string{"m"}}); err != nil {
		t.Fatal(err)
	}
	if err := provider.SetModelAPI("relay5713/m", "chat"); err != nil {
		t.Fatal(err)
	}
	gw := httptest.NewServer(New().Handler())
	defer gw.Close()

	history := `{"type":"message","role":"user","content":"Find the evidence and use the tool."},` +
		`{"type":"reasoning","id":"rs_diag","summary":[{"type":"summary_text","text":"UNIQUE_THINKING_X718"}]},` +
		`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"UNIQUE_ASSISTANT_X718"}]},` +
		`{"type":"web_search_call","id":"ws_diag","status":"completed","action":{"type":"search","query":"UNIQUE_SEARCH_QUERY_X718","queries":["UNIQUE_SEARCH_QUERY_X718"],"sources":[{"type":"url","url":"https://example.invalid/UNIQUE_SEARCH_RESULT_X718"}]}},` +
		`{"type":"function_call","id":"fc_diag","call_id":"call_diag","name":"noop","arguments":"{}"},` +
		`{"type":"function_call_output","call_id":"call_diag","output":"UNIQUE_TOOL_RESULT_X718"},` +
		`{"type":"message","role":"user","content":"Continue with the evidence already found."}`
	var previous json.RawMessage
	for turn := 0; turn < 2; turn++ {
		if turn == 1 {
			history += `,` + string(previous) + `,` +
				`{"type":"web_search_call","id":"ws_second","status":"completed","action":{"type":"search","query":"SECOND_QUERY","sources":[{"type":"url","url":"https://example.invalid/second"}]}},` +
				`{"type":"message","role":"user","content":"Continue again."}`
		}
		body := `{"model":"relay5713/m","stream":false,"reasoning":{"effort":"high"},"tools":[{"type":"function","name":"noop","parameters":{"type":"object","properties":{}}}],"input":[` + history + `]}`
		res, err := http.Post(gw.URL+"/v1/responses", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		response, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil || res.StatusCode != 200 {
			t.Fatalf("turn %d: status %d: %s, %v", turn, res.StatusCode, response, err)
		}
		var reply struct {
			Output []json.RawMessage `json:"output"`
		}
		if err := json.Unmarshal(response, &reply); err != nil || len(reply.Output) != 1 {
			t.Fatalf("turn %d: invalid response %s: %v", turn, response, err)
		}
		previous = reply.Output[0]
		var sent struct {
			Messages []struct {
				Role       string `json:"role"`
				Content    string `json:"content"`
				ToolCallID string `json:"tool_call_id"`
				ToolCalls  []struct {
					ID string `json:"id"`
				} `json:"tool_calls"`
			} `json:"messages"`
		}
		var wire []byte
		select {
		case wire = <-bodies:
		case <-time.After(5 * time.Second):
			t.Fatal("upstream did not receive the request")
		}
		if err := json.Unmarshal(wire, &sent); err != nil {
			t.Fatalf("invalid Chat request: %v: %s", err, wire)
		}
		var content strings.Builder
		calls, results := map[string]int{}, map[string]int{}
		for _, m := range sent.Messages {
			content.WriteString(m.Content)
			for _, c := range m.ToolCalls {
				calls[c.ID]++
			}
			if m.Role == "tool" {
				results[m.ToolCallID]++
			}
		}
		want := []string{"UNIQUE_SEARCH_QUERY_X718", "https://example.invalid/UNIQUE_SEARCH_RESULT_X718", "UNIQUE_ASSISTANT_X718", "UNIQUE_TOOL_RESULT_X718"}
		if turn == 1 {
			want = append(want, "SECOND_QUERY", "https://example.invalid/second")
		}
		for _, value := range want {
			if strings.Count(content.String(), value) != 1 {
				t.Errorf("turn %d: missing or duplicated %q in %s", turn, value, wire)
			}
		}
		if len(calls) != 1 || calls["call_diag"] != 1 || len(results) != 1 || results["call_diag"] != 1 {
			t.Errorf("turn %d: calls %v, results %v: %s", turn, calls, results, wire)
		}
	}
}
