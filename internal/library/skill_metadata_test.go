package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// #1072: opening a copied skill in a file manager isn't an edit, and its
// metadata doesn't belong in another agent's copy or a carried library.
func TestSkillIgnoresFileManagerMetadata(t *testing.T) {
	h := sandbox(t)
	src := filepath.Join(h, "src/pdf")
	skill(t, src, "pdf", "Read PDFs")
	ok(t)(InstallSkills(filepath.Dir(src), []string{"pdf"}, []string{"claude", "codex"}))
	ok(t)(SetSkillHow("", HowCopy))
	cl, cx := filepath.Join(h, ".claude/skills/pdf"), filepath.Join(h, ".codex/skills/pdf")
	before, stamp, _, good := hashWalk(cl, true, true)
	if !good {
		t.Fatal("the copy couldn't be hashed")
	}
	metadata := []string{"desktop.ini", "Thumbs.db", "nested/THUMBS.DB", "nested/DESKTOP.INI", ".DS_Store", "nested/.ds_store", ".git/HEAD"}
	for _, name := range metadata {
		write(t, filepath.Join(cl, name), "file manager")
	}
	if after, now, _, good := hashWalk(cl, true, true); !good || after != before || now != stamp {
		t.Error("file manager metadata changed the copy's hash or memo stamp")
	}
	if s := skillView(t, "pdf"); len(s.Behind) != 0 {
		t.Errorf("file manager metadata made the copy look changed: %v", s.Behind)
	}
	if !sameTree(src, cl) {
		t.Error("the copy's tree differs only by file manager metadata")
	}
	res := ok(t)(Sync())
	if len(res.Changed) != 0 || backedUp(t, "Read PDFs") {
		t.Errorf("file manager metadata was taken as an edit: %+v", res)
	}
	for _, name := range metadata {
		if _, err := os.Stat(filepath.Join(cl, name)); err != nil {
			t.Errorf("the file manager's own file was removed: %s: %v", name, err)
		}
	}

	// A real edit still reaches the library and the other agent, without
	// bringing the file manager's files along with it.
	notes := filepath.Join(cl, "notes.md")
	write(t, notes, "my notes")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(notes, later, later); err != nil {
		t.Fatal(err)
	}
	ok(t)(Sync())
	for _, dir := range []string{src, cx} {
		if read(t, filepath.Join(dir, "notes.md")) != "my notes" {
			t.Errorf("the edit didn't reach %s", dir)
		}
		for _, name := range metadata {
			if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
				t.Errorf("metadata copied to %s: %s", dir, name)
			}
		}
	}
	for _, name := range metadata {
		write(t, filepath.Join(src, name), "file manager")
	}
	b, err := Collect()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range metadata {
		if _, carried := b.Skills[0].Files[name]; carried || slices.Contains(b.Skills[0].Left, name) {
			t.Errorf("the bundle carries file manager metadata: %s", name)
		}
	}
	if string(b.Skills[0].Files["notes.md"]) != "my notes" {
		t.Fatal("the bundle lost the real edit")
	}
	ok(t)(SetSkillHow("claude", HowLink))
	ok(t)(SetSkillHow("claude", HowCopy))
	for _, name := range metadata {
		if _, err := os.Stat(filepath.Join(cl, name)); !os.IsNotExist(err) {
			t.Errorf("metadata copied from the library: %s", name)
		}
	}
}

// Earlier versions included metadata in both Skill.Hash and copy marks.
// Keep those hashes valid, and keep protecting edits to the actual skill.
func TestSkillKeepsRecordedMetadataHashes(t *testing.T) {
	for _, kind := range []string{"github", "git"} {
		t.Run(kind, func(t *testing.T) {
			h := sandbox(t)
			files := map[string]string{
				"skills/pdf/SKILL.md":    "---\nname: pdf\ndescription: PDFs\n---\n",
				"skills/pdf/.DS_Store":   "finder",
				"skills/pdf/desktop.ini": "explorer",
				"skills/pdf/Thumbs.db":   "thumbnails",
			}
			repo := "owner/metadata-hashes"
			if kind == "git" {
				repo, _ = gitRepo(t, files)
			} else {
				fakeSkillRepo(t, files)
			}
			ok(t)(InstallSkills(repo, []string{"skills/pdf"}, []string{"claude"}))
			if s := skillView(t, "pdf"); s.Edited {
				t.Error("a newly installed skill with source metadata reads as edited")
			}
			ok(t)(SetSkillHow("", HowCopy))
			copy := filepath.Join(h, ".claude/skills/pdf")
			// Recreate the files and the exact hash an earlier version kept,
			// independently of the current walker and copy implementation.
			x := sha256.New()
			for _, p := range slices.Sorted(maps.Keys(files)) {
				rel := strings.TrimPrefix(p, "skills/pdf/")
				fmt.Fprintf(x, "file %s %d\n%s", rel, len(files[p]), files[p])
				write(t, filepath.Join(skillDir("pdf"), rel), files[p])
				write(t, filepath.Join(copy, rel), files[p])
			}
			oldHash := hex.EncodeToString(x.Sum(nil))
			write(t, filepath.Join(copy, marker), "hash "+oldHash+"\n")
			l, err := load()
			if err != nil {
				t.Fatal(err)
			}
			l.skill("pdf").Hash, l.skill("pdf").Commit = oldHash, ""
			if err := l.save(); err != nil {
				t.Fatal(err)
			}
			if s := skillView(t, "pdf"); s.Edited || len(s.Behind) != 0 {
				t.Errorf("an untouched skill installed by an earlier version reads as changed: %+v", s)
			}
			if desktopEdited(copy) {
				t.Error("an untouched Desktop copy with an old hash reads as edited")
			}
			if c := check(t)["pdf"]; c.Status != "current" {
				t.Errorf("an unchanged source with an old hash reads as an update: %+v", c)
			}
			write(t, filepath.Join(skillDir("pdf"), "SKILL.md"), files["skills/pdf/SKILL.md"]+"my edit\n")
			if s := skillView(t, "pdf"); !s.Edited {
				t.Error("the old hash hides a real edit")
			}
			write(t, filepath.Join(skillDir("pdf"), "SKILL.md"), files["skills/pdf/SKILL.md"])
			ok(t)(UpdateSkill("pdf", false))
			if s := skillView(t, "pdf"); s.Edited || len(s.Behind) != 0 {
				t.Errorf("the updated skill reads as changed: %+v", s)
			}
		})
	}
}
