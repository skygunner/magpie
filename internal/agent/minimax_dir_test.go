package agent_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/sessions"
)

// #2998: both public readers must follow the same desktop bootstrap record.
func TestMiniMaxDirDesktopRecord(t *testing.T) {
	home, appdata := t.TempDir(), t.TempDir()
	moved := filepath.Join(t.TempDir(), "minimax data")
	override := filepath.Join(t.TempDir(), "explicit data")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", appdata)
	record := filepath.Join(appdata, "MiniMax", "minimax-agent-cn-config.json")
	if err := os.MkdirAll(filepath.Dir(record), 0o755); err != nil {
		t.Fatal(err)
	}
	movedJSON, err := json.Marshal(moved)
	if err != nil {
		t.Fatal(err)
	}
	valid := `{"config":{"localRuntimeDataParentDir":` + string(movedJSON) + `,"localRuntimePreviousDataParentDirs":["old parent"]}}`
	spacedJSON, err := json.Marshal("  " + moved + " \t")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, record, env, appdata, want string }{
		{"moved", valid, "", appdata, filepath.Join(moved, ".minimax")},
		{"whitespace record", `{"config":{"localRuntimeDataParentDir":` + string(spacedJSON) + `}}`, "", appdata, filepath.Join(moved, ".minimax")},
		{"explicit override", valid, override, appdata, override},
		{"whitespace override", valid, "  " + override + " ", appdata, filepath.Join(moved, ".minimax")},
		{"blank override", valid, " \t", appdata, filepath.Join(moved, ".minimax")},
		{"relative override", valid, "relative", appdata, filepath.Join(moved, ".minimax")},
		{"missing record", "", "", appdata, filepath.Join(home, ".minimax")},
		{"malformed record", "{", "", appdata, filepath.Join(home, ".minimax")},
		{"wrong type", `{"config":{"localRuntimeDataParentDir":42}}`, "", appdata, filepath.Join(home, ".minimax")},
		{"missing field", `{"config":{}}`, "", appdata, filepath.Join(home, ".minimax")},
		{"relative record", `{"config":{"localRuntimeDataParentDir":"relative"}}`, "", appdata, filepath.Join(home, ".minimax")},
		{"blank record", `{"config":{"localRuntimeDataParentDir":" \t"}}`, "", appdata, filepath.Join(home, ".minimax")},
		{"missing APPDATA", valid, "", "", filepath.Join(home, ".minimax")},
		{"relative APPDATA", valid, "", "relative", filepath.Join(home, ".minimax")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given: the app's documented settings file and an optional override.
			t.Setenv("APPDATA", tc.appdata)
			t.Setenv("MINIMAX_DATA_DIR", tc.env)
			if tc.record == "" {
				if err := os.Remove(record); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(record, []byte(tc.record), 0o600); err != nil {
				t.Fatal(err)
			}
			// When: configuration and session discovery resolve the data directory.
			gotAgent, gotSessions := agent.MiniMaxDir(home), sessions.MiniMaxDir()
			// Then: both use the same selected directory, with no helper dependency.
			if gotAgent != tc.want {
				t.Errorf("agent.MiniMaxDir = %q, want %q", gotAgent, tc.want)
			}
			if gotSessions != tc.want {
				t.Errorf("sessions.MiniMaxDir = %q, want %q", gotSessions, tc.want)
			}
		})
	}
}
