package gateway

import (
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// Cursor's plugin gives a native Run a lifetime of its own (15 minutes),
// and the deadline that cuts a turn mid-answer ends it as a failure "the
// gateway can retry or fail over" (plugins#38's contract): "native Run
// lifetime expired: the Run timed out; retry your request" (0.2.37,
// plugins#616). The lifetime is the Run's, not the account's — the next
// request starts a fresh Run on the same account and works — so as a cut
// (#2873) the account is as good as it was: the request is asked again,
// nobody rests, and no failure is counted (#4829).
const runLifeSaid = "native Run lifetime expired: the Run timed out; retry your request"

var runLifeStream = sse(
	`data: {"id":"c","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":""}}]}`,
	`data: {"error":{"message":"`+runLifeSaid+`","type":"api_error","code":null}}`,
	`data: [DONE]`)

// A plugin account whose Run outlived its lifetime before any of the
// answer was said is asked again — a fresh Run, which the plugin starts —
// and its second answer is the reply; the next member isn't needed, and
// nobody rests: the account did nothing wrong a minute of backoff would
// mend (#4829: the routing record showed fail "other" and a 60s backoff,
// although the next request on the same account works).
func TestPluginRunLifeExpiredIsRetriedNotRested(t *testing.T) {
	fresh(t)
	a := &scripted{replies: []reply{
		{200, "text/event-stream", runLifeStream},
		{200, "text/event-stream", chatStreamSaid("from a")},
	}}
	b := &scripted{replies: []reply{{200, "text/event-stream", chatStreamSaid("from b")}}}
	scriptedOn(t, "a", provider.Chat, a)
	scriptedOn(t, "b", provider.Chat, b)
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/m", "b/m"}, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	s := New()
	code, body := postAs(t, s, "", `{"model":"group/g","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	if code != 200 || !strings.Contains(body, "from a") || strings.Contains(body, "from b") || a.n != 2 || b.n != 0 {
		t.Fatalf("%d %s (a %d, b %d)", code, body, a.n, b.n)
	}
	r := s.trace.routes[len(s.trace.routes)-1]
	if len(r.Tries) != 2 || r.Tries[0].Fail != failCut || r.Tries[0].Rest != nil || r.Tries[1].Status != 200 {
		t.Fatalf("tries: %+v", r.Tries)
	}
	if s.resting("a") || s.resting("b") || len(routed.failures) != 0 {
		t.Fatalf("a Run's lifetime is nobody's fault: a resting %v, b %v, failures %v", s.resting("a"), s.resting("b"), routed.failures)
	}

	// the next request finds the account as good as it was: asked first
	// again, and answering
	code, body = postAs(t, s, "", `{"model":"group/g","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	if code != 200 || !strings.Contains(body, "from a") || b.n != 0 {
		t.Fatalf("next request: %d %s (a %d, b %d)", code, body, a.n, b.n)
	}
}

// The deadline may cut a turn after some of its answer was sent — the
// 200 is long gone, so nothing can be asked again for this request and
// the agent has the error as the plugin said it — but the account is no
// more at fault than before: it doesn't rest, its failure counts nowhere,
// and the agent's next request, a fresh Run, is answered by it again
// (#4829: the Run's 15-minute lifetime is per request).
func TestPluginRunLifeExpiredMidAnswerRestsNobody(t *testing.T) {
	fresh(t)
	a := &scripted{replies: []reply{
		{200, "text/event-stream", sse(
			`data: {"id":"c1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"Let me look"}}]}`,
			`data: {"error":{"message":"`+runLifeSaid+`","type":"api_error","code":null}}`)},
		{200, "text/event-stream", chatStreamSaid("from a")},
	}}
	b := &scripted{replies: []reply{{200, "text/event-stream", chatStreamSaid("from b")}}}
	scriptedOn(t, "a", provider.Chat, a)
	scriptedOn(t, "b", provider.Chat, b)
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"a/m", "b/m"}, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	s := New()
	code, body := postAs(t, s, "", `{"model":"group/g","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	if code != 200 || !strings.Contains(body, "Let me look") || !strings.Contains(body, runLifeSaid) || a.n != 1 || b.n != 0 {
		t.Fatalf("%d %s (a %d, b %d)", code, body, a.n, b.n)
	}
	r := s.trace.routes[len(s.trace.routes)-1]
	if len(r.Tries) != 1 || r.Tries[0].Fail != failCut {
		t.Fatalf("the try isn't told as a plugin's Run ending: %+v", r.Tries)
	}
	if s.resting("a") || s.resting("b") || len(routed.failures) != 0 {
		t.Fatalf("nobody rests for a Run's lifetime: a %v, b %v, failures %v", s.resting("a"), s.resting("b"), routed.failures)
	}

	// the next request: the account is as good as it was, so it answers
	code, body = postAs(t, s, "", `{"model":"group/g","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	if code != 200 || !strings.Contains(body, "from a") || a.n != 2 || b.n != 0 {
		t.Fatalf("next request: %d %s (a %d, b %d)", code, body, a.n, b.n)
	}
}

// The plugin's other endings keep their own lanes: a Run that went quiet,
// one the caller cancelled, a lifetime error from nowhere else — only the
// contract's own lifetime words are a cut.
func TestPluginRunLifeExpiredLeavesOtherErrorsAlone(t *testing.T) {
	for _, x := range []struct {
		status int
		body   string
		want   string
	}{
		{502, `{"error":{"message":"` + runLifeSaid + `","type":"api_error"}}`, failCut},
		// the words a plugin used before 0.2.37 said so plainly
		{502, `{"error":{"message":"native Run lifetime expired"}}`, failCut},
		// a Run that went quiet waiting for its checkpoint
		{502, `{"error":{"message":"the Run went quiet: no frame but heartbeats"}}`, failOther},
		// the caller cancelled the request
		{410, `{"error":{"message":"the Chat request was cancelled"}}`, failOther},
		// a vendor's own timeout words are its own, not the plugin's
		{502, `{"error":{"message":"upstream request timed out"}}`, failOther},
	} {
		if got := failure(x.status, []byte(x.body)); got != x.want {
			t.Errorf("failure(%d, %s) = %s, want %s", x.status, x.body, got, x.want)
		}
	}
}
