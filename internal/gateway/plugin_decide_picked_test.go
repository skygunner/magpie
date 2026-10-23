package gateway

import (
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// The plan's Jev without the plan listing it: Command Code's plugin lists
// 92 chat models and never its Jev, and its decision API is told by the
// user's decide= beside the pick (#3349). Saving must keep the plugin's
// identity, RouteDecider must answer the picked model, and the SystemOne
// request must go through the plugin's fetch to the plan's own host, with
// the plan's model and the account's sign-in. Local routing and protocol
// construction only: the real plan's endpoint was the reporter's own
// verification.
func TestPluginPickedDecisionModelRoutes(t *testing.T) {
	up := &decideUp{}
	pid := besideFake(t, "commandcode-plan", up)
	// the feed lists chat models only, as the real plan's does
	p0, err := provider.Find(pid)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range p0.Models {
		if m == "typesafe/jev" {
			t.Fatal("the feed already lists typesafe/jev; the fixture must not")
		}
	}
	// the plan's own host: in production api.commandcode.ai is where the
	// plugin's loader sends the account's requests; here the fake vendor's
	// base stands in for it (FAKE_BASE = the stub's URL + /v1)
	decide := strings.TrimSuffix(os.Getenv("FAKE_BASE"), "/v1") + "/provider/v1"
	if err := provider.Save(provider.Provider{ID: "commandcode-plan",
		Decide: decide, Models: []string{"typesafe/jev"}}); err != nil {
		t.Fatal(err)
	}
	// identity and setting survive the save and a re-read
	if got := provider.PluginID("commandcode-plan"); got != pid {
		t.Fatalf("PluginID = %q, want %q", got, pid)
	}
	p, err := provider.Find(pid)
	if err != nil || p.Account == nil || p.Decide != decide {
		t.Fatalf("re-read: %+v, %v", p, err)
	}
	// the picked model is a decider, kept from agents' lists
	if _, model, err := provider.RouteDecider(pid + "/typesafe/jev"); err != nil || model != "typesafe/jev" {
		t.Fatalf("RouteDecider: %q, %v", model, err)
	}
	if offered(pid + "/typesafe/jev") {
		t.Fatal("the decision model is offered to agents")
	}
	s := New()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/systemone", strings.NewReader(`{"model":"`+pid+`/typesafe/jev","state":{"state":"fix the login bug"},"questions":[{"type":"intent"}]}`))
	s.Handler().ServeHTTP(rec, req)
	code, out := rec.Code, rec.Body.String()
	if code != 200 {
		t.Fatalf("systemone: %d %s", code, out)
	}
	asked := up.seen()
	if !slices.ContainsFunc(asked, func(a string) bool {
		return strings.HasPrefix(a, "/provider/v1/systemone typesafe/jev Bearer k1")
	}) {
		t.Fatalf("the plan's decision API was asked %v", asked)
	}
}
