package provider

import (
	"slices"
	"testing"

	"github.com/yetone/magpie/internal/plugin"
)

// Selecting a conversation model must not change the kind of model it is.
// This covers both the plugin's declared decision endpoint and a user's
// decision endpoint saved beside conversation picks.
func TestReviewV2PickedChatStaysChat(t *testing.T) {
	for _, native := range []bool{true, false} {
		name := "user_decide"
		if native {
			name = "plugin_declared_decide"
		}
		t.Run(name, func(t *testing.T) {
			clinePlugin(t)
			if native {
				ps := plugin.Cached()
				ps[0].Decide = true
				plugin.UseCached(ps)
			}
			p, err := Find("cline")
			if err != nil {
				t.Fatal(err)
			}
			const model = "cline-pass/glm-5.3-flash"
			p.Models = []string{model}
			if !native {
				p.Decide = "https://fake.invalid/v1"
			}
			if err := Save(*p); err != nil {
				t.Fatal(err)
			}
			p, err = Find("cline")
			if err != nil {
				t.Fatal(err)
			}
			if p.DecidesModel(model) {
				t.Errorf("selected conversation model was reclassified as a decision model: %s", model)
			}
			if !slices.ContainsFunc(Catalog(), func(e Entry) bool { return e.ID == "cline/"+model }) {
				t.Errorf("selected conversation model disappeared from the agent catalog")
			}
			if _, _, err := RouteDecider("cline/" + model); err == nil {
				t.Error("conversation model was accepted as a System One model")
			}
		})
	}
}

// The selected decision model not in the chat feed must be discoverable
// in the classifier picker, in addition to routing by its full name.
func TestReviewV2ExplicitJevIsDiscoverable(t *testing.T) {
	clinePlugin(t)
	p, err := Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	p.Decide = "https://fake.invalid/v1"
	p.Models = []string{"typesafe/jev"}
	if err := Save(*p); err != nil {
		t.Fatal(err)
	}
	p, err = Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	if _, model, err := RouteDecider("cline/typesafe/jev"); err != nil || model != "typesafe/jev" {
		t.Fatalf("explicit route: %q %v", model, err)
	}
	for name, entries := range map[string][]Entry{"Deciders": Deciders(), "OfferedDeciders": OfferedDeciders(nil)} {
		if !slices.ContainsFunc(entries, func(e Entry) bool { return e.ID == "cline/typesafe/jev" }) {
			t.Errorf("%s omitted the explicitly configured decision model: %v", name, entries)
		}
	}
	if got := p.Jev(); got != "typesafe/jev" {
		t.Errorf("default decision model is %q, want explicit typesafe/jev", got)
	}
}

// Name was not required when reading an existing provider configuration.
// A key-bearing independent provider is not an account's nameless picks.
func TestReviewV2NamelessDecisionProviderSurvives(t *testing.T) {
	claudeHome(t)
	writeProviders(t, []Provider{{ID: "own-decision", Key: "synthetic-key", Decide: "https://fake.invalid/v1", Models: []string{"typesafe/jev"}}})
	p, err := Find("own-decision")
	if err != nil {
		t.Fatalf("independent decision provider disappeared: %v", err)
	}
	if p.Account != nil || p.Key != "synthetic-key" || !p.DecideOnly() {
		t.Fatalf("independent provider identity changed: %#v", p)
	}
}

// Positive identity control: a real independent provider named cline
// keeps its id, while the separately installed plugin receives its suffix.
func TestReviewV2NamedDecisionProviderCollision(t *testing.T) {
	clinePlugin(t)
	writeProviders(t, []Provider{{ID: "cline", Name: "Own Cline", Key: "synthetic-key", Decide: "https://fake.invalid/v1", Models: []string{"typesafe/jev"}}})
	if got := PluginID("cline"); got != "cline-plugin" {
		t.Fatalf("plugin id=%q", got)
	}
	p, err := Find("cline")
	if err != nil || p.Account != nil || p.Key != "synthetic-key" {
		t.Fatalf("independent provider=%+v err=%v", p, err)
	}
	a, err := Find("cline-plugin")
	if err != nil || !a.IsPlugin() {
		t.Fatalf("plugin provider=%+v err=%v", a, err)
	}
}
