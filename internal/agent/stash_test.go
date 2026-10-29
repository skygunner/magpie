package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/filelock"
	"github.com/yetone/magpie/internal/proc"
)

func TestStashCrossProcessLock(t *testing.T) {
	if os.Getenv("MAGPIE_STASH_LOCK_CHILD") == "1" {
		unlock, held := filelock.Lock(stashPath()+".lock", time.Second)
		if !held {
			t.Fatal("child could not acquire stash lock")
		}
		defer unlock()
		fmt.Println("LOCKED")
		_, _ = io.ReadAll(os.Stdin)
		return
	}
	stashHome(t)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := proc.CommandContext(ctx, exe, "-test.run=^TestStashCrossProcessLock$")
	cmd.Env = append(os.Environ(), "MAGPIE_STASH_LOCK_CHILD=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stdin.Close(); cmd.Wait() })
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "LOCKED\n" {
		t.Fatalf("child lock: %q, %v", line, err)
	}
	oldWait := stashWait
	stashWait = 20 * time.Millisecond
	t.Cleanup(func() { stashWait = oldWait })
	if err := stash(map[string]string{"agent.was": "original"}); err == nil {
		t.Fatal("stash wrote while another process held the lock")
	}
	if got := stashLoad()["agent.was"]; got != "" {
		t.Fatalf("stash changed while lock held: %q", got)
	}
}

func TestStashForgetIfUnchanged(t *testing.T) {
	stashHome(t)
	if err := stash(map[string]string{"model": "first"}); err != nil {
		t.Fatal(err)
	}
	if err := stash(map[string]string{"model": "newer"}); err != nil {
		t.Fatal(err)
	}
	forgetIfUnchanged("model", "first")
	if got := stashLoad()["model"]; got != "newer" {
		t.Fatalf("newer value removed: %q", got)
	}
	forgetIfUnchanged("model", "newer")
	if got := stashLoad()["model"]; got != "" {
		t.Fatalf("restored value not removed: %q", got)
	}
}

func TestPeekStashRejectsMalformedData(t *testing.T) {
	stashHome(t)
	if err := os.MkdirAll(filepath.Dir(stashPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`{"saved":`, `null`, `[]`} {
		if err := os.WriteFile(stashPath(), []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := peekStash("saved"); err == nil {
			t.Fatalf("malformed stash %q was treated as empty", raw)
		}
		if _, err := unstash("saved"); err == nil {
			t.Fatalf("malformed stash %q was treated as empty by unstash", raw)
		}
	}
}

func TestStashLockContention(t *testing.T) {
	stashHome(t)
	unlock, held := filelock.Lock(stashPath()+".lock", time.Second)
	if !held {
		t.Fatal("could not acquire test lock")
	}
	defer unlock()
	oldWait := stashWait
	stashWait = 20 * time.Millisecond
	t.Cleanup(func() { stashWait = oldWait })
	if err := stash(map[string]string{"agent.was": "original"}); err == nil {
		t.Fatal("stash succeeded while its lock was held")
	}
	if got := stashLoad()["agent.was"]; got != "" {
		t.Fatalf("value reported saved after lock failure: %q", got)
	}
}

func TestStashWriteErrorPreservesValues(t *testing.T) {
	stashHome(t)
	if err := os.MkdirAll(stashPath(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := stash(map[string]string{"agent.was": "original"}); err == nil {
		t.Fatal("stash reported success when its target is a directory")
	}
	if st, err := os.Stat(stashPath()); err != nil || !st.IsDir() {
		t.Fatalf("stash target changed: %v, %v", st, err)
	}
}

func TestStashReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix directory permissions required")
	}
	stashHome(t)
	dir := filepath.Dir(stashPath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Create the lock while writable so the attempted atomic write fails.
	unlock, held := filelock.Lock(stashPath()+".lock", time.Second)
	if !held {
		t.Fatal("could not acquire test lock")
	}
	unlock()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	if err := stash(map[string]string{"agent.was": "original"}); err == nil {
		t.Fatal("stash succeeded in a read-only directory")
	}
	if got := stashLoad()["agent.was"]; got != "" {
		t.Fatalf("value reported saved after write failure: %q", got)
	}
}

func stashHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
}

// Every agent's originals are in one stash.json, and each agent's connect
// and disconnect read, change and write it: two at once (the GUI's Connect
// all, a gateway move setting every agent again) must keep both changes.
func TestStashKeepsChangesMadeAtOnce(t *testing.T) {
	stashHome(t)
	var wg sync.WaitGroup
	for i := range 40 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stash(map[string]string{fmt.Sprintf("agent%d.was", i): "theirs"})
		}()
	}
	wg.Wait()
	if n := len(stashLoad()); n != 40 {
		t.Fatalf("%d of 40 agents' originals kept", n)
	}
}

// A malformed stash remains the active file when a mutation fails.
func TestUnreadStashKeptAside(t *testing.T) {
	stashHome(t)
	p := stashPath()
	os.MkdirAll(filepath.Dir(p), 0o755)
	half := `{"claude.was": "sk-their-own", "codex.wa`
	if err := os.WriteFile(p, []byte(half), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := stash(map[string]string{"pi.was": "x"}); err == nil {
		t.Fatal("malformed stash accepted")
	}
	forget("pi.was")
	if b, err := os.ReadFile(p); err != nil || string(b) != half {
		t.Fatalf("active stash changed: %q, %v", b, err)
	}
	if fi, err := os.Stat(p); err != nil {
		t.Fatal(err)
	} else if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Fatalf("stash.json is %v; it holds keys", fi.Mode().Perm())
	}
}

// A JSON array is not a stash and must not be replaced by a mutation.
func TestUnreadStashArrayKeptAside(t *testing.T) {
	stashHome(t)
	p := stashPath()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(`[]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := stash(map[string]string{"pi.was": "x"}); err == nil {
		t.Fatal("array stash accepted")
	}
	if b, err := os.ReadFile(p); err != nil || string(b) != `[]` {
		t.Fatalf("active stash changed: %q, %v", b, err)
	}
}
