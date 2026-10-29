package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// This fork's -pN builds compare as their base release, not as a
// pre-release of it: semver's order had 0.1.2051-p1 before 0.1.2051,
// which hid the base release from What's new at both ends of the range.
func TestNewerTreatsAProdBuildAsItsBase(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"0.1.2051-p1", "0.1.2051", false},
		{"0.1.2051", "0.1.2051-p1", false},
		{"0.1.2051-p2", "0.1.2051-p1", false},
		{"0.1.2052", "0.1.2051-p1", true},
		{"0.1.2051-p1", "0.1.2050", true},
		{"0.1.2050", "0.1.2051-p1", false},
		{"0.1.2051-p1", "0.1.1948-p1", true},
	} {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestReleasedOnAProdBuild(t *testing.T) {
	if !Released("0.1.2051-p1") {
		t.Error("a -p1 build is a release of its base, not a source build")
	}
	if Released("0.1.604-3-g0bcb2cc") {
		t.Error("git describe's output is not a release")
	}
}

// The range's bounds are the base releases themselves: 0.1.2051's notes
// are in "up to 0.1.2051-p1", and 0.1.1948's stay out of "since
// 0.1.1948-p1" — that release had already run, its notes already shown.
func TestBetweenAProdBuildRange(t *testing.T) {
	notes := []Note{
		{Version: "0.1.2052", Notes: "after"},
		{Version: "0.1.2051", Notes: "the base of upto"},
		{Version: "0.1.2050", Notes: "in between"},
		{Version: "0.1.1948", Notes: "the base of after"},
		{Version: "0.1.1947", Notes: "before"},
	}
	got := Between(notes, "0.1.1948-p1", "0.1.2051-p1")
	want := []string{"0.1.2051", "0.1.2050"}
	if len(got) != len(want) {
		t.Fatalf("got %d releases, want %d: %v", len(got), len(want), versions(got))
	}
	for i, v := range want {
		if got[i].Version != v {
			t.Errorf("release %d = %s, want %s", i, got[i].Version, v)
		}
	}
}

func versions(notes []Note) []string {
	var out []string
	for _, n := range notes {
		out = append(out, n.Version)
	}
	return out
}

// The site is asked in whole release numbers: its own newer() reads a -pN
// suffix as a pre-release, which would leave the range's base releases out
// at both ends.
func TestNotesAskedInWholeReleases(t *testing.T) {
	var after, upto string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		after, upto = q.Get("after"), q.Get("upto")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"releases": []Note{}})
	}))
	defer srv.Close()
	t.Setenv("MAGPIE_NOTES_FEED", srv.URL+"/api/notes")
	NotesBetween(context.Background(), "0.1.1948-p1", "0.1.2051-p1", "")
	if after != "0.1.1948" || upto != "0.1.2051" {
		t.Errorf("the site was asked after=%q upto=%q; want 0.1.1948, 0.1.2051", after, upto)
	}
	after, upto = "", ""
	NotesBack(context.Background(), "0.1.2051-p1", 10, "")
	if upto != "0.1.2051" {
		t.Errorf("NotesBack asked upto=%q; want 0.1.2051", upto)
	}
}

// ShowNotes between two builds of this fork's line still shows the notes
// since, from the last one's base.
func TestShowNotesBetweenProdBuilds(t *testing.T) {
	after, show := ShowNotes("0.1.1948-p1", "0.1.2051-p1", true)
	if !show || after != "0.1.1948-p1" {
		t.Errorf("ShowNotes = %q, %v; want 0.1.1948-p1, true", after, show)
	}
}
