package sessions

import (
	"path/filepath"
	"testing"
)

// Desktop Code tab files are Claude sessions and also match Desktop gateway IDs (#4333).
func TestNativeIDsDesktopCodeTab(t *testing.T) {
	for _, entry := range []string{"claude-desktop", "claude-desktop-3p"} {
		t.Run(entry, func(t *testing.T) {
			d := setupCalls(t)
			path := filepath.Join(d.claude, "projects", "-work-app", "desktop.jsonl")
			writeLines(t, path, `{"type":"user","timestamp":"`+stamp(0)+`","message":{"role":"user","content":"hello"},"cwd":"/work/app","sessionId":"desktop","entrypoint":"`+entry+`"}`)
			writeLines(t, filepath.Join(d.claude, "projects", "-work-app", "cli.jsonl"), `{"type":"user","timestamp":"`+stamp(0)+`","message":{"role":"user","content":"hello"},"cwd":"/work/app","sessionId":"cli","entrypoint":"cli"}`)
			want := map[string]bool{"claude-desktop": true, "claude": true}
			check := func(ids map[string]map[string]bool) {
				t.Helper()
				if !ids["claude-desktop"]["desktop"] || ids["claude-desktop"]["cli"] || !ids["claude"]["desktop"] || !ids["claude"]["cli"] {
					t.Fatalf("native IDs: %v", ids)
				}
			}
			old := quickBytes
			quickBytes = 0
			t.Cleanup(func() { behindRuns.Wait(); quickBytes = old })
			check(Manage(want, func([]AgentCount, map[string]map[string]bool) string { return "" }).Native)
			behindRuns.Wait()
			quickBytes = old
			check(NativeIDs(want))
			only := NativeIDs(map[string]bool{"claude-desktop": true})
			if !only["claude-desktop"]["desktop"] || only["claude-desktop"]["cli"] || len(only) != 1 {
				t.Fatalf("desktop-only IDs: %v", only)
			}
			writeLines(t, path, `{"type":"user","timestamp":"`+stamp(0)+`","message":{"role":"user","content":"changed"},"cwd":"/work/app","sessionId":"desktop","entrypoint":"cli"}`)
			quickBytes = 0
			ids := Manage(want, func([]AgentCount, map[string]map[string]bool) string { return "" }).Native
			behindRuns.Wait()
			quickBytes = old
			if ids["claude-desktop"]["desktop"] || !ids["claude"]["desktop"] {
				t.Fatalf("updated native IDs: %v", ids)
			}
		})
	}
}
