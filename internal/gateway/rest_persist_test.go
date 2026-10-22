package gateway

import (
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// A provider's quota-rest is a deadline, not a new duration after an update.
func TestQuotaRestSurvivesGatewayRestart(t *testing.T) {
	d, srv := deadChain(t, "dead-a", "dead-b")
	if err := provider.SetQuotaRest("mockdead", 18*60); err != nil {
		t.Fatal(err)
	}
	if got := postDead(t, srv); got.Code != 429 {
		t.Fatalf("initial quota refusal: %d", got.Code)
	}
	if got := d.took(); len(got) != 2 {
		t.Fatalf("initial vendor calls: %v", got)
	}
	id := "mockdead#" + provider.KeyID("dead-a")
	before, ok := restOf(id)
	if !ok || before.By != "quota" {
		t.Fatalf("initial rest: %+v, %v", before, ok)
	}

	// A process restart discards its in-memory state and starts a new gateway.
	restingUntil.Lock()
	restingUntil.m, restingUntil.note = map[string]time.Time{}, map[string]Rest{}
	restingUntil.from = ""
	restingUntil.Unlock()
	srv = New()
	after, ok := restOf(id)
	if !ok || !after.Until.Equal(before.Until) || after.Why != failQuota {
		t.Fatalf("rest after restart: %+v, %v; want %+v", after, ok, before)
	}
	if got := postDead(t, srv); got.Code != 429 {
		t.Fatalf("rested request: %d", got.Code)
	}
	if got := d.took(); len(got) != 0 {
		t.Fatalf("vendor was asked during saved rest: %v", got)
	}

	// A manual un-rest must also stay lifted after another restart.
	if !srv.Unrest(id) {
		t.Fatal("saved rest cannot be lifted")
	}
	restingUntil.Lock()
	restingUntil.m, restingUntil.note = map[string]time.Time{}, map[string]Rest{}
	restingUntil.from = ""
	restingUntil.Unlock()
	New()
	if r, ok := restOf(id); ok {
		t.Fatalf("lifted rest returned after restart: %+v", r)
	}
}
