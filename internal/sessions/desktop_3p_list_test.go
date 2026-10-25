package sessions

import (
	"os"
	"path/filepath"
	"testing"
)

// Claude Desktop's third-party mode writes Claude Code records with the
// entrypoint "claude-desktop-3p" (#3117, quiode). The list read them as
// Claude Code's: the Desktop tab was left with the gateway figures alone,
// and Desktop's tools, skills and active time counted under Claude Code.
// The reading is by the entrypoint's prefix, so Desktop's sessions, listed
// or counted, are Desktop's whatever mode it runs in.
func TestClaudeDesktop3pListedAsDesktop(t *testing.T) {
	claude, _ := setup(t)
	p := filepath.Join(claude, "projects", "-work-app", "p3.jsonl")
	if err := os.WriteFile(p, []byte(
		`{"type":"user","timestamp":"2026-10-24T10:00:00.000Z","message":{"role":"user","content":"hello"},"cwd":"/work/app","sessionId":"p3"}`+"\n"+
			`{"parentUuid":"p","isSidechain":false,"type":"assistant","uuid":"u1","timestamp":"2026-10-24T10:00:05.000Z","message":{"id":"m1","role":"assistant","model":"claude-sonnet-5-5","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":10,"cache_creation_input_tokens":2,"cache_read_input_tokens":3,"output_tokens":4}},"requestId":"req_1","sessionId":"p3","cwd":"/work/app","entrypoint":"claude-desktop-3p"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	Reset()
	ss := List(0)
	var found *Session
	for i := range ss {
		if ss[i].ID == "p3" {
			found = &ss[i]
		}
	}
	if found == nil {
		t.Fatalf("the 3p session is not listed: %+v", ss)
	}
	if found.Agent != "claude-desktop" {
		t.Fatalf("a claude-desktop-3p session lists as agent %q, want claude-desktop", found.Agent)
	}
}
