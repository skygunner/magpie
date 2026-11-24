package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// keyedSlow is slow to start on one key's one model while slow is set,
// and answers at once otherwise. It says who it was asked as, key/model.
type keyedSlow struct {
	key, model string
	slow       atomic.Bool
	arrived    chan struct{} // told of each slow request, when set
	mu         sync.Mutex
	asked      []string
}

func (k *keyedSlow) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	key, model := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), modelOf(body)
	k.mu.Lock()
	k.asked = append(k.asked, key+"/"+model)
	k.mu.Unlock()
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(200)
	if key == k.key && model == k.model && k.slow.Load() {
		io.WriteString(w, `data: {"id":"s","choices":[{"index":0,"delta":{"role":"assistant"}}]}`+"\n\n")
		w.(http.Flusher).Flush()
		if k.arrived != nil {
			k.arrived <- struct{}{}
		}
		<-r.Context().Done()
		return
	}
	io.WriteString(w, chatStreamOK)
}

func (k *keyedSlow) first() string {
	k.mu.Lock()
	defer k.mu.Unlock()
	if len(k.asked) == 0 {
		return ""
	}
	return k.asked[0]
}

func (k *keyedSlow) reset() {
	k.mu.Lock()
	k.asked = nil
	k.mu.Unlock()
}

// A model that keeps failing to start on one account rests there (小八 on
// Discord: a WorkBuddy account's kimi-k3-1 stalled 58 times, past the
// group's first-token time, and magpie asked it first again each time, 15
// times in 17 minutes). From the first slow start, that model on
// that account rests, and a success in between starts the count again.
// The same model on another key and another model on the same key are
// asked as before.
func TestSlowStartsRestTheModelOnItsAccount(t *testing.T) {
	fresh(t)
	up := &keyedSlow{key: "k1", model: "m"}
	up.slow.Store(true)
	serveOn(t, "a", "k1", []string{"m", "n"}, up, "k2")
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/m"}, Routing: provider.Ordered, FirstToken: 1}); err != nil {
		t.Fatal(err)
	}
	s := New()
	k1m, k1, k2m := "a#"+provider.KeyID("k1")+"/m", "a#"+provider.KeyID("k1"), "a#"+provider.KeyID("k2")+"/m"
	ask := func(model string) Try {
		t.Helper()
		up.reset()
		code, body := postAs(t, s, "", `{"model":"`+model+`","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
		if code != 200 || !strings.Contains(body, "from b") {
			t.Fatalf("%s: %d %s", model, code, body)
		}
		r := s.trace.routes[len(s.trace.routes)-1]
		return r.Tries[0]
	}

	tr := ask("group/g")
	if up.first() != "k1/m" || tr.Fail != failSlow || tr.Rest == nil || tr.Rest.Why != failSlow || tr.Rest.Model != "m" || tr.Rest.Key != k1m || tr.Rest.Failures != 1 {
		t.Fatalf("first slow start: %+v", tr)
	}
	if d := time.Until(tr.Rest.Until); d <= 0 || d > fallbackCooldown {
		t.Fatalf("rests %s, want up to %s", d, fallbackCooldown)
	}
	if _, ok := restOf(k1); ok {
		t.Fatal("the whole key rests")
	}
	if _, ok := restOf(k2m); ok {
		t.Fatal("the model rests on the other key")
	}
	if ask("group/g"); up.first() != "k2/m" {
		t.Fatalf("the resting model was asked first: %s", up.first())
	}
	if ask("a/n"); up.first() != "k1/n" {
		t.Fatalf("another model on the key: first %s", up.first())
	}

	// a second slow start (asked again once its rest is over) rests longer
	restingUntil.Lock()
	restingUntil.m[k1m] = time.Now().Add(-time.Second)
	restingUntil.Unlock()
	tr = ask("group/g")
	if tr.Rest == nil || tr.Rest.By != "backoff" || tr.Rest.Failures != 2 || time.Until(tr.Rest.Until) <= fallbackCooldown {
		t.Fatalf("again: %+v (rest %+v)", tr, tr.Rest)
	}
	// and an answer ends it
	restingUntil.Lock()
	restingUntil.m[k1m] = time.Now().Add(-time.Second)
	restingUntil.Unlock()
	up.slow.Store(false)
	ask("group/g")
	up.slow.Store(true)
	if tr := ask("group/g"); tr.Rest == nil || tr.Rest.Failures != 1 || tr.Rest.By != "cooldown" {
		t.Fatalf("after an answer: rest %+v", tr.Rest)
	}
}

func TestSlowStartBackoffToOneHour(t *testing.T) {
	fresh(t)
	synctest.Test(t, func(t *testing.T) {
		c := candidate{p: provider.Provider{ID: "a"}, model: "m", rest: "a"}
		for i, minutes := range []int{1, 2, 4, 8, 16, 32, 60, 60} {
			now := time.Now()
			r, ok := restSlow(c, now)
			if !ok || r.Failures != i+1 || r.Until.Sub(now) != time.Duration(minutes)*time.Minute {
				t.Fatalf("attempt %d: %+v", i+1, r)
			}
			time.Sleep(time.Second)
			probe, _ := restSlow(c, time.Now())
			if probe.Failures != r.Failures || !probe.Until.Equal(r.Until) {
				t.Fatalf("probe changed rest: %+v", probe)
			}
			time.Sleep(time.Until(r.Until) + 30*time.Second)
		}
		time.Sleep(slowForget)
		r, _ := restSlow(c, time.Now())
		if r.Failures != 1 || time.Until(r.Until) != time.Minute {
			t.Fatalf("after idle: %+v", r)
		}
	})
}

// A request its client gives up on while the model hasn't begun counts no
// slow start, however often.
func TestCanceledSlowStartRestsNothing(t *testing.T) {
	fresh(t)
	up := &keyedSlow{key: "k1", model: "m", arrived: make(chan struct{}, 1)}
	up.slow.Store(true)
	serveOn(t, "a", "k1", []string{"m"}, up, "k2")
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/m"}, Routing: provider.Ordered, FirstToken: 60}); err != nil {
		t.Fatal(err)
	}
	s := New()
	for i := range 3 {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"group/g","stream":true,"messages":[{"role":"user","content":"hi"}]}`)).WithContext(ctx)
			s.Handler().ServeHTTP(httptest.NewRecorder(), req)
		}()
		select {
		case <-up.arrived:
		case <-done: // answered by another: the slow one wasn't asked
			cancel()
			t.Fatalf("cancel %d: k1/m wasn't asked (rested?)", i)
		}
		cancel()
		<-done
		up.mu.Lock()
		asked := append([]string(nil), up.asked...)
		up.mu.Unlock()
		if len(asked) != i+1 {
			t.Fatalf("cancel %d: asked %v", i, asked)
		}
	}
	if _, ok := restOf("a#" + provider.KeyID("k1") + "/m"); ok {
		t.Fatal("a canceled request rests the model")
	}
	routed.Lock()
	defer routed.Unlock()
	if len(routed.slow) != 0 {
		t.Fatalf("counted: %v", routed.slow)
	}
}

