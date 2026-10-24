package netproxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// The v4 fallback's mark, on a host that has no IPv4 to fall back to
// (#3519, Tinkler-i: a NAS behind a TLS reverse proxy whose name has only
// an AAAA record; the marked tcp4 dial answered "dial tcp4: lookup …: no
// such host" for half an hour at a time).

// a AAAA-only name's dialer: tcp4 has no address for it, tcp reaches the
// v6 listener.
func aaaaOnly(v6 net.Addr) func(context.Context, string, string) (net.Conn, error) {
	var d net.Dialer
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if network == "tcp4" {
			return nil, &net.DNSError{Err: "no such host", Name: strings.Split(addr, ":")[0], IsNotFound: true}
		}
		return d.DialContext(ctx, "tcp", v6.String())
	}
}

// TestUnresolvableIPv4DialClearsTheMark: a host:port whose IPv4 dial
// answered no-such-host has no IPv4 to be dialed instead - the mark's
// reason is gone with that answer, and the mark goes with it.
func TestUnresolvableIPv4DialClearsTheMark(t *testing.T) {
	v6, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skip("no ::1:", err)
	}
	defer v6.Close()
	t.Cleanup(func() { v4Only.Lock(); v4Only.at = nil; v4Only.Unlock() })

	base := &http.Transport{DialContext: aaaaOnly(v6.Addr())}
	v4Fallback(base)
	markV4("aaaa-only.example.com:443")
	c, err := base.DialContext(context.Background(), "tcp", "aaaa-only.example.com:443")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	c.Close()
	if !v4Since("aaaa-only.example.com:443").IsZero() {
		t.Fatal("aaaa-only.example.com:443 is still marked IPv4-only after its IPv4 dial failed with \"dial tcp4: lookup aaaa-only.example.com: no such host\"; the mark outlives the reason it was made")
	}
}

// TestHostWithNoIPv4IsDialedOverIPv6Again: a request to a marked host
// with no IPv4 goes over IPv6 again, not out with tcp4's no-such-host.
func TestHostWithNoIPv4IsDialedOverIPv6Again(t *testing.T) {
	v6, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skip("no ::1:", err)
	}
	defer v6.Close()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	}))
	srv.Listener = v6
	srv.StartTLS()
	defer srv.Close()
	t.Cleanup(func() { v4Only.Lock(); v4Only.at = nil; v4Only.Unlock() })

	base := &http.Transport{
		Proxy:           func(*http.Request) (*url.URL, error) { return nil, nil },
		TLSClientConfig: srv.Client().Transport.(*http.Transport).TLSClientConfig.Clone(),
		DialContext:     aaaaOnly(v6.Addr()),
	}
	markV4("aaaa-only.example.com:443")
	c := &http.Client{Transport: Dispatch(base)}
	res, err := c.Get("https://aaaa-only.example.com/")
	if err != nil {
		t.Fatalf("dial: %v (dialed [tcp4])", err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(b) != "ok" {
		t.Fatalf("reply %q", b)
	}
}

// TestMarkedHostWithIPv4KeepsUsingIt: the fallback itself stays for a
// host whose IPv4 answers - the TUN case it was made for.
func TestMarkedHostWithIPv4KeepsUsingIt(t *testing.T) {
	v6, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skip("no ::1:", err)
	}
	defer v6.Close()
	t.Cleanup(func() { v4Only.Lock(); v4Only.at = nil; v4Only.Unlock() })

	var mu sync.Mutex
	var dials []string
	var d net.Dialer
	base := &http.Transport{DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		mu.Lock()
		dials = append(dials, network)
		mu.Unlock()
		if network == "tcp4" {
			return d.DialContext(ctx, "tcp4", "127.0.0.1:80")
		}
		return d.DialContext(ctx, "tcp", v6.Addr().String())
	}}
	v4Fallback(base)
	markV4("dual.example:443")
	c, err := base.DialContext(context.Background(), "tcp", "dual.example:443")
	if err != nil {
		t.Skipf("no 127.0.0.1 listener: %v", err)
	}
	c.Close()
	if got := strings.Join(dials, ","); got != "tcp4" {
		t.Fatalf("dials %s, want tcp4", got)
	}
}
