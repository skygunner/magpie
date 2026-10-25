package sessions

import (
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// Claude Desktop's third-party mode writes Claude Code records with the
// entrypoint "claude-desktop-3p" (#3117, quiode): its Code tab's sessions
// landed under Claude Code, so the Desktop tab showed the gateway figures
// alone and the CLI tab counted Desktop's tools, skills and active time.
// The Desktop reading is by the entrypoint's prefix, as Desktop's own
// deployment names it (3p is the third-party gateway mode).
func TestClaudeDesktop3pSessionsAreTheDesktops(t *testing.T) {
	d := setupCalls(t)
	writeLines(t, filepath.Join(d.claude, "projects", "-work-app", "p3.jsonl"),
		`{"type":"user","timestamp":"`+stamp(0)+`","message":{"role":"user","content":"hello"},"cwd":"/work/app","sessionId":"p3"}`,
		swap(swap(claudeMsg("d1", "claude-sonnet-5-5", 1, 2, 3, 4, 5), `"entrypoint":"cli"`, `"entrypoint":"claude-desktop-3p"`), `"sessionId":"sess1"`, `"sessionId":"p3"`),
		swap(swap(claudeMsg("d2", "claude-sonnet-5-5", 1, 2, 3, 4, 6), `"entrypoint":"cli"`, `"entrypoint":"claude-desktop-3p"`), `"sessionId":"sess1"`, `"sessionId":"p3"`),
	)
	cs := Calls(time.Time{})
	if len(cs) != 2 {
		t.Fatalf("calls: %+v", cs)
	}
	for _, c := range cs {
		if c.Agent != "claude-desktop" {
			t.Fatalf("a claude-desktop-3p call reads as agent %q, want claude-desktop", c.Agent)
		}
		if c.Session != "p3" || c.Tokens != (Tokens{1, 2, 3, 4, 0}) {
			t.Fatalf("call: %+v", c)
		}
	}
	// the CLI stays the CLI
	writeLines(t, filepath.Join(d.claude, "projects", "-work-app", "cli1.jsonl"),
		`{"type":"user","timestamp":"`+stamp(0)+`","message":{"role":"user","content":"hello"},"cwd":"/work/app","sessionId":"cli1"}`,
		swap(claudeMsg("c1", "claude-sonnet-5-5", 1, 2, 3, 4, 5), `"sessionId":"sess1"`, `"sessionId":"cli1"`))
	cs = Calls(time.Time{})
	cli := slices.IndexFunc(cs, func(c Call) bool { return c.Session == "cli1" })
	if cli < 0 || cs[cli].Agent != "claude" {
		t.Fatalf("the CLI call moved: %+v", cs)
	}
}
