package provider

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/yetone/magpie/internal/netproxy"
)

// droppingSystemOne answers a connection's first System One question and
// closes the connection on its second without a word, as a proxy or a
// vendor's front does with a keep-alive connection it has dropped
// meanwhile: the request was written into it, and the read finds EOF. The
// answer is the one SubQ gave the reporter's curl (#3665).
type droppingSystemOne struct {
	ln    net.Listener
	asked atomic.Int32
	keyed atomic.Bool // an idempotency key came on the wire
	wg    sync.WaitGroup
	mu    sync.Mutex
	open  []net.Conn
}

func newDroppingSystemOne(t *testing.T) *droppingSystemOne {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	u := &droppingSystemOne{ln: ln}
	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			u.mu.Lock()
			u.open = append(u.open, c)
			u.mu.Unlock()
			u.wg.Add(1)
			go func() {
				defer u.wg.Done()
				u.serve(c)
			}()
		}
	}()
	t.Cleanup(func() {
		// the client keeps its connections alive: they end with the test
		ln.Close()
		u.mu.Lock()
		for _, c := range u.open {
			c.Close()
		}
		u.mu.Unlock()
		u.wg.Wait()
	})
	return u
}

func (u *droppingSystemOne) serve(c net.Conn) {
	defer c.Close()
	br := bufio.NewReader(c)
	for n := 0; ; n++ {
		req, err := http.ReadRequest(br)
		if err != nil {
			return
		}
		io.Copy(io.Discard, req.Body)
		req.Body.Close()
		u.asked.Add(1)
		if _, ok := req.Header["X-Idempotency-Key"]; ok {
			u.keyed.Store(true)
		}
		if n > 0 {
			return // dropped: the question is read, and never answered
		}
		body := `{"model":"subq-snap-1.1","answers":{"escalate":{"type":"noul","noul":0.9758}},"usage":{"input_tokens":128,"output_tokens":0}}`
		fmt.Fprintf(c, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
	}
}

// installOwnRequests leaves http.DefaultClient as netproxy.Install does,
// so a request magpie makes of its own goes as it does in the app, and
// puts the global transport back when the test ends.
func installOwnRequests(t *testing.T) {
	t.Helper()
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatalf("http.DefaultTransport is %T, not *http.Transport", http.DefaultTransport)
	}
	proxy, h2 := base.Proxy, base.HTTP2
	client := http.DefaultClient.Transport
	netproxy.Install()
	t.Cleanup(func() {
		base.Proxy, base.HTTP2 = proxy, h2
		http.DefaultClient.Transport = client
		base.CloseIdleConnections()
	})
}

// A System One detection whose vendor dropped the keep-alive connection
// is answered: the transport sends the question again on a new one, as it
// does for the gateway's upstream requests, instead of the editor showing
// "Post …/systemone: EOF" for a vendor that answers curl (#3665).
func TestSystemOneDetectionOutlivesADroppedKeepAlive(t *testing.T) {
	installOwnRequests(t)
	up := newDroppingSystemOne(t)
	p := Provider{ID: "subq", Name: "SubQ", Key: "k", Decide: "http://" + up.ln.Addr().String() + "/v1"}
	d := p.DetectDecide(context.Background(), "subq-snap-1.1") // the connection's first question: answered, and kept alive
	if !d.OK {
		t.Fatalf("the first detection: %s", d.Error)
	}
	d = p.DetectDecide(context.Background(), "subq-snap-1.1") // goes on the kept connection, which drops it
	if !d.OK {
		t.Fatalf("the second detection: %s", d.Error)
	}
	if got := up.asked.Load(); got != 3 {
		t.Fatalf("the vendor was asked %d times, want 3 (the dropped one again on a new connection)", got)
	}
	if up.keyed.Load() {
		t.Fatal("the vendor was sent an X-Idempotency-Key: the mark is for Go's transport only")
	}
}
