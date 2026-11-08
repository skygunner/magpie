package library

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// #5240: the browser skill's daemon writes its own lock through a link or
// in a copy; that is not a user's edit and must not block an update.
func TestBrowserSkillRuntimeLock(t *testing.T) {
	for _, how := range []string{"link", "copy"} {
		t.Run(how, func(t *testing.T) {
			h := sandbox(t)
			version := "one"
			fakeSkillTarball(t, &version)
			ok(t)(InstallSkills("owner/repo", []string{"skills/pdf"}, []string{"claude"}))
			if how == "copy" {
				ok(t)(SetSkillHow("", HowCopy))
			}
			p := filepath.Join(h, ".claude/skills/pdf")
			if linked(p) != (how == "link") {
				t.Fatalf("unexpected install type at %s", p)
			}
			write(t, filepath.Join(p, ".bsk.lock"), "")
			if s := skillView(t, "pdf"); s.Edited {
				t.Error("runtime lock marked skill edited")
			}
			write(t, filepath.Join(p, ".bsk.lock"), "daemon state")
			if s := skillView(t, "pdf"); s.Edited {
				t.Error("updated runtime lock marked skill edited")
			}
			version = "two"
			if _, err := UpdateSkill("pdf", false); err != nil {
				t.Fatalf("update with only runtime lock: %v", err)
			}
			if got := read(t, filepath.Join(skillDir("pdf"), "forms.md")); got != "forms two" {
				t.Fatalf("update did not install new files: %q", got)
			}
			write(t, filepath.Join(p, "forms.md"), "my edit")
			later := time.Now().Add(time.Minute)
			if err := os.Chtimes(filepath.Join(p, "forms.md"), later, later); err != nil {
				t.Fatal(err)
			}
			version = "three"
			var edited *EditedError
			if _, err := UpdateSkill("pdf", false); !errors.As(err, &edited) {
				t.Fatalf("real edit must refuse update: %v", err)
			}
			if got := read(t, filepath.Join(p, "forms.md")); got != "my edit" {
				t.Fatalf("real edit lost: %q", got)
			}
		})
	}
}

// A lock belonging to the skill is still content; the bsk runtime mark
// alone is omitted from copies and carried bundles.
func TestBrowserSkillRuntimeLockNotCarried(t *testing.T) {
	sandbox(t)
	version := "one"
	fakeSkillTarball(t, &version)
	ok(t)(InstallSkills("owner/repo", []string{"skills/pdf"}, []string{"claude"}))
	dir := skillDir("pdf")
	write(t, filepath.Join(dir, ".bsk.lock"), "runtime")
	write(t, filepath.Join(dir, "ordinary.lock"), "content")
	if !skillView(t, "pdf").Edited {
		t.Error("ordinary lock must count as an edit")
	}
	if _, err := UpdateSkill("pdf", false); err == nil {
		t.Error("ordinary lock edit did not block update")
	}
	copy := filepath.Join(t.TempDir(), "copy")
	if err := copyDir(dir, copy); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(copy, ".bsk.lock")); !os.IsNotExist(err) {
		t.Errorf("runtime lock copied: %v", err)
	}
	if got := read(t, filepath.Join(copy, "ordinary.lock")); got != "content" {
		t.Errorf("ordinary lock not copied: %q", got)
	}
	bundle, err := Collect()
	if err != nil || bundle == nil || len(bundle.Skills) != 1 {
		t.Fatalf("collect skills: %v, %+v", err, bundle)
	}
	files := bundle.Skills[0].Files
	if _, ok := files[".bsk.lock"]; ok {
		t.Error("runtime lock carried in bundle")
	}
	if string(files["ordinary.lock"]) != "content" {
		t.Error("ordinary lock missing from bundle")
	}
	write(t, filepath.Join(copy, ".bsk.lock"), "local state")
	if err := mirrorDir(dir, copy); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(copy, ".bsk.lock")); got != "local state" {
		t.Errorf("mirror changed local runtime state: %q", got)
	}
}

// Existing fetched hashes without a runtime lock must remain valid, and
// Finder's historical inclusion remains controlled by noFinder.
func TestBrowserSkillLockPreservesHashModes(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "SKILL.md"), "content")
	write(t, filepath.Join(dir, ".DS_Store"), "finder")
	full, noFinder := hashFiles(dir, false), hashFiles(dir, true)
	if full == noFinder {
		t.Fatal("Finder entry must distinguish the two hash modes")
	}
	write(t, filepath.Join(dir, ".bsk.lock"), "state")
	if hashFiles(dir, false) != full || hashFiles(dir, true) != noFinder {
		t.Fatal("runtime lock changed an existing hash")
	}
}
