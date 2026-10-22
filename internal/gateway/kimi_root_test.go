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

// #2923 quotes Kimi's refusal verbatim. The envelope is OpenAI-shaped;
// the server checks the actual schema and accepts only its repaired form.
const kimiRootRefusal = `{"error":{"message":"tools.function.parameters.type is required and must be \"object\"","type":"invalid_request_error"}}`

func TestKimiUntypedToolRoot(t *testing.T) {
	schema := `{"properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}`
	for _, c := range []struct {
		api    string
		refuse bool
	}{
		{"chat", true}, {"responses", true}, {"chat", false}, {"responses", false},
	} {
		for name, body := range map[string]string{
			"chat":      `{"model":"kimi/m","stream":true,"messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"read","parameters":` + schema + `}}]}`,
			"responses": `{"model":"kimi/m","stream":false,"input":"hi","tools":[{"type":"function","name":"read","parameters":` + schema + `}]}`,
			"messages":  `{"model":"kimi/m","stream":true,"max_tokens":100,"messages":[{"role":"user","content":"hi"}],"tools":[{"name":"read","input_schema":` + schema + `}]}`,
		} {
			t.Run(name+" to "+c.api+map[bool]string{true: " refusing", false: " accepting"}[c.refuse], func(t *testing.T) {
				fresh(t)
				var mu sync.Mutex
				var seen []string
				up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					b, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
						return
					}
					var q struct {
						Stream bool `json:"stream"`
						Tools  []struct {
							Parameters json.RawMessage `json:"parameters"`
							Function   struct {
								Parameters json.RawMessage `json:"parameters"`
							} `json:"function"`
						} `json:"tools"`
					}
					if err := json.Unmarshal(b, &q); err != nil || len(q.Tools) != 1 {
						t.Errorf("tools: %s (%v)", b, err)
						w.WriteHeader(500)
						return
					}
					ps := q.Tools[0].Parameters
					if len(ps) == 0 {
						ps = q.Tools[0].Function.Parameters
					}
					mu.Lock()
					seen = append(seen, string(ps))
					mu.Unlock()
					var params map[string]any
					json.Unmarshal(ps, &params)
					if c.refuse && params["type"] != "object" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(400)
						io.WriteString(w, kimiRootRefusal)
						return
					}
					if c.api == "responses" && q.Stream {
						w.Header().Set("Content-Type", "text/event-stream")
						io.WriteString(w, sse(`data: {"type":"response.output_text.delta","delta":"ok"}`, `data: {"type":"response.completed","response":{"id":"r1","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}}`))
						return
					}
					answer(w, r)
				}))
				t.Cleanup(up.Close)
				p := provider.Provider{ID: "kimi", Name: "Kimi", Key: "k", Models: []string{"m"}}
				if c.api == "chat" {
					p.Chat = up.URL + "/v1"
				} else {
					p.Responses = up.URL + "/v1"
				}
				if err := provider.Save(p); err != nil {
					t.Fatal(err)
				}
				s := New()
				path := "/v1/" + name
				if name == "chat" {
					path = "/v1/chat/completions"
				}
				for turn := range 2 {
					rec := httptest.NewRecorder()
					s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", path, strings.NewReader(body)))
					if rec.Code != 200 || !strings.Contains(rec.Body.String(), "ok") {
						t.Fatalf("turn %d: %d %s", turn, rec.Code, rec.Body)
					}
					mu.Lock()
					got := append([]string(nil), seen...)
					mu.Unlock()
					want := turn + 1
					if c.refuse {
						want++
					}
					if len(got) != want {
						t.Fatalf("turn %d: %d upstream calls, want %d", turn, len(got), want)
					}
					if strings.Contains(got[0], `"type":"object"`) {
						t.Fatalf("schema changed before refusal: %s", got[0])
					}
					var fixed map[string]any
					json.Unmarshal([]byte(got[len(got)-1]), &fixed)
					if c.refuse && fixed["type"] != "object" {
						t.Fatalf("root not repaired: %s", got[len(got)-1])
					}
					if !c.refuse && fixed["type"] != nil {
						t.Fatalf("accepting vendor schema changed: %s", got[len(got)-1])
					}
					delete(fixed, "type")
					var original map[string]any
					json.Unmarshal([]byte(schema), &original)
					if jsonOf(fixed) != jsonOf(original) {
						t.Fatalf("schema constraints changed: %s", got[len(got)-1])
					}
				}
			})
		}
	}
}

func TestKimiRootRefusalWordings(t *testing.T) {
	for _, body := range []string{kimiRootRefusal, `tools.function.parameters.type is required and must be "object"`} {
		if !rootUnionRefusal.MatchString(body) {
			t.Errorf("not recognized: %s", body)
		}
	}
	for _, body := range []string{
		`tools.function.parameters.properties.path.type is required and must be "object"`,
		`tools.function.parameters.type is required and must be "string"`,
		`message.type is required and must be "object"`,
	} {
		if rootUnionRefusal.MatchString(body) {
			t.Errorf("unrelated refusal matched: %s", body)
		}
	}
}
