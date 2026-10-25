package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #3246: a ZCode connected to magpie leaves the connection after a while,
// and only connecting it again brings it back. ZCode rewrites its own files
// on a restart or an update — both config.json and provider_config.json,
// neither with magpie's provider in it — and ZCode's settings page edits
// provider_config.json alone. So magpie's provider gone from both is a
// rewrite by something else, not a removal by the user, and the sync puts
// it back; a disconnect through magpie stays disconnected.
func TestZCodeSyncRestoresARewrite(t *testing.T) {
	home := syncHome(t)
	t.Setenv("DSH_HOME", "")
	path := filepath.Join(home, ".zcode", "v2", "config.json")
	rules := filepath.Join(home, ".zcode", "v2", "provider_config.json")
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(`{"provider":{"builtin:bigmodel":{"name":"Bigmodel","kind":"anthropic","enabled":true}}}`), 0o644)
	os.WriteFile(rules, []byte(`{"schemaVersion":1,"config":{"providerConfigRules":{"providerRules":[{"providerId":"mine","config":{}}]}}}`), 0o600)
	a := zcode(home)
	if err := a.Apply("provider", "magpie"); err != nil {
		t.Fatal(err)
	}
	if got := a.Field("provider").Get(); got != "magpie" {
		t.Fatalf("connect: %q", got)
	}

	// ZCode rewrote both files: magpie's provider is in neither, its own are
	os.WriteFile(path, []byte(`{"provider":{"builtin:bigmodel":{"name":"Bigmodel","kind":"anthropic","enabled":true}}}`), 0o644)
	os.WriteFile(rules, []byte(`{"schemaVersion":1,"config":{"providerConfigRules":{"providerRules":[{"providerId":"mine","config":{}}]}}}`), 0o600)
	if err := a.Sync(); err != nil {
		t.Fatal(err)
	}
	if got := a.Field("provider").Get(); got != "magpie" {
		t.Fatalf("after a rewrite: %q, want magpie back", got)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), `"builtin:bigmodel"`) || !strings.Contains(string(b), `"magpie"`) {
		t.Fatalf("config.json: %s", b)
	}
	b, _ = os.ReadFile(rules)
	if !strings.Contains(string(b), `"providerId":"mine"`) || !strings.Contains(string(b), `"providerId":"magpie"`) || !strings.Contains(string(b), `"relay/glm-4.6"`) {
		t.Fatalf("provider_config.json: %s", b)
	}

	// disconnected through magpie: a rewrite is not undone
	if err := a.Apply("provider", ""); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, []byte(`{"provider":{"builtin:bigmodel":{"name":"Bigmodel","kind":"anthropic","enabled":true}}}`), 0o644)
	os.WriteFile(rules, []byte(`{"schemaVersion":1,"config":{"providerConfigRules":{"providerRules":[{"providerId":"mine","config":{}}]}}}`), 0o600)
	if err := a.Sync(); err != nil {
		t.Fatal(err)
	}
	if got := a.Field("provider").Get(); got != "" {
		t.Fatalf("a disconnect was undone: %q", got)
	}
}
