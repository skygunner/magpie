package provider

import (
	"slices"
	"testing"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/plugin"
)

// A local System One server is explicitly allowed to run without a key.
// The name is optional in stored configuration, just as in the R4 input.
func TestReviewR3KeylessLocalDecisionProviderSurvives(t *testing.T) {
	claudeHome(t)
	p := Provider{ID: "local-decision", Decide: "http://localhost:12345/v1", Models: []string{"typesafe/jev"}}
	if !keyOptional(p) {
		t.Fatal("fixture is not a valid keyless local provider")
	}
	// Confirm the endpoint/key combination is accepted by the product's Save.
	if err := Save(p); err != nil {
		t.Fatalf("valid local provider rejected on save: %v", err)
	}
	if _, err := Find(p.ID); err != nil {
		t.Fatalf("saved local provider not readable: %v", err)
	}
	// Existing JSON configurations do not require the optional display name.
	writeProviders(t, []Provider{p})
	got, err := Find(p.ID)
	if err != nil {
		t.Fatalf("independent keyless local provider disappeared: %v", err)
	}
	if got.Account != nil || got.Decide != p.Decide || !got.DecideOnly() {
		t.Fatalf("local decision provider identity changed: %+v", got)
	}
}

// The same explicit decision selection must be classified consistently by
// the routing picker and the model metadata consumed by the provider editor.
func TestReviewR3PickedDecisionMetadataAgreesWithRouting(t *testing.T) {
	clinePlugin(t)
	ps := plugin.Cached()
	ps[0].Decide = true
	ps[0].Models = append(ps[0].Models, plugin.Model{ID: "native-decision", Name: "Native Decision", Decides: true})
	plugin.UseCached(ps)
	p, err := Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	const chat, selected = "cline-pass/glm-5.3-flash", "typesafe/jev"
	p.Decide = "https://fake.invalid/v1"
	p.Models = []string{chat, selected}
	if err := Save(*p); err != nil {
		t.Fatal(err)
	}
	p, err = Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	if p.DecidesModel(chat) {
		t.Fatal("mixed picks reclassified chat as decision")
	}
	if !slices.ContainsFunc(Catalog(), func(e Entry) bool { return e.ID == "cline/"+chat }) {
		t.Fatal("mixed picks hid the selected chat model")
	}
	if _, model, err := RouteDecider("cline/" + selected); err != nil || model != selected {
		t.Fatalf("explicit decision route failed: %q %v", model, err)
	}
	if !slices.ContainsFunc(Deciders(), func(e Entry) bool { return e.ID == "cline/"+selected }) {
		t.Fatal("routing picker omitted the selected decision model")
	}
	ms := p.DecisionModels()
	if !slices.ContainsFunc(ms, func(m catalog.Model) bool { return m.ID == "native-decision" }) {
		t.Fatal("fixture did not expose native decision metadata")
	}
	if !slices.ContainsFunc(ms, func(m catalog.Model) bool { return m.ID == selected }) {
		t.Fatalf("DecisionModels omits explicit %s while Deciders and RouteDecider accept it: %+v", selected, ms)
	}
}

// A decision-shaped pick alone supplies no endpoint or authorization.
func TestReviewR3PickWithoutDecisionEndpointStaysUnroutable(t *testing.T) {
	clinePlugin(t)
	p, err := Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	p.Models = []string{"typesafe/jev", "unknown-chat-model"}
	if err := Save(*p); err != nil {
		t.Fatal(err)
	}
	p, err = Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	if p.Decides() || p.DecidesModel("typesafe/jev") {
		t.Fatal("a pick manufactured a decision endpoint")
	}
	if _, _, err := RouteDecider("cline/typesafe/jev"); err == nil {
		t.Fatal("decision route accepted a plugin without a decision endpoint")
	}
}
