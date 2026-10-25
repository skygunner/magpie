package sessions

import (
	"path/filepath"
	"strings"
	"testing"
)

// A second Codex folder keeps its own session_index.jsonl and its own
// config.toml (#3530, yetone): a session started there showed its first
// prompt instead of the name the user gave it in Codex, and a sqlite_home
// that folder set in its config.toml was ignored, its state DBs looked for
// in the folder itself. Both readers go over every Codex folder now.
func TestCodexOtherHomeTitlesAndStateDBs(t *testing.T) {
	d := setupCalls(t)
	home := filepath.Join(filepath.Dir(d.codex), "home")
	work := filepath.Join(home, ".codex-work")
	writeLines(t, filepath.Join(work, "auth.json"), `{"auth_mode":"chatgpt","tokens":{}}`)
	writeLines(t, filepath.Join(d.codex, "session_index.jsonl"),
		`{"id":"main-one","thread_name":"Main Name"}`)
	writeLines(t, filepath.Join(work, "session_index.jsonl"),
		`{"id":"work-only","thread_name":"Work Name"}`)
	t.Cleanup(Reset)

	got := CodexTitles([]string{"main-one", "work-only"})
	if got["work-only"] != "Work Name" {
		t.Fatalf("the other folder's name is not read: %#v", got)
	}
	if got["main-one"] != "Main Name" {
		t.Fatalf("the main folder's name changed: %#v", got)
	}

	// the state DBs of the other folder follow its own config.toml
	writeLines(t, filepath.Join(work, "config.toml"), "sqlite_home = \"states\"\n")
	writeLines(t, filepath.Join(work, "states", "state_1.sqlite"), "x")
	writeLines(t, filepath.Join(work, "state_2.sqlite"), "x")
	dbs := codexStateDBs(work)
	if len(dbs) != 1 || !strings.HasSuffix(filepath.ToSlash(dbs[0]), "/.codex-work/states/state_1.sqlite") {
		t.Fatalf("the other folder's sqlite_home is ignored: %v", dbs)
	}
	// a folder with no sqlite_home of its own is looked at in the folder
	writeLines(t, filepath.Join(d.codex, "state_9.sqlite"), "x")
	if dbs := codexStateDBs(d.codex); len(dbs) != 1 || !strings.HasSuffix(filepath.ToSlash(dbs[0]), "/state_9.sqlite") {
		t.Fatalf("the main folder's DBs: %v", dbs)
	}
}

// The main folder's CODEX_SQLITE_HOME override stays the main folder's.
func TestCodexSqliteHomeEnvIsTheMainFolders(t *testing.T) {
	d := setupCalls(t)
	env := t.TempDir()
	writeLines(t, filepath.Join(env, "state_e.sqlite"), "x")
	t.Setenv("CODEX_SQLITE_HOME", env)
	dbs := codexStateDBs(d.codex)
	if len(dbs) != 1 || !strings.HasSuffix(filepath.ToSlash(dbs[0]), "state_e.sqlite") {
		t.Fatalf("the env override no longer works: %v", dbs)
	}
	home := filepath.Join(filepath.Dir(d.codex), "home")
	work := filepath.Join(home, ".codex-work")
	writeLines(t, filepath.Join(work, "auth.json"), `{"auth_mode":"chatgpt","tokens":{}}`)
	if dbs := codexStateDBs(work); len(dbs) != 0 {
		t.Fatalf("the env override leaked to the other folder: %v", dbs)
	}
}
