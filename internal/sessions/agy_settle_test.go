package sessions

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// agy writes a conversation's title once its first reply is in, rewriting a
// page of conversation_summaries.db in place: the size stays the same, and a
// write in the same clock tick as the read that saw it leaves the same mtime
// too (TestAgyTitleLater went red 2 of 200 on a coarse Linux clock, #3708).
// The rows are read again then: a database written within the settle window
// keeps no revision to be reused from, as git keeps no stat for a file it
// wrote in the same tick. One written long before is still read once.
func TestAgySummariesSettle(t *testing.T) {
	dir := putAgy(t)
	path := filepath.Join(dir, "conversation_summaries.db")
	title := func() string {
		rows := agySummariesOf(dir)
		return rows[agyRead].Title
	}
	if got := title(); got != "Listing the folder" {
		t.Fatalf("title %q", got)
	}
	// the same rewrite twice: a page changed in place, its time put back
	rewrite := func(to string) {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE conversation_summaries SET title = ? WHERE conversation_id = ?`, to, agyRead); err != nil {
			t.Fatal(err)
		}
		db.Close()
		if err := os.Chtimes(path, fi.ModTime(), fi.ModTime()); err != nil {
			t.Fatal(err)
		}
	}
	// written just now: the revision it has is not kept, so the next look
	// reads it again and sees the new title
	rewrite("Reading a.txt")
	if got := title(); got != "Reading a.txt" {
		t.Fatalf("a title written in the settle window: %q", got)
	}
	// written long before: a rewrite so close to its own time isn't told
	// from the one read, and the rows read are kept
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if got := title(); got != "Reading a.txt" {
		t.Fatalf("a settled database: %q", got)
	}
	rewrite("Listing the folder")
	if got := title(); got != "Reading a.txt" {
		t.Fatalf("a settled database was read again: %q", got)
	}
}
