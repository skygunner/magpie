package gateway

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type tlsHandoverListener struct {
	net.Listener
	reading chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func (l *tlsHandoverListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &tlsHandoverConn{Conn: c, reading: l.reading}, nil
}

func (l *tlsHandoverListener) Close() error {
	err := l.Listener.Close()
	l.once.Do(func() { close(l.closed) })
	return err
}

type tlsHandoverConn struct {
	net.Conn
	reading chan struct{}
	once    sync.Once
}

func (c *tlsHandoverConn) Read(b []byte) (int, error) {
	c.once.Do(func() { close(c.reading) })
	return c.Conn.Read(b)
}

func TestHandedOffTLSGatewayAnswersAConnectionWaitingForItsFirstByte(t *testing.T) {
	fresh(t)
	cert, key := tlsFiles(t)
	writeCert(t, cert, key, "relay.example")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	addr := ln.Addr().String()
	t.Setenv("MAGPIE_ADDR", addr)
	tracked := &tlsHandoverListener{Listener: ln, reading: make(chan struct{}), closed: make(chan struct{})}
	ready := make(chan struct{})
	Inherit(tracked, func() { close(ready) })
	old := New()
	ctx, stopOld := context.WithCancel(context.Background())
	oldDone := make(chan error, 1)
	go func() { oldDone <- old.ListenAndServe(ctx) }()
	t.Cleanup(func() {
		stopOld()
		select {
		case err := <-oldDone:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(10 * time.Second):
			t.Error("the old gateway never ended")
		}
	})
	wait := func(signal <-chan struct{}, state string) {
		t.Helper()
		select {
		case <-signal:
		case <-time.After(10 * time.Second):
			t.Fatalf("the gateway never %s", state)
		}
	}
	wait(ready, "inherited its listener")

	// Given: the sniff goroutine has begun reading a silent connection.
	silent, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()
	wait(tracked.reading, "started reading the connection's first byte")

	// When: handover closes accepting before the client sends its request.
	old.HandOff()
	stopOld()
	wait(tracked.closed, "stopped accepting for handover")
	if err := silent.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("GET", "http://"+addr+"/v1/models", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := req.Write(silent); err != nil {
		t.Fatalf("the connection was cut at the handover: %v", err)
	}

	// Then: the old gateway answers the request accepted before handover.
	res, err := http.ReadResponse(bufio.NewReader(silent), req)
	if err != nil {
		t.Fatalf("no answer to a request sent once the handover began: %v", err)
	}
	b, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || !strings.Contains(string(b), `"data"`) {
		t.Fatalf("the silent connection's request: %d %s", res.StatusCode, b)
	}
}
