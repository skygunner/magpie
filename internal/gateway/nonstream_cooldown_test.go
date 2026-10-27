package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// The client allows five times the gateway's budget. Before the fix it
// cancels with 499 instead of receiving the second key's JSON response.
func TestNonstreamFirstResponseFallsBack(t *testing.T) {
	for _, mode := range []string{"silent", "headers_only"} {
		t.Run(mode, func(t *testing.T) {
			fresh(t)
			quietFast(t) // The one-second deadline must cross the keepalive threshold.
			up := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				if r.Header.Get("Authorization") == "Bearer k1" {
					if mode == "headers_only" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(200)
						w.(http.Flusher).Flush() // headers alone must not count as content
					}
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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"group/g","stream":false,"messages":[{"role":"user","content":"hi"}]}`)).WithContext(ctx)
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, req)
			if w.Code != 200 || w.Result().Header.Get("Content-Type") != "application/json" || !json.Valid(w.Body.Bytes()) || !strings.Contains(w.Body.String(), "fallback answered") {
				t.Fatalf("want fallback JSON, got %d %s %s", w.Code, w.Result().Header, w.Body.String())
			}
			route := s.trace.routes[len(s.trace.routes)-1]
			if len(route.Tries) != 2 || route.Tries[0].Fail != failSlow {
				t.Fatalf("tries: %+v", route.Tries)
			}
		})
	}
}

// A group watches non-streaming first content too, but its 15s keepalive
// must not commit 200/SSE before a delayed JSON answer or error arrives.
func TestNonstreamWaitKeepsStatusAndJSON(t *testing.T) {
	for _, proto := range []provider.Protocol{provider.Chat, provider.Responses, provider.Anthropic} {
		for _, status := range []int{200, 503} {
			t.Run(string(proto)+"/"+http.StatusText(status), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					w := httptest.NewRecorder()
					h := newHoldWriter(w, true)
					h.ctx, h.stop = ctx, cancel
					h.alive = &keptAlive{proto: proto}
					h.firstWait = time.Minute
					stop := h.watch()
					defer stop()
					synctest.Wait()
					time.Sleep(20 * time.Second)
					synctest.Wait()
					if w.Flushed || w.Body.Len() != 0 {
						t.Errorf("non-streaming wait sent an early reply: %s %q", w.Result().Header, w.Body.String())
					}
					body := map[provider.Protocol]string{
						provider.Chat:      `{"id":"ok","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`,
						provider.Responses: `{"id":"ok","object":"response","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}`,
						provider.Anthropic: `{"id":"ok","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn"}`,
					}[proto]
					if status != 200 {
						body = `{"error":{"message":"temporarily unavailable"}}`
					}
					h.Header().Set("Content-Type", "application/json")
					h.WriteHeader(status)
					io.WriteString(h, body)
					h.release()
					if w.Code != status || w.Result().Header.Get("Content-Type") != "application/json" || w.Body.String() != body {
						t.Errorf("want %d JSON %s; got %d %s %s", status, body, w.Code, w.Result().Header, w.Body.String())
					}
				})
			})
		}
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
