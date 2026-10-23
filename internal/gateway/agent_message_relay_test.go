package gateway

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func relayReply() string {
	return sse(
		`data: {"type":"response.completed","response":{"id":"r1","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"PONG"}]}],"usage":{"input_tokens":10,"output_tokens":1}}}`,
		`data: [DONE]`)
}

// A subagent's plain agent_message reaches a provider of OpenAI's public
// Responses as the receiving agent's user turn, not as the ChatGPT
// backend's own item such a provider drops unheard-of: a subagent on
// DeepSeek started with nothing to do (#3374, bigTsingLung — the pair the
// reporter posted, the same task as a user message and as an
// agent_message).
func TestAgentMessageRelaysAsUserTurn(t *testing.T) {
	f := &fake{t: t, reply: relayReply()}
	setup(t, provider.Responses, f)
	s := New()
	task := "Message Type: NEW_TASK\nTask name: /root/format_probe\nSender: /root\nPayload:\nReply with exactly FORMAT_PROBE_OK_842. Do not ask for a task."
	tb, _ := json.Marshal(task)
	body := `{"model":"fake/m1","stream":false,"input":[` +
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"You are ready to receive a task."}]},` +
		`{"type":"agent_message","id":"amsg_format_probe_842","author":"/root","recipient":"/root/format_probe","content":[{"type":"input_text","text":` + string(tb) + `}]}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", CodexPath+"/responses", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer chatgpt-token")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 || f.calls != 1 {
		t.Fatalf("status %d, upstream calls %d", rec.Code, f.calls)
	}
	var sent struct {
		Input []map[string]any `json:"input"`
	}
	if err := json.Unmarshal(f.got, &sent); err != nil {
		t.Fatal(err)
	}
	if len(sent.Input) != 2 {
		t.Fatalf("input = %s", f.got)
	}
	last := sent.Input[1]
	if last["type"] != "message" || last["role"] != "user" || strings.Contains(string(f.got), "agent_message") {
		t.Fatalf("the agent_message went as it came: %s", f.got)
	}
	parts, _ := last["content"].([]any)
	first, _ := parts[0].(map[string]any)
	if first["text"] != task {
		t.Fatalf("the task text was lost: %s", f.got)
	}
	for _, k := range []string{"author", "recipient", "id"} {
		if _, ok := last[k]; ok {
			t.Fatalf("the backend's own field %s stayed on the message: %s", k, f.got)
		}
	}
}

// A sealed agent_message (its encrypted_content) is left as it came:
// what it says is not magpie's to read. Serve turns a sealed task away
// from a provider that cannot open it (hasSealedAgentMessage), so this is
// the rewrite's own guard.
func TestSealedAgentMessageStaysSealedInRelay(t *testing.T) {
	body := []byte(`{"model":"m","input":[{"type":"agent_message","id":"amsg_1","author":"/root","recipient":"/root/w","content":[{"type":"encrypted_content","encrypted_content":"gAAAAABc9zF0Q2lzY28"}]}]}`)
	if got := agentMessagesAsUserTurns(body); string(got) != string(body) {
		t.Fatalf("the sealed item was touched:\n%s", got)
	}
}
