package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// Cached Responses search must reach the selected searcher's wire (#5018).
func TestCachedSearchReachesCodexModel(t *testing.T) {
	codexSignedIn(t)
	search := newFakeChatGPT(t)
	st := settings.Load()
	st.Searcher = "codex"
	if err := settings.Save(st); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	rounds := 0
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		rounds++
		n := rounds
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		if n%2 == 1 {
			io.WriteString(w, sse(`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"web_search","arguments":""}}`,
				`data: {"type":"response.function_call_arguments.delta","output_index":0,"item_id":"fc_1","delta":"{\"query\":\"latest go\"}"}`,
				`data: {"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"web_search","arguments":"{\"query\":\"latest go\"}"}}`,
				`data: {"type":"response.completed","response":{"id":"r1","status":"completed","output":[{"type":"function_call","id":"fc_1","call_id":"call_1","name":"web_search","arguments":"{\"query\":\"latest go\"}"}],"usage":{"input_tokens":30,"output_tokens":7}}}`))
			return
		}
		io.WriteString(w, sse(`data: {"type":"response.output_text.delta","item_id":"m","output_index":0,"content_index":0,"delta":"Go is out."}`, `data: {"type":"response.completed","response":{"id":"r2","status":"completed","output":[{"type":"message","id":"m","role":"assistant","content":[{"type":"output_text","text":"Go is out."}]}],"usage":{"input_tokens":10,"output_tokens":5}}}`))
	}))
	t.Cleanup(model.Close)
	if err := provider.Save(provider.Provider{ID: "xiaomi", Name: "Xiaomi", Key: "k", Preset: "xiaomi", Chat: model.URL + "/v1", Responses: model.URL + "/v1", Models: []string{"mimo-v2.6-flash"}}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(New().Handler())
	t.Cleanup(server.Close)
	for _, tc := range []struct {
		name, field string
		want        any
	}{
		{"cached", `,"external_web_access":false`, false},
		{"live", `,"external_web_access":true`, true},
		{"default", ``, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"model":"xiaomi/mimo-v2.6-flash","stream":true,"input":"latest go","tools":[{"type":"web_search"` + tc.field + `}]}`
			resp, err := http.Post(server.URL+"/v1/responses", "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			out, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 200 || !strings.Contains(string(out), "Go is out.") {
				t.Fatalf("status %d: %s", resp.StatusCode, out)
			}
			paths, bodies := search.asked()
			if len(paths) != 1 || !strings.HasSuffix(paths[0], "/responses") {
				t.Fatalf("search paths %v", paths)
			}
			var tools []map[string]json.RawMessage
			b, err := json.Marshal(bodies[0]["tools"])
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &tools); err != nil {
				t.Fatal(err)
			}
			if len(tools) != 1 {
				t.Fatalf("tools %s", b)
			}
			var got any
			if raw := tools[0]["external_web_access"]; raw != nil {
				if err := json.Unmarshal(raw, &got); err != nil {
					t.Fatal(err)
				}
			}
			if got != tc.want {
				t.Errorf("external_web_access = %v, want %v; tools %s", got, tc.want, b)
			}
		})
	}
}
