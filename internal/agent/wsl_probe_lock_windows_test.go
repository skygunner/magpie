package agent

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWSLProbeConcurrentChanges(t *testing.T) {
	for _, change := range []string{"list", "rescan", "restart", "duplicate"} {
		t.Run(change, func(t *testing.T) {
			syncHome(t)
			fakeWSL(t, "", "", nil, nil)
			wsl.Lock()
			wsl.probing = nil
			wsl.Unlock()
			t.Cleanup(func() { wsl.Lock(); wsl.probing = nil; wsl.Unlock() })
			var releaseOnce sync.Once
			var release chan struct{}

			started, listed := make(chan struct{}), make(chan struct{})
			release = make(chan struct{})
			var mu sync.Mutex
			installed, running, calls := "Ubuntu", "Ubuntu", 0
			wslRun = func(_ time.Duration, args ...string) ([]byte, error) {
				mu.Lock()
				switch strings.Join(args, " ") {
				case "-l -q":
					s := installed
					mu.Unlock()
					return []byte(s), nil
				case "-l --running -q":
					s := running
					mu.Unlock()
					return []byte(s), nil
				}
				if len(args) > 2 && args[0] == "-d" {
					calls++
					first := calls == 1
					mu.Unlock()
					if first {
						close(started)
						<-release
						return []byte("home:/old\ndir:.codex\n"), nil
					}
					if args[1] == "Debian" {
						close(listed)
					}
					return []byte("home:/new\ndir:.codex\n"), nil
				}
				mu.Unlock()
				return nil, errors.New("unexpected command")
			}
			wslRoot = func(name string) string { return name }
			done := make(chan []distro, 1)
			go func() { done <- wslDistros() }()
			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })
				select {
				case <-done:
				case <-time.After(10 * time.Second):
					t.Error("first reader did not stop")
				}
			})
			select {
			case <-started:
			case <-time.After(10 * time.Second):
				t.Fatal("probe did not start")
			}

			unlocked := make(chan struct{})
			go func() { wsl.Lock(); wsl.Unlock(); close(unlocked) }()
			select {
			case <-unlocked:
			case <-time.After(10 * time.Second):
				t.Error("probe held the state lock")
			}
			second := make(chan []distro, 1)
			secondDone := make(chan struct{})
			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })
				select {
				case <-secondDone:
				case <-time.After(10 * time.Second):
					t.Error("second reader did not stop")
				}
			})
			go func() {
				defer close(secondDone)
				switch change {
				case "list":
					mu.Lock()
					installed, running = "Ubuntu\nDebian", "Ubuntu\nDebian"
					mu.Unlock()
					wsl.Lock()
					wsl.at, wsl.runAt = time.Time{}, time.Time{}
					wsl.Unlock()
				case "rescan":
					Rescan()
				case "restart":
					mu.Lock()
					running = ""
					mu.Unlock()
					wsl.Lock()
					wsl.runAt = time.Time{}
					wsl.Unlock()
					wslUp("Ubuntu")
					mu.Lock()
					running = "Ubuntu"
					mu.Unlock()
					wsl.Lock()
					wsl.runAt = time.Time{}
					wsl.Unlock()
				case "duplicate":
				}
				second <- wslDistros()
			}()
			if change == "list" {
				select {
				case <-listed:
				case <-time.After(10 * time.Second):
					t.Error("new distro was not probed")
				}
			}
			if change == "rescan" || change == "restart" {
				select {
				case ds := <-second:
					if len(ds) != 1 || ds[0].Home != "/new" {
						t.Errorf("fresh result after %s: %+v", change, ds)
					}
					second <- ds
				case <-time.After(10 * time.Second):
					t.Error("fresh probe did not finish")
				}
			}
			releaseOnce.Do(func() { close(release) })
			var ds []distro
			select {
			case ds = <-done:
				done <- ds
			case <-time.After(10 * time.Second):
				t.Fatal("first reader did not finish")
			}
			select {
			case got := <-second:
				if len(got) == 0 || len(ds) == 0 || got[0].Home != ds[0].Home {
					t.Errorf("concurrent reader got %+v, first got %+v", got, ds)
				}
			case <-time.After(10 * time.Second):
				t.Error("second reader did not finish")
			}
			if change == "list" {
				if len(ds) != 2 || ds[0].Name != "Ubuntu" || ds[1].Name != "Debian" {
					t.Fatalf("distros after listing changed: %+v", ds)
				}
				b, err := os.ReadFile(wslStatePath())
				if err != nil {
					t.Fatal(err)
				}
				var saved map[string]*distro
				if err := json.Unmarshal(b, &saved); err != nil || saved["Debian"] == nil {
					t.Fatalf("Debian not persisted: %s, %v", b, err)
				}
			} else if len(ds) != 1 || ((change == "rescan" || change == "restart") && ds[0].Home != "/new") {
				t.Fatalf("distros after %s: %+v", change, ds)
			}
			mu.Lock()
			count := calls
			mu.Unlock()
			if change == "duplicate" && count != 1 {
				t.Fatalf("concurrent readers started %d probes", count)
			}
		})
	}
}
