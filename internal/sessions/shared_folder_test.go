package sessions

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Two agents told to share one folder under different spellings — Pi's
// variable on omp's folder through a link, Qoder CN's on Qoder's — leave
// one folder for both: its sessions are read once, as the folder's own
// agent's (omp's, Qoder's), on the Sessions page and in usage alike, and
// the folder is named once (#3258). Read as both's, each was listed twice
// under the two spellings and counted twice, and one refresh took each
// path twice and closed a nil channel (#3255). Spelled the same, the
// sessions had been read once, as the pointer's (Pi's); they are the
// folder's own agent's now, whichever way it is spelled.
func TestSharedFolderReadOnce(t *testing.T) {
	t.Run("pi on omp's folder", func(t *testing.T) {
		for _, how := range []string{"named", "linked"} {
			t.Run(how, func(t *testing.T) {
				inZone(t, 0)
				home := setupAgents(t)
				dir := filepath.Join(home, ".omp", "agent")
				if how == "linked" {
					dir = filepath.Join(home, "omp-link")
					if err := os.Symlink(filepath.Join(home, ".omp", "agent"), dir); err != nil {
						t.Skip("no symlinks here:", err)
					}
				}
				t.Setenv("PI_CODING_AGENT_DIR", dir)
				// the start's warm-up, which the duplicate paths crashed
				if st, _ := StatsAndNative(0); len(st.Sessions) == 0 {
					t.Fatal("the warm-up read no sessions")
				}
				ss := List(0)
				if o, p := count(ss, "omp"), count(ss, "pi"); o != 3 || p != 0 {
					t.Fatalf("listed %d omp and %d Pi sessions, want 3 and 0", o, p)
				}
				n := map[string]int{}
				for _, c := range Calls(time.Time{}) {
					n[c.Agent]++
				}
				if n["omp"] != 8 || n["pi"] != 0 {
					t.Fatalf("calls %v, want omp's 8 and no Pi's", n)
				}
				for _, s := range StatsFor(0).Sessions {
					if s.Agent == "pi" {
						t.Fatalf("stats counted an omp session as Pi's too: %s", s.Key)
					}
				}
				named := 0
				for _, d := range Dirs() {
					if sameFolder(d, filepath.Join(home, ".omp", "agent")) {
						named++
					}
				}
				if named != 1 {
					t.Fatalf("omp's folder named %d times in %v", named, Dirs())
				}
			})
		}
	})

	t.Run("qoder-cn on Qoder's folder", func(t *testing.T) {
		for _, how := range []string{"named", "linked"} {
			t.Run(how, func(t *testing.T) {
				inZone(t, 0)
				home := setupAgents(t)
				dir := filepath.Join(home, ".qoder")
				if how == "linked" {
					dir = filepath.Join(home, "qoder-link")
					if err := os.Symlink(filepath.Join(home, ".qoder"), dir); err != nil {
						t.Skip("no symlinks here:", err)
					}
				}
				t.Setenv("QODERCN_CONFIG_DIR", dir)
				if st, _ := StatsAndNative(0); len(st.Sessions) == 0 {
					t.Fatal("the warm-up read no sessions")
				}
				ss := List(0)
				if q, c := count(ss, "qoder"), count(ss, "qoder-cn"); q != 1 || c != 0 {
					t.Fatalf("listed %d Qoder and %d Qoder CN sessions, want 1 and 0", q, c)
				}
				for _, s := range StatsFor(0).Sessions {
					if s.Agent == "qoder-cn" {
						t.Fatalf("stats counted a Qoder session as Qoder CN's too: %s", s.Key)
					}
				}
				named := 0
				for _, d := range Dirs() {
					if sameFolder(d, filepath.Join(home, ".qoder")) {
						named++
					}
				}
				if named != 1 {
					t.Fatalf("Qoder's folder named %d times in %v", named, Dirs())
				}
			})
		}
	})
}
