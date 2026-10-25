package gateway

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// codexPostsTurn posts a Codex Responses request on group/r for the turns'
// words, naming the native turn and, for a subagent's own task, its header:
// what Codex's client_metadata carries.
func codexPostsTurn(t *testing.T, s *Server, session string, turns []string, native, sub string) Route {
	t.Helper()
	var in []string
	for i, w := range turns {
		if i > 0 {
			in = append(in, `{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]}`)
		}
		in = append(in, `{"type":"message","role":"user","content":[{"type":"input_text","text":`+quote(w)+`}]}`)
	}
	body := `{"model":"group/r","stream":false,"client_metadata":{"x-codex-turn-metadata":` +
		quote(`{"turn_id":"`+native+`"}`) + `},"input":[` + strings.Join(in, ",") + `]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body))
	req.Header.Set("x-session-id", session)
	if sub != "" {
		req.Header.Set("x-openai-subagent", sub)
	}
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	return lastRoute(s)
}

// Codex names the turn it is in, in client_metadata's x-codex-turn-metadata
// (turn_id). A compaction replaces a conversation's first user message with
// a summary and its parsed turn count with 1, and the group's rules
// classified the same native turn afresh (#3434, e2720515-cloud): another
// classifier call on the summary, and a route decided by it rather than by
// the task. The turn's decision is kept by the agent's own turn; a new
// native turn, and a subagent's task within one, are new tasks as before.
func TestCodexCompactionKeepsTheTurnsRule(t *testing.T) {
	s, _, _, c := intented(t, provider.Rule{Use: "b/big", Intent: "writing or fixing tests"})
	c.set("1", 0)
	// the turn begins: the classifier is asked once, and the rule sends it on
	r := codexPostsTurn(t, s, "s1", []string{"the parser test fails on a tab", "and the fixture is stale"}, "turn-a", "")
	if r.Rule == nil || r.Rule.N != 1 || r.Rule.Use != "b/big" || r.Rule.Classified == nil || c.n() != 1 {
		t.Fatalf("the turn's start: %+v, classifier asked %d times", r.Rule, c.n())
	}
	// the same native turn, compacted: a summary for the first words, one
	// parsed turn instead of two
	r = codexPostsTurn(t, s, "s1", []string{"Summary of the conversation so far: the parser test fails on a tab; the fixture is stale"}, "turn-a", "")
	if r.Rule == nil || r.Rule.N != 1 || r.Rule.Use != "b/big" || !r.Rule.Held || r.Rule.Classified != nil || c.n() != 1 {
		t.Fatalf("after compaction: %+v, classifier asked %d times", r.Rule, c.n())
	}
	// another native turn is another task, classified afresh
	r = codexPostsTurn(t, s, "s1", []string{"summary", "now rename the package"}, "turn-b", "")
	if r.Rule == nil || r.Rule.Classified == nil || c.n() != 2 {
		t.Fatalf("the next turn: %+v, classifier asked %d times", r.Rule, c.n())
	}
	// a subagent's task within the same native turn is its own, classified
	r = codexPostsTurn(t, s, "s1", []string{"look over the diff and say what breaks"}, "turn-b", "explore")
	if r.Rule == nil || r.Rule.Classified == nil || c.n() != 3 {
		t.Fatalf("the delegation: %+v, classifier asked %d times", r.Rule, c.n())
	}
}
