package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// #3191 (LuwenOosten): a request whose body the vendor stops taking — its
// HTTP/2 stream's flow-control window never opens again, while the
// connection itself answers and stays open — sat in the write phase for as
// long as the client waited: ResponseHeaderTimeout begins only once a
// request is written, and nothing else bounded the write, so the group
// never asked its next member. The transport now lets such a request go,
// and the group moves on.
func TestStalledRequestWriteLetsTheGroupMoveOn(t *testing.T) {
	fresh(t)
	was := upstreamWriteStall
	upstreamWriteStall = 100 * time.Millisecond
	t.Cleanup(func() { upstreamWriteStall = was })

	asked := make(chan struct{}, 1) // the vendor has the request's headers
	release := make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	vendor := h2Vendor(t, func(w http.ResponseWriter, r *http.Request) {
		// nothing of the body is read, so its window closes and never
		// opens again, as a vendor's that took the request and stopped
		select {
		case asked <- struct{}{}:
		default:
		}
		<-release
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"a","choices":[{"message":{"role":"assistant","content":"from a"}}]}`)
	})
	quickPings(t, vendor)
	if err := provider.Save(provider.Provider{ID: "a", Name: "A", Key: "k", Models: []string{"m"},
		Chat: "https://" + vendor.Listener.Addr().String() + "/v1"}); err != nil {
		t.Fatal(err)
	}
	serveOn(t, "b", "k", []string{"m"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"b","choices":[{"message":{"role":"assistant","content":"from b"}}]}`)
	}))
	if err := provider.SaveGroup(provider.Group{Name: "Auto", Members: []string{"a/m", "b/m"}, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	// the reporter's turn: a long prompt, a few MB, more than the vendor's
	// window takes before it stops
	body := `{"model":"group/auto","messages":[{"role":"user","content":"` + strings.Repeat("x", 4<<20) + `"}]}`
	s := New()
	t.Cleanup(s.client.CloseIdleConnections)
	// the client's patience, ten seconds, against the hundred milliseconds
	// the transport gives a write to go on with
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body)).WithContext(ctx)
	began := time.Now()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "from b") {
		t.Fatalf("a group request whose first member stopped taking its body: %d %s after %v; want b's reply",
			rec.Code, rec.Body.String(), time.Since(began))
	}
	select {
	case <-asked:
	default:
		t.Fatal("the first member was never asked: the group didn't move on from a stalled write")
	}
}
