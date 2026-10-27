package provider

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/testenv"
)

// yiyiSmallti's report (#2838): a Claude account upgraded to Max kept
// showing plan "pro". Claude Code's `auth status` misses the new Max
// subscription identifiers in Anthropic's payload and answers "pro" for a
// Max account (anthropics/claude-code#47647), and magpie preferred that
// over the credential. The credential's rate-limit tier is what Anthropic's
// API enforces and names the Max plans (default_claude_max_5x,
// default_claude_max_20x): a Max tier says Max whatever subscriptionType
// and the CLI say.

func claudeMaxTierHome(t *testing.T, sub, tier, status string) {
	t.Helper()
	shellFakes(t)
	home := claudeHome(t)
	oauth := map[string]any{
		"accessToken": "sk-ant-oat01-old", "refreshToken": "sk-ant-ort01-old",
		"expiresAt": time.Now().Add(time.Hour).UnixMilli(),
		"scopes":    []string{"user:inference", "user:profile"},
	}
	if sub != "" {
		oauth["subscriptionType"] = sub
	}
	if tier != "" {
		oauth["rateLimitTier"] = tier
	}
	writeFile(t, filepath.Join(home, ".claude", ".credentials.json"), map[string]any{"claudeAiOauth": oauth})
	writeFile(t, filepath.Join(home, ".claude.json"), map[string]any{"oauthAccount": map[string]any{"emailAddress": "me@example.com"}})
	exe := filepath.Join(t.TempDir(), "claude")
	testenv.Program(t, exe, "#!/bin/sh\necho '{\"loggedIn\":true,\"email\":\"me@example.com\",\"subscriptionType\":\""+status+"\"}'\n")
	old := claudeExecutable
	claudeExecutable = func() string { return exe }
	t.Cleanup(func() {
		claudeExecutable = old
		forgetClaudeStatus()
		forgetAccountCaches()
	})
	forgetClaudeStatus()
	forgetAccountCaches()
}

func TestClaudeMaxTierSaysMax(t *testing.T) {
	for _, c := range []struct {
		name, sub, tier, status, want, login string
	}{
		// the report: upgraded to Max, the CLI's status still answers pro
		{"max 20x tier, the CLI says pro", "pro", "default_claude_max_20x", "pro", "max", "max"},
		{"max 5x tier, the CLI says pro", "pro", "default_claude_max_5x", "pro", "max", "max"},
		{"max tier, subscriptionType left empty", "", "default_claude_max_20x", "pro", "max", "max"},
		{"subscriptionType max, the CLI says pro", "max", "", "pro", "max", "max"},
		// a Pro account is not made Max by this
		{"pro tier, the CLI says pro", "pro", "default_claude_pro", "pro", "pro", "pro"},
		{"no tier, the CLI says pro", "pro", "", "pro", "pro", "pro"},
		// the CLI's reading still corrects the provider's account, which
		// always took it over the credential's subscriptionType; the saved
		// login never did, before this change either
		{"no tier, the CLI says max", "pro", "", "max", "max", "pro"},
	} {
		t.Run(c.name, func(t *testing.T) {
			claudeMaxTierHome(t, c.sub, c.tier, c.status)
			p, ok := claudeAccount()
			if !ok || p.Account == nil || p.Account.Plan != c.want {
				t.Fatalf("account plan: %q, want %q (%+v %v)", p.Account.Plan, c.want, p.Account, ok)
			}
			l, ok := liveLogin("claude")
			if !ok || l.Plan != c.login {
				t.Fatalf("saved login plan: %q, want %q (%+v %v)", l.Plan, c.login, l, ok)
			}
		})
	}
}
