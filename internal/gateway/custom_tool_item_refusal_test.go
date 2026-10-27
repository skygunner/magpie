package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// vllmCustomToolCallRefusal is vLLM's Responses API turning a custom_tool_call
// input item away, the message as #3849's reporter got it from a local vLLM.
const vllmCustomToolCallRefusal = `{"error":{"code":null,"message":"Unsupported input item type: custom_tool_call (parameter=input)","param":"input","type":"invalid_request_error"}}`

// compactionWithCustomToolCall is a Codex compaction request that carries
// custom_tool_call items in its history (input array). The tools array is
// empty as codexInputAs clears it for compaction.
const compactionWithCustomToolCall = `{"model":%q,"stream":%t,"input":[` +
	`{"type":"custom_tool_call","id":"ctc_1","call_id":"call_1","name":"apply_patch","input":"print(1)"},` +
	`{"type":"custom_tool_call_output","call_id":"call_1","output":"done"},` +
	`{"role":"user","content":"Summarize this."}],` +
	`"tools":[]}`

// refusingVLLM is a Responses API that, like vLLM, refuses custom_tool_call
// input items, but accepts function calls.
type refusingVLLM struct {
	mu    sync.Mutex
	asked []map[string]any
	paths []string
}

func (u *refusingVLLM) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	var q map[string]any
	json.Unmarshal(b, &q)
	u.mu.Lock()
	u.asked = append(u.asked, q)
	u.paths = append(u.paths, r.URL.Path)
	u.mu.Unlock()
	items, _ := q["input"].([]any)
	for _, it := range items {
		if im, _ := it.(map[string]any); im != nil {
			typ, _ := im["type"].(string)
			if typ == "custom_tool_call" || typ == "custom_tool_call_output" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				io.WriteString(w, vllmCustomToolCallRefusal)
				return
			}
		}
	}
	call := `{"type":"function_call","id":"fc_1","call_id":"call_1","name":"apply_patch","arguments":"{\"input\":\"print(1)\"}","status":"completed"}`
	if q["stream"] == true {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse(
			`data: {"type":"response.created","response":{"id":"r1","status":"in_progress","output":[]}}`,
			`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"apply_patch","arguments":""}}`,
			`data: {"type":"response.function_call_arguments.delta","output_index":0,"item_id":"fc_1","delta":"{\"input\":\"print(1)\"}"}`,
			`data: {"type":"response.output_item.done","output_index":0,"item":`+call+`}`,
			`data: {"type":"response.completed","response":{"id":"r1","status":"completed","output":[`+call+`],"usage":{"input_tokens":9,"output_tokens":3}}}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"id":"r1","object":"response","status":"completed","output":[`+call+`],"usage":{"input_tokens":9,"output_tokens":3}}`)
}

// #3849 (yorhaha): a Codex compaction request carries custom_tool_call
// items in its history; the upstream vLLM refuses them with "Unsupported
// input item type: custom_tool_call". The refusal is recognized, the model
// is marked as refusing custom tools, and the request is retried translated
// on the same API: the custom tool calls become function calls and the
// summary is returned.
func TestCodexCompactionOverCustomToolCalls(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("stream %t", stream), func(t *testing.T) {
			fresh(t)
			noRests()
			t.Cleanup(noRests)
			refusing := &refusingVLLM{}
			ds := httptest.NewServer(refusing)
			t.Cleanup(ds.Close)
			var taking []map[string]any
			var mu sync.Mutex
			lax := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				var q map[string]any
				json.Unmarshal(b, &q)
				mu.Lock()
				taking = append(taking, q)
				mu.Unlock()
				answer(w, r)
			}))
			t.Cleanup(lax.Close)
			for _, p := range []provider.Provider{
				{ID: "vllm", Name: "vLLM", Key: "k", Models: []string{"m1"}, Responses: ds.URL + "/v1"},
				{ID: "lax", Name: "Lax", Key: "k", Models: []string{"m1"}, Responses: lax.URL + "/v1"},
			} {
				if err := provider.Save(p); err != nil {
					t.Fatal(err)
				}
			}
			s := New()
			ask := func(model string) *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/responses", strings.NewReader(fmt.Sprintf(compactionWithCustomToolCall, model, stream))))
				return rec
			}
			rec := ask("vllm/m1")
			if rec.Code != 200 {
				t.Fatalf("vllm: %d %s", rec.Code, rec.Body)
			}
			// the refusing vLLM was asked twice: once with custom_tool_call,
			// once translated with function_call
			if len(refusing.asked) != 2 || refusing.paths[1] != "/v1/responses" {
				t.Fatalf("vllm asked %d times, at %v", len(refusing.asked), refusing.paths)
			}
			items, _ := refusing.asked[1]["input"].([]any)
			if len(items) < 1 {
				t.Fatalf("second ask missing input: %v", refusing.asked[1])
			}
			// the second ask should have function_call, not custom_tool_call
			hasFunction := false
			for _, it := range items {
				if im, _ := it.(map[string]any); im != nil {
					if im["type"] == "function_call" {
						hasFunction = true
					}
					if im["type"] == "custom_tool_call" || im["type"] == "custom_tool_call_output" {
						t.Fatalf("second ask still has custom_tool_call: %v", im)
					}
				}
			}
			if !hasFunction {
				t.Fatalf("second ask has no function_call: %v", refusing.asked[1])
			}
			// the lax provider was never asked
			if len(taking) != 0 {
				t.Fatalf("lax was asked: %v", taking)
			}
		})
	}
}

// #3849: the customToolRefusal regex also matches vLLM's "Unsupported
// input item type: custom_tool_call" wording. Unrelated input item type
// errors must not match.
func TestCustomToolRefusalWordings_VLLM(t *testing.T) {
	for _, b := range []string{
		vllmCustomToolCallRefusal,
		`{"error":{"message":"Unsupported input item type: 'custom_tool_call' (parameter=input)"}}`,
		`{"error":{"message":"Invalid input item type: custom_tool_call_output (parameter=input)"}}`,
		`{"error":{"message":"Unknown item type: custom_tool_call in input"}}`,
	} {
		if !customToolRefusal.Match(wordsOf([]byte(b))) {
			t.Errorf("not known: %s", b)
		}
	}
	for _, b := range []string{
		`{"error":{"message":"Unsupported input item type: 'function_call' (parameter=input)"}}`,
		`{"error":{"message":"Invalid input item type: some_other_thing"}}`,
	} {
		if customToolRefusal.Match(wordsOf([]byte(b))) {
			t.Errorf("taken for one: %s", b)
		}
	}
}