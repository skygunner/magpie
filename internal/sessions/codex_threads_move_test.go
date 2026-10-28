package sessions

import (
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/filelock"
)

// cxRollout writes a Codex rollout of the thread id on the provider p, as
// Codex 0.162 starts one, with the time at.
func cxRollout(t *testing.T, home, sub, id, p string, at time.Time) string {
	t.Helper()
	dir := filepath.Join(home, sub, "2026", "10", "21")
	if sub == "archived_sessions" {
		dir = filepath.Join(home, sub)
	}
	os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "rollout-2026-10-21T10-00-00-"+id+".jsonl")
	b := `{"timestamp":"2026-10-21T10:00:00.000Z","type":"session_meta","payload":{"id":"` + id + `","timestamp":"2026-10-21T10:00:00.000Z","cwd":"/work/proj","originator":"codex_cli_rs","cli_version":"0.162.1","source":"cli","model_provider":"` + p + `","base_instructions":{"text":"x"}}}` + "\n" +
		`{"timestamp":"2026-10-21T10:00:01.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"remember: the build uses make"}]}}` + "\n"
	if err := os.WriteFile(path, []byte(b), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(path, at, at)
	return path
}

func cxThreadRows(t *testing.T, home string, rows map[string]string) string {
	t.Helper()
	p := filepath.Join(home, "state_5.sqlite")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE threads (id TEXT PRIMARY KEY, rollout_path TEXT NOT NULL, model_provider TEXT NOT NULL, title TEXT NOT NULL, archived INTEGER NOT NULL DEFAULT 0)`); err != nil {
		t.Fatal(err)
	}
	for id, prov := range rows {
		if _, err := db.Exec(`INSERT INTO threads VALUES (?, 'x', ?, 't', 0)`, id, prov); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

// Codex lists only the threads of the provider it is on now, so the threads
// started while magpie was its provider ("magpie") were gone from Codex
// once it was off magpie (kaizaoji on X). MoveCodexThreads moves every
// thread on one provider to another, in its files and its row, a copy of
// each kept: an archived one too; one Codex wrote to in the last minute
// waits; a thread of another provider (CC Switch's "custom") stays.
func TestMoveCodexThreads(t *testing.T) {
	_, home := manageSetup(t)
	const (
		old      = "019a0000-0000-7000-8000-00000000000a"
		archived = "019a0000-0000-7000-8000-00000000000b"
		hot      = "019a0000-0000-7000-8000-00000000000c"
		theirs   = "019a0000-0000-7000-8000-00000000000d"
	)
	day := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	oldPath := cxRollout(t, home, "sessions", old, "magpie", day)
	arPath := cxRollout(t, home, "archived_sessions", archived, "magpie", day)
	hotPath := cxRollout(t, home, "sessions", hot, "magpie", time.Now())
	theirPath := cxRollout(t, home, "sessions", theirs, "custom", day)
	db := cxThreadRows(t, home, map[string]string{old: "magpie", archived: "magpie", hot: "magpie", theirs: "custom"})

	moved, err := MoveCodexThreads(home, "magpie", "openai")
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 2 {
		t.Fatalf("moved %+v, want the two not written to lately", moved)
	}
	for _, mv := range moved {
		if mv.From != "magpie" || mv.To != "openai" || len(mv.DB) != 1 || mv.Backup == "" {
			t.Fatalf("moved %+v", mv)
		}
		if _, err := os.Stat(filepath.Join(mv.Backup, "move.json")); err != nil {
			t.Fatalf("no copy kept: %v", err)
		}
	}
	want := map[string]string{oldPath: "openai", arPath: "openai", hotPath: "magpie", theirPath: "custom"}
	for p, prov := range want {
		if got := codexMetaProvider(p); got != prov {
			t.Errorf("%s: file on %q, want %q", filepath.Base(p), got, prov)
		}
		if fi, _ := os.Stat(p); p != hotPath && !fi.ModTime().Equal(day) {
			t.Errorf("%s: time %v, want %v kept", filepath.Base(p), fi.ModTime(), day)
		}
	}
	for id, prov := range map[string]string{old: "openai", archived: "openai", hot: "magpie", theirs: "custom"} {
		if got, _, _ := threadProvider(t, db, id); got != prov {
			t.Errorf("row %s on %q, want %q", id, got, prov)
		}
	}
	b, _ := os.ReadFile(oldPath)
	if !strings.Contains(string(b), "remember: the build uses make") {
		t.Fatalf("the thread's messages went:\n%s", b)
	}

	// the one written to lately goes once Codex is done with it
	hour := time.Now().Add(-time.Hour)
	os.Chtimes(hotPath, hour, hour)
	if moved, err = MoveCodexThreads(home, "magpie", "openai"); err != nil || len(moved) != 1 || moved[0].ID != hot {
		t.Fatalf("again: %+v %v", moved, err)
	}
	if moved, err = MoveCodexThreads(home, "magpie", "openai"); err != nil || len(moved) != 0 {
		t.Fatalf("a third time: %+v %v", moved, err)
	}
}

// A Codex folder with no state database, or none of its threads on the
// provider, is left as it is, with nothing kept in the trash.
func TestMoveCodexThreadsNone(t *testing.T) {
	_, home := manageSetup(t)
	day := time.Now().Add(-24 * time.Hour)
	p := cxRollout(t, home, "sessions", "019a0000-0000-7000-8000-00000000000e", "magpie", day)
	if moved, err := MoveCodexThreads(home, "magpie", "openai"); err != nil || len(moved) != 0 {
		t.Fatalf("no database: %+v %v", moved, err)
	}
	if got := codexMetaProvider(p); got != "magpie" {
		t.Fatalf("file on %q", got)
	}
	if _, err := MoveCodexThreads(home, "magpie", "magpie"); err == nil {
		t.Fatal("a move to the same provider went")
	}
}

// bringBack's move leaves a thread a Codex has open for its next call, as
// it leaves one written to in the last minute: the other threads move, this
// one keeps its file and its row, and the next call moves it once Codex has
// let go (#3751). A Codex home without writer locks, as a Codex from before
// them keeps its threads, moves as before and is given none.
func TestMoveCodexThreadsLeavesAnOpenThread(t *testing.T) {
	_, home := manageSetup(t)
	const (
		open  = "019a0000-0000-7000-8000-000000000011"
		other = "019a0000-0000-7000-8000-000000000012"
	)
	day := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	openPath := cxRollout(t, home, "sessions", open, "custom", day)
	otherPath := cxRollout(t, home, "sessions", other, "custom", day)
	db := cxThreadRows(t, home, map[string]string{open: "custom", other: "custom"})
	locks := filepath.Join(home, "thread-writer-locks")

	if moved, err := MoveCodexThreads(home, "custom", "magpie"); err != nil || len(moved) != 2 {
		t.Fatalf("no writer locks: %+v %v", moved, err)
	}
	if _, err := os.Stat(locks); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a home without writer locks was given one: %v", err)
	}
	if moved, err := MoveCodexThreads(home, "magpie", "custom"); err != nil || len(moved) != 2 {
		t.Fatalf("back: %+v %v", moved, err)
	}

	// a Codex that resumed one thread holds its writer lock
	if err := os.MkdirAll(locks, 0o755); err != nil {
		t.Fatal(err)
	}
	lf, err := os.OpenFile(filepath.Join(locks, open+".lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer lf.Close()
	if got, err := filelock.TryLock(lf); !got || err != nil {
		t.Fatalf("lock: %v %v", got, err)
	}
	moved, err := MoveCodexThreads(home, "custom", "magpie")
	if err != nil || len(moved) != 1 || moved[0].ID != other {
		t.Fatalf("with a thread open: %+v %v", moved, err)
	}
	if got := codexMetaProvider(otherPath); got != "magpie" {
		t.Errorf("the other thread's file is on %q", got)
	}
	if got := codexMetaProvider(openPath); got != "custom" {
		t.Fatalf("the open thread's file is on %q", got)
	}
	if got, _, _ := threadProvider(t, db, open); got != "custom" {
		t.Fatalf("the open thread's row is on %q", got)
	}

	// once Codex has let go, the next call moves it
	filelock.Unlock(lf)
	if moved, err := MoveCodexThreads(home, "custom", "magpie"); err != nil || len(moved) != 1 || moved[0].ID != open {
		t.Fatalf("after its Codex let go: %+v %v", moved, err)
	}
	if got := codexMetaProvider(openPath); got != "magpie" {
		t.Fatalf("the file is on %q", got)
	}
	if got, _, _ := threadProvider(t, db, open); got != "magpie" {
		t.Fatalf("the row is on %q", got)
	}
}
