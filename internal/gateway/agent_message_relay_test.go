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
		`data: {"type":"response.completed","response":{"id":"r1","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"PONG"}]}],"usage":{"input_tokens":10,"output_tokens":1}}}`,
		`data: [DONE]`)
}

// A subagent's plain agent_message reaches a provider of OpenAI's public
// Responses as the receiving agent's user turn, author and recipient said
// first and its images carried, through the relay's agent_message reading
// (forVendor, #3183) — from Codex's own endpoint, where the review's pair
// showed the item going missing on a DeepSeek preset (#3374, bigTsingLung:
// the same task as a user message worked, as an agent_message the subagent
// asked for a task).
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
	head, _ := first["text"].(string)
	if head != "From /root to /root/format_probe\n\n" {
		t.Fatalf("the sender and recipient were lost: %s", f.got)
	}
	second, _ := parts[1].(map[string]any)
	if second["text"] != task {
		t.Fatalf("the task text was lost: %s", f.got)
	}
	for _, k := range []string{"author", "recipient", "id"} {
		if _, ok := last[k]; ok {
			t.Fatalf("the backend's own field %s stayed on the message: %s", k, f.got)
		}
	}
}

// A sealed agent_message (its encrypted_content) never reaches a
// provider that cannot open it: serve refuses the request with the
// sealed-task explanation (hasSealedAgentMessage, sealedTaskError) —
// what the item says is not magpie's to read, and forwarding the
// ciphertext as a user turn would hand the subagent nothing.
func TestSealedAgentMessageIsRefusedNotForwarded(t *testing.T) {
	f := &fake{t: t, reply: relayReply()}
	setup(t, provider.Responses, f)
	s := New()
	body := `{"model":"fake/m1","stream":false,"input":[{"type":"agent_message","id":"amsg_1","author":"/root","recipient":"/root/w","content":[{"type":"encrypted_content","encrypted_content":"gAAAAABc9zF0Q2lzY28"}]}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", CodexPath+"/responses", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer chatgpt-token")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 400 || f.calls != 0 {
		t.Fatalf("status %d, upstream calls %d, body %s", rec.Code, f.calls, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "sealed") {
		t.Fatalf("the refusal says nothing of the seal: %s", rec.Body.String())
	}
}
