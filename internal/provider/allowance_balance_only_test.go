package provider

import (
	"context"
	"testing"
	"time"
)

// A reading that answers with a balance alone (MiniMax Code's credits:
// windows [], balance "1740.72 credits") still knows the account — there
// is just no window that stops it. Kept out, the Routing page said
// "remaining allowance not known" for an account it had just read, and
// routing ranked it behind those known (#3352, 1234fsdc).
func TestBalanceOnlyReadingKnowsTheAccount(t *testing.T) {
	f := &fakeGoogle{load: `{"cloudaicompanionProject":"test-project"}`}
	googleSandbox(t, f)
	auth := googleAuth{AccessToken: "tok", RefreshToken: "rt-bal", Expiry: time.Now().Add(time.Hour).UnixMilli()}
	if err := addGoogleLogin("antigravity", "bal@example.com", "", auth); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		lastQuotas.Lock()
		lastQuotas.loaded, lastQuotas.m = false, nil
		lastQuotas.Unlock()
	})
	q := keepLast(SubscriptionQuota{Provider: "antigravity", Name: "Antigravity", User: "bal@example.com", Balance: "1740.72 credits"}, "bal@example.com")
	lastQuotas.Lock()
	lastQuotas.loaded, lastQuotas.m = false, nil
	lastQuotas.Unlock()
	a, ok := lastAllowances("antigravity")[q.User]
	if !ok {
		t.Fatalf("a balance-only reading left the account not known")
	}
	if len(a) != 0 {
		t.Fatalf("the balance-only account carries windows: %+v", a)
	}
	if used, _ := a.For("", time.Now()); used != 0 {
		t.Fatalf("the balance-only account counts as used: %v", used)
	}
}

// The live reading takes a balance-only account the same way, so the
// router weighs it as known and unused, not as not-known behind the rest.
func TestBalanceOnlyLiveReadingKnowsTheAccount(t *testing.T) {
	startWithNoAllowancesRead(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	LoginUsageVia(func(ctx context.Context, agent string) map[string]SubscriptionQuota {
		return map[string]SubscriptionQuota{"MiniMax336627": {Provider: "minimax-code", Windows: []QuotaWindow{}, Balance: "1740.72 credits"}}
	})
	t.Cleanup(func() { LoginUsageVia(nil) })
	a, ok := Allowances("minimax-code")["MiniMax336627"]
	if !ok {
		t.Fatalf("the live balance-only reading left the account not known")
	}
	if len(a) != 0 {
		t.Fatalf("the balance-only account carries windows: %+v", a)
	}
}
