package mcpauth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/mcpauth"
	"github.com/yetone/magpie/internal/mcpauth/mcpauthtest"
	"github.com/yetone/magpie/internal/settings"
)

// #3194: an authorization server that advertises a resource in its
// protected-resource metadata may still answer 400 invalid_target to a
// token request that carries it (Semgrep). The token stage then goes on
// once without the resource; everything else about the request stands,
// and the record keeps the resource it advertised.
func TestTokenStageGoesOnWithoutAnUnacceptedResource(t *testing.T) {
	for _, tt := range []struct {
		name  string
		fresh bool // refuse the resource from the start, at the exchange
	}{
		{"exchange", true},
		{"refresh", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home(t)
			f := mcpauthtest.New(t)
			f.RefuseResource = tt.fresh
			f.SignIn(t, "semgrep")
			got, _ := mcpauth.Get("semgrep")
			if !tt.fresh {
				f.RefuseResource = true
				stale := got
				access, err := mcpauth.Renew(t.Context(), "semgrep", stale.Access)
				if err != nil {
					t.Fatalf("renewal failed despite a usable retry without resource: %v", err)
				}
				got, _ = mcpauth.Get("semgrep")
				if access == stale.Access || got.Access != access || got.Refresh == stale.Refresh {
					t.Fatal("renewal did not persist rotated tokens")
				}
			}
			calls, resources := f.TokenCalls, f.TokenResources
			if !tt.fresh {
				calls-- // the successful setup exchange
				resources = resources[1:]
			}
			if calls != 2 || len(resources) != 2 || resources[0] != f.URL || resources[1] != "" {
				t.Fatalf("token endpoint received %d requests with resources %q", calls, resources)
			}
			rec, _ := mcpauth.Get("semgrep")
			if rec.Resource != f.URL {
				t.Fatalf("the record lost the resource the server advertised: %q", rec.Resource)
			}
			if rec.Access != got.Access || rec.Access == "" || rec.Refresh == "" {
				t.Fatalf("the renewed tokens were not kept: %+v", rec)
			}
		})
	}
}

// A refusal of anything else than invalid_target is not a word about the
// resource: an invalid_grant stands, and the request is not sent again.
func TestInvalidGrantIsNotRetried(t *testing.T) {
	home(t)
	f := mcpauthtest.New(t)
	f.SignIn(t, "neon")
	f.Revoke()
	f.ForgetRefresh()
	stale, _ := mcpauth.Get("neon")
	if _, err := mcpauth.Renew(t.Context(), "neon", stale.Access); err == nil {
		t.Fatal("renewed with a refresh token the server forgot")
	}
	if f.TokenCalls != 2 {
		t.Fatalf("the token endpoint saw %d requests, want setup plus one refused answer", f.TokenCalls)
	}
}

// A record without a resource is sent as it is: a server that keeps
// answering invalid_target gets one request, not a loop.
func TestNoResourceIsNotRetried(t *testing.T) {
	home(t)
	f := mcpauthtest.New(t)
	f.SignIn(t, "neon")
	p := filepath.Join(settings.Dir(), "mcp-signins.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]*mcpauth.Record
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	r := m["neon"]
	r.Resource = ""
	if b, err = json.MarshalIndent(m, "", "  "); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(b, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	f.RefuseTarget = true
	stale, _ := mcpauth.Get("neon")
	if _, err := mcpauth.Renew(t.Context(), "neon", stale.Access); err == nil {
		t.Fatal("renewed against a server that refuses every target")
	}
	if f.TokenCalls != 2 {
		t.Fatalf("the token endpoint saw %d requests, want setup plus one with no retry", f.TokenCalls)
	}
}

// A server that refuses invalid_target whatever the request carries gets
// the first one and the one without the resource, and then the answer:
// the retry is bounded at one.
func TestPersistentRefusalIsBounded(t *testing.T) {
	home(t)
	f := mcpauthtest.New(t)
	f.RefuseTarget = true
	st, err := mcpauth.Start(t.Context(), "neon", f.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer mcpauth.Cancel(st.ID)
	resp, err := http.Get(st.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got, _ := mcpauth.Progress(st.ID); got.State != "failed" {
		t.Fatalf("sign-in state %s: %s", got.State, got.Error)
	}
	if f.TokenCalls != 2 {
		t.Fatalf("the token endpoint saw %d requests, want the refused one and the one retry", f.TokenCalls)
	}
	if _, ok := mcpauth.Get("neon"); ok {
		t.Fatal("a refused sign-in was kept")
	}
}

// The retry is the same request without the resource: a client registered
// for client_secret_post still sends its secret on it.
func TestRetryKeepsTheClientSecret(t *testing.T) {
	home(t)
	f := mcpauthtest.New(t)
	f.AuthMethods = []string{"client_secret_post"}
	f.RefuseResource = true
	f.SignIn(t, "neon")
	if f.TokenCalls != 2 {
		t.Fatalf("the token endpoint saw %d requests, want the refused one and the retry", f.TokenCalls)
	}
	if r, _ := mcpauth.Get("neon"); r.AuthMethod != "client_secret_post" || r.Secret == "" {
		t.Fatalf("record %+v", r)
	}
}

// A cancellation during the held retry keeps the sign-in out: no token is
// saved for a flow the user ended while its request was in flight.
func TestCanceledRetrySavesNothing(t *testing.T) {
	home(t)
	f := mcpauthtest.New(t)
	f.RefuseResource = true
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	calls := 0
	oldClient := mcpauth.Client
	base := http.DefaultTransport
	mcpauth.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		resp, err := base.RoundTrip(r)
		if err == nil && r.URL.Path == "/oauth/token" {
			mu.Lock()
			calls++
			n := calls
			mu.Unlock()
			if n == 2 {
				once.Do(func() { close(entered) })
				<-release
			}
		}
		return resp, err
	})}
	t.Cleanup(func() { mcpauth.Client = oldClient })

	st, err := mcpauth.Start(t.Context(), "demo", f.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer mcpauth.Cancel(st.ID)
	done := make(chan error, 1)
	go func() {
		resp, gerr := http.Get(st.URL)
		if gerr == nil {
			resp.Body.Close()
		}
		done <- gerr
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("callback finished before the retry: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	mcpauth.Cancel(st.ID)
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got, _ := mcpauth.Progress(st.ID); got.State != "canceled" {
		t.Fatalf("state after cancellation = %q", got.State)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 2 {
		t.Fatalf("the token endpoint saw %d requests, want the refused one and the held retry", calls)
	}
	if _, ok := mcpauth.Get("demo"); ok {
		t.Fatal("a canceled sign-in was saved")
	}
}

// roundTripFunc adapts a function to an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