// A group's serial walk has a total budget (#5113, mycf_711): nine members
// at 60s each took 660s, past the 600s a client waits, and the client's
// hang-up was told as the failure of the member it was on. The member on
// when the budget runs out gets what is left, the walk stops there, nobody
// else is asked, nobody rests, and the error names the budget.
func TestGroupWalkStopsAtItsBudget(t *testing.T) {
	fresh(t)
	was := groupBudget
	groupBudget = 100 * time.Millisecond
	t.Cleanup(func() { groupBudget = was })
	up := &keyedSlow{key: "k1", model: "m"}
	up.slow.Store(true)
	serveOn(t, "a", "k1", []string{"m", "n"}, up, "k2")
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/m", "a/n"}, Routing: provider.Ordered, FirstToken: 60}); err != nil {
		t.Fatal(err)
	}
	s := New()
	// slow twice over: a rest on the first would only show on the second
	for i := range 2 {
		up.reset()
		code, body := postAs(t, s, "", `{"model":"group/g","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
		r := s.trace.routes[len(s.trace.routes)-1]
		if code != 504 || !strings.Contains(body, "budget") {
			t.Fatalf("round %d: %d %s", i, code, body)
		}
		if len(r.Tries) != 1 || r.Tries[0].Fail != failSlow || r.Tries[0].Rest != nil || up.first() != "k1/m" {
			t.Fatalf("round %d: asked %v, tries %+v", i, up.asked, r.Tries)
		}
	}
	if _, ok := restOf("a#" + provider.KeyID("k1") + "/m"); ok {
		t.Fatal("the budget's end rests the member")
	}
}
