package provider

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/testenv"
)

// A refused credential write must not change saved login marks (#4592).
func TestClaudeSwitchCredentialFailureKeepsLogins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stands in for the keychain")
	}
	home := claudeHome(t)
	cred := claudeSignIn(t, home, time.Now().Add(time.Hour))
	profile := filepath.Join(home, ".claude.json")
	writeFile(t, profile, map[string]any{"oauthAccount": map[string]any{"emailAddress": "a@example.com", "accountUuid": "u-a"}})
	rememberLogins(true)
	writeFile(t, cred, map[string]any{"claudeAiOauth": map[string]any{"accessToken": "b", "refreshToken": "b", "expiresAt": time.Now().Add(time.Hour).UnixMilli(), "subscriptionType": "pro"}})
	writeFile(t, profile, map[string]any{"oauthAccount": map[string]any{"emailAddress": "b@example.com", "accountUuid": "u-b"}})
	forgetClaudeCredential()
	rememberLogins(true)
	before, err := os.ReadFile(loginsPath())
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	testenv.Program(t, filepath.Join(bin, "security"), "#!/bin/sh\ncase \"$1\" in find-*) printf '%s' '{\"claudeAiOauth\":{\"accessToken\":\"b\",\"refreshToken\":\"b\",\"expiresAt\":4102444800000,\"subscriptionType\":\"pro\"}}';; *) echo attempt >> \"$SWITCH_ATTEMPTS\"; echo refused; exit 51;; esac\n")
	t.Setenv("SWITCH_ATTEMPTS", filepath.Join(bin, "attempts"))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	claudeKeychain = true
	forgetClaudeCredential()
	err = SwitchLogin("claude", "a@example.com")
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("credential failure = %v", err)
	}
	after, err := os.ReadFile(loginsPath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("logins.json changed after credential write failed")
	}
	attempts, err := os.ReadFile(filepath.Join(bin, "attempts"))
	if err != nil || string(attempts) != "attempt\n" {
		t.Fatalf("credential writes = %q, %v; want one", attempts, err)
	}
}
