package library

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func symlink(t *testing.T, to, at string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(to, at); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
}

// #227: the user-wide ~/.agents/skills is found, links to it (symlinks,
// junctions on Windows) are the same skill, and bringing one in leaves the
// shared folder as it is.
func TestSharedAgentsSkills(t *testing.T) {
	h := sandbox(t)
	// no agent here whose own folder the shared one is (Goose's): its CLI
	// of the sandbox too, since that alone would have it here (#2177)
	os.RemoveAll(filepath.Join(h, ".config/goose"))
	os.Remove(filepath.Join(h, "bin/goose"))
	os.Remove(filepath.Join(h, "bin/goose.exe"))
	shared := filepath.Join(h, ".agents/skills")
	// a skill kept elsewhere (D:\aimer-skills\…) linked into the shared folder
	ext := filepath.Join(h, "aimer-skills/grilling")
	skill(t, ext, "grilling", "Grill")
	symlink(t, ext, filepath.Join(shared, "grilling"))
	// real folders in the shared folder
	skill(t, filepath.Join(shared, "orchestration"), "orchestration", "Orchestrate")
	skill(t, filepath.Join(shared, "orca-cli"), "orca-cli", "Orca")
	// agents' links to it: absolute, relative, and to a link
	symlink(t, filepath.Join(shared, "orchestration"), filepath.Join(h, ".pi/agent/skills/orchestration"))
	symlink(t, filepath.Join("..", "..", ".agents", "skills", "orchestration"), filepath.Join(h, ".codex/skills/orchestration"))
	symlink(t, filepath.Join(shared, "grilling"), filepath.Join(h, ".gemini/skills/grilling"))
	// a byte copy of its own
	skill(t, filepath.Join(h, ".codex/skills/orca-cli"), "orca-cli", "Orca")
	// not a skill, and a hidden one
	write(t, filepath.Join(shared, "notes/README.md"), "")
	skill(t, filepath.Join(shared, ".hidden"), "hidden", "")

	found := func() map[string]FoundSkill {
		v, err := Read(nil)
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]FoundSkill{}
		for _, f := range v.FoundSkills {
			m[f.Name] = f
		}
		return m
	}
	byName := found()
	if len(byName) != 3 {
		t.Fatalf("found: %+v", byName)
	}
	if f := byName["grilling"]; f.Shared != filepath.Join(shared, "grilling") || f.Link != realDir(ext) || !slices.Equal(f.Agents, []string{"gemini"}) || len(f.Others) != 0 {
		t.Errorf("grilling: %+v", f)
	}
	f := byName["orchestration"]
	agents := slices.Sorted(slices.Values(f.Agents))
	if f.Shared == "" || f.Link != "" || !slices.Equal(agents, []string{"codex", "pi"}) || len(f.Others) != 0 {
		t.Errorf("orchestration is one skill in the shared folder, pi and codex: %+v", f)
	}
	if f := byName["orca-cli"]; f.Shared == "" || len(f.Agents) != 0 || len(f.Others) != 0 || !slices.Equal(f.Copies, []string{"codex"}) {
		t.Errorf("orca-cli: %+v", f)
	}

	ok(t)(ImportSkill("orchestration"))
	ok(t)(ImportSkill("grilling"))
	ok(t)(ImportSkill("orca-cli"))
	// #3657: a folder of the user's kept in the shared one is moved into
	// the library, so the agents' ticks decide which of them have it; a
	// link there stays where it points
	for _, n := range []string{"orchestration", "orca-cli"} {
		if _, err := os.Lstat(filepath.Join(shared, n)); !os.IsNotExist(err) {
			t.Errorf("%s is still in the shared folder: %v", n, err)
		}
		if linked(skillDir(n)) {
			t.Errorf("the library's %s is a link into the shared folder", n)
		}
		if _, err := os.Stat(filepath.Join(skillDir(n), "scripts/run.sh")); err != nil {
			t.Errorf("%s's files: %v", n, err)
		}
	}
	if !linked(filepath.Join(shared, "grilling")) {
		t.Error("the shared folder's link went")
	}
	if _, err := os.Stat(filepath.Join(ext, "SKILL.md")); err != nil {
		t.Error("the folder the shared link pointed to went")
	}
	// the library keeps the moved-in folders itself and links to where
	// the shared folder's link pointed (#3657)
	if linked(skillDir("orchestration")) || linked(skillDir("orca-cli")) {
		t.Error("a moved-in skill is the library's own folder")
	}
	if !linked(skillDir("grilling")) || realDir(skillDir("grilling")) != realDir(ext) {
		t.Errorf("the library's grilling isn't a link to %s", ext)
	}
	// the agents that had it have the library's; Codex and Gemini CLI read
	// ~/.agents/skills, where all three are kept, so a link in their own
	// folders would be each skill twice (#595): they get none, and Codex's
	// byte copy of its own is kept aside with the backups
	if !ours(filepath.Join(h, ".pi/agent/skills/orchestration"), "orchestration") {
		t.Error("pi's orchestration isn't the library's")
	}
	for _, d := range []string{".codex/skills/orchestration", ".codex/skills/orca-cli"} {
		if !ours(filepath.Join(h, d), filepath.Base(d)) {
			t.Errorf("%s: not the library's link", d)
		}
	}
	if m, _ := filepath.Glob(filepath.Join(BackupDir(), "*", "codex", "skills", "orca-cli", "SKILL.md")); len(m) != 1 {
		t.Errorf("codex's own copy isn't kept aside: %v", m)
	}
	v, _ := Read(nil)
	for _, s := range v.Skills {
		// a link into the shared folder is still had by every agent reading
		// it, ticked or not; one moved in is not (#3657)
		if s.Name == "grilling" {
			for _, id := range []string{"codex", "gemini"} {
				if !slices.Contains(s.Always, id) {
					t.Errorf("grilling: %s isn't said to have it always: %v", id, s.Always)
				}
			}
		} else if len(s.Always) != 0 {
			t.Errorf("%s: always %v", s.Name, s.Always)
		}
	}
	if byName := found(); len(byName) != 0 {
		t.Errorf("still found: %+v", byName)
	}
}

