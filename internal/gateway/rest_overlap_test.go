package gateway

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// #3289: replies to requests already in flight can arrive in either order.
// A short rate limit must not replace an earlier credit, quota or restricted rest.
func TestOverlappingRestsKeepTheLaterDeadline(t *testing.T) {
	for _, preset := range []string{"", provider.RemoteMagpiePreset} {
		for _, status := range []int{http.StatusPaymentRequired, http.StatusTooManyRequests, http.StatusForbidden} {
			for _, rateFirst := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/status=%d/rate-first=%t", preset, status, rateFirst), func(t *testing.T) {
					s := &Server{}
					c := candidate{p: provider.Provider{ID: "overlap", Preset: preset, Key: "k"}, model: "m", rest: "overlap#k"}
					s.UnrestAll(c.restKey())
					t.Cleanup(func() { s.UnrestAll(c.restKey()) })
					rate := func() Rest {
						return s.restAfter(c, http.StatusTooManyRequests, http.Header{"Retry-After": {"1"}}, []byte(`{"error":{"type":"rate_limit_error","message":"Too many requests"}}`))
					}
					if rateFirst {
						rate()
					}
					body := `{"error":{"code":"insufficient_balance","message":"Insufficient balance"}}`
					if status == http.StatusTooManyRequests {
						body = `{"error":{"message":"You have reached your weekly usage limit"}}`
					} else if status == http.StatusForbidden {
						body = `{"error":{"message":"403 11140: ` + provider.WBRestrictedHint + `"}}`
					}
					long := s.restAfter(c, status, nil, []byte(body))
					if !rateFirst {
						if short := rate(); short.Why != failRate || !short.Until.Before(long.Until) {
							t.Fatalf("later reply lost its own failure: %+v", short)
						}
					}
					key := c.restKey()
					if preset != "" {
						key = c.restID()
						if _, ok := RestOf(c.restKey()); ok {
							t.Fatal("one model's refusal rested the whole key")
						}
					}
					if got, ok := RestOf(key); !ok || got != long {
						t.Fatalf("overlapping replies replaced the longer rest: got %+v, want %+v", got, long)
					}
					s.Unrest(key)
					short := rate()
					if got, ok := RestOf(key); !ok || got != short {
						t.Fatalf("explicit wake kept an old rest: got %+v, want %+v", got, short)
					}
				})
			}
		}
	}
}

func TestUnforbidKeepsAnEarlierCreditRest(t *testing.T) {
	s := &Server{}
	c := candidate{p: provider.Provider{ID: "overlap-403", Key: "k"}, model: "m", rest: "overlap-403#k"}
	t.Cleanup(func() { servedCandidate(c, 0) })
	credit := s.restAfter(c, http.StatusPaymentRequired, nil, []byte(`{"error":{"message":"Insufficient balance"}}`))
	refused := s.restAfter(c, http.StatusForbidden, nil, []byte(`{"error":{"message":"Forbidden"}}`))
	unforbid(forbidden{c: c, rest: refused})
	if got, ok := RestOf(c.restKey()); !ok || got != credit {
		t.Fatalf("taking back a later 403 lifted the credit rest: got %+v, want %+v", got, credit)
	}
}
