package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// DSH run only through npx (#3303, Moody-Sin): the package sits in
// ~/.npm/_npx/<hash>/node_modules, no dsh on PATH, and ~/.dsh is its
// config. magpie's model and thinking settings are file writes to
// cordis.patch.yml - they went nowhere, the CLI gate dead-ending the row
// before them, while dsh itself ran fine on its own DeepSeek. The files
// are written whatever the CLI's absence; nothing that would run the CLI
// is claimed.
func TestDshWithoutItsCLIWritesItsSettings(t *testing.T) {
	home, dir, web := dshRouteHome(t)
	dshKeyFixture(dir)
	os.WriteFile(web, []byte("# Your patch layer for this dsh profile.\n"), 0o644)
	// nothing named dsh anywhere on PATH
	empty := t.TempDir()
	t.Setenv("PATH", empty)
	t.Setenv("CODEX_HOME", "")
	a := dsh(home)
	f := a.Field("model")
	if f == nil {
		t.Fatalf("no model field: %+v", a)
	}
	if err := f.Set("magpie/deepseek/pro"); err != nil {
		t.Fatalf("the model could not be set without the CLI: %v", err)
	}
	b, _ := os.ReadFile(web)
	if !strings.Contains(string(b), "deepseek/pro") {
		t.Fatalf("the patch file was not written:\\n%s", b)
	}
}

// The CLI-missing gate the reporter met: with ~/.dsh in place and dsh
// only in the npx cache, the row was CLI not found and the settings were
// dead. The cache's bins are the user's tool folders now, so the gate
// opens the way it does when dsh is on PATH - and stays shut when it
// isn't anywhere (an uninstall left the folder, #843's own case).
func TestDshNpxCacheOpensTheCLIGate(t *testing.T) {
	home, dir, _ := dshRouteHome(t)
	t.Setenv("PATH", t.TempDir())
	bin := filepath.Join(home, ".npm", "_npx", "ebf017b61addb8bd", "node_modules", ".bin")
	os.MkdirAll(bin, 0o755)
	os.WriteFile(filepath.Join(bin, "dsh"), []byte("#!/usr/bin/env node\n"), 0o755)
	a := &Agent{ID: "dsh", Name: "DeepSeek Harness", Bin: "dsh", Dir: dir, Path: filepath.Join(dir, "profiles", "web", "cordis.patch.yml")}
	if !a.Detected() {
		t.Fatal("not detected with ~/.dsh in place")
	}
	if a.CLIMissing() {
		t.Fatal("dsh in the npx cache is still CLI not found")
	}
	os.RemoveAll(filepath.Join(home, ".npm"))
	if !a.CLIMissing() {
		t.Fatal("dsh nowhere on this machine is not CLI not found")
	}
}
