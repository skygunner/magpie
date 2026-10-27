package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/sessions"
	"github.com/yetone/magpie/internal/usage"
)

// #2705: Claude Desktop's third-party gateway mode runs Claude Code and
// writes its session where Claude Code's are (~/.claude/projects), with
// "entrypoint":"claude-desktop-3p", while the gateway records the same calls
// as Claude Desktop's. magpie read the session as Claude Code's, so Usage >
// Sessions listed it under both, counted its tokens twice and could not pair
// the two rows. It is Claude Desktop's session, once, with the folder, the
// duration and the conversation its own log has, and a Claude Code CLI
// session beside it stays Claude Code's.
func TestClaudeDesktop3pSessionsListedOnceAsDesktop(t *testing.T) {
	h := sandboxHome(t)
	claude := filepath.Join(h, ".claude")
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	proj := filepath.Join(claude, "projects", "-work-app")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	at := now.Add(-time.Hour).Format(time.RFC3339)
	// the reporter's own record of one response, both logs naming it the same
	const id = "6f1b9f1e-1f1e-4d3a-9d0c-3b0a8a0f9c11"
	desktop := fmt.Sprintf(`{"type":"user","message":{"role":"user","content":"hello from the desktop"},"uuid":"u1","timestamp":%q,"cwd":"/work/app","sessionId":%q,"entrypoint":"claude-desktop-3p"}`+"\n"+
		`{"type":"assistant","uuid":"a1","timestamp":%q,"cwd":"/work/app","sessionId":%q,"requestId":"req_1","entrypoint":"claude-desktop-3p","message":{"id":"msg_1","role":"assistant","model":"gpt-6.1-sol","content":[{"type":"text","text":"done"}],"usage":{"input_tokens":51901,"output_tokens":296,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`+"\n",
		at, id, at, id)
	if err := os.WriteFile(filepath.Join(proj, id+".jsonl"), []byte(desktop), 0o600); err != nil {
		t.Fatal(err)
	}
	// the subagent it started, as Claude Code keeps it
	sub := fmt.Sprintf(`{"type":"assistant","uuid":"s1","timestamp":%q,"cwd":"/work/app","sessionId":%q,"isSidechain":true,"entrypoint":"claude-desktop-3p","message":{"id":"msg_2","role":"assistant","model":"gpt-6.1-sol","content":[{"type":"text","text":"searched"}],"usage":{"input_tokens":100,"output_tokens":5}}}`+"\n", at, id)
	if err := os.MkdirAll(filepath.Join(proj, id, "subagents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, id, "subagents", "agent-a1.jsonl"), []byte(sub), 0o600); err != nil {
		t.Fatal(err)
	}
	// a Claude Code CLI session in the same folder: Claude Code's, as before
	cli := fmt.Sprintf(`{"type":"user","message":{"role":"user","content":"hello from the cli"},"uuid":"u2","timestamp":%q,"cwd":"/work/cli","sessionId":"cli-1","entrypoint":"cli"}`+"\n"+
		`{"type":"assistant","uuid":"a2","timestamp":%q,"cwd":"/work/cli","sessionId":"cli-1","requestId":"req_2","entrypoint":"cli","message":{"id":"msg_2","role":"assistant","model":"claude-sonnet-5","content":[{"type":"text","text":"done"}],"usage":{"input_tokens":10,"output_tokens":2}}}`+"\n",
		at, at)
	if err := os.WriteFile(filepath.Join(proj, "cli-1.jsonl"), []byte(cli), 0o600); err != nil {
		t.Fatal(err)
	}
	sessions.Reset()
	t.Cleanup(sessions.Reset)
	t.Cleanup(func() { statsMemo.Lock(); statsMemo.m = nil; statsMemo.Unlock() })
	statsMemo.Lock()
	statsMemo.m = nil
	statsMemo.Unlock()
	usage.Append(usage.Record{Time: now.Add(-time.Minute), Agent: "claude-desktop", Session: id, Provider: "deepseek", Model: "gpt-6.1-sol", Input: 51901, Output: 296, Status: 200})

	mux := http.NewServeMux()
	sessionRoutes(mux, folderOnly{})
	sessionManageRoutes(mux, folderOnly{})
	get := func(path string, v any) {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body)
		}
		if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}

	var list struct {
		Sessions []sessions.Session
	}
	get("/api/sessions", &list)
	byKey := map[string]sessions.Session{}
	for _, s := range list.Sessions {
		byKey[s.Agent+"|"+s.ID] = s
	}
	if len(list.Sessions) != 2 {
		t.Fatalf("want the desktop session and the CLI one, got %d: %+v", len(list.Sessions), list.Sessions)
	}
	if _, ok := byKey["claude|"+id]; ok {
		t.Errorf("the 3p session is also listed as Claude Code's: %+v", list.Sessions)
	}
	desk, ok := byKey["claude-desktop|"+id]
	if !ok {
		t.Fatalf("the 3p session is not Claude Desktop's: %+v", list.Sessions)
	}
	if desk.Path == "" || desk.Gateway || desk.Cwd != "/work/app" {
		t.Errorf("the 3p session lost its own file, folder or duration: %+v", desk)
	}
	if desk.Input != 51901+100 || desk.Output != 296+5 {
		t.Errorf("the 3p session's tokens are %d in, %d out, want 52001 in, 301 out", desk.Input, desk.Output)
	}
	if !desk.Transcript {
		t.Error("the 3p session's conversation can't be read")
	}
	if len(desk.Subagents) != 1 || desk.Subagents[0].ID != "a1" {
		t.Errorf("its subagents are %+v, want the one Claude Code kept", desk.Subagents)
	}
	if !strings.Contains(desk.Resume, "claude --resume "+id) {
		t.Errorf("resume %q, want Claude Code's", desk.Resume)
	}
	if c, ok := byKey["claude|cli-1"]; !ok || c.Input != 10 {
		t.Errorf("the Claude Code CLI session is %+v, want 10 tokens in", byKey["claude|cli-1"])
	}
	// its conversation is the one its own file has, and its row is found by
	// the key the page builds from the row
	var one struct{ sessions.Session }
	get("/api/sessions/one?key=claude-desktop:"+id, &one)
	if one.Path != desk.Path {
		t.Errorf("one?key=claude-desktop:%s is %+v, want the file the row names", id, one.Session)
	}
	var tr sessions.Transcript
	get("/api/sessions/transcript?agent=claude-desktop&id="+id, &tr)
	text := ""
	for _, p := range tr.Parts {
		text += p.Text
	}
	if !strings.Contains(text, "hello from the desktop") {
		t.Errorf("transcript %q, want what the session's file says", text)
	}

	// the page's summary: one session, its tokens once
	var st sessions.Stats
	get("/api/sessions/stats?days=0", &st)
	mine := []string{}
	for _, k := range st.Keys {
		if k == "claude-desktop:"+id || k == "claude:"+id {
			mine = append(mine, k)
		}
	}
	if len(mine) != 1 || mine[0] != "claude-desktop:"+id {
		t.Errorf("the range counts %v, want claude-desktop:%s once", mine, id)
	}
	dayIn := 0
	for _, d := range st.Days {
		for _, u := range d.Usage {
			if u.Model == "gpt-6.1-sol" {
				dayIn += u.Input
			}
		}
	}
	if dayIn != desk.Input {
		t.Errorf("the range counts %d input tokens, the session %d: its response is counted twice", dayIn, desk.Input)
	}

	// the page's trash takes it, as the Claude Code file it is
	old := now.Add(-time.Hour)
	for _, p := range []string{filepath.Join(proj, id+".jsonl"), filepath.Join(proj, id, "subagents", "agent-a1.jsonl"), filepath.Join(proj, id, "subagents"), filepath.Join(proj, id)} {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/sessions/delete", strings.NewReader(`{"agent":"claude-desktop","ids":["`+id+`"]}`)))
	var del struct {
		Deleted []string
		Refused []struct{ ID, Error string }
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &del) != nil || len(del.Deleted) != 1 || del.Deleted[0] != id || len(del.Refused) != 0 {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	if _, err := os.Stat(filepath.Join(proj, id+".jsonl")); !os.IsNotExist(err) {
		t.Errorf("the session's file is still there: %v", err)
	}
}
