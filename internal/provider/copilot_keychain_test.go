package provider

// The Copilot CLI's token is behind a macOS keychain prompt, and a user who
// answers Deny meant "don't ask again": a ten-minute pause let macOS show
// the prompt every ten minutes for as long as magpie ran, and again after
// every restart (#3058). The refusal is kept, and dropped when the CLI's
// own sign-in changes or magpie asks afresh.

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// copilotCLIOnly is a machine whose only Copilot sign-in is the standalone
// CLI's: who from its config.json, the token from the keychain.
func copilotCLIOnly(t *testing.T, login string) string {
	t.Helper()
	home := signIn(t)
	os.Remove(filepath.Join(home, ".config", "github-copilot", "apps.json"))
	cli := filepath.Join(home, "copilot-home")
	t.Setenv("COPILOT_HOME", cli)
	os.MkdirAll(cli, 0o700)
	writeCLISignIn(t, cli, login)
	return cli
}

// writeCLISignIn writes the CLI's config.json as `copilot login` leaves it.
func writeCLISignIn(t *testing.T, cli, login string) {
	t.Helper()
	b := `{
  "lastLoggedInUser": {"host": "https://github.com", "login": "` + login + `"},
  "loggedInUsers": [{"host": "https://github.com", "login": "` + login + `"}]
}`
	if err := os.WriteFile(filepath.Join(cli, "config.json"), []byte(b), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ageCopilotSecrets has every kept read as old as d more: the ten minutes
// a refusal used to be kept for.
func ageCopilotSecrets(d time.Duration) {
	copilotSecrets.Lock()
	defer copilotSecrets.Unlock()
	for k, at := range copilotSecrets.at {
		copilotSecrets.at[k] = at.Add(-d)
	}
}

// restartCopilot loses what a process keeps in memory, as a restart does.
func restartCopilot() {
	copilotSecrets.Lock()
	copilotSecrets.at, copilotSecrets.tok = map[string]time.Time{}, map[string]string{}
	copilotSecrets.Unlock()
}

// A refusal is asked once, and not again ten minutes later nor after a
// restart; the CLI signing in again asks it once more.
func TestCopilotKeychainDeniedIsNotAskedAgain(t *testing.T) {
	cli := copilotCLIOnly(t, "chengbot-ai")
	var denied atomic.Bool
	denied.Store(true)
	var asks atomic.Int32
	old := copilotKeychainRead
	copilotKeychainRead = func(string) ([]byte, error) {
		asks.Add(1)
		if denied.Load() {
			return nil, errKeychainRefused
		}
		return []byte("gho_cli"), nil
	}
	defer func() { copilotKeychainRead = old }()
	// TestMain stands in for the read, as no test may reach the runner's
	// keychain; this one reads through the reader itself
	oldSecret := copilotCLISecret
	copilotCLISecret = copilotCLISecretFrom
	defer func() { copilotCLISecret = oldSecret }()
	cfg := copilotConfigDir()

	if app, ok := copilotLogin(cfg); ok {
		t.Fatalf("a refused keychain read is no sign-in: %+v", app)
	}
	if n := asks.Load(); n != 1 {
		t.Fatalf("the first read asks the keychain once, asked %d times", n)
	}

	// ten minutes on, and the prompt stays away
	ageCopilotSecrets(11 * time.Minute)
	if app, ok := copilotLogin(cfg); ok {
		t.Fatalf("a refused keychain read is no sign-in: %+v", app)
	}
	if n := asks.Load(); n != 1 {
		t.Fatalf("a refused read was asked again ten minutes later: %d asks", n)
	}

	// and a restart keeps it too: the user's answer outlives the process
	restartCopilot()
	if app, ok := copilotLogin(cfg); ok {
		t.Fatalf("a refused keychain read is no sign-in: %+v", app)
	}
	if n := asks.Load(); n != 1 {
		t.Fatalf("a restart asked the keychain again: %d asks", n)
	}

	// the CLI signs in again: that refusal answered for the sign-in before
	// it, and this one is asked
	denied.Store(false)
	writeCLISignIn(t, cli, "hubot")
	app, ok := copilotLogin(cfg)
	if !ok || app.User != "hubot" || app.Token != "gho_cli" {
		t.Fatalf("a new CLI sign-in isn't read: %+v %v", app, ok)
	}
	if n := asks.Load(); n != 2 {
		t.Fatalf("a new CLI sign-in was asked about %d times, want 2", n)
	}
}

// A prompt nobody answers doesn't hold the readers behind it: the ask runs
// outside the cache every reader takes, so the account list and the quota
// poll don't wait out a dialog (#3058).
func TestCopilotKeychainAskDoesNotHoldReaders(t *testing.T) {
	copilotCLIOnly(t, "chengbot-ai")
	up := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	old := copilotKeychainRead
	copilotKeychainRead = func(string) ([]byte, error) {
		once.Do(func() { close(up) })
		<-release
		return nil, errKeychainNoAnswer
	}
	defer func() { copilotKeychainRead = old }()
	defer close(release)
	account := "https://github.com:chengbot-ai"

	first := make(chan string, 1)
	go func() { first <- copilotCLISecretFrom(account) }()
	<-up // the prompt is up

	second := make(chan string, 1)
	go func() { second <- copilotCLISecretFrom(account) }()
	select {
	case <-second:
	case <-time.After(10 * time.Second):
		t.Fatal("a second reader waited on a keychain prompt nobody answered")
	}
	select {
	case <-first:
		t.Fatal("the ask finished, so the second reader's return proves nothing")
	default:
	}
}
