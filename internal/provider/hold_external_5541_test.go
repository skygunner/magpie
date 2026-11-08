package provider

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/yetone/magpie/internal/filememo"
)

func TestOverlappingHoldSeesExternalRemovalAfterWindow(t *testing.T) {
	isolate(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("MAGPIE_NO_ACCOUNTS", "1")
	if err := Save(Provider{ID: "external-removal", Name: "External", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"m"}}); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	synctest.Test(t, func(t *testing.T) {
		ready := make(chan []Provider, 1)
		release := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer Hold()()
			defer filememo.Hold()()
			ready <- All()
			<-release
		}()
		old := <-ready
		defer func() { close(release); <-done }()
		contains := func(ps []Provider) bool {
			return slices.ContainsFunc(ps, func(p Provider) bool { return p.ID == "external-removal" })
		}
		if !contains(old) {
			t.Fatal("initial snapshot missing provider")
		}
		time.Sleep(heldFor + time.Nanosecond)
		if !contains(All()) {
			t.Fatal("provider missing when refreshing during the first hold")
		}
		if err := os.WriteFile(Path(), []byte(`{"providers":[]}`), 0o600); err != nil {
			t.Fatal(err)
		}
		func() {
			defer Hold()()
			defer filememo.Hold()()
			if contains(All()) {
				t.Error("overlapping request retained externally removed provider")
			}
		}()
		// A second edit within the same window must not trigger another stat.
		if err := os.WriteFile(Path(), original, 0o600); err != nil {
			t.Fatal(err)
		}
		func() {
			defer Hold()()
			defer filememo.Hold()()
			if contains(All()) {
				t.Error("within-window request looked at the root again")
			}
		}()
		time.Sleep(heldFor - time.Nanosecond)
		func() {
			defer Hold()()
			defer filememo.Hold()()
			if contains(All()) {
				t.Error("root checked before the window expired")
			}
		}()
		time.Sleep(time.Nanosecond)
		func() {
			defer Hold()()
			defer filememo.Hold()()
			if !contains(All()) {
				t.Error("exact window boundary missed externally restored provider")
			}
		}()
		held.Lock()
		checkedAt, generation := held.rootsAt, held.gen
		held.Unlock()
		func() {
			defer Hold()()
			defer filememo.Hold()()
			if !contains(All()) {
				t.Error("same-file fast path lost restored provider")
			}
		}()
		held.Lock()
		if held.rootsAt != checkedAt || held.gen != generation {
			t.Error("same-window hold repeated root check or invalidated cache")
		}
		held.Unlock()
		time.Sleep(heldFor)
		func() {
			defer Hold()()
			if !contains(All()) {
				t.Error("unchanged root lost provider at next window")
			}
		}()
		held.Lock()
		if held.rootsAt != checkedAt.Add(heldFor) || held.gen != generation {
			t.Error("unchanged root did not keep generation after the next check")
		}
		held.Unlock()
		if !contains(old) {
			t.Error("active request snapshot changed")
		}
	})
}

func TestExternalRootChangeInvalidatesInFlightBuild(t *testing.T) {
	isolate(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	if err := Save(Provider{ID: "relay", Name: "Relay", Key: "k", Chat: "http://127.0.0.1:1/v1"}); err != nil {
		t.Fatal(err)
	}
	synctest.Test(t, func(t *testing.T) {
		defer Hold()()
		started, proceed, result := make(chan struct{}), make(chan struct{}), make(chan string, 1)
		go func() {
			result <- heldOf("external-5541", func() string {
				close(started)
				<-proceed
				return "old"
			})
		}()
		<-started
		waited := make(chan string, 1)
		go func() { waited <- heldOf("external-5541", func() string { return "new" }) }()
		synctest.Wait()
		finished := false
		defer func() {
			if !finished {
				close(proceed)
				<-result
			}
		}()
		time.Sleep(heldFor + time.Nanosecond)
		if err := os.WriteFile(Path(), []byte(`{"providers":[]}`), 0o600); err != nil {
			t.Fatal(err)
		}
		release := Hold()
		defer release()
		timedOut := make(chan string, 1)
		go func() { timedOut <- heldOf("external-5541", func() string { return "timeout-own" }) }()
		synctest.Wait()
		time.Sleep(heldWait)
		if got := <-timedOut; got != "timeout-own" {
			t.Errorf("timed-out waiter after refresh got %q", got)
		}
		close(proceed)
		got := <-result
		finished = true
		if got != "old" {
			t.Errorf("original builder got %q", got)
		}
		if got := <-waited; got != "new" {
			t.Errorf("waiter took stale build: %q", got)
		}
		if got := heldOf("external-5541", func() string { return "new" }); got != "new" {
			t.Errorf("stale in-flight build was cached: %q", got)
		}
	})
}
