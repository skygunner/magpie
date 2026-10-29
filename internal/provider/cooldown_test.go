package provider

import (
	"testing"
	"time"

	"github.com/yetone/magpie/internal/settings"
)

func cooldownPtr[T any](v T) *T { return &v }
func TestCooldownInheritanceAndLegacy(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	s := settings.Load()
	s.Cooldown = settings.CooldownPolicy{RateCap: cooldownPtr(80)}
	if err := settings.Save(s); err != nil {
		t.Fatal(err)
	}
	defer func() { s.Cooldown = settings.CooldownPolicy{}; _ = settings.Save(s) }()
	p := Provider{ID: "policy", Key: "key", RestCap: 40, QuotaRest: 540, ModelCooldown: map[string]int{"m": 60}, Cooldown: settings.CooldownPolicy{RateCap: cooldownPtr(64)}, AccountCooldown: map[string]settings.CooldownPolicy{KeyID("key"): {RateCap: cooldownPtr(32), Models: map[string]int{"m": 0}}}}
	v := p.CooldownFor(KeyID("key"))
	if v.RateStart != 2 || v.RateCap != 32 || v.Source["rateCap"] != "account" || v.SlowCap != 40 || v.QuotaRest != 540 || v.Models["m"] != 0 {
		t.Fatalf("effective %#v", v)
	}
	if p.CooldownFor("other").RateCap != 64 {
		t.Fatal("account override leaked")
	}
	p.Cooldown = settings.CooldownPolicy{}
	p.RestCap = 0
	if p.CooldownFor("other").RateCap != 80 {
		t.Fatal("global ignored")
	}
	if (Provider{}).OwnQuotaRest(time.Now()) != 0 {
		t.Fatal("default must leave gateway fallback/probes in charge")
	}
}
func TestCooldownSignedInPicks(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	p := Provider{RestCap: 7, QuotaRest: 540, ModelCooldown: map[string]int{"m": 60}, LimitRestOff: true, Cooldown: settings.CooldownPolicy{RateCap: cooldownPtr(64)}, AccountCooldown: map[string]settings.CooldownPolicy{"user": {RateStart: cooldownPtr(4)}}}
	a := withPicks(Provider{Account: &Account{User: "user"}}, p)
	v := a.CooldownFor(a.CooldownIdentity())
	if v.RateStart != 4 || v.RateCap != 64 || v.QuotaRest != 540 || v.LimitRest || v.Models["m"] != 60 {
		t.Fatalf("lost account settings: %#v", v)
	}
}
func TestCooldownValidation(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	for _, p := range []settings.CooldownPolicy{{RateStart: cooldownPtr(0)}, {RateStart: cooldownPtr(3), RateCap: cooldownPtr(2)}, {QuotaResetAt: cooldownPtr("25:00")}, {Models: map[string]int{"m": 61}}} {
		if CheckCooldown(p) == nil {
			t.Fatalf("accepted %#v", p)
		}
	}
}

func TestCooldownKeyRotation(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	p := Provider{ID: "cooldown-rotation", Chat: "https://example.invalid/v1", Key: "before", AccountCooldown: map[string]settings.CooldownPolicy{KeyID("before"): {RateCap: cooldownPtr(32)}}}
	if err := Save(p); err != nil {
		t.Fatal(err)
	}
	_, now, err := ReplaceKey(p.ID, KeyID("before"), "after")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Find(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CooldownFor(now).RateCap != 32 {
		t.Fatal("rotation lost the policy")
	}
	if _, ok := got.AccountCooldown[KeyID("before")]; ok {
		t.Fatal("old fingerprint retained")
	}
}

func TestCooldownLegacyCLIStillChangesProviderPolicy(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	p := Provider{ID: "cooldown-cli", Chat: "https://example.invalid/v1", Key: "k", Cooldown: settings.CooldownPolicy{RateCap: cooldownPtr(64), SlowCap: cooldownPtr(10), QuotaRest: cooldownPtr(30)}}
	if err := Save(p); err != nil {
		t.Fatal(err)
	}
	if err := SetRestCap(p.ID, 20); err != nil {
		t.Fatal(err)
	}
	if err := SetQuotaRest(p.ID, 90); err != nil {
		t.Fatal(err)
	}
	got, err := Find(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	v := got.CooldownFor("")
	if v.RateCap != 20 || v.SlowCap != 20 || v.QuotaRest != 90 {
		t.Fatalf("CLI ignored: %#v", v)
	}
}
