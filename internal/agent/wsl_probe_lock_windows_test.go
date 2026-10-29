package agent

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestWSLProbeConcurrentReaderWaits(t *testing.T) {
	syncHome(t)
	fakeWSL(t, "", "", nil, nil)
	wsl.Lock()
	wsl.probing = nil
	wsl.Unlock()
	t.Cleanup(func() { wsl.Lock(); wsl.probing = nil; wsl.Unlock() })

	watchdogDone := make(chan struct{})
	watchdogStop := make(chan struct{})
	watchdogFired := make(chan struct{})
	abortProbe := make(chan struct{})
	var releaseOnce sync.Once
	finishProbe := func() { releaseOnce.Do(func() { close(abortProbe) }) }
	go func() {
		defer close(watchdogDone)
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		select {
		case <-watchdogStop:
		case <-timer.C:
			close(watchdogFired)
			finishProbe()
		}
	}()
	t.Cleanup(func() {
		close(watchdogStop)
		<-watchdogDone
	})

	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		release := abortProbe
		defer finishProbe()
		var mu sync.Mutex
		calls := 0
		wslRun = func(_ time.Duration, args ...string) ([]byte, error) {
			switch strings.Join(args, " ") {
			case "-l -q", "-l --running -q":
				return []byte("Ubuntu\n"), nil
			}
			if len(args) > 2 && args[0] == "-d" {
				mu.Lock()
				calls++
				first := calls == 1
				mu.Unlock()
				if first {
					close(started)
					<-release
				}
				return []byte("home:/home/me\ndir:.codex\n"), nil
			}
			return nil, errors.New("unexpected command")
		}
		wslRoot = func(name string) string { return name }
		first, second := make(chan []distro, 1), make(chan []distro, 1)
		go func() { first <- wslDistros() }()
		select {
		case <-started:
		case <-time.After(10 * time.Second):
			t.Fatal("first probe did not start")
		}
		go func() { second <- wslDistros() }()
		settled := make(chan struct{})
		go func() { synctest.Wait(); close(settled) }()
		select {
		case <-settled:
		case <-time.After(10 * time.Second):
			t.Fatal("readers did not reach a blocked state")
		}
		// Wait returns only when both readers are blocked on bubble channels.

		select {
		case got := <-second:
			t.Fatalf("second reader returned before probe completed: %+v", got)
		default:
		}
		finishProbe()
		for _, result := range []chan []distro{first, second} {
			select {
			case got := <-result:
				if len(got) != 1 || got[0].Name != "Ubuntu" || got[0].Home != "/home/me" || !got[0].Running {
					t.Errorf("unexpected distros: %+v", got)
				}
			case <-time.After(10 * time.Second):
				t.Error("reader did not finish")
			}
		}
		mu.Lock()
		count := calls
		mu.Unlock()
		if count != 1 {
			t.Errorf("concurrent readers started %d probes", count)
		}
	})
	select {
	case <-watchdogFired:
		t.Error("real-time watchdog released the blocked probe")
	default:
	}
}

func TestWSLProbeConcurrentChanges(t *testing.T) {
	for _, change := range []string{"list", "rescan", "restart"} {
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
				if len(got) != len(ds) {
					t.Errorf("concurrent reader got %+v, first got %+v", got, ds)
				} else {
					for i := range ds {
						if got[i].Name != ds[i].Name || got[i].Home != ds[i].Home || got[i].Running != ds[i].Running {
							t.Errorf("concurrent reader got %+v, first got %+v", got, ds)
						}
					}
				}
			case <-time.After(10 * time.Second):
				t.Error("second reader did not finish")
			}
			if change == "list" {
				if len(ds) != 2 || ds[0].Name != "Ubuntu" || ds[0].Home != "/old" || !ds[0].Running || ds[1].Name != "Debian" || ds[1].Home != "/new" || !ds[1].Running {
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
		})
	}
}
