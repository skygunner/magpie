package gateway

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

const reasoningReadCall = `{"id":"call_1","type":"function","function":{"name":"read","arguments":"{}"}}`

// carriesBothReasoning says whether a message an upstream was sent carries
// the thinking in both of DeepSeek's spellings, which a self-hosted
// engine's serde reads as one field twice (#3165).
func carriesBothReasoning(m json.RawMessage) bool {
	return bytes.Contains(m, []byte(`"reasoning":`)) && bytes.Contains(m, []byte(`"reasoning_content":`))
}

// A tool turn that carries its own reasoning_content loses the reasoning
// echoed beside it: withReasoningOnToolTurns leaves such a message as the
// client sent it (1bb8143b), so both spellings reached a self-hosted
// DeepSeek engine, whose serde reads them as one field and turns the whole
// request away as "duplicate field `reasoning`" (#3165, ggbdpq's face 1).
// The thinking goes once, in reasoning_content, the spelling the replay
// path exists to send.
func TestChatToolTurnReasoningBothSpellingsKeptOnce(t *testing.T) {
	f := &fake{t: t, reply: chatWholeOK, ctype: "application/json", refuse: selfHostedRefuseDuplicate}
	up := setup(t, provider.Chat, f)
	chatProvider(t, up.URL, provider.Provider{ID: "ds-alias", Models: []string{"deepseek-v4"}})
	srv := New()
	for _, tt := range []struct {
		name string
		turn string
		want string
	}{
		{
			name: "own reasoning_content beside echoed reasoning",
			turn: `{"role":"assistant","content":null,"tool_calls":[` + reasoningReadCall + `],"reasoning_content":"Own thinking.","reasoning":"Echoed thinking."}`,
			want: `{"role":"assistant","content":null,"tool_calls":[` + reasoningReadCall + `],"reasoning_content":"Own thinking."}`,
		},
		{
			// the pair is refused on the second spelling of the field, so
			// it goes even beside an empty reasoning_content
			name: "own empty reasoning_content beside echoed reasoning",
			turn: `{"role":"assistant","content":null,"tool_calls":[` + reasoningReadCall + `],"reasoning_content":"","reasoning":"Echoed thinking."}`,
			want: `{"role":"assistant","content":null,"tool_calls":[` + reasoningReadCall + `],"reasoning_content":""}`,
		},
		{
			// the landed fold's own face, as it was: reasoning alone moves in
			name: "thinking sent in reasoning alone moves in",
			turn: `{"role":"assistant","content":null,"tool_calls":[` + reasoningReadCall + `],"reasoning":"Echoed thinking."}`,
			want: `{"role":"assistant","content":null,"tool_calls":[` + reasoningReadCall + `],"reasoning_content":"Echoed thinking."}`,
		},
		{
			// reasoning_details is a list of typed entries, not another name
			// for the text, and stays (1bb8143b)
			name: "own reasoning_content and reasoning_details, no reasoning",
			turn: `{"role":"assistant","content":null,"tool_calls":[` + reasoningReadCall + `],"reasoning_content":"Own thinking.","reasoning_details":[{"type":"reasoning.encrypted","data":"gAAAA"}]}`,
			want: `{"role":"assistant","content":null,"tool_calls":[` + reasoningReadCall + `],"reasoning_content":"Own thinking.","reasoning_details":[{"type":"reasoning.encrypted","data":"gAAAA"}]}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f.calls = 0
			code, body := postTo(t, srv, "/v1/chat/completions", `{"model":"ds-alias/deepseek-v4","stream":false,"tools":[{"type":"function","function":{"name":"read","parameters":{"type":"object","properties":{}}}}],"messages":[
				{"role":"user","content":"read note.txt"},
				`+tt.turn+`,
				{"role":"tool","tool_call_id":"call_1","content":"hello"}]}`)
			if code != 200 || f.calls != 1 {
				t.Fatalf("%d after %d calls: %s", code, f.calls, body)
			}
			ms := sentMessages(t, f.got)
			sameJSON(t, ms[1], tt.want)
			if carriesBothReasoning(ms[1]) {
				t.Errorf("both spellings reached the engine: %s", f.got)
			}
		})
	}
}

// A body with no tool calls is relayed as the client sent it, so both
// spellings on an ordinary message reach a self-hosted DeepSeek engine too
// (#3165, ggbdpq's face 2), and its serde turns the request away as
// "duplicate field `reasoning`". The message keeps reasoning_content alone;
// one that carries a single spelling is left exactly as it came.
func TestChatBothReasoningSpellingsRelayed(t *testing.T) {
	f := &fake{t: t, reply: chatWholeOK, ctype: "application/json", refuse: selfHostedRefuseDuplicate}
	up := setup(t, provider.Chat, f)
	chatProvider(t, up.URL, provider.Provider{ID: "ds-alias", Models: []string{"deepseek-v4"}})
	srv := New()
	for _, tt := range []struct {
		name string
		turn string
		want string
	}{
		{
			name: "both spellings on a reply",
			turn: `{"role":"assistant","content":"It is 12:00.","reasoning":"the thinking","reasoning_content":"the thinking"}`,
			want: `{"role":"assistant","content":"It is 12:00.","reasoning_content":"the thinking"}`,
		},
		{
			name: "both spellings, reasoning_content written first",
			turn: `{"role":"assistant","content":"It is 12:00.","reasoning_content":"the thinking","reasoning":"the thinking"}`,
			want: `{"role":"assistant","content":"It is 12:00.","reasoning_content":"the thinking"}`,
		},
		{
			// the message's own reasoning_content is the one kept
			name: "both spellings disagree, reasoning_content kept",
			turn: `{"role":"assistant","content":"It is 12:00.","reasoning":"older thinking","reasoning_content":"the thinking"}`,
			want: `{"role":"assistant","content":"It is 12:00.","reasoning_content":"the thinking"}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f.calls = 0
			code, body := postTo(t, srv, "/v1/chat/completions", `{"model":"ds-alias/deepseek-v4","stream":false,"messages":[
				{"role":"user","content":"what time is it?"},
				`+tt.turn+`,
				{"role":"user","content":"thanks"}]}`)
			if code != 200 || f.calls != 1 {
				t.Fatalf("%d after %d calls: %s", code, f.calls, body)
			}
			ms := sentMessages(t, f.got)
			sameJSON(t, ms[1], tt.want)
			if carriesBothReasoning(ms[1]) {
				t.Errorf("both spellings reached the engine: %s", f.got)
			}
		})
	}

	// one spelling alone goes byte for byte
	turn := `{"role":"assistant","content":"Hello!","reasoning":"Greet back."}`
	f.calls = 0
	code, body := postTo(t, srv, "/v1/chat/completions", `{"model":"ds-alias/deepseek-v4","stream":false,"messages":[{"role":"user","content":"hi"},`+turn+`,{"role":"user","content":"bye"}]}`)
	if code != 200 || f.calls != 1 {
		t.Fatalf("%d after %d calls: %s", code, f.calls, body)
	}
	if got := sentMessages(t, f.got)[1]; !bytes.Equal(got, []byte(turn)) {
		t.Errorf("a message with one spelling changed:\nsent %s\nwant %s", got, turn)
	}
}
