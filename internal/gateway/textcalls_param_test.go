package gateway

import (
	"encoding/json"
	"strings"
	"testing"
)

// #917 (Moody-Sin, DeepSeek through group/deepseek in Pi): a call written
// half as Hermes' block and half as DeepSeek's template — the name as JSON
// closed with >, then <parameter name="command"> with no close, the reply
// ending there — reached Pi as text and the turn ended. It is the call.
func TestTextWrittenParamCallRelayed(t *testing.T) {
	body := `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"hi"}],` + textCallTools + `}`
	text, calls, finish, raw := relayedChat(t, body,
		chunkOf("Let me run it.\n\n<tool_call>{\"name\":\"bash\">\n<param"),
		chunkOf("eter name=\"command\">cd /e/Project/synara && node .review-tmp\n"),
		stopChunk)
	var bash struct{ Command string }
	if text != "Let me run it.\n\n" || json.Unmarshal([]byte(calls["bash"]), &bash) != nil ||
		bash.Command != "cd /e/Project/synara && node .review-tmp" || finish != "tool_calls" || strings.Contains(raw, "parameter") {
		t.Fatalf("text %q calls %v finish %q\n%s", text, calls, finish, raw)
	}

	// closed, with } and Qwen's <parameter=k>, a JSON value and a raw
	// multi-line one, then a second call
	text, calls, finish, raw = relayedChat(t, body,
		chunkOf("<tool_call>{\"name\":\"edit\"}\n<parameter=path>a.ts</parameter>\n<parameter=edits>[{\"oldText\":\"x\",\"newText\":\"y\"}]</parameter>\n</tool_call>"),
		chunkOf("<tool_call>{\"name\":\"bash\"}<parameter name=\"command\">for f in *; do\n  echo \"$f\"\ndone</parameter></tool_call>"),
		stopChunk)
	if text != "" || calls["edit"] != `{"edits":[{"newText":"y","oldText":"x"}],"path":"a.ts"}` || finish != "tool_calls" {
		t.Fatalf("text %q calls %v finish %q\n%s", text, calls, finish, raw)
	}
	if json.Unmarshal([]byte(calls["bash"]), &bash) != nil || bash.Command != "for f in *; do\n  echo \"$f\"\ndone" {
		t.Fatalf("bash %q\n%s", calls["bash"], raw)
	}

	// a tool not offered, or a name with no parameters after it, stays text
	block := "<tool_call>{\"name\":\"rm_rf\">\n<parameter name=\"path\">/</parameter>"
	text, calls, finish, _ = relayedChat(t, body, chunkOf(block), stopChunk)
	if text != block || len(calls) != 0 || finish != "stop" {
		t.Fatalf("unknown tool: text %q calls %v finish %q", text, calls, finish)
	}
	block = "<tool_call>{\"name\":\"bash\"> and then"
	text, calls, finish, _ = relayedChat(t, body, chunkOf(block), stopChunk)
	if text != block || len(calls) != 0 || finish != "stop" {
		t.Fatalf("no parameters: text %q calls %v finish %q", text, calls, finish)
	}
}

// #2789 (turtle4512, trae-cn): the model writes its calls as
// <call:tool name="Bash">…</call:tool> or <tool_call name="Bash">…</tool_call>,
// with the arguments as XML children, and sometimes as {"arguments":{…}}.
// Those must be rescued like the Hermes/DeepSeek spellings.
func TestTextCallXMLSpellingsRelayed(t *testing.T) {
	body := `{"model":"a/m","stream":true,"messages":[{"role":"user","content":"hi"}],` + textCallTools + `}`

	// <call:tool name="Bash">\n<command>ls</command>\n</call:tool>
	text, calls, finish, raw := relayedChat(t, body,
		chunkOf("Let me run it.\n\n<call:tool name=\"Bash\">\n<command>ls</command>\n</call:tool>"),
		stopChunk)
	var bash struct{ Command string }
	if text != "Let me run it.\n\n" || json.Unmarshal([]byte(calls["bash"]), &bash) != nil ||
		bash.Command != "ls" || finish != "tool_calls" {
		t.Fatalf("call:tool: text %q calls %v finish %q\n%s", text, calls, finish, raw)
	}

	// <tool_call name="Bash">\n<parameter name="command">ls</parameter>
	text, calls, finish, raw = relayedChat(t, body,
		chunkOf("Let me run it.\n\n<tool_call name=\"Bash\">\n<parameter name=\"command\">ls</parameter>"),
		stopChunk)
	if text != "Let me run it.\n\n" || json.Unmarshal([]byte(calls["bash"]), &bash) != nil ||
		bash.Command != "ls" || finish != "tool_calls" {
		t.Fatalf("tool_call: text %q calls %v finish %q\n%s", text, calls, finish, raw)
	}

	// a name the request did not offer stays text
	block := "<call:tool name=\"rm_rf\">\n<command>rm -rf /</command>\n</call:tool>"
	text, calls, finish, _ = relayedChat(t, body, chunkOf(block), stopChunk)
	if text != block || len(calls) != 0 || finish != "stop" {
		t.Fatalf("unknown tool: text %q calls %v finish %q", text, calls, finish)
	}

	// {"name":"Edit","arguments":{"arguments":{"file_path":"a.ts"}}} peels one level
	text, calls, finish, raw = relayedChat(t, body,
		chunkOf("<tool_call>{\"name\":\"edit\",\"arguments\":{\"arguments\":{\"file_path\":\"a.ts\",\"oldText\":\"x\",\"newText\":\"y\"}}}</tool_call>"),
		stopChunk)
	var ed struct {
		FilePath string `json:"file_path"`
		OldText  string `json:"oldText"`
		NewText  string `json:"newText"`
	}
	if text != "" || json.Unmarshal([]byte(calls["edit"]), &ed) != nil ||
		ed.FilePath != "a.ts" || ed.OldText != "x" || ed.NewText != "y" || finish != "tool_calls" {
		t.Fatalf("peeled arguments: text %q calls %v finish %q\n%s", text, calls, finish, raw)
	}
}
