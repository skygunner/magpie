package provider

import "testing"

// A raw plugin's saved decide override is account settings, not a
// standalone endpoint that makes PluginID reserve the suffixed id.
func TestReviewR6IndependentSuffixSurvivesRawDecisionPicks(t *testing.T) {
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
	if got := PluginID("cline"); got != "cline" {
		t.Fatalf("raw picks unexpectedly changed public plugin id: %q", got)
	}
	own := Provider{ID: "cline-plugin", Decide: "http://localhost:12345/v1", Models: []string{"typesafe/jev"}}
	if !keyOptional(own) {
		t.Fatal("fixture must be a valid keyless local provider")
	}
	if err := Save(own); err != nil {
		t.Fatal(err)
	}
	if q, err := Find(own.ID); err != nil || q.Account != nil {
		t.Fatalf("saved independent provider missing: %+v %v", q, err)
	}
	stored := load().Providers
	for i := range stored {
		if stored[i].ID == own.ID {
			stored[i].Name = ""
		}
	}
	writeProviders(t, stored)
	if got := PluginID("cline"); got != "cline" {
		t.Fatalf("tested suffix is a real public alias: %q", got)
	}
	if q, err := Find("cline"); err != nil || !q.IsPlugin() || q.Decide != p.Decide {
		t.Fatalf("raw plugin's decision picks did not survive: %+v %v", q, err)
	}
	q, err := Find(own.ID)
	if err != nil {
		t.Fatalf("saved raw decision picks claimed an unused plugin alias: %v", err)
	}
	if q.Account != nil || q.Decide != own.Decide || !q.DecideOnly() {
		t.Fatalf("independent local provider changed identity: %+v", q)
	}
	if got, model, err := RouteDecider(own.ID + "/typesafe/jev"); err != nil || got.ID != own.ID || model != "typesafe/jev" || got.Account != nil {
		t.Fatalf("independent decision route changed: provider=%+v model=%q err=%v", got, model, err)
	}
}
