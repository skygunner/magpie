package netproxy

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/settings"
)

// stubLookup answers names from m for the test, and forgets what earlier
// lookups found.
func stubLookup(t *testing.T, m map[string]string) {
	old := lanLookup
	lanLookup = func(_ context.Context, host string) ([]net.IPAddr, error) {
		ip, ok := m[host]
		if !ok {
			return nil, errors.New("no such host")
		}
		return []net.IPAddr{{IP: net.ParseIP(ip)}}, nil
	}
	clear(lanCache.m)
	t.Cleanup(func() { lanLookup = old; clear(lanCache.m) })
}

// #982: with a proxy set, a Remote magpie on the user's network is asked
// directly; a vendor still goes through the proxy, and so does a provider
// with a proxy of its own.
func TestLANGoesAroundTheProxy(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	settings.Save(settings.Settings{Proxy: "127.0.0.1:7890"})
	stubLookup(t, map[string]string{"server.lan": "192.168.131.20", "nas.tail1234.ts.net": "100.101.102.103", "api.openai.com": "162.159.140.245"})
	req := func(host, choice string) *http.Request {
		return (&http.Request{URL: &url.URL{Scheme: "http", Host: host}}).WithContext(With(context.Background(), choice))
	}
	for _, h := range []string{"192.168.131.20:3425", "10.0.0.5:3425", "172.20.1.1", "100.101.102.103:3425", "[fd7a:115c:a1e0::1]:3425", "server.lan:3425", "nas.tail1234.ts.net:3425", "mac-mini.local:3425"} {
		if p, err := Func(req(h, "")); p != nil || err != nil {
			t.Errorf("%s: through %v (%v), want direct", h, p, err)
		}
	}
	for _, h := range []string{"api.openai.com", "unresolvable.example", "8.8.8.8"} {
		if p, _ := Func(req(h, "")); p == nil || p.Host != "127.0.0.1:7890" {
			t.Errorf("%s: %v, want the proxy", h, p)
		}
	}
	if p, _ := Func(req("192.168.131.20:3425", "socks5://127.0.0.1:1080")); p == nil || p.Host != "127.0.0.1:1080" {
		t.Errorf("a provider's own proxy: %v", p)
	}
}

// #1139: a relay's name that resolves to a loopback address — a hosts
// file line or a proxy tool's DNS takeover on this computer — is not on
// the user's own network, so the system proxy the user follows still
// carries it, instead of a direct dial to 127.x that times out.
func TestLoopbackResolvedNameGoesThroughTheProxy(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	settings.Save(settings.Settings{Proxy: "127.0.0.1:7890"})
	old := lanLookup
	lanLookup = func(_ context.Context, host string) ([]net.IPAddr, error) {
		seen := map[string][]string{
			"relay.example":         {"127.115.2.250"},
			"relay6.example":        {"::1"},
			"relay-and-nas.example": {"127.0.0.1", "192.168.131.20"},
			"localhost":             {"127.0.0.1", "::1"},
			"server.lan":            {"192.168.131.20"},
		}[host]
		if seen == nil {
			return nil, errors.New("no such host")
		}
		addrs := make([]net.IPAddr, 0, len(seen))
		for _, s := range seen {
			addrs = append(addrs, net.IPAddr{IP: net.ParseIP(s)})
		}
		return addrs, nil
	}
	clear(lanCache.m)
	t.Cleanup(func() { lanLookup = old; clear(lanCache.m) })
	req := func(host string) *http.Request {
		return (&http.Request{URL: &url.URL{Scheme: "http", Host: host}}).WithContext(context.Background())
	}
	for _, h := range []string{"relay.example:443", "relay6.example:443"} {
		if p, err := Func(req(h)); p == nil || p.Host != "127.0.0.1:7890" {
			t.Errorf("%s: through %v (%v), want the system proxy", h, p, err)
		}
	}
	// a loopback address beside one on the network still dials directly:
	// the dial moves on to the address that answers
	if p, err := Func(req("relay-and-nas.example:443")); p != nil || err != nil {
		t.Errorf("relay-and-nas.example: through %v (%v), want direct", p, err)
	}
	// localhost, a .local name and an address written out are untouched
	for _, h := range []string{"localhost:3425", "server.lan:3425", "192.168.131.20:3425", "127.0.0.1:3425", "[::1]:3425", "mac-mini.local:3425"} {
		if p, err := Func(req(h)); p != nil || err != nil {
			t.Errorf("%s: through %v (%v), want direct", h, p, err)
		}
	}
}

// The reporter's case end to end: a proxy that answers 502 Bad Gateway to
// whatever it is asked (as Clash does for a host it can't reach), set in
// magpie, and another magpie at this computer's LAN address. The list
// comes from the other magpie, not the proxy's 502.
func TestLANModelListNotThroughProxy(t *testing.T) {
	var lan net.IP
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil && n.IP.IsPrivate() {
			lan = n.IP
			break
		}
	}
	if lan == nil {
		t.Skip("no private IPv4 address on this computer")
	}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer proxy.Close()
	ln, err := net.Listen("tcp", net.JoinHostPort(lan.String(), "0"))
	if err != nil {
		t.Skip("can't listen on", lan, err)
	}
	remote := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-magpie-key" {
			http.Error(w, "no API key was sent", http.StatusUnauthorized)
			return
		}
		io.WriteString(w, `{"object":"list","data":[{"id":"codex/gpt-6.1-sol"}]}`)
	}))
	remote.Listener = ln
	remote.Start()
	defer remote.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	settings.Save(settings.Settings{Proxy: proxy.URL})
	clear(lanCache.m)
	c := &http.Client{Transport: Dispatch(&http.Transport{Proxy: Func})}
	req, _ := http.NewRequest("GET", remote.URL+"/v1/models", nil)
	req.Header.Set("Authorization", "Bearer sk-magpie-key")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("%s: %s %s", remote.URL, res.Status, b)
	}
	if !strings.Contains(string(b), "codex/gpt-6.1-sol") {
		t.Fatalf("not the other magpie's list: %s", b)
	}
}
