package proc

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yetone/magpie/internal/testenv"
)

// A package run with npx (@deepseek-ai/dsh's dsh, #3303, Moody-Sin) lives
// under ~/.npm/_npx/<hash>, its bins in that tree's .bin - never on PATH,
// so magpie said CLI not found while dsh itself ran. The cache's bins are
// among the user's tool folders now; nothing in it is run to find this,
// only its files are looked at.
func TestNpxCacheBinsAreUserToolFolders(t *testing.T) {
	home := os.Getenv(testenv.Marker)
	if home == "" {
		home = t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
	}
	t.Setenv("PATH", t.TempDir())
	bin := filepath.Join(home, ".npm", "_npx", "ebf017b61addb8bd", "node_modules", ".bin")
	os.MkdirAll(bin, 0o755)
	os.WriteFile(filepath.Join(bin, "dsh"), []byte("#!/usr/bin/env node\n"), 0o755)
	if runtime.GOOS == "windows" {
		os.WriteFile(filepath.Join(bin, "dsh.cmd"), nil, 0o755)
	}
	dirs := userDirs(true)
	found := false
	for _, d := range dirs {
		if d == bin {
			found = true
		}
	}
	if !found {
		t.Fatalf("the npx cache's .bin is not among the user's tool folders: %v", dirs)
	}
	if p := toolIn("dsh", runtime.GOOS, dirs); p == "" {
		t.Error("dsh not found in the npx cache's .bin")
	}
	// a folder isn't the tool
	os.RemoveAll(filepath.Join(bin, "dsh"))
	os.MkdirAll(filepath.Join(bin, "dsh"), 0o755)
	if p := toolIn("dsh", runtime.GOOS, dirs); p != "" {
		t.Errorf("a folder named dsh was taken for the tool: %q", p)
	}
	// no cache, no tool
	os.RemoveAll(filepath.Join(home, ".npm"))
	for _, d := range userDirs(true) {
		if d == bin {
			t.Fatal("the emptied cache is still listed")
		}
	}
	if p := toolIn("dsh", runtime.GOOS, userDirs(true)); p != "" {
		t.Errorf("dsh found without a cache: %q", p)
	}
}
