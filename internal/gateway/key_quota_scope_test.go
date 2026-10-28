package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// reportedQuotaBody is the refusal the reporter's own gateway passed on
// from the ChatGPT account behind one of its models (#2744): Codex's own
// usage_limit_reached, with when the account is back.
var reportedQuotaBody = []byte(`{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached","plan_type":"plus","resets_in_seconds":7200}}`)

// A key out of a model's allowance rests that model alone: behind one key
// a relay may serve each of its models from an account of its own, so one
// model's used-up allowance says nothing of the others (#2744, Goalonez:
// sol is served from his plus account and luna from another, and sol out
// of its allowance held luna until 21:50 with it).
func TestKeyQuotaRestsItsModelOnly(t *testing.T) {
	fresh(t)
	old := allowances
	defer func() { allowances = old }()
	allowances = func(string) map[string]provider.Allowance { return nil }
	s := &Server{}
	p := provider.Provider{ID: "gw", Name: "Goalonez", Chat: "https://gw.test/v1", Key: "k"}
	sol := candidate{p: p, model: "gpt-5.6-sol", rest: "gw"}
	luna := candidate{p: p, model: "gpt-5.6-luna", rest: "gw"}
	r := s.restAfter(sol, 429, http.Header{}, reportedQuotaBody)
	if d := time.Until(r.Until); r.Why != failQuota || r.By != "resets" || d <= 2*time.Hour-time.Minute || d > 2*time.Hour {
		t.Fatalf("sol out of its allowance rests %v by %s (%s)", time.Until(r.Until), r.By, r.Why)
	}
	if r.Key != sol.restID() || r.Model != sol.model {
		t.Fatalf("sol out of its allowance rests %s (model %q), not the model %s", r.Key, r.Model, sol.restID())
	}
	if !resting(sol) {
		t.Fatal("sol out of its allowance isn't resting")
	}
	if resting(luna) {
		t.Fatal("luna rests beside sol out of its allowance")
	}

	// the same refusal as a relay may pass it on, its message alone
	if r := s.restAfter(sol, 429, http.Header{}, []byte(`{"error":{"message":"The usage limit has been reached","resets_in_seconds":7200}}`)); r.Key != sol.restID() {
		t.Fatalf("its words alone rest %s, not the model %s", r.Key, sol.restID())
	}

	// a key whose own limit is what the vendor named rests whole, as
	// before: sub2api's "api key 7天限额已用完" is no account behind it
	k := candidate{p: provider.Provider{ID: "sub", Chat: "https://sub.test/v1", Key: "sk"}, model: "gpt-5.6-sol", rest: "sub"}
	if r := s.restAfter(k, 429, http.Header{}, []byte(`{"error":{"message":"api key 7天限额已用完","type":"rate_limit_exceeded"}}`)); r.Key != k.restKey() {
		t.Fatalf("a key's own limit rests %s, not the key %s", r.Key, k.restKey())
	}
}

// Through the gateway, as the reporter's agent asked it: once sol is out
// of its allowance, a request for luna still reaches the gateway and is
// answered, and sol is answered with its refusal without asking again
// (#2744).
func TestKeyQuotaLeavesItsOtherModels(t *testing.T) {
	fresh(t)
	old := allowances
	defer func() { allowances = old }()
	allowances = func(string) map[string]provider.Allowance { return nil }
	p := provider.Provider{ID: "gw", Name: "Goalonez", Chat: "https://gw.test/v1", Key: "k", Models: []string{"gpt-5.6-sol", "gpt-5.6-luna"}}
	const completion = `{"id":"chatcmpl_1","object":"chat.completion","model":"gpt-5.6-luna","choices":[{"index":0,"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":1,"total_tokens":6}}`
	var mu sync.Mutex
	var asked []string
	s := New()
	s.client = &http.Client{Transport: countTransport(func(r *http.Request) (*http.Response, error) {
		var q struct {
			Model string `json:"model"`
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &q)
		mu.Lock()
		asked = append(asked, q.Model)
		mu.Unlock()
		h := http.Header{"Content-Type": {"application/json; charset=utf-8"}}
		if q.Model == "gpt-5.6-sol" {
			return &http.Response{StatusCode: http.StatusTooManyRequests, Header: h,
				Body: io.NopCloser(strings.NewReader(string(reportedQuotaBody)))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Header: h,
			Body: io.NopCloser(strings.NewReader(completion))}, nil
	})}
	if err := provider.Save(p); err != nil {
		t.Fatal(err)
	}
	ask := func(model string) (int, string) {
		t.Helper()
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/chat/completions",
			strings.NewReader(`{"model":"gw/`+model+`","stream":false,"messages":[{"role":"user","content":"Say pong"}]}`)))
		return rec.Code, rec.Body.String()
	}
	if code, body := ask("gpt-5.6-sol"); code != http.StatusTooManyRequests || !strings.Contains(body, "usage limit") {
		t.Fatalf("sol out of its allowance: %d %s", code, body)
	}
	if code, body := ask("gpt-5.6-luna"); code != http.StatusOK || !strings.Contains(body, "pong") {
		t.Fatalf("luna, served from another account: %d %s", code, body)
	}
	if code, body := ask("gpt-5.6-sol"); code != http.StatusTooManyRequests || !strings.Contains(body, "usage limit") {
		t.Fatalf("sol out of its allowance, again: %d %s", code, body)
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(asked, ",") != "gpt-5.6-sol,gpt-5.6-luna" {
		t.Fatalf("the gateway was asked for %v, want sol once, then luna", asked)
	}
}
