package gateway

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func TestReviewV2RelayFixtureIsValidJSON(t *testing.T) {
	for _, line := range strings.Split(relayReply(), "\n") {
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || data == "[DONE]" {
			continue
		}
		if !json.Valid([]byte(data)) {
			t.Fatalf("relayReply emitted invalid JSON: %s", data)
		}
	}
}

func TestReviewV2ResponsesEndings(t *testing.T) {
	for _, tc := range []struct{ name, delta, finish, status, reason string }{
		{"text", `{"content":"Done."}`, "stop", "completed", ""},
		{"tool-only", `{"tool_calls":[{"index":0,"id":"call_review","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.go\"}"}}]}`, "tool_calls", "completed", ""},
		{"reasoning-length", `{"reasoning_content":"Review thoughts."}`, "length", "incomplete", "max_output_tokens"},
		{"filter-text", `{"content":"Review refusal."}`, "content_filter", "incomplete", "content_filter"},
		{"filter-empty", `{}`, "content_filter", "incomplete", "content_filter"},
	} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stream=%v", tc.name, stream), func(t *testing.T) {
				f := &fake{t: t, reply: sse(`data: {"id":"c_review","choices":[{"index":0,"delta":`+tc.delta+`}]}`, `data: {"id":"c_review","choices":[{"index":0,"delta":{},"finish_reason":"`+tc.finish+`"}],"usage":{"prompt_tokens":10,"completion_tokens":3}}`, `data: [DONE]`)}
				setup(t, provider.Chat, f)
				code, out := post(t, "/v1/responses", fmt.Sprintf(`{"model":"fake/m1","stream":%v,"input":[{"role":"user","content":"continue"}]}`, stream))
				if tc.name == "filter-empty" {
					if code != 400 || !strings.Contains(out, "content_filter") || strings.Contains(out, "an empty reply") {
						t.Fatalf("empty filter should retain pre-stream refusal: %d %s", code, out)
					}
					return
				}
				if code != 200 {
					t.Fatalf("HTTP %d %s", code, out)
				}
				var response map[string]any
				if stream {
					terminals := 0
					for _, line := range strings.Split(out, "\n") {
						data, ok := strings.CutPrefix(line, "data: ")
						if !ok || data == "[DONE]" {
							continue
						}
						var event map[string]any
						if err := json.Unmarshal([]byte(data), &event); err != nil {
							t.Fatal(err)
						}
						if typ, _ := event["type"].(string); typ == "response.completed" || typ == "response.incomplete" || typ == "response.failed" {
							terminals++
							if typ != "response."+tc.status {
								t.Errorf("terminal type %q want response.%s", typ, tc.status)
							}
							response, _ = event["response"].(map[string]any)
						}
					}
					if terminals != 1 {
						t.Fatalf("terminal count=%d: %s", terminals, out)
					}
				} else if err := json.Unmarshal([]byte(out), &response); err != nil {
					t.Fatal(err)
				}
				if response["status"] != tc.status {
					t.Fatalf("status=%v want %s: %s", response["status"], tc.status, out)
				}
				if tc.reason != "" {
					details, _ := response["incomplete_details"].(map[string]any)
					if details["reason"] != tc.reason {
						t.Errorf("reason=%v want %s", details["reason"], tc.reason)
					}
				}
				if response["error"] != nil {
					t.Errorf("unexpected response error=%v", response["error"])
				}
				if f.calls != 1 {
					t.Errorf("positive control retried %d calls", f.calls)
				}
				output, _ := json.Marshal(response["output"])
				switch tc.name {
				case "text":
					if !strings.Contains(string(output), "Done.") {
						t.Errorf("text missing: %s", output)
					}
				case "tool-only":
					if !strings.Contains(string(output), `"type":"function_call"`) || !strings.Contains(string(output), `"call_id":"call_review"`) || !strings.Contains(string(output), `"name":"read_file"`) || !strings.Contains(string(output), `a.go`) {
						t.Errorf("tool output missing: %s", output)
					}
				}
			})
		}
	}
}
