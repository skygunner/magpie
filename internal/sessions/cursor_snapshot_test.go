package sessions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func cursorTempCopies(t *testing.T) []string {
	t.Helper()
	d, _ := filepath.Glob(filepath.Join(os.TempDir(), "magpie-cursor-*"))
	return d
}

// #3228: Cursor's state.vscdb is often several GB. A copy of it is written
// out in full on every change where the file system can't clone it, which on
// NTFS cost hundreds of GB of writes a day. A large store with a live log is
// read in place instead, read-only: no copy is made, and its newest chats,
// which only the log has yet, are still read.
func TestCursorLargeStoreIsReadInPlace(t *testing.T) {
	cursorSetup(t)
	t.Setenv("TMPDIR", t.TempDir())
	minWas := cursorSnapMin
	cursorSnapMin = 0
	t.Cleanup(func() { cursorSnapMin = minWas })

	config, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(config, "Cursor", "User", "globalStorage", "state.vscdb")
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	for _, q := range []string{"PRAGMA journal_mode=WAL", "PRAGMA wal_autocheckpoint=0", "CREATE TABLE cursorDiskKV (key TEXT PRIMARY KEY, value BLOB)"} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	put := func(k string, v any) {
		b, _ := json.Marshal(v)
		if _, err := db.Exec("INSERT OR REPLACE INTO cursorDiskKV VALUES (?,?)", k, b); err != nil {
			t.Fatal(err)
		}
	}
	at := time.Now().Add(-time.Hour)
	var ids []string
	for i := range 6 {
		id := fmt.Sprintf("0000000%d-1111-2222-3333-444444444444", i)
		ids = append(ids, id)
		put("composerData:"+id, map[string]any{"composerId": id, "name": "Chat " + id[:8], "createdAt": at.UnixMilli(), "lastUpdatedAt": at.Add(time.Minute).UnixMilli(), "fullConversationHeadersOnly": []any{map[string]any{"bubbleId": "u"}, map[string]any{"bubbleId": "a"}}})
		put("bubbleId:"+id+":u", map[string]any{"type": 1, "text": "question " + id, "createdAt": at.Format(time.RFC3339Nano)})
		put("bubbleId:"+id+":a", map[string]any{"type": 2, "text": "answer", "createdAt": at.Add(time.Minute).Format(time.RFC3339Nano)})
	}
	// the writer is still open and its log is not checkpointed: every row,
	// the schema's included, is in the -wal, so a read that ignores the log
	// finds no chats at all
	if fi, err := os.Stat(p + "-wal"); err != nil || fi.Size() == 0 {
		t.Fatalf("fixture has no live -wal: %v", err)
	}

	// a visit to Sessions: the list, the counts, each chat, a search
	visit := func() {
		ss := List(All)
		Agents()
		for _, id := range ids {
			s, ok := findSession(ss, "cursor", id)
			if !ok {
				t.Fatalf("chat %s not listed", id)
			}
			if tr, err := TranscriptOf(s); err != nil || len(tr.Parts) != 2 {
				t.Fatalf("transcript %+v: %v", tr, err)
			}
			if _, _, err := SearchSession(context.Background(), s, "nothing like this"); err != nil {
				t.Fatal(err)
			}
		}
	}
	before := cursorCopies.Load()
	visit()
	visit()
	if n := cursorCopies.Load() - before; n != 0 {
		t.Fatalf("a live state.vscdb was copied %d times, want 0", n)
	}
	if d := cursorTempCopies(t); len(d) != 0 {
		t.Fatalf("copies in the temp folder: %v", d)
	}

	put("composerData:"+ids[0], map[string]any{"composerId": ids[0], "name": "Renamed", "createdAt": at.UnixMilli(), "lastUpdatedAt": at.Add(2 * time.Minute).UnixMilli(), "fullConversationHeadersOnly": []any{map[string]any{"bubbleId": "u"}, map[string]any{"bubbleId": "a"}}})
	if s := find(t, List(All), "cursor", ids[0]); s.Title != "Renamed" {
		t.Fatalf("a change to the store wasn't read: %q", s.Title)
	}
	visit()
	if n := cursorCopies.Load() - before; n != 0 {
		t.Fatalf("copies after one change: %d, want 0", n)
	}
	if d := cursorTempCopies(t); len(d) != 0 {
		t.Fatalf("copies in the temp folder: %v", d)
	}
}

// A store too small for a copy to cost real disk is still read from a copy
// of it and its log, so Cursor's folder keeps its files and times untouched,
// and the copy is gone when the read is done.
func TestCursorSmallStoreCopiedAndRemoved(t *testing.T) {
	dir := cursorSetup(t)
	t.Setenv("TMPDIR", t.TempDir())
	_, _, live, _ := makeCursorChats(t, dir)
	before := snapshot(dir)
	copies := cursorCopies.Load()
	db, done, err := openCursorStore(filepath.Join(live, "store.db"))
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM blobs").Scan(&n); err != nil || n == 0 {
		t.Fatalf("the copy of a chat's store: %d rows, %v", n, err)
	}
	done()
	if n := cursorCopies.Load() - copies; n != 1 {
		t.Fatalf("a small store was copied %d times, want 1", n)
	}
	if after := snapshot(dir); !maps.Equal(before, after) {
		t.Fatalf("Cursor's folder changed:\n%v\n%v", before, after)
	}
	if left := cursorTempCopies(t); len(left) != 0 {
		t.Fatalf("copies left in the temp folder: %v", left)
	}
}

// A copy left by a magpie that was killed mid-read is removed; one a
// running magpie made a moment ago is not.
func TestCursorLeftCopiesSwept(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	left := filepath.Join(os.TempDir(), "magpie-cursor-left")
	fresh := filepath.Join(os.TempDir(), "magpie-cursor-fresh")
	for _, d := range []string{left, fresh} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "store.db"), make([]byte, 1024), 0600); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * cursorSnapStale)
	if err := os.Chtimes(left, old, old); err != nil {
		t.Fatal(err)
	}
	sweepCursorSnaps()
	if _, err := os.Stat(left); !os.IsNotExist(err) {
		t.Fatalf("a left copy is still there: %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatalf("a copy in use was removed: %v", err)
	}
}
