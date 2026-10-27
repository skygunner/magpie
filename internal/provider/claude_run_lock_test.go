package provider

// #3254 (jstar0): Claude Code runs in a saved account's own config
// directory for magpie's /usage reads and warm-ups. Another magpie's switch
// handed the account's sign-in to Claude Code itself and removed that
// directory under the running Claude Code — one refresh token then held in
// two places, whichever refreshed first signing the other out — and another
// magpie's removal took the directory away under it.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/filelock"
)

// claudeRunLockIn is the lock a magpie holds on the saved account user's
// folder while Claude Code runs in it. Spelled out here rather than through
// the code that takes it, so the test builds against a tree without the
// guard too, and fails on the behavior rather than on a compile.
func claudeRunLockIn(user string) string {
	return filepath.Join(filepath.Dir(claudeDirsRoot()), "claude-runs", filepath.Base(claudeAccountDir(user))+".lock")
}

// openClaudeRunLock opens that lock's file, made with its folder.
func openClaudeRunLock(t *testing.T, user string) *os.File {
	t.Helper()
	path := claudeRunLockIn(user)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// anotherMagpieRuns takes that lock, as another magpie's Claude Code in the
// folder holds it, and returns the end of that run.
func anotherMagpieRuns(t *testing.T, user string) (done func()) {
	t.Helper()
	f := openClaudeRunLock(t, user)
	got, err := filelock.TryLock(f)
	if err != nil || !got {
		f.Close()
		t.Fatalf("another magpie's run could not take the lock of %s: %v %v", user, got, err)
	}
	var once sync.Once
	return func() { once.Do(func() { filelock.Unlock(f); f.Close() }) }
}

// claudeWithAnotherAccountSignedIn sets a temp home whose Claude Code is
// signed in to own@example.com, and whose logins.json saves side@example.com
// as well: one magpie's account, run in its own folder.
func claudeWithAnotherAccountSignedIn(t *testing.T) (home, own string) {
	t.Helper()
	home = claudeHome(t)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	noAnthropic(t)
	own = claudeSignIn(t, home, time.Now().Add(time.Hour))
	writeFile(t, filepath.Join(home, ".claude.json"), map[string]any{"oauthAccount": map[string]any{"emailAddress": "own@example.com"}})
	far := time.Now().Add(24 * time.Hour).UnixMilli()
	writeFile(t, loginsPath(), []map[string]any{
		{"agent": "claude", "user": "side@example.com", "plan": "max", "seen": time.Now().Add(-time.Hour).UTC(),
			"profile": map[string]any{"emailAddress": "side@example.com"},
			"auth":    map[string]any{"claudeAiOauth": map[string]any{"accessToken": "tok-side", "refreshToken": "r-side", "expiresAt": far, "subscriptionType": "max"}}},
	})
	ForgetAccounts()
	return home, own
}

// A switch to an account another magpie runs Claude Code in is refused: the
// sign-in would be Claude Code's own and the folder's run's at once, and the
// folder taken away under the run. Nothing is handed over, and the switch
// goes through once that run ends.
func TestClaudeSwitchRefusedWhileAnotherMagpieRunsTheFolder(t *testing.T) {
	_, own := claudeWithAnotherAccountSignedIn(t)
	dir, err := claudeSavedDir("side@example.com")
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(own)
	if err != nil {
		t.Fatal(err)
	}
	done := anotherMagpieRuns(t, "side@example.com")

	err = SwitchLogin("claude", "side@example.com")
	if err == nil {
		t.Fatal("switched to an account another magpie runs Claude Code in")
	}
	if !strings.Contains(err.Error(), "side@example.com") {
		t.Fatalf("the refusal names no account: %v", err)
	}
	// Claude Code is still signed in to its own account, and the folder the
	// run is in still holds the account's sign-in
	if after, err := os.ReadFile(own); err != nil || string(after) != string(before) {
		t.Fatalf("Claude Code was signed in to the account under the run: %v\n%s", err, after)
	}
	if _, err := os.Stat(filepath.Join(dir, ".credentials.json")); err != nil {
		t.Fatalf("the folder went out from under the run: %v", err)
	}

	done()
	if err := SwitchLogin("claude", "side@example.com"); err != nil {
		t.Fatalf("the switch after the run ended: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("the folder stayed after the switch: %v", err)
	}
}

// A removal of an account another magpie runs Claude Code in is refused, and
// the account is removed once that run ends.
func TestClaudeForgetRefusedWhileAnotherMagpieRunsTheFolder(t *testing.T) {
	claudeWithAnotherAccountSignedIn(t)
	dir, err := claudeSavedDir("side@example.com")
	if err != nil {
		t.Fatal(err)
	}
	done := anotherMagpieRuns(t, "side@example.com")

	if err := ForgetLogin("claude", "side@example.com"); err == nil {
		t.Fatal("removed an account another magpie runs Claude Code in")
	}
	if _, err := os.Stat(filepath.Join(dir, ".credentials.json")); err != nil {
		t.Fatalf("the folder went out from under the run: %v", err)
	}
	still := false
	for _, l := range Logins("claude") {
		still = still || l.User == "side@example.com"
	}
	if !still {
		t.Fatal("the account was dropped although its folder was in use")
	}

	done()
	if err := ForgetLogin("claude", "side@example.com"); err != nil {
		t.Fatalf("the removal after the run ended: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("the folder stayed after the removal: %v", err)
	}
}

// The other half of it: a run magpie makes in a saved account's folder holds
// that lock while Claude Code runs, so another magpie's switch or removal
// finds it held.
func TestClaudeSavedRunHoldsItsFolder(t *testing.T) {
	claudeWithAnotherAccountSignedIn(t)
	err := inSavedClaudeDir("side@example.com", func(dir string) error {
		if dir != claudeAccountDir("side@example.com") {
			t.Fatalf("run in %s", dir)
		}
		f := openClaudeRunLock(t, "side@example.com")
		defer f.Close()
		if got, err := filelock.TryLock(f); err != nil || got {
			t.Fatalf("another magpie took the lock of a folder Claude Code runs in: %v %v", got, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// and lets it go when the run ends
	f := openClaudeRunLock(t, "side@example.com")
	defer f.Close()
	if got, err := filelock.TryLock(f); err != nil || !got {
		t.Fatalf("the lock stayed after the run: %v %v", got, err)
	}
	filelock.Unlock(f)
}

// Signing an account in again keeps a folder another magpie runs Claude Code
// in: it is an older sign-in of it, and goes at the next chance rather than
// out from under the run.
func TestClaudeSignInKeepsAFolderInUse(t *testing.T) {
	home := claudeHome(t)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	noAnthropic(t)
	claudeSignIn(t, home, time.Now().Add(time.Hour))
	writeFile(t, filepath.Join(home, ".claude.json"), map[string]any{"oauthAccount": map[string]any{"emailAddress": "side@example.com"}})
	writeFile(t, loginsPath(), []map[string]any{
		{"agent": "claude", "user": "side@example.com", "plan": "max", "seen": time.Now().Add(-time.Hour).UTC(),
			"auth": map[string]any{"claudeAiOauth": map[string]any{"accessToken": "tok-side", "refreshToken": "r-side", "expiresAt": time.Now().Add(24 * time.Hour).UnixMilli()}}},
	})
	ForgetAccounts()
	dir, err := claudeSavedDir("side@example.com")
	if err != nil {
		t.Fatal(err)
	}
	done := anotherMagpieRuns(t, "side@example.com")
	defer done()
	using, err := addLogin(savedLogin{Agent: "claude", User: "side@example.com",
		Auth: []byte(`{"claudeAiOauth":{"accessToken":"tok-again","refreshToken":"r-again","expiresAt":` +
			strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10) + `}}`)})
	if err != nil {
		t.Fatal(err)
	}
	if !using {
		t.Fatal("the account Claude Code is signed in to, signed in again, isn't the one in use")
	}
	if _, err := os.Stat(filepath.Join(dir, ".credentials.json")); err != nil {
		t.Fatalf("the folder went out from under the run: %v", err)
	}
}
