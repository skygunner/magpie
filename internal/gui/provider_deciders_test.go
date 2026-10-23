package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// The editor's deciders metadata agrees with routing: a decision model the
// user picked beside an explicit decide= is in the JSON deciders list the
// page reads by name, and the selected conversation model is not — the
// page's decidesModel does a plain includes and never falls back to Jev
// name shapes, so an omission here made the Test-as menus and the
// routing-only hints treat the picked decision model as chat (#3349
// review, r3 follow-up). Metadata and frontend semantics only: no native
// GUI was driven.
func TestProviderEditorDecidersIncludeThePickedDecisionModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("MAGPIE_BUN", "")
	os.MkdirAll(filepath.Dir(catalog.CachePath()), 0o755)
	os.WriteFile(catalog.CachePath(), []byte(`{}`), 0o644)
	catalog.Reset()
	t.Cleanup(catalog.Reset)
	write := func(name string, v any) {
		t.Helper()
		b, _ := json.Marshal(v)
		os.MkdirAll(settings.Dir(), 0o700)
		if err := os.WriteFile(filepath.Join(settings.Dir(), name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("plugins.json", map[string]any{"plugins": []map[string]any{{"spec": "@magpie-community/opencode-cline-auth"}}})
	write("plugin-auth.json", map[string]map[string]any{"cline": {"type": "api", "key": "sk-cline-1234"}})
	plugin.UseCached([]plugin.Provider{{ID: "cline", Spec: "@magpie-community/opencode-cline-auth", Name: "Cline",
		Methods:  []plugin.Method{{Type: "api", Label: "API key"}},
		Accounts: []plugin.Account{{Key: "cline", Type: "api"}},
		Models: []plugin.Model{
			{ID: "cline-pass/glm-5.3-flash", Name: "GLM 5.3 Flash (ClinePass)"},
			{ID: "native-decision", Name: "Native Decision", Decides: true},
		}}})
	t.Cleanup(func() { plugin.UseCached(nil) })

	p, err := provider.Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	const chat, picked = "cline-pass/glm-5.3-flash", "typesafe/jev"
	p.Decide = "https://fake.invalid/v1"
	p.Models = []string{chat, picked}
	if err := provider.Save(*p); err != nil {
		t.Fatal(err)
	}
	p, err = provider.Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	info := providerInfo(*p, nil)
	if info.Deciders == nil {
		t.Fatal("the editor's deciders list is absent")
	}
	if !slices.Contains(*info.Deciders, picked) {
		t.Fatalf("deciders omits the picked decision model: %v", *info.Deciders)
	}
	if slices.Contains(*info.Deciders, chat) {
		t.Fatalf("deciders list the conversation model: %v", *info.Deciders)
	}
}

// An absent deciders list (no decision API) stays absent: the page's
// includes never runs, and the remote magpie's explicit empty list is
// untouched.
func TestProviderEditorDecidersAbsentWithoutADecideEndpoint(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	p := provider.Provider{ID: "plain", Name: "Plain", Chat: "https://example.invalid/v1", Key: "k",
		Models: []string{"typesafe/jev", "other"}}
	if err := provider.Save(p); err != nil {
		t.Fatal(err)
	}
	got, err := provider.Find("plain")
	if err != nil {
		t.Fatal(err)
	}
	info := providerInfo(*got, nil)
	if info.Deciders != nil {
		t.Fatalf("a plain provider grew a deciders list: %v", *info.Deciders)
	}
}
