package provider

import (
	"testing"
)

// decide= saved on a plugin account's provider is that account's setting,
// not a self-built provider: the stored entry must not turn customIDs on,
// or the plugin's provider id reads as "cline-plugin" and every reference
// to it by id breaks (#3349 review).
func TestAccountDecideStaysOffCustomIDs(t *testing.T) {
	clinePlugin(t)
	if err := Save(Provider{ID: "cline", Decide: "https://api.commandcode.ai/provider/v1", Models: []string{"typesafe/jev"}}); err != nil {
		t.Fatal(err)
	}
	if got := PluginID("cline"); got != "cline" {
		t.Fatalf("PluginID(cline) = %q, want cline", got)
	}
	p, err := Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	if p.Decide != "https://api.commandcode.ai/provider/v1" || p.Account == nil {
		t.Fatalf("the saved decide: %+v", p)
	}
	if !p.DecidesModel("typesafe/jev") {
		t.Fatalf("typesafe/jev is not a decision model of the provider: %+v", p)
	}
	// a self-built decision-only provider of the same id is still custom,
	// collision rules and all
	if err := Save(Provider{ID: "sole", Name: "Sole", Decide: "https://decide.example/v1", Key: "sk-sole"}); err != nil {
		t.Fatal(err)
	}
	if got := PluginID("sole"); got != "sole-plugin" {
		t.Fatalf("PluginID(sole) = %q, want sole-plugin", got)
	}
}
