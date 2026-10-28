package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/provider"
)

// A Claude Code subscription runs its own binary, so the provider has no
// native API of its own; the gateway still serves its models on Anthropic
// Messages (serveClaudeSubscription). dsh used to leave such a model on the
// openai-completions magpie route, where a Copilot-style client fails. It
// goes on magpie-anthropic now (#3514).
func TestDshRoutesASubscriptionClaudeOnMessages(t *testing.T) {
	home, _, web := dshProtocolHome(t)
	t.Cleanup(provider.ForgetAccounts)
	provider.ForgetAccounts()
	dir := appdir.ClaudeDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cred := `{"claudeAiOauth":{"accessToken":"at","refreshToken":"rt","expiresAt":1999999999000,"subscriptionType":"max"}}`
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), []byte(cred), 0o600); err != nil {
		t.Fatal(err)
	}
	state := `{"oauthAccount":{"emailAddress":"sub@example.com","accountUuid":"u-sub"},"additionalModelOptionsCache":[{"value":"claude-sonnet-5[1m]","label":"Sonnet 5"}],"additionalModelOptionsAnsweredAt":9999999999999}`
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(web, []byte("# Your patch layer for this dsh profile.\n"+dshMineRow), 0o644)
	a := dsh(home)
	f := a.Field("model")
	id := "magpie/claude/claude-sonnet-5"
	if err := f.Set(id); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(web)
	s := string(b)
	ps := dshPiProviders(t, web)
	r, ok := ps["magpie-anthropic"]
	if !ok {
		t.Fatalf("no magpie-anthropic route:\n%s", s)
	}
	if r.API != "anthropic-messages" {
		t.Errorf("the subscription Claude is served on %q, want anthropic-messages", r.API)
	}
	if got := dshRouteModelIDs(r); !contains(got, "claude/claude-sonnet-5") {
		t.Errorf("magpie-anthropic lacks claude/claude-sonnet-5: %v", got)
	}
	if !strings.Contains(s, "    provider: magpie-anthropic\n") || a.Check() != "" {
		t.Fatalf("start: check %q\n%s", a.Check(), s)
	}
	_ = home
}
