package gateway

import (
	"net/http"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// #3847 (MYCF711): a seat whose model its vendor serves at no cost keeps
// Weighed.Free when it sits inside a nested group. The outer group is set
// Free first, the group inside it left on the default: the seat is planned
// by the inner group's routing, which doesn't weigh by free, so the flag
// was lost and the Routing page read the seat that always can answer as
// "100% used · all but used up" again — what #3763 set out to fix. Free is
// the seat's own property: it follows the seat wherever it sits.
func TestNestedFreeSeatKeepsItsFreeFlag(t *testing.T) {
	fresh(t)
	vendor := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"ok","choices":[{"message":{"role":"assistant","content":"hi"}}]}`))
	})
	serveOn(t, "zenfree", "sk-free", []string{"m"}, vendor)
	serveOn(t, "paidv", "sk-paid", []string{"m"}, vendor)
	liveModels(t, "zenfree", `{"models":[{"id":"m","free":true}]}`)
	liveModels(t, "paidv", `{"models":[{"id":"other"},{"id":"m"}]}`)

	if err := provider.SaveGroup(provider.Group{
		Name: "inner", Members: []string{"paidv/m", "zenfree/m"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := provider.SaveGroup(provider.Group{
		Name: "outer", Members: []string{"group/inner"}, Routing: provider.FreeFirst,
	}); err != nil {
		t.Fatal(err)
	}

	s := New()
	code, body := postAs(t, s, "", `{"model":"group/outer","messages":[{"role":"user","content":"x"}]}`)
	if code != 200 {
		t.Fatalf("code %d: %s", code, body)
	}
	last := s.trace.routes[len(s.trace.routes)-1]
	var freeRow, paidRow *Weighed
	for i := range last.Order {
		switch last.Order[i].Provider {
		case "zenfree":
			freeRow = &last.Order[i]
		case "paidv":
			paidRow = &last.Order[i]
		}
	}
	if freeRow == nil || paidRow == nil {
		t.Fatalf("the trace should weigh both members: %+v", last.Order)
	}
	if !freeRow.Free {
		t.Fatalf("the free seat inside the nested group lost its flag: %+v", *freeRow)
	}
	if paidRow.Free {
		t.Fatalf("the paid seat's row says Free: %+v", *paidRow)
	}
}
