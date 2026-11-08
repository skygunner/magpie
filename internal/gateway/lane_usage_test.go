package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/usage"
)

// A queue-full refusal has its own recent call and rejected ledger row even
// when another request has already timed out in the same queue (#2541).
func TestQueueRefusalIsInTheUsageLedger(t *testing.T) {
	fresh(t)
	_, up := slowUp(t)
	one := 1
	if err := provider.Save(provider.Provider{ID: "slow", Name: "Slow", Key: "k", Models: []string{"m"}, Chat: up + "/v1", Anthropic: up,
		MaxConcurrency: &one, QueueLimit: 1, QueueWait: 1}); err != nil {
		t.Fatal(err)
	}
	s := New()
	who := "slow#" + provider.KeyID("k")
	release, ok := s.lanes.tryTake(who, 1)
	if !ok {
		t.Fatal("could not hold the lane")
	}
	defer release()
	h := s.Handler()
	ask := func(tag string) *httptest.ResponseRecorder {
		body := `{"model":"slow/m","messages":[{"role":"user","content":"` + tag + `"}]}`
		r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
		r.Header.Set("X-Magpie-Session", tag)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	// b's real queue timeout has completed before c's refusal is counted.
	if w := ask("b"); w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), "waited") {
		t.Fatalf("b status = %d, want 429", w.Code)
	}
	ctx, cancel := context.WithCancel(context.Background())
	queued, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = s.lanes.take(ctx, who, 1, 1, 0, func() (bool, time.Time) {
			select {
			case <-queued:
			default:
				close(queued)
			}
			return false, time.Time{}
		})
	}()
	defer func() { cancel(); <-done }()
	select {
	case <-queued:
	case <-time.After(5 * time.Second):
		t.Fatal("queue waiter did not enter")
	}
	if w := ask("c"); w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), "queue is full") {
		t.Fatalf("c status = %d, want 429", w.Code)
	}
	var route int64
	for _, call := range s.Recent() {
		if call.Status == http.StatusTooManyRequests && strings.Contains(call.RequestBody, `"content":"c"`) {
			if route != 0 {
				t.Fatal("c has multiple recent-call entries")
			}
			route = call.Route
		}
	}
	if route == 0 {
		t.Fatal("c has no recent-call entry")
	}
	count := 0
	for _, rec := range usage.Load(time.Time{}) {
		if rec.Session == "c" && rec.Status == http.StatusTooManyRequests && rec.Rejected {
			count++
		}
	}
	if count != 1 {
		t.Errorf("c has %d rejected 429 rows in the usage ledger, want 1", count)
	}
}

// The contrast that makes this a defect rather than a design choice: a refusal
// turned away the same way, for a model magpie does not know, does reach the
// ledger, because it goes through turnedAway().
func TestUnknownModelRefusalIsInTheUsageLedger(t *testing.T) {
	fresh(t)
	_, up := slowUp(t)
	if err := provider.Save(provider.Provider{ID: "slow", Name: "Slow", Key: "k", Models: []string{"m"}, Chat: up + "/v1"}); err != nil {
		t.Fatal(err)
	}
	g := newLaneRig(t)

	g.send(context.Background(), "/v1/chat/completions", "nope/nothing", "a")
	r := g.reply()
	if r.status == 200 {
		t.Fatalf("a = %+v, want a refusal", r)
	}
	time.Sleep(500 * time.Millisecond)

	for _, rec := range usage.Load(time.Time{}) {
		if rec.Rejected {
			return // an unknown model is logged as the failure it was
		}
	}
	t.Fatalf("an unknown model's refusal is in no usage record either (status %d): the contrast this test needs is gone", r.status)
}
