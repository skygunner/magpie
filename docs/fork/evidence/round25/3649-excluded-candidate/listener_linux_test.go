package proc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
)

// A process of a fixture /proc: its pid, the program its exe names, and
// one link for each of the descriptors it holds.
type fixturePid struct {
	pid int
	exe string
	fds []string
}

// nodeFds are the descriptors of a node process holding the port beside a
// lot it holds besides it: every one of them a library or a socket, the
// last the port's, so the walk reads the whole directory to find it.
func nodeFds(n, inode int) []string {
	fds := make([]string, 0, n)
	for i := range n {
		if i == n-1 {
			fds = append(fds, fmt.Sprintf("socket:[%d]", inode))
			continue
		}
		fds = append(fds, fmt.Sprintf("/usr/lib/node/module-%d.so", i))
	}
	return fds
}

// fixtureProc writes a /proc of the test's own and points the walk at it:
// the net tables naming port listening at inode, and one dir per pid with
// its exe and its descriptors. No process of the machine's is read while
// the variable naming is set, so the test does not wait for the walk of
// the machine it runs on.
func fixtureProc(t *testing.T, port, inode int, procs ...fixturePid) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("MAGPIE_TEST_PROC", root)
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "net", "tcp"),
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n"+
			fmt.Sprintf("   0: 0100007F:%04X 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 %d 1\n", port, inode))
	write(filepath.Join(root, "net", "tcp6"),
		"  sl  local_address                                    remote_address                                  st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n")
	for _, p := range procs {
		dir := filepath.Join(root, strconv.Itoa(p.pid))
		if err := os.MkdirAll(filepath.Join(dir, "fd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(p.exe, filepath.Join(dir, "exe")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "comm"), []byte("node\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		for i, link := range p.fds {
			if err := os.Symlink(link, filepath.Join(dir, "fd", strconv.Itoa(i))); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// cutAfter is a context that is done once the walk has asked it more than
// after times: a deadline that passes inside the walk, in a place a test
// can put it without a clock.
type cutAfter struct {
	context.Context
	asked atomic.Int64
	after int64
}

func (c *cutAfter) Err() error {
	if c.asked.Add(1) > c.after {
		return context.DeadlineExceeded
	}
	return nil
}

// The walk names the process a magpie is run under however little of the
// machine it managed to look at: a magpie's own descriptors are read
// first, before the other processes' millions of them, and that pass does
// not stop for the context, so the pid the take-over ends is in the list
// with the context's own error beside it (#3649: a 19.68m-descriptor
// machine, the 30s spent, and the holder never named).
func TestListeningOnNamesTheMagpieHolderWhenTheWalkIsCutShort(t *testing.T) {
	fixtureProc(t, 45441, 12345,
		fixturePid{pid: 300, exe: "/usr/bin/node", fds: nodeFds(2000, 12345)},
		fixturePid{pid: 900, exe: "/opt/magpie", fds: []string{"socket:[12345]"}},
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pids, err := ListeningOn(ctx, 45441)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a walk cut short: %v, want the context's own error", err)
	}
	if want := []int{900}; !slices.Equal(pids, want) {
		t.Fatalf("pids before the cut: %v, want %v", pids, want)
	}
}

// Every holder of the port is named, however many there are and whatever
// their programs: a magpie's own process, a program of another name with
// two thousand descriptors, and the child that inherited the same socket
// from it. Nothing here ends the walk, so this is the whole list.
func TestListeningOnNamesEveryHolderOfThePort(t *testing.T) {
	fixtureProc(t, 45441, 12345,
		fixturePid{pid: 900, exe: "/opt/magpie", fds: []string{"socket:[12345]"}},
		fixturePid{pid: 300, exe: "/usr/bin/node", fds: nodeFds(2000, 12345)},
		fixturePid{pid: 301, exe: "/usr/bin/node", fds: []string{"socket:[12345]"}},
		fixturePid{pid: 302, exe: "/usr/bin/node", fds: []string{"socket:[999]"}},
	)
	pids, err := ListeningOn(context.Background(), 45441)
	if err != nil {
		t.Fatalf("a walk that finished: %v", err)
	}
	slices.Sort(pids)
	if want := []int{300, 301, 900}; !slices.Equal(pids, want) {
		t.Fatalf("holders: %v, want %v", pids, want)
	}
}

// A deadline that passes while one process's two thousand descriptors are
// being read ends that process's reading between its batches: the socket
// its last descriptor holds is not found, and nothing else is reported.
// The walk of the process is what the take-over's 30s is spent on when a
// machine holds a great many descriptors.
func TestListeningOnEndsAProcessBetweenItsDescriptorBatches(t *testing.T) {
	fixtureProc(t, 45441, 12345,
		fixturePid{pid: 300, exe: "/usr/bin/node", fds: nodeFds(2000, 12345)},
	)
	ctx := &cutAfter{Context: context.Background(), after: 1}
	pids, err := ListeningOn(ctx, 45441)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a walk cut short: %v, want the context's own error", err)
	}
	if len(pids) != 0 {
		t.Fatalf("the batch the deadline came in was not read through: %v", pids)
	}
}
