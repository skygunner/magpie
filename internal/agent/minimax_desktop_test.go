package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// miniMaxDesktopPlist is MiniMax Code Desktop 3.1.10's Info.plist (MiniMax
// Code-3.1.10-arm64-mac.zip from filecdn.minimax.chat, the China build) cut
// to the keys magpie reads, in their order and indentation. The
// international build (file.cdn.minimax.io) is the same bundle name with
// the id com.minimax.agent (app-identity.js).
const miniMaxDesktopPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>CFBundleDisplayName</key>
    <string>MiniMax Code</string>
    <key>CFBundleExecutable</key>
    <string>MiniMax Code</string>
    <key>CFBundleIdentifier</key>
    <string>com.minimax.agent.cn</string>
    <key>CFBundleName</key>
    <string>MiniMax Code</string>
    <key>CFBundleShortVersionString</key>
    <string>3.1.10</string>
  </dict>
</plist>
`

// 教父 on Discord (MiniMax Code 3.1.10, magpie v0.1.2629 and v0.1.2679):
// MiniMax Code turned on in magpie with a model picked, and the desktop app
// went on with the user's own MiniMax-M3.1-Flash; Sessions showed nothing.
// Desktop's window keeps the model last picked in it and shows that, not
// config.yaml's defaultModel, while that model is still listed. So with the
// desktop app installed, either build, magpie's notice says to restart it
// and pick a magpie model once in its menu, and its version is shown.
// Without it the mcode advice stays as it was.
func TestMiniMaxDesktopSaysPickInTheApp(t *testing.T) {
	home, _ := miniMaxHome(t)
	apps := t.TempDir()
	wasFolders, wasSpot, wasRunning := appFolders, spotlightApps, Running
	appFolders = func() []string { return []string{apps} }
	spotlightApps = func(string) []string { return nil }
	t.Cleanup(func() { appFolders, spotlightApps, Running = wasFolders, wasSpot, wasRunning })
	mcode := false
	Running = func(...string) bool { return mcode }

	a := miniMax(home)
	if n := a.Notice(); n != "" {
		t.Fatalf("no app, no mcode running: notice %q", n)
	}
	mcode = true
	if n := a.Notice(); n != noticeMinimax.String() {
		t.Fatalf("no app, mcode running: notice %q", n)
	}

	for _, id := range []string{"com.minimax.agent.cn", "com.minimax.agent"} {
		bundle := filepath.Join(apps, "MiniMax Code.app")
		os.RemoveAll(bundle)
		if err := os.MkdirAll(filepath.Join(bundle, "Contents"), 0o755); err != nil {
			t.Fatal(err)
		}
		plist := strings.Replace(miniMaxDesktopPlist, "com.minimax.agent.cn", id, 1)
		if err := os.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, running := range []bool{false, true} {
			mcode = running
			if n := a.Notice(); n != noticeMinimaxDesktop.String() {
				t.Errorf("%s installed, mcode running %v: notice %q", id, running, n)
			}
		}
		if v := a.AppVersion(); v != "3.1.10" {
			t.Errorf("%s installed: app version %q", id, v)
		}
	}

	// another app under the same name is not MiniMax Code's
	plist := strings.Replace(miniMaxDesktopPlist, "com.minimax.agent.cn", "com.example.other", 1)
	if err := os.WriteFile(filepath.Join(apps, "MiniMax Code.app", "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	mcode = false
	if n := a.Notice(); n != "" {
		t.Errorf("another bundle id: notice %q", n)
	}
}

// #4870 (教父's report, read from the 3.1.10 download): MiniMax Code
// Desktop lets the user move its data folder in its settings — its data-dir
// relocation, Windows only there. Desktop keeps the parent in its own
// store (config.localRuntimeDataParentDir in the build's userData
// electron-store: MiniMax/minimax-agent-cn-config.json for the China
// build, MiniMax Agent/minimax-agent-config.json for the overseas one) and
// pins its runtime to <parent>/.minimax through MINIMAX_DATA_DIR, so the
// switch magpie writes has to land there, not in ~/.minimax Desktop no
// longer reads. Without Desktop installed its store is dead state, and the
// default stays; a MINIMAX_DATA_DIR of the user's own still wins, as it
// does for Desktop's own runtime.
func TestMiniMaxDesktopMovedDataDir(t *testing.T) {
	home, _ := miniMaxHome(t)
	apps, stores := t.TempDir(), t.TempDir()
	cn := filepath.Join(stores, "MiniMax", "minimax-agent-cn-config.json")
	en := filepath.Join(stores, "MiniMax Agent", "minimax-agent-config.json")
	moved := filepath.Join(home, "mmdata", ".minimax")
	for _, f := range []string{cn, en} {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	wasFolders, wasSpot, wasStores := appFolders, spotlightApps, miniMaxDesktopStores
	appFolders = func() []string { return []string{apps} }
	spotlightApps = func(string) []string { return nil }
	miniMaxDesktopStores = func() []string { return []string{cn, en} }
	t.Cleanup(func() { appFolders, spotlightApps, miniMaxDesktopStores = wasFolders, wasSpot, wasStores })
	installApp := func(id string) {
		bundle := filepath.Join(apps, "MiniMax Code.app")
		os.RemoveAll(bundle)
		if err := os.MkdirAll(filepath.Join(bundle, "Contents"), 0o755); err != nil {
			t.Fatal(err)
		}
		plist := strings.Replace(miniMaxDesktopPlist, "com.minimax.agent.cn", id, 1)
		if err := os.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	setStore := func(f, parent string) {
		if parent == "" {
			if err := os.WriteFile(f, []byte(`{}`), 0o644); err != nil {
				t.Fatal(err)
			}
			return
		}
		b, err := json.Marshal(map[string]any{"config": map[string]any{"localRuntimeDataParentDir": parent}})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Desktop installed, the folder moved in its settings: its data folder
	// is the moved one, and the row's config.yaml there
	installApp("com.minimax.agent.cn")
	setStore(cn, filepath.Join(home, "mmdata"))
	if got := MiniMaxDir(home); got != moved {
		t.Fatalf("the moved folder is not read: %q", got)
	}
	if got := miniMax(home).Path; got != filepath.Join(moved, "config.yaml") {
		t.Fatalf("the switch would land in %q", got)
	}

	// the overseas build's store says it too
	os.Remove(cn)
	setStore(en, filepath.Join(home, "mmdata"))
	if got := MiniMaxDir(home); got != moved {
		t.Fatalf("the overseas build's store is not read: %q", got)
	}

	// no parent in the store: the default
	setStore(en, "")
	if got := MiniMaxDir(home); got != filepath.Join(home, ".minimax") {
		t.Fatalf("the default is not kept: %q", got)
	}

	// Desktop gone: its store is dead state
	installApp("com.example.other")
	setStore(cn, filepath.Join(home, "mmdata"))
	if got := MiniMaxDir(home); got != filepath.Join(home, ".minimax") {
		t.Fatalf("a store of another app is read: %q", got)
	}

	// the user's own MINIMAX_DATA_DIR wins over the store, as Desktop's
	// own runtime takes it over the config
	installApp("com.minimax.agent.cn")
	t.Setenv("MINIMAX_DATA_DIR", filepath.Join(home, "envdir"))
	if got := MiniMaxDir(home); got != filepath.Join(home, "envdir") {
		t.Fatalf("MINIMAX_DATA_DIR does not win: %q", got)
	}
}
