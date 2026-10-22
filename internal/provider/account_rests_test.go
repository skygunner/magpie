package provider

import (
	"testing"
	"time"
)

// The rests set on a signed-in provider reach the provider the gateway
// routes by, not only the record in providers.json: rest-cap, quota-rest
// and its until-time are carried onto the account like the queue and
// concurrency settings are (#3176). Dropped by the merge, `magpie provider
// rest-cap <id> 6h` wrote the record and read back magpie's own cap, and a
// repeatedly failing subscription member kept probing at the default cap.
func TestAccountKeepsItsRests(t *testing.T) {
	googleSandbox(t, &fakeGoogle{})
	if err := addGoogleLogin("antigravity", "u@example.com", "", googleAuth{
		AccessToken: "t", RefreshToken: "rt", Project: "p",
		Expiry: time.Now().Add(time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := addGoogleLogin("antigravity", "second@example.com", "", googleAuth{
		AccessToken: "t2", RefreshToken: "rt2", Project: "p2",
		Expiry: time.Now().Add(time.Hour).UnixMilli(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := switchGoogleLogin("antigravity", "u@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := setGoogleLoginOn("antigravity", "second@example.com", true); err != nil {
		t.Fatal(err)
	}
	if err := Save(Provider{ID: "antigravity", RestCap: 360, QuotaRest: 120, QuotaResetAt: "23:30"}); err != nil {
		t.Fatal(err)
	}
	p, err := Find("antigravity")
	if err != nil {
		t.Fatal(err)
	}
	if p.RestCap != 360 || p.QuotaRest != 120 || p.QuotaResetAt != "23:30" {
		t.Fatalf("the account's rests: cap %d, quota %d, at %q", p.RestCap, p.QuotaRest, p.QuotaResetAt)
	}
	// every account of the provider keeps them, as it keeps the queue (#1899)
	if len(p.AlsoOn()) == 0 {
		t.Fatal("no second account to check")
	}
	for _, q := range p.AlsoOn() {
		if q.RestCap != 360 || q.QuotaRest != 120 || q.QuotaResetAt != "23:30" {
			t.Fatalf("also on: cap %d, quota %d, at %q", q.RestCap, q.QuotaRest, q.QuotaResetAt)
		}
	}
}
