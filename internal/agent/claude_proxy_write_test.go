package agent

import (
	"net/http"
	"net/http/httptest"
	atomicpkg "sync/atomic"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/gateway"
)

// #3512: a magpie that has only just started, Claude Code behind a proxy
// that forwards to the gateway and takes a moment to answer (a company's
// compliance proxy over a VPN: 0.3s, 1.5s). The gateway's answer there is
// only ever asked for in the background, so the first write that keeps the
// address must wait for it: taken for the user's own endpoint, it put
// magpie's own address on Claude Code, out from under the proxy every
// request has to go through.
func TestClaudeFirstWriteWaitsForTheProxy(t *testing.T) {
	reachForget()
	for _, late := range []time.Duration{300 * time.Millisecond, 1500 * time.Millisecond} {
		t.Run(late.String(), func(t *testing.T) {
			t.Setenv("MAGPIE_ADDR", "")
			var hits atomicpkg.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				time.Sleep(late)
				w.Write([]byte(`{"name":"magpie"}`))
			}))
			t.Cleanup(srv.Close)
			relay := srv.URL + "/corp-relay"
			home, path := claudeSettings(t, `{"env":{"ANTHROPIC_BASE_URL":"`+relay+`","ANTHROPIC_AUTH_TOKEN":"`+gateway.Token+`"}}`)
			saveDeepseek(t)
			// magpie's answer at the proxy is asked over the network, as it
			// is when magpie runs: the test's server is the slow part
			was := magpieAt
			magpieAt = func(base string) (bool, bool) {
				return probed("magpie "+base, func() bool { return magpieProbe(base) })
			}
			t.Cleanup(func() { magpieAt = was })
			if err := claude(home).Apply("model", "deepseek/pro"); err != nil {
				t.Fatal(err)
			}
			if got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); got != relay {
				t.Fatalf("ANTHROPIC_BASE_URL became %q, not the proxy's %q", got, relay)
			}
			if got, _ := edit.GetJSON(path, "model"); got != "deepseek/pro" {
				t.Fatalf("model is %q", got)
			}
			asked := hits.Load()

			// A write with no proxy in the way waits for nothing: magpie's
			// own address is no proxyBase, so nothing is probed and the
			// write never asks a proxy.
			home, _ = claudeSettings(t, `{}`)
			if err := claude(home).Apply("model", "deepseek/pro"); err != nil {
				t.Fatal(err)
			}
			if n := hits.Load(); n != asked {
				t.Fatalf("a local write probed the proxy %d more times", n-asked)
			}
		})
	}
}
