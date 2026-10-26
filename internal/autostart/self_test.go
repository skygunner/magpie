package autostart

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// The record names the magpie on PATH when that is this same file: a
// package manager's link (Nix's profile, Homebrew's bin) still leads to
// the version it updates to, while the versioned file it points at is
// gone once the old version is collected (#3621).
func TestSelfNamesThePathLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("links")
	}
	t.Setenv("APPIMAGE", "")
	bin := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(bin, "magpie")
	if err := os.Symlink(exe, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	got, err := self()
	if err != nil {
		t.Fatal(err)
	}
	if got != link {
		t.Fatalf("self() = %q, want the link on PATH %q", got, link)
	}
}

// A build `go run` deletes when it ends is no program to start at login:
// the record naming it starts nothing from the next login on. This test
// binary is itself such a build (`go test` puts it in a go-build temp
// folder), so self() of this file is refused.
func TestSelfRefusesAGoRunBuild(t *testing.T) {
	t.Setenv("APPIMAGE", "")
	// no magpie on PATH: what self() answers is about this file alone
	t.Setenv("PATH", t.TempDir())
	if got, err := self(); err == nil {
		t.Fatalf("self() took %q, a build `go run` deletes when it ends", got)
	}
}
