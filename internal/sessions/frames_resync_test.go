package sessions

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
)

// #4144: a damaged frame in a packed session file — a header that is no
// frame's, lengths that run past the file, blocks that do not decode — is
// passed, and the frames after it count as before, the tail moving past the
// damage instead of holding there while the file grows. The file is built
// the way dsh writes one, from the real session lines the dsh fixtures
// carry; the damage is what a torn or overwritten frame looks like on disk.
func TestFramesResyncPastDamaged(t *testing.T) {
	lines := resyncLines(t)
	frame := func(lines []string) []byte {
		enc, err := zstd.NewWriter(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer enc.Close()
		return enc.EncodeAll([]byte(strings.Join(lines, "\n")+"\n"), nil)
	}
	f1 := frame(lines[0:4])
	f2 := frame(lines[4:6])
	f3 := frame(lines[6:8])
	p := filepath.Join(t.TempDir(), "session.v3.jsonl.zstd")

	// the damages: blocks that do not decode (the checksum flipped), a
	// start that is no frame's (the magic overwritten), lengths that run
	// past the file (the first block header made to ask for megabytes)
	damages := map[string]func([]byte) []byte{
		"blocks that do not decode": func(b []byte) []byte {
			out := slices.Clone(b)
			out[len(out)-4] ^= 0xFF
			return out
		},
		"a start that is no frame's": func(b []byte) []byte {
			out := slices.Clone(b)
			copy(out, "JUNK")
			return out
		},
		"lengths that run past the file": func(b []byte) []byte {
			out := slices.Clone(b)
			out[5], out[6], out[7] = 0xF9, 0xFF, 0xFF // last block, ~2 MiB
			return out
		},
	}
	for name, damage := range damages {
		t.Run(name, func(t *testing.T) {
			write := func(parts ...[]byte) {
				t.Helper()
				var b []byte
				for _, x := range parts {
					b = append(b, x...)
				}
				if err := os.WriteFile(p, b, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			st := func(old *state) *state {
				t.Helper()
				fi, err := os.Stat(p)
				if err != nil {
					t.Fatal(err)
				}
				return parseDsh(file{agent: "dsh", path: p, main: true, size: fi.Size(), mod: modOf(fi.ModTime())}, old)
			}
			write(f1, damage(f2), f3)

			// the whole read: the head and the frames after the damage
			// count; the read goes on from the first frame past it
			s := st(nil)
			if s.Title == "" && s.more().First == "" {
				t.Fatalf("%s: the frames after the damage were not read: %+v", name, s)
			}
			if tail := s.more().Dsh.Tail; tail == nil || tail.Off < int64(len(f1)+len(f2)) {
				t.Fatalf("%s: the tail did not move past the damage: %+v", name, tail)
			}
			// the calls the frame after the damage holds are counted, the
			// damaged one's lost with the frame: the fixture's two
			// assistant messages spent 100 and then 10 input tokens
			calls := func(old *callFile) []int {
				st, _, ok := readDshCalls(file{agent: "dsh", path: p, size: mustSize(t, p), mod: modOf(mustMod(t, p))}, old)
				if !ok {
					t.Fatalf("%s: not read for calls", name)
				}
				var in []int
				for _, c := range st.Calls {
					in = append(in, c.Input)
				}
				return in
			}
			got := calls(nil)
			if !slices.Contains(got, 10) {
				t.Fatalf("%s: the call the frame after the damage holds is not counted: %v", name, got)
			}
			if slices.Contains(got, 100) {
				t.Fatalf("%s: the damaged frame's call was counted anyway: %v", name, got)
			}

			// the file grows: the next read starts past the damage and
			// takes the new frame alone
			f4 := frame(lines[4:5])
			write(f1, damage(f2), f3, f4)
			s = st(s)
			if tail := s.more().Dsh.Tail; tail == nil || tail.Off < int64(len(f1)+len(f2)+len(f3)) {
				t.Fatalf("%s: the appended frame was not read on from past the damage: %+v", name, tail)
			}
		})
	}
}

// Nothing after the damage: a file that ends in the damaged frame is still
// read as one being written, the read waiting where it stood — no resync
// into nothing, and no error out of parseDsh.
func TestFramesResyncAtTheEndWaits(t *testing.T) {
	lines := resyncLines(t)
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer enc.Close()
	f1 := enc.EncodeAll([]byte(strings.Join(lines[0:4], "\n")+"\n"), nil)
	f2 := enc.EncodeAll([]byte(strings.Join(lines[4:8], "\n")+"\n"), nil)
	p := filepath.Join(t.TempDir(), "session.v3.jsonl.zstd")
	var b []byte
	b = append(b, f1...)
	dmg := slices.Clone(f2)
	copy(dmg, "JUNK")
	b = append(b, dmg...)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	s := parseDsh(file{agent: "dsh", path: p, main: true, size: fi.Size(), mod: modOf(fi.ModTime())}, nil)
	if tail := s.more().Dsh.Tail; tail == nil || tail.Off != int64(len(f1)) {
		t.Fatalf("the read did not wait at the damaged frame: %+v", tail)
	}
}

// resyncLines is the real session lines the dsh fixture carries, as dsh
// wrote them.
func resyncLines(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "dsh", "sessions", "work-dsh-3f2a", "dsh-a", "session.v3.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) != 8 {
		t.Fatalf("the fixture's lines changed: %d", len(lines))
	}
	return lines
}

func mustSize(t *testing.T, p string) int64 {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Size()
}

func mustMod(t *testing.T, p string) time.Time {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.ModTime()
}
