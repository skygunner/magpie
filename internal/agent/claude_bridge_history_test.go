package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// #5656: the subscription bridge's later history entry is not another use
// of the user's Claude Code, while real user prompts still count.
func TestClaudeLastUsedLeavesOutBridgeHistory(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	name := "magpie-claude-work"
	if runtime.GOOS != "windows" {
		name += "-" + strconv.Itoa(os.Getuid())
	}
	bridge := filepath.Join(os.TempDir(), name)
	user := time.Unix(1700000000, 0)
	for _, c := range []struct {
		name    string
		project string
		want    time.Time
	}{
		{"bridge", bridge, user},
		{"ordinary project", filepath.Join(home, "project"), user.Add(time.Minute)},
		{"similar project", bridge + "-user", user.Add(time.Minute)},
	} {
		t.Run(c.name, func(t *testing.T) {
			var data []byte
			for _, row := range []struct {
				Project   string `json:"project"`
				Display   string `json:"display"`
				Timestamp int64  `json:"timestamp"`
			}{
				{filepath.Join(home, "project"), "hello", user.UnixMilli()},
				{c.project, "Perform a web search for the query: test", user.Add(time.Minute).UnixMilli()},
			} {
				b, err := json.Marshal(row)
				if err != nil {
					t.Fatal(err)
				}
				data = append(data, append(b, '\n')...)
			}
			if err := os.WriteFile(filepath.Join(dir, "history.jsonl"), data, 0o600); err != nil {
				t.Fatal(err)
			}
			if got := claude(home).LastUsed(); !got.Equal(c.want) {
				t.Fatalf("last use %v, want %v", got, c.want)
			}
		})
	}
}
