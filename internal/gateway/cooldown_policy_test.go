package gateway

import (
	"net/http"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

func TestCooldownDefaultRateLadder(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	c := candidate{p: provider.Provider{ID: "new-rate-default", Key: "k"}, rest: "new-rate-default", model: "m"}
	defer servedCandidate(c, 1)
	for _, minutes := range []int{2, 4, 8, 16, 32, 64, 64} {
		rest := (&Server{}).restAfter(c, 429, http.Header{}, []byte(`{"error":{"message":"Too many requests","type":"rate_limit_error"}}`))
		d := time.Until(rest.Until)
		if d > time.Duration(minutes)*time.Minute || d < time.Duration(minutes)*time.Minute-time.Second {
			t.Fatalf("rest %v; want %dm", d, minutes)
		}
		restingUntil.Lock()
		restingUntil.m[c.restKey()] = time.Now().Add(-time.Second)
		restingUntil.Unlock()
	}
}
func TestCooldownDefaultQuotaAndModelScope(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	c := candidate{p: provider.Provider{ID: "new-quota-default", Key: "k"}, rest: "new-quota-default", model: "m"}
	defer servedCandidate(c, 1)
	r := (&Server{}).restAfter(c, 429, http.Header{}, []byte(`{"error":{"message":"quota exhausted"}}`))
	if d := time.Until(r.Until); d < 30*time.Minute-time.Second || d > 30*time.Minute {
		t.Fatalf("quota wait %v", d)
	}
	servedCandidate(c, 1)
	cap := 3
	c.p.Cooldown = settings.CooldownPolicy{SlowCap: &cap}
	if _, ok := restSlow(c, time.Now()); ok {
		t.Fatal("first timeout should not rest")
	}
	r, ok := restSlow(c, time.Now())
	if !ok || r.Key != c.restID() {
		t.Fatalf("model rest: %#v", r)
	}
	if _, ok := restOf(c.restKey()); ok {
		t.Fatal("model timeout rested entire key")
	}
}

func TestCooldownTryNowKeepsRateLadder(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	c := candidate{p: provider.Provider{ID: "try-now-rate", Key: "k"}, rest: "try-now-rate", model: "m"}
	defer servedCandidate(c, 1)
	s := New()
	body := []byte(`{"error":{"message":"Too many requests","type":"rate_limit_error"}}`)
	s.restAfter(c, 429, nil, body)
	if !s.Unrest(c.restKey()) || resting(c) {
		t.Fatal("Try now did not lift rest")
	}
	r := s.restAfter(c, 429, nil, body)
	if r.Failures != 2 || time.Until(r.Until) < 4*time.Minute-time.Second {
		t.Fatalf("lost ladder: %#v", r)
	}
}
func TestCooldownLargeStartDoesNotOverflow(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	cap := time.Duration(provider.MaxQuotaRest) * time.Minute
	if got := backoffWait(cap/2, cap, 100); got != cap {
		t.Fatalf("got %v", got)
	}
}

func TestCooldownLongTimeoutCapKeepsItsLadder(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	cap := 64
	c := candidate{p: provider.Provider{ID: "long-timeout-policy", Key: "k", Cooldown: settings.CooldownPolicy{SlowCap: &cap}}, rest: "long-timeout-policy", model: "m"}
	defer servedCandidate(c, 1)
	now := time.Now()
	if _, ok := restSlow(c, now); ok {
		t.Fatal("first timeout rested")
	}
	for _, minutes := range []int{1, 2, 4, 8, 16, 32, 64, 64} {
		r, ok := restSlow(c, now.Add(time.Second))
		if !ok {
			t.Fatal("missing timeout rest")
		}
		d := r.Until.Sub(now.Add(time.Second))
		if d != time.Duration(minutes)*time.Minute {
			t.Fatalf("rest %v, want %dm", d, minutes)
		}
		now = r.Until
		restingUntil.Lock()
		restingUntil.m[c.restID()] = time.Now().Add(-time.Second)
		restingUntil.Unlock()
	}
}
