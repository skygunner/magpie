package gateway

import (
	"net/http"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// A 5xx that answers it queued the request for the model it names is that
// model's own congestion (Qoder's 503 "Qoder queued the request for
// <model> (this plan's queue: p3); try again in 30s", #3213,
// liubai0106): the model rests alone and the account's other models are
// still asked, instead of the account resting and the group failing over
// between accounts on the one model that is out. A 5xx that says nothing
// of a queue rests the account as before.
func TestQueuedModelRestsAlone(t *testing.T) {
	old := allowances
	defer func() { allowances = old }()
	allowances = func(string) map[string]provider.Allowance { return nil }
	s := &Server{}
	acct := func(model string) candidate {
		return candidate{p: provider.Provider{ID: "qoder", Account: &provider.Account{Agent: "qoder", User: "u@x.com"}}, model: model, rest: "qoder#u"}
	}
	body := []byte("Qoder: Qoder queued the request for nemotron-3-ultra (this plan's queue: p3); try again in 30s")
	if f := failure(503, body); f != failOther {
		t.Fatalf("read as %s, want other", f)
	}
	q := acct("nemotron-3-ultra")
	clearRest(q.restKey())
	r := s.restAfter(q, 503, http.Header{}, body)
	if r.Key != q.restID() {
		t.Fatalf("rests %s, want the model %s", r.Key, q.restID())
	}
	if _, ok := restingOf(acct("glm-5.3")); ok {
		t.Fatal("the queued model's rest sat the account's other models")
	}
	clearRest(q.restKey())

	// a 5xx that says nothing of a queue rests the account, as before
	other := acct("glm-5.3")
	r = s.restAfter(other, 503, http.Header{}, []byte("Qoder: upstream is overloaded, try again"))
	if r.Key != other.restKey() {
		t.Fatalf("a plain 503 rested %s, want the account %s", r.Key, other.restKey())
	}
	clearRest(other.restKey())
}
