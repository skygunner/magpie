package library

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// #1903: a skill set aside across disks (rename can't cross) whose removal
// stops partway (a folder it can't unlink inside, a file held open) is left
// half-emptied where it was, and setAside must still say where the whole
// copy it made first is kept — not return no path at all.
func TestSetAsideLeftBehindSaysWhereTheCopyIs(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root unlinks inside a read-only folder all the same")
	}
	sandbox(t)
	at, err := os.MkdirTemp("/dev/shm", "magpie-skill-")
	if err != nil {
		t.Skip("no writable shared-memory filesystem:", err)
	}
	t.Cleanup(func() { os.Chmod(at, 0o755); os.RemoveAll(at) })
	write(t, filepath.Join(at, "SKILL.md"), "mine")
	write(t, filepath.Join(at, "refs/deep/note.md"), "kept")
	stuck := filepath.Join(at, "refs")
	if err := os.Chmod(stuck, 0o555); err != nil {
		t.Fatal(err)
	}
	// the move must reach its copy-and-remove path: rename across the two
	// filesystems refuses
	probe := filepath.Join(t.TempDir(), "probe")
	if err := os.Rename(at, probe); err == nil {
		t.Skip("temporary folders are on the same filesystem")
	} else if !errors.Is(err, syscall.EXDEV) {
		t.Fatalf("probe rename: %v", err)
	}

	dst, err := setAside("claude", at)
	if err == nil {
		t.Fatal("a move that couldn't take the skill away whole, and no error")
	}
	if dst == "" {
		t.Fatal("no path named for where the whole copy went")
	}
	if !strings.Contains(err.Error(), dst) {
		t.Fatalf("the error doesn't say the copy is at %s: %v", dst, err)
	}
	if _, err := os.Lstat(at); err != nil {
		t.Fatalf("what couldn't be taken away is gone anyway: %v", err)
	}
	for _, f := range []string{"SKILL.md", "refs/deep/note.md"} {
		if b, err := os.ReadFile(filepath.Join(dst, f)); err != nil {
			t.Errorf("the copy at %s isn't whole: %v", dst, err)
		} else if f == "SKILL.md" && string(b) != "mine" || f != "SKILL.md" && string(b) != "kept" {
			t.Errorf("%s in the copy reads %q", f, b)
		}
	}
}
