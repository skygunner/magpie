package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/sessions"
	"github.com/yetone/magpie/internal/usage"
)

// #3470 follow-up: the totals tile's rate reads cache_unknown out of its
// denominator, and the gateway-only sessions' days carried it - but a
// native session's day rows (Stats.Days) counted the prompt of the calls
// whose vendor counted no cache as a miss, so a Kiro-heavy native session
// still diluted the tile. The day rows join the amount back from the kept
// projection, per day, and the tile sums it.
func TestSessionStatsDaysCarryCacheUnknown(t *testing.T) {
	h := sandboxHome(t)
	claude := filepath.Join(h, ".claude")
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	proj := filepath.Join(claude, "projects", "-work")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	line := func(id, ts string) string {
		return fmt.Sprintf(`{"type":"user","message":{"role":"user","content":"hello"},"uuid":"u-%s","timestamp":%q,"cwd":"/work","sessionId":"native"}`+"\n"+
			`{"type":"assistant","message":{"model":"claude-sonnet-5","id":"%s","role":"assistant","content":[{"type":"text","text":"done"}],"usage":{"input_tokens":10,"output_tokens":2}},"uuid":"a-%s","timestamp":%q,"cwd":"/work","sessionId":"native"}`+"\n", id, ts, id, id, ts)
	}
	if err := os.WriteFile(filepath.Join(proj, "native.jsonl"),
		[]byte(line("d1", yesterday.Add(-time.Hour).Format(time.RFC3339))+line("d2", now.Add(-time.Hour).Format(time.RFC3339))), 0o600); err != nil {
		t.Fatal(err)
	}
	sessions.Reset()
	t.Cleanup(sessions.Reset)
	t.Cleanup(func() { statsMemo.Lock(); statsMemo.m = nil; statsMemo.Unlock() })
	statsMemo.Lock()
	statsMemo.m = nil
	statsMemo.Unlock()
	// what the gateway recorded of the native session's calls: a Kiro call
	// that counted no cache, and a counted one, on two days
	usage.Append(usage.Record{Time: yesterday.Add(-30 * time.Minute), Agent: "claude", Session: "native", Requested: "kiro/claude-sonnet-5", Provider: "kiro", Model: "claude-sonnet-5", Input: 400, Status: 200, CacheUnknown: true})
	usage.Append(usage.Record{Time: now.Add(-30 * time.Minute), Agent: "claude", Session: "native", Requested: "kiro/claude-sonnet-5", Provider: "kiro", Model: "claude-sonnet-5", Input: 500, Status: 200, CacheUnknown: true})
	// a session only the gateway saw: its unknown must count once, not twice
	usage.Append(usage.Record{Time: now.Add(-20 * time.Minute), Agent: "claude", Session: "gwonly", Requested: "kiro/claude-sonnet-5", Provider: "kiro", Model: "claude-sonnet-5", Input: 700, Status: 200, CacheUnknown: true})

	mux := http.NewServeMux()
	sessionRoutes(mux, folderOnly{})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/sessions/stats?days=7", nil))
	if w.Code != 200 {
		t.Fatalf("stats: %d %s", w.Code, w.Body)
	}
	var st struct {
		Days []struct {
			Date  string `json:"date"`
			Usage []struct {
				Agent        string `json:"agent"`
				CacheUnknown int    `json:"cache_unknown"`
			} `json:"usage"`
		} `json:"days"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	sum := 0
	for _, d := range st.Days {
		for _, u := range d.Usage {
			if u.Agent == "claude" {
				sum += u.CacheUnknown
			}
		}
	}
	if sum != 1600 {
		t.Fatalf("the days carry %d of cache_unknown, want the three calls' 1,600 prompt once each", sum)
	}
}
