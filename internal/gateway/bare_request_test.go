package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// The short messages reported in #3035, with NVIDIA's generic envelope.
const nvidiaBareRequest = `{"message":"Invalid request","type":"invalid_request_error","code":400}`
const senseNovaBareRequest = `{"error":{"message":"inference request is invalid","type":"invalid_request_error","param":null,"code":null}}`

func TestBareRequestRefused(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   bool
	}{
		{400, nvidiaBareRequest, true},
		{400, senseNovaBareRequest, true},
		{422, `{"message":"Invalid request"}`, true},
		{422, `{"error":{"message":"Invalid request","code":"422"}}`, true},
		{400, `{"type":"error","error":{"message":"  INVALID REQUEST  ","code":"invalid_request_error"}}`, true},
		{400, `{"message":"\u0049nvalid request"}`, true},
		{400, "inference request is invalid", true},
		{400, "Invalid request", true},
		{200, nvidiaBareRequest, false},
		{401, nvidiaBareRequest, false},
		{403, nvidiaBareRequest, false},
		{429, nvidiaBareRequest, false},
		{500, nvidiaBareRequest, false},
		{400, `{"message":"Invalid request parameters"}`, false},
		{400, `{"error":"Invalid request content: A tool_choice was set on the request but no tools were specified."}`, false},
		{400, `{"message":"Invalid request: messages: field required"}`, false},
		{400, `{"message":"request is invalid"}`, false},
		{400, `{"message":"Invalid request","param":"messages"}`, false},
		{400, `{"message":"Invalid request","code":"context_length_exceeded"}`, false},
		{400, `{"message":"Invalid request","type":"quota_exceeded"}`, false},
		{400, `{"message":"Invalid request","detail":"messages required"}`, false},
		{400, `{"message":"Invalid request","details":[{"field":"messages"}]}`, false},
		{400, `{"error":{"message":"Invalid request","code":"invalid_api_key"}}`, false},
		{400, `{"error":{"message":"Invalid request","type":"authentication_error"}}`, false},
		{400, `{"error":{"message":"Invalid request","param":"temperature"}}`, false},
		{400, `{"error":{"message":"Invalid request","status":"RESOURCE_EXHAUSTED"}}`, false},
		{400, `{"code":"insufficient_quota","error":{"message":"Invalid request"}}`, false},
		{400, `{"error":{"message":"Invalid request"},"param":"tools"}`, false},
		{400, `{"message":"Invalid request"`, false},
		{400, `{"detail":{"message":"Invalid request"}}`, false},
		{400, `{"message":"other error mentioning Invalid request"}`, false},
	} {
		if got := bareRequestRefused(tc.status, []byte(tc.body)); got != tc.want {
			t.Errorf("bareRequestRefused(%d, %s) = %v, want %v", tc.status, tc.body, got, tc.want)
		}
	}
}

// The failed request moves on, but neither another request nor a sibling
// model loses its working key. Run through error translation as well as relay.
func TestBareRequestFailsOverWithoutRest(t *testing.T) {
	for _, from := range []provider.Protocol{provider.Chat, provider.Anthropic, provider.Responses} {
		for _, stream := range []bool{false, true} {
			for _, refusal := range []string{nvidiaBareRequest, senseNovaBareRequest} {
				t.Run(fmt.Sprintf("%s/stream=%v/%s", from, stream, refusal), func(t *testing.T) {
					fresh(t)
					var bad, good, spare atomic.Int32
					serveOn(t, "a", "key", []string{"bad", "good"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var req struct{ Model string }
						json.NewDecoder(r.Body).Decode(&req)
						if req.Model == "bad" {
							bad.Add(1)
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(400)
							io.WriteString(w, refusal)
							return
						}
						good.Add(1)
						w.Header().Set("Content-Type", "text/event-stream")
						io.WriteString(w, chatStreamOK)
					}))
					serveOn(t, "b", "key", []string{"m"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						spare.Add(1)
						w.Header().Set("Content-Type", "text/event-stream")
						io.WriteString(w, chatStreamOK)
					}))
					for _, g := range []provider.Group{
						{Name: "Bad", Members: []string{"a/bad", "b/m"}, Routing: provider.Ordered},
						{Name: "Good", Members: []string{"a/good", "b/m"}, Routing: provider.Ordered},
					} {
						if err := provider.SaveGroup(g); err != nil {
							t.Fatal(err)
						}
					}
					path, request := refusalErrorAsk(from)
					var body map[string]any
					json.Unmarshal([]byte(request), &body)
					body["stream"] = stream
					s := New()
					for _, model := range []string{"group/bad", "group/bad", "group/good"} {
						body["model"] = model
						b, _ := json.Marshal(body)
						code, reply := sendRefusalError(t, s, path, string(b))
						if code != 200 {
							t.Fatalf("%s: %d %s", model, code, reply)
						}
						route := s.trace.routes[len(s.trace.routes)-1]
						if model == "group/bad" && (len(route.Tries) != 2 || route.Tries[0].Fail != failOther || route.Tries[0].Rest != nil) {
							t.Fatalf("tries: %+v", route.Tries)
						}
						routed.Lock()
						failures := routed.failures["a"] + routed.failures["a/bad"]
						routed.Unlock()
						if failures != 0 {
							t.Fatalf("request refusal counted against the key/model: %d", failures)
						}
					}
					if bad.Load() != 2 || good.Load() != 1 || spare.Load() != 2 {
						t.Fatalf("calls: bad=%d good=%d spare=%d", bad.Load(), good.Load(), spare.Load())
					}
				})
			}
		}
	}
}

