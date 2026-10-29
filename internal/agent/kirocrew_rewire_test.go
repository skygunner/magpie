package agent

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/edit"
)

func TestKiroCrewRewireRejectsChangedHarness(t *testing.T) {
	_, a := kiroCrewSandbox(t, []string{"codex", "claude"}, []string{"codex", "claude"})
	if err := os.WriteFile(a.Path, []byte(kiroCrewUser), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ConnectHow(); err != nil {
		t.Fatal(err)
	}
	if err := edit.SetJSON(a.Path, edit.KV{Path: kiroCrewBackend, Value: "claude"}); err != nil {
		t.Fatal(err)
	}
	g := &rewireGuard{}
	if err := a.Field("provider").setRewire(g, "magpie"); err == nil || !g.blocked {
		t.Fatal("changed harness accepted")
	}
}

func TestKiroCrewRewireConcurrentStash(t *testing.T) {
	_, a := kiroCrewSandbox(t, []string{"codex"}, []string{"codex"})
	if err := os.WriteFile(a.Path, []byte(kiroCrewUser), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ConnectHow(); err != nil {
		t.Fatal(err)
	}
	f := a.Field("provider")
	set := f.setRewire
	f.setRewire = func(g *rewireGuard, v string) error {
		done := make(chan error, 1)
		go func() { done <- stash(map[string]string{"other.value": "concurrent"}) }()
		select {
		case err := <-done:
			if err != nil {
				return err
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("unrelated stash blocked")
		}
		return set(g, v)
	}
	moved, errs := setAgain([]*Agent{a})
	if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.value"] != "concurrent" {
		t.Fatalf("rewire: %v %v", moved, errs)
	}
	if err := a.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if b := readFile(a.Path); b != kiroCrewUser {
		t.Fatalf("original lost: %s", b)
	}
}