// #3657: a skill of the user's kept in ~/.agents/skills and brought in is
// moved into the library, so the ticks decide which agents have it: an
// agent reading the shared folder gets the library's while ticked and
// loses it once unticked, where a copy left there was read by all of them
// whatever was ticked.
func TestImportSharedSkillUnticksForItsReaders(t *testing.T) {
	h := sandbox(t)
	shared := filepath.Join(h, ".agents/skills/drawio-skill")
	skill(t, shared, "drawio-skill", "Draw diagrams")
	ok(t)(ImportSkill("drawio-skill"))
	// the folder is the library's own now, and Goose, whose folder the
	// shared one is, has the library's linked there: #3657, not a copy of
	// the user's the ticks can never take away
	if linked(skillDir("drawio-skill")) {
		t.Error("the library's is a link into the shared folder")
	}
	if !ours(shared, "drawio-skill") {
		t.Error("goose's entry in ~/.agents/skills isn't the library's link")
	}
	cx := filepath.Join(h, ".codex/skills/drawio-skill")
	ok(t)(SkillAgents("drawio-skill", []string{"codex"}))
	if !ours(cx, "drawio-skill") {
		t.Error("codex hasn't it while ticked")
	}
	if v, err := Read(nil); err != nil || len(v.Skills) != 1 || slices.Contains(v.Skills[0].Always, "codex") || slices.Contains(v.Skills[0].Always, "goose") {
		t.Errorf("said to have it always: %+v %v", v.Skills, err)
	}
	ok(t)(SkillAgents("drawio-skill", []string{}))
	for _, p := range []string{cx, shared} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Errorf("%s is there unticked: %v", p, err)
		}
	}
}

func TestRealDir(t *testing.T) {
	d := t.TempDir()
	skill(t, filepath.Join(d, "a/real"), "real", "")
	symlink(t, filepath.Join(d, "a/real"), filepath.Join(d, "b/abs"))
	symlink(t, filepath.Join("..", "b", "abs"), filepath.Join(d, "c/rel"))
	symlink(t, filepath.Join(d, "c"), filepath.Join(d, "dir"))
	want, _ := filepath.EvalSymlinks(filepath.Join(d, "a/real"))
	for _, p := range []string{"a/real", "b/abs", "c/rel", "dir/rel", "dir/rel/scripts/.."} {
		if got := realDir(filepath.Join(d, p)); got != want {
			t.Errorf("%s: %s, want %s", p, got, want)
		}
	}
	if got := realDir(filepath.Join(d, "dir/rel/scripts")); got != filepath.Join(want, "scripts") {
		t.Errorf("inside: %s", got)
	}
	// a loop, and what isn't there: the path as it is
	symlink(t, filepath.Join(d, "loop2"), filepath.Join(d, "loop1"))
	symlink(t, filepath.Join(d, "loop1"), filepath.Join(d, "loop2"))
	for _, p := range []string{filepath.Join(d, "loop1"), filepath.Join(d, "gone/x")} {
		if got := realDir(p); got != p {
			t.Errorf("%s: %s", p, got)
		}
	}
	if linked(filepath.Join(d, "a/real")) || !linked(filepath.Join(d, "c/rel")) {
		t.Error("isLink")
	}
	if !within(filepath.Join(d, "a", "x"), filepath.Join(d, "a")) || within(filepath.Join(d, "ab"), filepath.Join(d, "a")) || within(d, filepath.Join(d, "a")) {
		t.Error("within")
	}
}

// A junction (what Windows makes without the right to symlink, and what
// the issue's ZCode and Pi folders were) is a link to the folder it points to.
func TestJunction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("junctions are Windows'")
	}
	d := t.TempDir()
	skill(t, filepath.Join(d, "shared/orchestration"), "orchestration", "")
	j := filepath.Join(d, "zcode", "orchestration")
	os.MkdirAll(filepath.Dir(j), 0o755)
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", j, filepath.Join(d, "shared", "orchestration")).CombinedOutput(); err != nil {
		t.Skipf("mklink /J: %v %s", err, out)
	}
	if !linked(j) {
		t.Error("a junction isn't a link")
	}
	if realDir(j) != realDir(filepath.Join(d, "shared", "orchestration")) {
		t.Errorf("%s != %s", realDir(j), realDir(filepath.Join(d, "shared", "orchestration")))
	}
}
