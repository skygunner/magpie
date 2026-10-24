package provider

import "testing"

// The existence of raw provider cline does not reserve cline-plugin:
// without a collision PluginID(cline) is still cline. An independently
// configured local server at the otherwise free id keeps its identity,
// including when its optional display name is absent from stored JSON.
func TestReviewR5IndependentSuffixIDIsNotAnUnusedPluginAlias(t *testing.T) {
	clinePlugin(t)
	if got := PluginID("cline"); got != "cline" {
		t.Fatalf("fixture has a real plugin alias collision: %q", got)
	}
	own := Provider{ID: "cline-plugin", Decide: "http://localhost:12345/v1", Models: []string{"typesafe/jev"}}
	if !keyOptional(own) {
		t.Fatal("fixture is not a valid keyless local provider")
	}
	if err := Save(own); err != nil {
		t.Fatalf("independent id or local endpoint rejected by Save: %v", err)
	}
	if p, err := Find(own.ID); err != nil || p.Account != nil {
		t.Fatalf("saved independent provider not readable: %+v %v", p, err)
	}
	if got := PluginID("cline"); got != "cline" {
		t.Fatalf("saved provider unexpectedly changed plugin identity: %q", got)
	}
	writeProviders(t, []Provider{own})
	if got := PluginID("cline"); got != "cline" {
		t.Fatalf("the plugin actually claimed the tested alias: %q", got)
	}
	p, err := Find(own.ID)
	if err != nil {
		t.Fatalf("independent provider disappeared behind an alias no plugin uses: %v", err)
	}
	if p.Account != nil || p.Decide != own.Decide || !p.DecideOnly() {
		t.Fatalf("independent local provider changed identity: %+v", p)
	}
	if got, model, err := RouteDecider(own.ID + "/typesafe/jev"); err != nil || got.ID != own.ID || model != "typesafe/jev" || got.Account != nil {
		t.Fatalf("independent local decision route changed: provider=%+v model=%q err=%v", got, model, err)
	}
}
