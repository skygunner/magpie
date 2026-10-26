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
// with the real signal. Every step is gated on an event rather than a
// duration: the backend is seen to take the connection (held.took), the
// signal is seen by the process and stopServing cancels the test's backend
// context, and only then is the held connection handed on with its request
// unread — as a busy machine would hold one — so the old code cuts it.
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

	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tcp.Close() })
	held := &heldListener{Listener: tcp, took: make(chan struct{}), let: make(chan struct{})}
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

	if err := syscall.Kill(os.Getpid(), handoverSignal.(syscall.Signal)); err != nil {
		t.Fatal(err)
	}
	// The end has begun: the signal was seen, the backend's context was
	// cancelled by stopServing, and only then is the connection handed on
	// with its request unread. The order is on events (the signal handler
	// has run, and stopServing has cancelled the context the test replaced),
	// not on a duration, so a loaded runner cannot turn the precondition
	// into a red for the wrong reason.
	select {
	case <-backendCtx.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("the handover never began")
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
// net/http notes it.
type heldListener struct {
	net.Listener
	hold atomic.Bool
	took chan struct{} // closed once it holds one
	let  chan struct{} // closed to hand it on
}

func (l *heldListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil && l.hold.Swap(false) {
		close(l.took)
		<-l.let
	}
	return c, err
}

// gatewayHandoverForTest puts gateway.Handover back after the test; the
// dev backend sets it for its own listener.
func gatewayHandoverForTest() func() {
	was := gateway.Handover
	return func() { gateway.Handover = was }
}
