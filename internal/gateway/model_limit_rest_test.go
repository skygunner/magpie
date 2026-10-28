package gateway

import (
	"net/http"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// A rate limit the vendor puts on the model alone rests that model and
// not its key: SophNet answers "Model tpm limit exceeded" for one model
// of a key (its console limits each model's tokens a minute), and the
// key's other models are asked as before (#3619).
func TestModelRateLimitRestsTheModelAlone(t *testing.T) {
	forgetRouting()
	c := candidate{p: provider.Provider{ID: "sophnet"}, model: "DeepSeek-V4.1-Flash-1017", rest: "sophnet"}
	s := &Server{}
	req := []byte(`{"error":{"message":"SophNet: Model tpm limit exceeded. Please try again later."}}`)
	r := s.restAfter(c, http.StatusTooManyRequests, http.Header{}, req)
	if r.Why != failRate {
		t.Fatalf("why = %s (%s)", r.Why, r.said)
	}
	if r.Key != c.restID() {
		t.Fatalf("Key=%q restKey=%s restID=%s why=%s said=%q modelRate=%v", r.Key, c.restKey(), c.restID(), r.Why, r.said, modelRateWords.Match(wordsOf(req)))
	}
	if r.Model != c.model {
		t.Fatalf("the rest names the model %q, want %q", r.Model, c.model)
	}
	if r, ok := restOf(c.restKey()); ok {
		t.Fatalf("the whole key rested: %+v", r)
	}
	if _, ok := restOf(c.restID()); !ok {
		t.Fatal("the model's own rest is not there")
	}
	// the wording the other way round is the model's own too
	forgetRouting()
	req = []byte(`{"error":{"message":"RPM limit exceeded for model DeepSeek-Flash-1017, please retry later"}}`)
	r = s.restAfter(candidate{p: provider.Provider{ID: "sophnet"}, model: "DeepSeek-Flash-1017", rest: "sophnet"},
		http.StatusTooManyRequests, http.Header{}, req)
	if r.Key != c.restKey()+"/DeepSeek-Flash-1017" {
		t.Fatalf("rested by %s, want the model's own", r.Key)
	}
	// a second model of the same key that 429s rests its own model too:
	// the key is not rested for one model's limit (#3619)
	forgetRouting()
	req = []byte(`{"error":{"message":"SophNet: Model tpm limit exceeded. Please try again later."}}`)
	c2 := candidate{p: provider.Provider{ID: "sophnet"}, model: "DeepSeek-Flash-1017", rest: "sophnet"}
	r = s.restAfter(c2, http.StatusTooManyRequests, http.Header{}, req)
	if r.Key != c2.restID() {
		t.Fatalf("second model rested by %s, want the model's own %s", r.Key, c2.restID())
	}
	if _, ok := restOf(c.restKey()); ok {
		t.Fatalf("the whole key rested for one model's limit: %+v", r)
	}
}

// A rate limit the vendor puts on the key or account rests the whole
// key, as before.
func TestKeyRateLimitRestsTheKey(t *testing.T) {
	forgetRouting()
	c := candidate{p: provider.Provider{ID: "relay"}, model: "gpt-6.1-sol", rest: "relay"}
	s := &Server{}
	r := s.restAfter(c, http.StatusTooManyRequests, http.Header{}, []byte(`{"error":{"message":"Too Many Requests"}}`))
	if r.Key != c.restKey() {
		t.Fatalf("rested by %s, want the key %s", r.Key, c.restKey())
	}
	if _, ok := restOf(c.restID()); ok {
		t.Fatal("the model rested on its own")
	}
}
