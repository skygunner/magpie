package gateway

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

const kiloToolCount400 = `{"error":{"message":"Kilo Gateway: Invalid value for 'tools': maximum number of items is 128.","type":"invalid_request_error"}}`

func TestGroupFailsOverAToolCountLimit(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			fresh(t)
			a := &fake{t: t, ctype: "application/json", reply: chatOK}
			a.refuse = func(body []byte) (int, string) {
				var request struct{ Tools []json.RawMessage }
				if err := json.Unmarshal(body, &request); err != nil {
					t.Error(err)
				}
				if len(request.Tools) > 128 {
					return 400, kiloToolCount400
				}
				return 0, ""
			}
			b := &fake{t: t, ctype: "application/json", reply: chatOK}
			if stream {
				b.ctype, b.reply = "text/event-stream", chatStreamOK
				a.ctype, a.reply = b.ctype, b.reply
			}
			twoProviders(t, a, b)
			if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"plan/m1", "spare/m2"}, Routing: provider.Ordered}); err != nil {
				t.Fatal(err)
			}
			tools := make([]json.RawMessage, 129)
			for i := range tools {
				tools[i] = json.RawMessage(fmt.Sprintf(`{"type":"function","function":{"name":"tool_%d","parameters":{"type":"object","properties":{}}}}`, i))
			}
			request := func(n int) string {
				body, err := json.Marshal(map[string]any{"model": "group/g", "messages": []any{map[string]string{"role": "user", "content": "hi"}}, "stream": stream, "tools": tools[:n]})
				if err != nil {
					t.Fatal(err)
				}
				return string(body)
			}
			s := New()
			for i := 1; i <= 2; i++ {
				if code, body := postAs(t, s, "", request(129)); code != 200 || a.calls != i || b.calls != i {
					t.Fatalf("request %d: %d %s (calls %d/%d)", i, code, body, a.calls, b.calls)
				}
				var sent struct{ Tools []json.RawMessage }
				if err := json.Unmarshal(b.got, &sent); err != nil || len(sent.Tools) != 129 {
					t.Fatalf("fallback lost tools: %d, %v", len(sent.Tools), err)
				}
				r := s.trace.routes[len(s.trace.routes)-1]
				if len(r.Tries) != 2 || r.Tries[0].Fail != failShape || r.Tries[0].Rest != nil {
					t.Fatalf("tries %+v", r.Tries)
				}
			}
			if code, body := postAs(t, s, "", request(128)); code != 200 || a.calls != 3 || b.calls != 2 {
				t.Fatalf("within the limit: %d %s (calls %d/%d)", code, body, a.calls, b.calls)
			}
			if err := provider.SaveGroup(provider.Group{Name: "Solo", Members: []string{"plan/m1"}, Routing: provider.Ordered}); err != nil {
				t.Fatal(err)
			}
			if code, body := postAs(t, s, "", strings.Replace(request(129), "group/g", "group/solo", 1)); code != 400 || !strings.Contains(body, "maximum number of items is 128") {
				t.Fatalf("no fallback: %d %s", code, body)
			}
		})
	}
}

func TestToolCountRefusalIsNarrow(t *testing.T) {
	for _, status := range []int{400, 422} {
		if !shapeRefused(status, []byte(kiloToolCount400)) || !retryable(status, []byte(kiloToolCount400)) {
			t.Errorf("tool count refusal at %d was terminal", status)
		}
	}
	for _, body := range []string{
		strings.ReplaceAll(kiloToolCount400, "'tools'", "'messages'"),
		`{"error":{"message":"tools: field required"}}`,
		`{"error":{"message":"Invalid schema for function 'x': expected an object"}}`,
		`{"error":{"message":"Invalid value for 'tools': maximum nesting depth is 10"}}`,
	} {
		if shapeRefused(400, []byte(body)) || retryable(400, []byte(body)) {
			t.Errorf("not a tool count refusal: %s", body)
		}
	}
	if shapeRefused(200, []byte(kiloToolCount400)) {
		t.Fatal("a successful response is not a refusal")
	}
}
