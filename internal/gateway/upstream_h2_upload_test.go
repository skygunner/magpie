package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// slowUplink sits between the gateway and a vendor like a saturated
// uplink: the gateway's bytes get to the vendor at rate bytes a second,
// while the vendor's own pass at once. The gateway's writes still land in
// a socket, as they do on a slow link until its send buffer fills, and the
// vendor hears nothing for as long as the body takes.
type slowUplink struct {
	ln   net.Listener
	to   string
	rate int
	wg   sync.WaitGroup
	mu   sync.Mutex
	open []net.Conn
}

func newSlowUplink(t *testing.T, to string, rate int) *slowUplink {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := &slowUplink{ln: ln, to: to, rate: rate}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			a, err := ln.Accept()
			if err != nil {
				return
			}
			b, err := net.Dial("tcp", to)
			if err != nil {
				a.Close()
				continue
			}
			p.mu.Lock()
			p.open = append(p.open, a, b)
			p.mu.Unlock()
			p.wg.Add(2)
			go p.paced(b, a)
			go p.splat(a, b)
		}
	}()
	t.Cleanup(func() {
		p.ln.Close()
		p.mu.Lock()
		for _, c := range p.open {
			c.Close()
		}
		p.mu.Unlock()
		p.wg.Wait()
	})
	return p
}

// paced carries what the gateway wrote to the vendor a twentieth of a
// second of the uplink at a time.
func (p *slowUplink) paced(dst, src net.Conn) {
	defer p.wg.Done()
	chunk := p.rate / 20
	if chunk < 1 {
		chunk = 1
	}
	buf := make([]byte, chunk)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		if err != nil {
			return
		}
	}
}

func (p *slowUplink) splat(dst, src net.Conn) {
	defer p.wg.Done()
	io.Copy(dst, src)
}

// #3909 (yxd-ym): a 175k-token turn (~680KB) on a 2-10KB/s uplink failed
// every try with "http2: client connection lost" after 35-45s, while the
// same uplink answered a small turn. The body is still going out, so the
// vendor sends nothing; the read-idle ping is written behind the body's
// own DATA and can only be answered once the last byte arrives, so the
// health check closed a connection that was working. A body that may still
// be under way is not watched by that ping.
func TestUploadStillUnderwayIsNotCutByHealthPing(t *testing.T) {
	fresh(t)
	body := []byte(`{"model":"gemini-4-argon","messages":[{"role":"user","content":"` +
		strings.Repeat("a", 256<<10) + `"}]}`)
	vendor := h2Vendor(t, func(w http.ResponseWriter, r *http.Request) {
		n, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			t.Errorf("the vendor reading the body: %v", err)
		}
		fmt.Fprintf(w, "%s %d", r.Proto, n)
	})
	// 256KB at 64KB/s: the body is still being delivered for about 4s, and
	// the vendor's frames arrive more than the 0.5s a ping is given here
	// apart, as they do at 2KB/s in the report
	path := newSlowUplink(t, vendor.Listener.Addr().String(), 64<<10)
	quickPings(t, vendor)
	s := New()
	t.Cleanup(s.client.CloseIdleConnections)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://"+path.ln.Addr().String()+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("a %d-byte body an uplink was still delivering failed: %v; want the vendor's reply", len(body), err)
	}
	defer res.Body.Close()
	got, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("HTTP/2.0 %d", len(body)); string(got) != want {
		t.Fatalf("the vendor read %q; want %q", got, want)
	}
}
