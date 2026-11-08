package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/sessions"
	"github.com/yetone/magpie/internal/usage"
)

// The management API must omit a Desktop gateway row already kept by Claude (#4333).
func TestManageDesktopCodeTabNativeWins(t *testing.T) {
	h := sandboxHome(t)
	sessions.Reset()
	t.Cleanup(sessions.Reset)
	dir := filepath.Join(h, ".claude", "projects", "-work-app")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	id := "desktop4333"
	line := `{"type":"user","timestamp":"2026-09-19T08:00:00.000Z","message":{"role":"user","content":"hello"},"cwd":"/work/app","sessionId":"desktop4333","entrypoint":"claude-desktop-3p"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, id+".jsonl"), []byte(line), 0600); err != nil {
		t.Fatal(err)
	}
	for _, session := range []string{id, "remote4333"} {
		usage.Append(usage.Record{Time: time.Now().Add(-2 * time.Hour), Agent: "claude-desktop", Session: session, Provider: "fake", Model: "m", Status: 200})
	}
	mux := http.NewServeMux()
	sessionManageRoutes(mux, folderOnly{})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/sessions/manage?agent=claude-desktop", nil))
	var got struct {
		Agents []struct {
			Agent string `json:"agent"`
			Count int    `json:"count"`
		} `json:"agents"`
		Sessions []struct {
			ID string `json:"id"`
		} `json:"sessions"`
	}
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Sessions) != 1 || got.Sessions[0].ID != "remote4333" {
		t.Fatalf("sessions: %+v", got.Sessions)
	}
	for _, a := range got.Agents {
		if a.Agent == "claude-desktop" && a.Count == 1 {
			return
		}
	}
	t.Fatalf("counts: %+v", got.Agents)
}
