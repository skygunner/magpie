package sessions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// dshOwn is a session in this computer's own dsh folder, not a distro's: the
// store a WSL delete must leave alone.
const dshOwn = "session-0b7d5c31-9e42-4f68-a3d5-1c8e6f2a4b70"

// wslDsh is a WSL distro with DeepSeek Harness's own sessions in its ~/.dsh
// (#2771): a compressed v4 session and its subagent's, another in the same
// project, each with the row dsh caches of it — and this computer's own dsh
// folder, DSH_HOME, holding a session of its own. Every file is a day old.
func wslDsh(t *testing.T) (home, own string, running *bool) {
	manageSetup(t)
	dir := t.TempDir()
	own = filepath.Join(dir, "own-dsh")
	t.Setenv("DSH_HOME", own)
	home = filepath.Join(dir, "Ubuntu", "home", "me")
	distro := filepath.Join(home, ".dsh")
	dshSession(t, distro, dshP, `{"type":"session","version":4,"id":"`+dshP+`","createdAt":1790503200000,"cwd":"`+dshCwd+`","isSeeded":false,"delegationDepth":0}`, "Tidy the build script")
	dshSession(t, distro, dshC, `{"type":"session","version":4,"id":"`+dshC+`","createdAt":1790503230000,"cwd":"`+dshCwd+`","isSeeded":false,"delegationDepth":1,"parentSession":"`+dshP+`","origin":"subagent"}`, "Find the flaky test")
	dshSession(t, distro, dshO, `{"type":"session","version":4,"id":"`+dshO+`","createdAt":1790503300000,"cwd":"`+dshCwd+`","isSeeded":false,"delegationDepth":0}`, "Write the release notes")
	dshSession(t, own, dshOwn, `{"type":"session","version":4,"id":"`+dshOwn+`","createdAt":1790503100000,"cwd":"/work/host","isSeeded":false,"delegationDepth":0}`, "This computer's own")
	ageDsh(home)
	ageDsh(own)
	on := true
	WSLHomes = func() []WSLHome { return []WSLHome{{Distro: "Ubuntu", Home: home, Running: on}} }
	t.Cleanup(func() { wslSettle(); WSLHomes = nil; Reset() })
	Reset()
	return home, own, &on
}

// DeepSeek Harness's sessions in a WSL distro are listed with it — its
// titles, times, Linux working directories, compressed records and its
// subagent under the session it ran in — and its direct calls are read,
// while a stopped distro's are still listed as last read (#2771).
func TestWSLDshSessionsListed(t *testing.T) {
	home, _, running := wslDsh(t)
	ss := List(0)
	wslSettle()
	a, ok := findSession(ss, "dsh", dshP)
	if !ok {
		t.Fatalf("the distro's dsh session isn't listed: %+v", ss)
	}
	if a.WSL != "Ubuntu" || !strings.HasPrefix(a.Path, home) || a.Cwd != dshCwd ||
		a.Title != "Tidy the build script" || a.Tokens.zero() || a.Start.IsZero() || a.Last.IsZero() {
		t.Fatalf("the distro's dsh session: %+v", a)
	}
	// the subagent is the session it ran under's, as this computer's is
	m, ok := findManaged(ListAgent("dsh"), dshP)
	if !ok || m.WSL != "Ubuntu" || !m.Deletable || m.Files != 2 {
		t.Fatalf("the distro's dsh session listed as %+v, %v", m, ok)
	}
	by := map[string][]Call{}
	for _, c := range Calls(time.Time{}) {
		if c.Agent == "dsh" {
			by[c.Session] = append(by[c.Session], c)
		}
	}
	if n := len(by[dshP]); n != 2 {
		t.Fatalf("the distro's session and its subagent: %d calls, want 2: %+v", n, by)
	}
	for _, c := range append(by[dshP], by[dshO]...) {
		if c.Model != "deepseek-v4-pro" || c.Tokens != (Tokens{100, 20, 1000, 0, 0}) {
			t.Fatalf("a call of the distro's: %+v", c)
		}
	}
	if len(by[dshOwn]) != 1 {
		t.Fatalf("this computer's own dsh call: %+v", by[dshOwn])
	}

	// a distro stopped since the last listing keeps its dsh sessions as
	// last read: nothing in it is opened, which would start it
	Saved()
	*running = false
	Reset()
	if err := os.RemoveAll(home); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		ss := List(0)
		wslSettle()
		s, ok := findSession(ss, "dsh", dshP)
		if !ok || s.WSL != "Ubuntu" || s.Title != "Tidy the build script" || s.Tokens.zero() {
			t.Fatalf("pass %d: the stopped distro's dsh session: %+v", i, s)
		}
	}
}

// A dsh session in a WSL distro is deleted and restored as this computer's
// are: its folder, its subagent's and their cached rows leave the distro's
// ~/.dsh and come back there, and this computer's own store is untouched
// (#2771).
func TestWSLDshSessionDeleted(t *testing.T) {
	home, own, _ := wslDsh(t)
	List(0)
	wslSettle()
	if m, ok := findManaged(ListAgent("dsh"), dshP); !ok || m.WSL != "Ubuntu" || m.Files != 2 {
		t.Fatalf("the distro's dsh session listed as %+v, %v", m, ok)
	}
	proj := filepath.Join(home, ".dsh", "sessions", dshProjD)
	cache := filepath.Join(home, ".dsh", "storages", "session_projcache", "sessions")
	pDir, cDir := filepath.Join(proj, dshP), filepath.Join(proj, dshC)
	tr, err := Delete("dsh", dshP)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{pDir, cDir, filepath.Join(cache, dshP+".json"), filepath.Join(cache, dshC+".json")} {
		if _, err := os.Lstat(p); err == nil {
			t.Fatalf("%s is still in the distro", p)
		}
	}
	if _, err := os.Stat(filepath.Join(own, "sessions", dshProjD, dshOwn, "session.v4.jsonl.zstd")); err != nil {
		t.Fatalf("this computer's own session was moved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cache, dshO+".json")); err != nil {
		t.Fatalf("another session's cached row was moved: %v", err)
	}
	if _, err := Restore(tr.Key); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(pDir, "session.v4.jsonl.zstd"), filepath.Join(pDir, "session.lock"),
		filepath.Join(cDir, "session.v4.jsonl.zstd"),
		filepath.Join(cache, dshP+".json"), filepath.Join(cache, dshC+".json"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s isn't back in the distro: %v", p, err)
		}
	}
	if m, ok := findManaged(ListAgent("dsh"), dshP); !ok || m.WSL != "Ubuntu" || m.Files != 2 {
		t.Fatalf("the restored session: %+v, %v", m, ok)
	}
}
