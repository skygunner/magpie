package sessions

import (
	"path/filepath"
	"testing"
	"time"
)

// #2301 (LukeZ1988): MiniMax Code's sessions are on the Sessions page, under
// the mvs_ id Usage's calls of them carry, with the tokens and models its own
// messages.jsonl records. Its folder holds neither the folder the session ran
// in nor a resume command, so its row is listed and nothing more.
func TestMiniMaxCodeSessionsListed(t *testing.T) {
	claude, _ := setup(t)
	copyTree(t, "testdata/minimax", filepath.Join(filepath.Dir(claude), "home", ".minimax"))
	got := ListAgent("minimax-code")
	if len(got) != 2 {
		t.Fatalf("want MiniMax Code's two sessions, got %d: %+v", len(got), got)
	}
	newest := got[0]
	if newest.ID != "mvs_82c610ef258f4350b76250bda12cacb5" {
		t.Errorf("id %q, want the one Usage's calls carry", newest.ID)
	}
	if newest.Title != "say pong again" {
		t.Errorf("title %q", newest.Title)
	}
	if want := (Tokens{Input: 369, Output: 21, CacheRead: 150, CacheWrite: 60}); newest.Tokens != want {
		t.Errorf("tokens %+v, want %+v", newest.Tokens, want)
	}
	if !newest.Last.Equal(time.UnixMilli(1791717506102)) {
		t.Errorf("last %v", newest.Last)
	}
	if len(newest.Models) != 2 || newest.Models[0].Model != "m1" || newest.Models[1].Model != "m2" {
		t.Errorf("models %+v", newest.Models)
	}
	if newest.Messages != 6 {
		t.Errorf("messages %d, want its six", newest.Messages)
	}
	if newest.Resume != "" || newest.Deletable {
		t.Errorf("resume %q deletable %v: MiniMax Code's files hold neither", newest.Resume, newest.Deletable)
	}
}
