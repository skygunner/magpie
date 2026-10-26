//go:build dev && !windows

package gui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/gateway"
)

// A make dev handover must answer a connection the old backend took, whose
// first request it reads only after the handover began: net/http's Shutdown
// closes such a connection without an answer, and the page sees the
// connection end with nothing (#3648). The test drives the real devBackend
// with the real signal, and every step is gated on an event rather than a
// duration or a goroutine-scheduling guess: the backend is seen to take the
// connection (held.took), the handover is seen to have begun (held.closed —
// see heldListener.Close for why that event orders the release after the
// old code's shutdown mark and before the fix's Shutdown), and only then is
// the held connection handed on with its request unread, as a busy machine
// would hold one.
func TestDevBackendHandoverAnswersAConnectionItTook(t *testing.T) {
	if handoverSignal == nil {
		t.Fatal("no handover signal on this platform")
	}
	// devBackend's handover calls stopServing, which cancels the process's
	// backend context for good: the test runs against its own, and puts the
	// real one back after.
	oldCtx, oldEnd := backendCtx, endBackend
	backendCtx, endBackend = context.WithCancel(context.Background())
	t.Cleanup(func() {
		endBackend()
		backendCtx, endBackend = oldCtx, oldEnd
	})
	wasHandover := gatewayHandoverForTest()
	t.Cleanup(wasHandover)

	// The test listens for the handover signal itself, before the backend
	// can: the base registers its own Notify only after `go srv.Serve`, so
	// a signal sent too early would otherwise take the process down. A
	// signal the backend misses (it wasn't listening yet) leaves it in
	// <-sig, which the bounded wait below reports loudly — never a quiet
	// pass.
	sigSeen := make(chan os.Signal, 8)
	signal.Notify(sigSeen, handoverSignal)
	t.Cleanup(func() { signal.Stop(sigSeen) })

	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tcp.Close() })
	held := &heldListener{Listener: tcp, took: make(chan struct{}), let: make(chan struct{}), closed: make(chan struct{})}
	letGo := sync.OnceFunc(func() { close(held.let) })
	t.Cleanup(letGo)
	t.Setenv("MAGPIE_DEV_BACKEND", tcp.Addr().String())
	// The backend's listener comes from Inherit, so the test sees the very
	// accept devBackend's server makes.
	gateway.Inherit(held, func() {})

	done := make(chan error, 1)
	go func() {
		done <- devBackend(func(Windows) http.Handler {
			return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				io.WriteString(rw, "page-served")
			})
		})
	}()

	held.hold.Store(true)
	c, err := net.Dial("tcp", tcp.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	select {
	case <-held.took:
	case <-time.After(10 * time.Second):
		t.Fatal("the backend never took the connection")
	}

	// The end must have marked the server shutting down before the
	// connection is handed on with its request unread — that is the moment
	// the old code cuts it. The old code's Shutdown marks the server
	// (go1.26.8 net/http/server.go:3151) before it closes its listeners
	// (:3154), and the fix's finish closes its listener before it calls
	// Shutdown, so the listener's Close beginning is the barrier in both
	// directions. The base registers its signal handler after Serve starts;
	// retry on observed signal delivery until Close starts, with one failure deadline.
	if err := syscall.Kill(os.Getpid(), handoverSignal.(syscall.Signal)); err != nil {
		t.Fatal(err)
	}
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
waitForClose:
	for {
		select {
		case <-held.closed:
			break waitForClose
		case <-sigSeen:
			if err := syscall.Kill(os.Getpid(), handoverSignal.(syscall.Signal)); err != nil {
				t.Fatal(err)
			}
		case <-deadline.C:
			t.Fatal("the handover never began")
		}
	}
	letGo()
	if _, err := fmt.Fprintf(c, "GET /api/state HTTP/1.1\r\nHost: dev\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	c.SetReadDeadline(time.Now().Add(10 * time.Second))
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatalf("a request on a connection the backend took, read once its handover began: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || string(body) != "page-served" {
		t.Fatalf("a request on a connection the backend took, read once its handover began: %d %q", resp.StatusCode, body)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("the handover ended with: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the handover never ended once its connection was answered")
	}
}

// heldListener holds the next connection it takes, once told to, until let
// go: as a busy machine can hold the goroutine that took one, before
// net/http notes it. Its Close tells the test it has begun, then lets the
// real listener go: Shutdown calls Close holding net/http's own lock, so it
// must not block here, and the event alone orders the test's release —
// after the old code's shutdown mark, before the fix's Shutdown.
type heldListener struct {
	net.Listener
	hold    atomic.Bool
	took    chan struct{} // closed once it holds one
	let     chan struct{} // closed to hand it on
	closed  chan struct{} // closed once Close has begun
	closeOn sync.Once
}

func (l *heldListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil && l.hold.Swap(false) {
		close(l.took)
		<-l.let
	}
	return c, err
}

func (l *heldListener) Close() error {
	l.closeOn.Do(func() { close(l.closed) })
	return l.Listener.Close()
}

// gatewayHandoverForTest puts gateway.Handover back after the test; the
// dev backend sets it for its own listener.
func gatewayHandoverForTest() func() {
	was := gateway.Handover
	return func() { gateway.Handover = was }
}
