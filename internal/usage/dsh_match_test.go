package usage

import (
	"testing"
	"time"

	"github.com/yetone/magpie/internal/sessions"
)

// #3674: a dsh call that went through magpie is counted once. dsh sends the
// gateway no session header, but its session file names the vendor's reply
// by responseId, which matches the gateway's response_id — the pairing
// #678 gave Codex, now for dsh too. A dsh call magpie never saw keeps
// counting on its own.
func TestDshCallThroughMagpieCountedOnce(t *testing.T) {
	began := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	rec := Record{Time: began.Add(time.Second), Agent: "dsh", Provider: "workbuddy", Model: "deepseek-v4.1-flash",
		Input: 700, CacheRead: 5000, Output: 90, Status: 200, ResponseID: "resp-gw-dsh"}
	served := sessions.Call{Time: began, Agent: "dsh", Session: "dsh-gw", Model: "deepseek-v4.1-flash",
		ResponseID: "resp-gw-dsh", Tokens: sessions.Tokens{Input: 700, CacheRead: 5000, Output: 90}}
	direct := sessions.Call{Time: began.Add(10 * time.Minute), Agent: "dsh", Session: "dsh-out",
		Model: "deepseek-v4-flash", Tokens: sessions.Tokens{Input: 30, Output: 7}}

	m := gatewayMatches([]Record{rec}, []sessions.Call{served, direct})
	if !m[0] {
		t.Error("the call the gateway served is counted again from dsh's session file")
	}
	if m[1] {
		t.Error("a call dsh made without magpie was taken for the gateway's")
	}

	// the guards #678 set stay: a rejected or client-reported record is
	// not the pair's keeper, and a codex record is not paired with a dsh
	// call (codex identity fields would be carried onto the wrong agent)
	rejected := rec
	rejected.Rejected = true
	if gatewayMatches([]Record{rejected}, []sessions.Call{served})[0] {
		t.Error("a rejected gateway record consumed dsh's own call")
	}
	reported := rec
	reported.Source = SourceReported
	if gatewayMatches([]Record{reported}, []sessions.Call{served})[0] {
		t.Error("a client-reported record consumed dsh's own call")
	}
	codex := rec
	codex.Agent = "codex"
	if gatewayMatches([]Record{codex}, []sessions.Call{served})[0] {
		t.Error("a codex record was paired with a dsh call")
	}

	// the Requests page pairs rows by the same id: the served call's row
	// is matched and hidden, the direct one stays
	gateway, local := &rowChunk{}, &rowChunk{}
	gateway.add(Row{Record: rec}, "", 0, false)
	local.add(Row{Record: logRecord(served), Source: "log"}, "", 0, false)
	local.add(Row{Record: logRecord(direct), Source: "log"}, "", 1, false)
	pairs, _ := codexChunkResponsePairs([]*rowChunk{gateway}, []*rowChunk{local}, nil)
	if len(pairs) != 1 {
		t.Fatalf("the request page paired %d rows by response id, want 1", len(pairs))
	}
	for g, l := range pairs {
		if g.Index != 0 || l.Index != 0 {
			t.Fatalf("paired the wrong rows: gateway %d with local %d", g.Index, l.Index)
		}
	}
}
