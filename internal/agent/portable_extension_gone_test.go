package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// A portable editor's extension keeps the shared agent folder in use (#4672).
func TestPortableVSCodeExtensionsKeepAgents(t *testing.T) {
	for _, tc := range []struct{ id, folder, extension string }{
		{"claude", ".claude", "anthropic.claude-code"},
		{"codex", ".codex", "openai.chatgpt"},
		{"cline", ".cline", "saoudrizwan.claude-dev"},
	} {
		t.Run(tc.id, func(t *testing.T) {
			home := t.TempDir()
			if resolved, err := filepath.EvalSymlinks(home); err == nil {
				home = resolved
			}
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
			t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
			judgeOffWindows(t)
			noApps(t)
			portableVSCode(t, home, true, false)
			dir := filepath.Join(home, tc.folder)
			writeT(t, filepath.Join(dir, "settings.json"), "{}")
			a := &Agent{ID: tc.id, Name: tc.id, Bin: "magpie-test-gone-" + tc.id, Dir: dir, Path: filepath.Join(dir, "settings.json")}
			if !a.Detected() || !a.CLIMissing() {
				t.Fatalf("no extension: detected %v missing %v", a.Detected(), a.CLIMissing())
			}
			exts := filepath.Join(home, "Editor With Spaces", "data", "extensions")
			mkdirT(t, filepath.Join(exts, "unrelated.extension-1.0.0"))
			if !a.CLIMissing() {
				t.Fatal("unrelated extension hid missing CLI")
			}
			mkdirT(t, filepath.Join(exts, tc.extension+"-1.0.0"))
			if a.CLIMissing() {
				t.Fatal("portable extension exists but CLI is marked missing")
			}
		})
	}
}

func TestPortableVSCodeChannelsKeepCodex(t *testing.T) {
	for _, k := range []vscodeKind{vscodeStable, vscodeInsiders, vscodiumKind} {
		t.Run(k.id, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			judgeOffWindows(t)
			noApps(t)
			portableVSCode(t, home, true, false)
			if k.bin != "code" {
				bin := filepath.Join(home, "Editor With Spaces", "bin")
				from, to := "code", k.bin
				if runtime.GOOS == "windows" {
					from += ".cmd"
					to += ".cmd"
				}
				if err := os.Rename(filepath.Join(bin, from), filepath.Join(bin, to)); err != nil {
					t.Fatal(err)
				}
			}
			writeT(t, filepath.Join(home, ".codex", "config.toml"), "")
			a := &Agent{ID: "codex", Bin: "magpie-test-gone-codex", Dir: filepath.Join(home, ".codex"), Path: filepath.Join(home, ".codex", "config.toml")}
			if !a.CLIMissing() {
				t.Fatalf("%s without an extension is not missing", k.id)
			}
			mkdirT(t, filepath.Join(home, "Editor With Spaces", "data", "extensions", "openai.chatgpt-1.0.0"))
			if a.CLIMissing() {
				t.Fatalf("%s portable extension ignored", k.id)
			}
		})
	}
}
