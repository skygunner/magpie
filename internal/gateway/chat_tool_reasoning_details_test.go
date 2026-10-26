package gateway

import (
	"encoding/json"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// #3165: foldReasoningAliases handles reasoning, so reasoning_details must
// independently prove the tool-turn cleanup, including after a tool result.
func TestChatToolTurnReasoningDetailsCarriedOnce(t *testing.T) {
	for _, c := range []struct {
		name, own, want string
	}{
		{"injected", "", "Let me read it."},
		{"already present", `,"reasoning_content":"Keep my own thinking."`, "Keep my own thinking."},
	} {
		t.Run(c.name, func(t *testing.T) {
			// Given: pi's structured thinking, with a distinct canonical value.
			f := &fake{t: t, reply: chatWholeOK, ctype: "application/json"}
			up := setup(t, provider.Chat, f)
			chatProvider(t, up.URL, provider.Provider{ID: "ds", Models: []string{"deepseek-v4.1-flash"}})
			details := `,"reasoning_details":[{"type":"reasoning.text","text":"Let me read it.","signature":null,"format":"unknown","index":0}]`
			request := `{"model":"ds/deepseek-v4.1-flash","stream":false,"messages":[
				{"role":"user","content":"What time is it?"},
				{"role":"assistant","content":null,"tool_calls":[` + timeCall + `]` + details + c.own + `},
				{"role":"tool","tool_call_id":"call_1","content":"12:00"},
				{"role":"assistant","content":"It is 12:00."` + details + c.own + `}]}`

			// When: the real Chat passthrough sends the replay to its upstream.
			code, body := postTo(t, New(), "/v1/chat/completions", request)

			// Then: both assistant turns carry the thinking once, canonically.
			if code != 200 || f.calls != 1 {
				t.Fatalf("%d after %d calls: %s", code, f.calls, body)
			}
			messages := sentMessages(t, f.got)
			if len(messages) != 4 {
				t.Fatalf("sent %d messages, want 4: %s", len(messages), f.got)
			}
			for _, i := range []int{1, 3} {
				var m map[string]json.RawMessage
				if err := json.Unmarshal(messages[i], &m); err != nil {
					t.Fatal(err)
				}
				var think string
				if err := json.Unmarshal(m["reasoning_content"], &think); err != nil {
					t.Fatalf("message %d reasoning_content: %v", i, err)
				}
				if think != c.want {
					t.Errorf("message %d reasoning_content = %q, want %q", i, think, c.want)
				}
				if details, ok := m["reasoning_details"]; ok {
					t.Errorf("message %d still carries reasoning_details beside reasoning_content: %s", i, details)
				}
			}
		})
	}
}
