package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/netproxy"
)

// A daily check-in is a signed call made once a day per account, and it
// went out on the global proxy while the account's requests and reads took
// its own (#2777, a1286743184): the check-in loop carried no account in
// its ctx, so the plugin's fetch and magpie's own vendor calls answered
// proxyOf with the global. CheckinVia is the wrap the runner builders
// apply: the do runs as the account, through the proxy its reads take.
func TestCheckinViaCarriesTheAccountsProxy(t *testing.T) {
	isolate(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	// written as it stands: Save rebuilds a key provider's AccountProxies
	// by key, and this is a signed-in account's map
	os.MkdirAll(filepath.Dir(Path()), 0o700)
	if err := os.WriteFile(Path(), []byte(`{"providers":[{"id":"trae-cn","name":"Trae CN","chat":"http://127.0.0.1:1/v1","key":"k",
		"proxy":"http://global.example:8080","accountProxies":{"hu":"http://hu-exit.example:8080"}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var got string
	do := CheckinVia("trae-cn", "hu", func(ctx context.Context) WorkBuddyCheckin {
		got = netproxy.Choice(ctx)
		return WorkBuddyCheckin{Outcome: CheckinClaimed}
	})
	if r := do(context.Background()); r.Outcome != CheckinClaimed {
		t.Fatalf("check-in: %+v", r)
	}
	if got != "http://hu-exit.example:8080" {
		t.Fatalf("the check-in ran on %q, want the account's own exit", got)
	}
	// an account with no proxy of its own follows the provider's, never
	// another account's
	var bare string
	do = CheckinVia("trae-cn", "mei", func(ctx context.Context) WorkBuddyCheckin {
		bare = netproxy.Choice(ctx)
		return WorkBuddyCheckin{}
	})
	do(context.Background())
	if bare != "http://global.example:8080" {
		t.Fatalf("an account without its own proxy ran on %q, want the provider's", bare)
	}
	// and a provider with no proxies at all leaves the choice empty
	var none string
	do = CheckinVia("qoder", "hu", func(ctx context.Context) WorkBuddyCheckin {
		none = netproxy.Choice(ctx)
		return WorkBuddyCheckin{}
	})
	do(context.Background())
	if none != "" {
		t.Fatalf("a provider without proxies was given %q", none)
	}
}
