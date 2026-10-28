package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// A conversation kept on a member that answers its next request with an
// upstream error isn't kept there: the stick goes, as it does for a reply
// that broke off (#733), so the next request of that conversation is
// routed again rather than back to the one that just failed it (#3703: a
// Grok account out of its weekly balance, 402 on every try, sixteen times
// over, until the stick was cleared from affinity.json by hand). A key's
// 401 stands for it: the last one left is rested for neither, so only the
// stick held the conversation there.
func TestUpstreamErrorNotKept(t *testing.T) {
	fresh(t)
	var mu sync.Mutex
	var tried []string
	down := false // the kept member refuses the request as its key is gone
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		who := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		mu.Lock()
		tried = append(tried, who)
		refuse := down && who == "broken"
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if refuse {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, `{"error":{"message":"invalid api key"}}`)
			return
		}
		io.WriteString(w, `{"id":"c1","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"from `+who+`"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`)
	}))
	defer up.Close()
	for _, p := range []provider.Provider{
		{ID: "flaky", Name: "Flaky", Key: "broken", Chat: up.URL + "/v1", Models: []string{"m"}},
		{ID: "steady", Name: "Steady", Key: "fine", Chat: up.URL + "/v1", Models: []string{"m"}},
	} {
		if err := provider.Save(p); err != nil {
			t.Fatal(err)
		}
	}
	group := func(members ...string) {
		t.Helper()
		if err := provider.SaveGroup(provider.Group{Name: "G", Members: members, Routing: provider.Ordered}); err != nil {
			t.Fatal(err)
		}
	}
	// the group is left with flaky alone, as the reporter's was once its
	// others were out
	group("flaky/m")
	s := New()
	post := func() string {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"group/g","messages":[{"role":"user","content":"hi"}]}`))
		req.Header.Set("x-session-id", "s1")
		s.Handler().ServeHTTP(rec, req)
		return rec.Body.String()
	}
	if out := post(); !strings.Contains(out, "from broken") {
		t.Fatalf("the first reply isn't flaky's: %s", out)
	}
	mu.Lock()
	down = true
	mu.Unlock()
	if out := post(); !strings.Contains(out, "invalid api key") {
		t.Fatalf("the failure isn't the member's as it came: %s", out)
	}
	sticks.Lock()
	kept := len(sticks.m)
	sticks.Unlock()
	if kept != 0 {
		t.Fatalf("the conversation is still kept on the member that just failed it")
	}
	// steady is back, and the group is asked in its own order again: not
	// the one that failed the conversation
	group("steady/m", "flaky/m")
	if out := post(); !strings.Contains(out, "from fine") {
		t.Fatalf("the next request: %s", out)
	}
	mu.Lock()
	got := strings.Join(tried, ",")
	mu.Unlock()
	if got != "broken,broken,fine" {
		t.Fatalf("tried %s, want broken,broken,fine (the failed member asked first)", got)
	}
	if a := s.trace.routes[len(s.trace.routes)-1].Affinity; a == nil || a.Kept {
		t.Fatalf("the next request was kept on the member that failed the last one: %+v", a)
	}
}
