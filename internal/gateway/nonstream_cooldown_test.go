package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// The client allows three times the gateway's budget. Before the fix it
// cancels with 499 instead of receiving the second key's JSON response.
func TestNonstreamFirstResponseFallsBack(t *testing.T) {
	fresh(t)
	up := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		if r.Header.Get("Authorization") == "Bearer k1" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.(http.Flusher).Flush() // headers alone must not count as content
			<-r.Context().Done()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"ok","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"fallback answered"},"finish_reason":"stop"}]}`)
	})
	serveOn(t, "a", "k1", []string{"m"}, up, "k2")
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/m"}, Routing: provider.Ordered, FirstToken: 1}); err != nil {
		t.Fatal(err)
	}
	s := New()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"group/g","stream":false,"messages":[{"role":"user","content":"hi"}]}`)).WithContext(ctx)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "fallback answered") {
		t.Fatalf("want fallback JSON, got %d %s", w.Code, w.Body.String())
	}
	route := s.trace.routes[len(s.trace.routes)-1]
	if len(route.Tries) != 2 || route.Tries[0].Fail != failSlow {
		t.Fatalf("tries: %+v", route.Tries)
	}
}

func TestOptedInCancellationRestsOnlyModel(t *testing.T) {
	fresh(t)
	up := &keyedSlow{key: "k1", model: "m", arrived: make(chan struct{}, 1)}
	up.slow.Store(true)
	serveOn(t, "a", "k1", []string{"m", "n"}, up, "k2")
	// Save and reload the opt-in through the same persisted Provider schema.
	p, err := provider.Find("a")
	if err != nil {
		t.Fatal(err)
	}
	p.ModelCooldown = map[string]int{"m": 60}
	if err := provider.Save(*p); err != nil {
		t.Fatal(err)
	}
	s := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"a/m","stream":true,"messages":[{"role":"user","content":"hi"}]}`)).WithContext(ctx)
		s.Handler().ServeHTTP(httptest.NewRecorder(), r)
	}()
	select {
	case <-up.arrived:
	case <-done:
		t.Fatal("did not reach upstream")
	}
	cancel()
	<-done
	id := "a#" + provider.KeyID("k1") + "/m"
	r, ok := restOf(id)
	if !ok || r.Status != 499 || r.Failures != 1 {
		t.Fatalf("cancel rest: %+v %v", r, ok)
	}
	for _, other := range []string{"a#" + provider.KeyID("k1"), "a#" + provider.KeyID("k1") + "/n", "a#" + provider.KeyID("k2") + "/m"} {
		if _, ok := restOf(other); ok {
			t.Fatalf("rest leaked to %s", other)
		}
	}
}

func TestModelCooldownReachesAndKeepsSixtyMinutes(t *testing.T) {
	fresh(t)
	c := candidate{rest: "a", p: provider.Provider{ID: "a", Key: "k", ModelCooldown: map[string]int{"m": 60}}, model: "m"}
	now := time.Now()
	id := c.restID()
	for i, minutes := range []int{1, 2, 4, 8, 16, 32, 60, 60} {
		restingUntil.Lock()
		restingUntil.m[id] = time.Now().Add(-time.Second)
		restingUntil.Unlock()
		r, ok := restUnresponsive(c, now, failCanceled, 499, 1)
		if !ok || r.Until.Sub(now) != time.Duration(minutes)*time.Minute || r.Failures != i+1 {
			t.Fatalf("step %d: %+v %v", i, r, ok)
		}
		// The next attempt can consume the client's two-minute timeout too.
		now = r.Until.Add(2 * time.Minute)
	}
	servedCandidate(c, 1)
	r, ok := restUnresponsive(c, now, failCanceled, 499, 1)
	if !ok || r.Failures != 1 || r.Until.Sub(now) != time.Minute {
		t.Fatalf("success did not reset: %+v", r)
	}
}

func TestModelCooldownDoesNotProbeLastCandidate(t *testing.T) {
	fresh(t)
	var calls atomic.Int32
	serveOn(t, "a", "k", []string{"m"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	p, err := provider.Find("a")
	if err != nil {
		t.Fatal(err)
	}
	p.ModelCooldown = map[string]int{"m": 60}
	if err := provider.Save(*p); err != nil {
		t.Fatal(err)
	}
	c := candidate{rest: "a", p: *p, model: "m"}
	rest, ok := restUnresponsive(c, time.Now(), failCanceled, 499, 1)
	if !ok {
		t.Fatal("no rest")
	}
	s := New()
	for range 3 {
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"a/m","stream":false,"messages":[{"role":"user","content":"hi"}]}`))
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, req)
		if w.Code != 503 || w.Header().Get("Retry-After") == "" {
			t.Fatalf("got %d %s %s", w.Code, w.Header(), w.Body)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("cooling model was probed %d times", calls.Load())
	}
	after, _ := restOf(c.restID())
	if after.Until != rest.Until || after.Failures != 1 {
		t.Fatalf("held requests extended rest: %+v", after)
	}
}
