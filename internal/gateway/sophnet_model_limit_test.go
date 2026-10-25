package gateway

import (
	"net/http"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// sophnetLimit is SophNet's model-scoped 429, as the reporter's gateway
// trace has it (#3619, zisen123): the console limits each model by TPM and
// RPM of its own, and the key's other models still answer.
const sophnetLimit = `{"error":{"message":"SophNet: Model tpm limit exceeded. Please try again later."}}`

// A 429 that says the limit is the model's ("Model tpm limit exceeded")
// rests that model alone, as a refusal pointing at the other models does:
// the key's sibling models are greyed out on the Routing page otherwise,
// answered with a local 429 that names another model's error. A rate
// limit that doesn't say it is the model's rests the key as before.
func TestModelRateLimitRestsTheModel(t *testing.T) {
	clineRestsCleared(t)
	s := &Server{}
	p := provider.Provider{ID: "sophnet", Chat: "https://sophnet.test/v1", Key: "sk-sophnet"}
	at := func(model string) candidate { return candidate{p: p, model: model, rest: p.ID} }
	c := at("DeepSeek-V4.1-Flash-1017")
	for _, body := range []string{
		sophnetLimit,
		`sophnet: Model tpm limit exceeded`,
		`{"error":{"message":"Model DeepSeek-R1 rpm limit exceeded"}}`,
		escaped("Model tpm limit exceeded. Please try again later.", "429"),
	} {
		if f := failure(429, []byte(body)); f != failRate {
			t.Fatalf("%s: %s, want %s", body, f, failRate)
		}
		r := s.restAfter(c, 429, http.Header{}, []byte(body))
		if r.Key != c.restID() || r.Model != c.model {
			t.Fatalf("%s: rests %s (model %q), not the model %s", body, r.Key, r.Model, c.restID())
		}
		for _, m := range []string{"DeepSeek-V4.1-Flash-1017-global", "DeepSeek-Flash-1017"} {
			if resting(at(m)) {
				t.Fatalf("%s: %s rests beside the model over its own TPM", body, m)
			}
		}
		if !resting(c) {
			t.Fatalf("%s: the model over its TPM isn't resting", body)
		}
	}
	// a rate limit that names no model is the key's, as before
	plain := `{"error":{"message":"SophNet: Too many requests"}}`
	if r := s.restAfter(c, 429, http.Header{}, []byte(plain)); r.Key != c.restKey() {
		t.Fatalf("%s: rests %s, want the key %s", plain, r.Key, c.restKey())
	}
}
