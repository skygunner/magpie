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
)

// codexCollabTools is Codex's collaboration namespace, spawn_agent in it.
const codexCollabTools = `"tools":[
	{"type":"function","name":"exec_command","parameters":{"type":"object"}},
	{"type":"namespace","name":"collaboration","description":"Tools in the collaboration namespace.","tools":[
		{"type":"function","name":"spawn_agent","description":"Spawns an agent.","parameters":{"type":"object","properties":{"message":{"type":"string","encrypted":true}}}}
	]}
]`

// #3183 (john-zeng-233): a vendor serving the Responses API natively is
// relayed Codex's request as it came, and its reply goes back as it came:
// the call to collaboration.spawn_agent reached Codex without the empty
// encrypted_function_args list, and Codex read the plain-text message as
// sealed — the subagent was handed its task in an encrypted_content and
// asked what the task was. Codex picks DirectPlaintextMessage only when
// the list is there and empty, as the translated path's callTo gives it.
func TestNativeRelayNamespacedCallSaysUnsealed(t *testing.T) {
	fresh(t)
	var mu sync.Mutex
	var sent []map[string]any
	call := `{"type":"function_call","id":"fc_1","call_id":"c9","name":"spawn_agent","namespace":"collaboration","arguments":"{\"message\":\"go\"}","status":"completed"}`
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var q map[string]any
		json.Unmarshal(b, &q)
		mu.Lock()
		sent = append(sent, q)
		mu.Unlock()
		done := `{"id":"r1","object":"response","status":"completed","output":[` + call + `],"usage":{"input_tokens":5,"output_tokens":3}}`
		if q["stream"] != true {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, done)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse(
			`event: response.created`+"\n"+`data: {"type":"response.created","response":{"id":"r1","object":"response","status":"in_progress","output":[]}}`,
			`event: response.output_item.added`+"\n"+`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"c9","name":"spawn_agent","namespace":"collaboration","arguments":""}}`,
			`event: response.output_item.done`+"\n"+`data: {"type":"response.output_item.done","output_index":0,"item":`+call+`}`,
			`event: response.completed`+"\n"+`data: {"type":"response.completed","response":`+done+`}`))
	}))
	defer up.Close()
	if err := provider.Save(provider.Provider{ID: "ucloud", Name: "UCloud", Key: "k", Models: []string{"m"}, Responses: up.URL + "/v1"}); err != nil {
		t.Fatal(err)
	}
	s := New()
	for _, stream := range []bool{true, false} {
		mu.Lock()
		sent = nil
		mu.Unlock()
		body := `{"model":"ucloud/m","stream":` + map[bool]string{true: "true", false: "false"}[stream] + `,` + codexCollabTools + `,"input":[{"type":"message","role":"user","content":"go"}]}`
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body)))
		if rec.Code != 200 {
			t.Fatalf("stream %v: %d %s", stream, rec.Code, rec.Body)
		}
		// relayed, not translated: the namespace went up as Codex sent it
		mu.Lock()
		if len(sent) != 1 {
			t.Fatalf("stream %v: the vendor was asked %d times", stream, len(sent))
		}
		q := sent[0]
		mu.Unlock()
		tools, _ := q["tools"].([]any)
		if len(tools) != 2 {
			t.Fatalf("stream %v: the vendor was offered %v", stream, q["tools"])
		}
		if tm, _ := tools[1].(map[string]any); tm["type"] != "namespace" || tm["name"] != "collaboration" {
			t.Fatalf("stream %v: the request was translated, its tools went as %v", stream, q["tools"])
		}

		// every function_call Codex reads says its arguments are plain text
		var items []map[string]any
		if stream {
			for _, line := range strings.Split(rec.Body.String(), "\n") {
				data, ok := strings.CutPrefix(line, "data: ")
				if !ok {
					continue
				}
				var ev struct {
					Item     map[string]any `json:"item"`
					Response struct {
						Output []map[string]any `json:"output"`
					} `json:"response"`
				}
				if err := json.Unmarshal([]byte(data), &ev); err != nil {
					t.Fatalf("%v: %s", err, data)
				}
				if ev.Item != nil {
					items = append(items, ev.Item)
				}
				items = append(items, ev.Response.Output...)
			}
		} else {
			var res struct {
				Output []map[string]any `json:"output"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
				t.Fatalf("%v: %s", err, rec.Body)
			}
			items = res.Output
		}
		if len(items) == 0 || stream && len(items) != 3 {
			t.Fatalf("stream %v: %s", stream, rec.Body)
		}
		for _, it := range items {
			sealed, ok := it["encrypted_function_args"].([]any)
			if it["name"] != "spawn_agent" || it["namespace"] != "collaboration" || !ok || len(sealed) != 0 {
				t.Fatalf("stream %v: Codex was given %v", stream, it)
			}
		}
	}
}

// The seal a reply carries is the seal the client reads: an upstream that
// answered a namespaced call with its arguments sealed hands that seal on
// untouched, and a call to a tool outside every namespace says nothing
// about seals either way (#3183).
func TestNativeRelayLeavesOtherCallsBe(t *testing.T) {
	fresh(t)
	sealed := `{"type":"function_call","id":"fc_1","call_id":"c9","name":"spawn_agent","namespace":"collaboration","arguments":"","encrypted_function_args":[{"content":"gAAAAAsealed"}],"status":"completed"}`
	plain := `{"type":"function_call","id":"fc_2","call_id":"c8","name":"exec_command","arguments":"{}","status":"completed"}`
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"r1","object":"response","status":"completed","output":[`+sealed+`,`+plain+`],"usage":{"input_tokens":5,"output_tokens":3}}`)
	}))
	defer up.Close()
	if err := provider.Save(provider.Provider{ID: "ucloud", Name: "UCloud", Key: "k", Models: []string{"m"}, Responses: up.URL + "/v1"}); err != nil {
		t.Fatal(err)
	}
	s := New()
	body := `{"model":"ucloud/m","stream":false,` + codexCollabTools + `,"input":[{"type":"message","role":"user","content":"go"}]}`
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body)))
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	var res struct {
		Output []map[string]any `json:"output"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Output) != 2 {
		t.Fatalf("%s", rec.Body.String())
	}
	if got, ok := res.Output[0]["encrypted_function_args"].([]any); !ok || len(got) != 1 {
		t.Fatalf("the seal was lost: %v", res.Output[0])
	}
	if _, ok := res.Output[1]["encrypted_function_args"]; ok {
		t.Fatalf("a plain call was given a seal: %v", res.Output[1])
	}
}

// #3183's other half: a plaintext V2 task must reach a native third-party
// Responses endpoint as a user message, with its sender and image intact.
func TestNativeRelayPlainAgentMessage(t *testing.T) {
	fresh(t)
	f := &fake{reply: `{"id":"r1","object":"response","status":"completed","output":[],"usage":{"input_tokens":5,"output_tokens":0}}`}
	setup(t, provider.Responses, f)
	const input = `{"type":"agent_message","author":"/root","recipient":"/root/worker","internal_chat_message_metadata_passthrough":{"secret":"drop me"},"content":[{"type":"input_text","text":"Reply PONG"},{"type":"input_image","image_url":"data:image/png;base64,AAAA"}]}`
	code, response := post(t, "/v1/responses", `{"model":"fake/m1","stream":false,"input":[`+input+`]}`)
	if code != 200 {
		t.Fatalf("%d %s", code, response)
	}
	var q struct {
		Input []map[string]any `json:"input"`
	}
	if err := json.Unmarshal(f.got, &q); err != nil {
		t.Fatal(err)
	}
	if len(q.Input) != 1 {
		t.Fatalf("input: %s", f.got)
	}
	m := q.Input[0]
	if m["type"] != "message" || m["role"] != "user" {
		t.Fatalf("task wasn't converted: %s", f.got)
	}
	b, _ := json.Marshal(m["content"])
	if string(b) != `[{"text":"From /root to /root/worker\n\n","type":"input_text"},{"text":"Reply PONG","type":"input_text"},{"image_url":"data:image/png;base64,AAAA","type":"input_image"}]` {
		t.Fatalf("task content: %s", b)
	}
	// The native OpenAI path and sealed/unsupported items keep their bytes.
	for _, body := range []string{
		`{"input":[` + input + `]}`,
		`{"input":[{"type":"agent_message","content":[{"type":"encrypted_content","encrypted_content":"gAAAAAsealed"}]}]}`,
	} {
		if got := forVendor(provider.Provider{Responses: "https://api.openai.com/v1"}, []byte(body)); string(got) != body {
			t.Fatalf("changed OpenAI: %s", got)
		}
	}
	for _, body := range []string{
		`{"input":[{"type":"agent_message","content":[{"type":"encrypted_content","encrypted_content":"gAAAAAsealed"}]}]}`,
		`{"input":[{"type":"agent_message","content":[{"type":"other","text":"unknown"}]}]}`,
	} {
		if got := forVendor(provider.Provider{Responses: "https://relay.example/v1"}, []byte(body)); string(got) != body {
			t.Fatalf("changed opaque task: %s", got)
		}
	}

}

func TestNativeRelayOpenAICallUnchanged(t *testing.T) {
	fresh(t)
	if err := provider.Save(provider.Provider{ID: "native", Name: "Native", Key: "k", Models: []string{"m"}, Responses: "https://api.openai.com/v1"}); err != nil {
		t.Fatal(err)
	}
	const call = `{"type":"function_call","id":"fc_1","call_id":"c1","name":"spawn_agent","namespace":"collaboration","arguments":"{}","status":"completed"}`
	s := New()
	s.client = &http.Client{Transport: countTransport(func(r *http.Request) (*http.Response, error) {
		io.Copy(io.Discard, r.Body)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"r1","object":"response","status":"completed","output":[` + call + `]}`))}, nil
	})}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{"model":"native/m","stream":false,"input":"hi",`+codexCollabTools+`}`)))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"call_id":"c1"`) || strings.Contains(rec.Body.String(), "encrypted_function_args") {
		t.Fatalf("native OpenAI call changed: %d %s", rec.Code, rec.Body)
	}
}
