package gateway

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// A slow-start rest reaches the ledger as it is made, like any other
// rest: the review of the persistence work found a Why=slow, Status=504
// rest live in the process and gone after it (restSlow wrote
// restingUntil and never saved) (#2008 review). The restart here is
// simulated the way the tests above do it — in-memory state cleared,
// the ledger loaded again — not a real daemon restart.
func TestSlowRestReachesTheLedger(t *testing.T) {
	fresh(t)
	up := &keyedSlow{key: "k1", model: "m"}
	up.slow.Store(true)
	serveOn(t, "a", "k1", []string{"m", "n"}, up, "k2")
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/m"}, Routing: provider.Ordered, FirstToken: 1}); err != nil {
		t.Fatal(err)
	}
	s := New()
	k1m := "a#" + provider.KeyID("k1") + "/m"
	ask := func() Try {
		t.Helper()
		up.reset()
		code, body := postAs(t, s, "", `{"model":"group/g","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
		if code != 200 || !strings.Contains(body, "from b") {
			t.Fatalf("%d %s", code, body)
		}
		return s.trace.routes[len(s.trace.routes)-1].Tries[0]
	}
	ask() // once: no rest
	tr := ask()
	if tr.Rest == nil || tr.Rest.Why != failSlow {
		t.Fatalf("no slow rest formed: %+v", tr)
	}
	before := *tr.Rest

	// the ledger on disk holds the rest already, without the test
	// saving anything itself
	if _, err := os.Stat(restsPath()); err != nil {
		t.Fatalf("no rests file after a slow rest: %v", err)
	}

	// a restart: the in-memory rests are gone, the ledger is loaded again
	restingUntil.Lock()
	restingUntil.m, restingUntil.note = map[string]time.Time{}, map[string]Rest{}
	restingUntil.from = ""
	restingUntil.Unlock()
	New()
	after, ok := restOf(k1m)
	if !ok || !after.Until.Equal(before.Until) || after.Why != failSlow || after.Status != before.Status || after.Failures != before.Failures || after.Model != before.Model {
		t.Fatalf("rest after restart: %+v, %v; want %+v", after, ok, before)
	}
}
