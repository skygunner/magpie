package gateway

import (
	"context"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/usage"
)

// relayAsks is a Chat request relayed to a provider that speaks Chat itself
// (a plugin's OpenAI-compatible fetch included, #4855).
func relayAsks(t *testing.T, s *Server, reply string) *httptest.ResponseRecorder {
	t.Helper()
	fresh(t)
	setup(t, provider.Chat, &fake{t: t, reply: reply})
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest("POST", "/v1/chat/completions",
		strings.NewReader(`{"model":"fake/m1","stream":true,"messages":[{"role":"user","content":"hi"}]}`)))
	return w
}

// A relayed stream's error frame names its own status (#4855): a plugin
// that already sent 200 can only say what went wrong in the frame, and its
// code and status are the routing signal. The frame is relayed as it came
// and the turn fails as it would at the status it named, not as a 200
// whose words name nothing.
func TestRelayStreamErrorFrameSaysItsStatus(t *testing.T) {
	const frame = `{"error":{"message":"the upstream timed out","type":"upstream_error","code":"upstream_timeout","status":504}}`
	for _, c := range []struct {
		name               string
		reply              string
		wantStatus         int    // the try's failure, and the rest's
		wantFail, wantText string // what the rest says of it
	}{
		{"mid-reply", sse(
			`data: {"id":"c1","choices":[{"index":0,"delta":{"content":"partial text"}}]}`,
			`data: `+frame), 504, failOther, "Gateway Timeout"},
		{"terse 429", sse(
			`data: {"id":"c1","choices":[{"index":0,"delta":{"content":"partial text"}}]}`,
			`data: {"error":{"message":"busy","type":"upstream_error","code":"rate_limit","status":429}}`), 429, failRate, "Too Many Requests"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := New()
			w := relayAsks(t, s, c.reply)
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"partial text"`) || !strings.Contains(w.Body.String(), `"status":`+strconv.Itoa(c.wantStatus)) {
				t.Fatalf("the frame was not relayed: %d %s", w.Code, w.Body)
			}
			state := s.Trace(context.Background(), 0, 0)
			if len(state.Routes) != 1 {
				t.Fatalf("routes %d", len(state.Routes))
			}
			var try *Try
			for i := range state.Routes[0].Tries {
				if state.Routes[0].Tries[i].Rest != nil {
					try = &state.Routes[0].Tries[i]
				}
			}
			if try == nil {
				t.Fatalf("the turn rested nobody: %+v", state.Routes[0].Tries)
			}
			if try.Fail != c.wantFail || try.Rest.Status != c.wantStatus {
				t.Fatalf("failed as %s at %d, want %s at %d", try.Fail, try.Rest.Status, c.wantFail, c.wantStatus)
			}
			if !strings.Contains(try.Rest.Error, c.wantText) {
				t.Fatalf("the rest says %q, want it to name %q", try.Rest.Error, c.wantText)
			}
			restingUntil.Lock()
			var rest *Rest
			for _, r := range restingUntil.note {
				rest = &r
			}
			restingUntil.Unlock()
			if rest == nil || rest.Status != c.wantStatus || rest.Why != c.wantFail {
				t.Fatalf("rested %+v, want %s at %d", rest, c.wantFail, c.wantStatus)
			}
		})
	}

	// the frame as the stream's first event, nothing said before it: the
	// frame is relayed as it came, and the try fails at the status it
	// named, not the 502 its words guess
	t.Run("first event", func(t *testing.T) {
		s := New()
		w := relayAsks(t, s, sse(`data: `+frame))
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":504`) {
			t.Fatalf("stream: %d %s", w.Code, w.Body)
		}
		state := s.Trace(context.Background(), 0, 0)
		if len(state.Routes) != 1 || state.Routes[0].Status != 504 {
			t.Fatalf("route: %+v", state.Routes)
		}
		for _, u := range usage.Load(time.Time{}) {
			if u.Status != 504 || !u.Failed() {
				t.Fatalf("usage: %+v", u)
			}
		}
	})
}

// An error frame that names no status reads as before: the words guess,
// and a reply whose words name nothing is a 502.
func TestRelayStreamErrorFrameWithoutStatusAsBefore(t *testing.T) {
	s := New()
	w := relayAsks(t, s, sse(
		`data: {"id":"c1","choices":[{"index":0,"delta":{"content":"partial text"}}]}`,
		`data: {"error":{"type":"server_error","message":"synthetic stream failure"}}`))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "synthetic stream failure") {
		t.Fatalf("stream: %d %s", w.Code, w.Body)
	}
	state := s.Trace(context.Background(), 0, 0)
	var try *Try
	for i := range state.Routes[0].Tries {
		if state.Routes[0].Tries[i].Rest != nil {
			try = &state.Routes[0].Tries[i]
		}
	}
	if try == nil || try.Fail != failOther || try.Rest.Status != 200 {
		t.Fatalf("a frame that named no status failed as %+v, want other at 200 as before", try)
	}
}

// magpie's own Chat failure frame carries what a plugin's does (#4855):
// a code and a status beside its message, so a client reads one shape.
func TestStreamFailureChatCarriesCodeAndStatus(t *testing.T) {
	if got := string(streamFailure(provider.Chat, "Fake: connection lost mid-reply (unexpected EOF)")); !strings.Contains(got, `"code":"server_error"`) || !strings.Contains(got, `"status":502`) {
		t.Fatalf("streamFailure(chat) = %s", got)
	}
}