// Metadata must be judged before a client's error shape drops it, even
// when the resulting message would otherwise be indistinguishable.
func TestBareRequestKeepsSpecificErrors(t *testing.T) {
	for _, from := range []provider.Protocol{provider.Chat, provider.Anthropic, provider.Responses} {
		for _, refusal := range []string{
			`{"message":"Invalid request: messages: field required"}`,
			`{"message":"Invalid request parameters"}`,
			`{"message":"Invalid request","code":"invalid_argument"}`,
			`{"message":"Invalid request","type":"validation_error"}`,
			`{"error":{"message":"Invalid request","param":"tools"}}`,
		} {
			t.Run(fmt.Sprintf("%s/%s", from, refusal), func(t *testing.T) {
				fresh(t)
				a := &scripted{replies: []reply{{400, "", refusal}}}
				b := &scripted{replies: []reply{{200, "text/event-stream", chatStreamOK}}}
				scriptedOn(t, "a", provider.Chat, a)
				scriptedOn(t, "b", provider.Chat, b)
				if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/m", "b/m"}, Routing: provider.Ordered}); err != nil {
					t.Fatal(err)
				}
				path, body := refusalErrorAsk(from)
				s := New()
				code, got := sendRefusalError(t, s, path, body)
				if code != 400 || a.n != 1 || b.n != 0 {
					t.Fatalf("%d %s (a=%d b=%d)", code, got, a.n, b.n)
				}
			})
		}
	}
}

func TestBareRequestAllMembersRefuse(t *testing.T) {
	fresh(t)
	a := &scripted{replies: []reply{{422, "", `{"message":"Invalid request"}`}}}
	b := &scripted{replies: []reply{{400, "", senseNovaBareRequest}}}
	scriptedOn(t, "a", provider.Chat, a)
	scriptedOn(t, "b", provider.Chat, b)
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/m", "b/m"}, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	s := New()
	code, body := postAs(t, s, "", `{"model":"group/g","messages":[{"role":"user","content":"hi"}]}`)
	if code != 400 || !strings.Contains(body, "inference request is invalid") || a.n != 1 || b.n != 1 {
		t.Fatalf("%d %s (a=%d b=%d)", code, body, a.n, b.n)
	}
	for _, try := range s.trace.routes[len(s.trace.routes)-1].Tries {
		if try.Rest != nil || try.Fail != failOther {
			t.Fatalf("try: %+v", try)
		}
	}
}

func TestBareRequestDecisionGroup(t *testing.T) {
	fresh(t)
	a := &scripted{replies: []reply{{400, "", nvidiaBareRequest}, {200, "", `{"model":"jev-latest","answers":{}}`}}}
	b := &scripted{replies: []reply{{200, "", `{"model":"jev-latest","answers":{}}`}}}
	for id, upstream := range map[string]*scripted{"a": a, "b": b} {
		srv := httptest.NewServer(upstream)
		t.Cleanup(srv.Close)
		if err := provider.Save(provider.Provider{ID: id, Name: id, Key: "k", Decide: srv.URL + "/v1"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/jev-latest", "b/jev-latest"}, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	s := New()
	for range 2 {
		code, body := sendRefusalError(t, s, "/v1/systemone", `{"model":"group/g","state":{"message":"hi"},"questions":{"intent":{"type":"choice"}}}`)
		if code != 200 {
			t.Fatalf("%d %s", code, body)
		}
	}
	if a.n != 2 || b.n != 1 {
		t.Fatalf("a=%d b=%d", a.n, b.n)
	}
	if try := s.trace.routes[0].Tries[0]; try.Rest != nil || try.Fail != failOther {
		t.Fatalf("try: %+v", try)
	}
}
