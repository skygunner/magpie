package agent

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCodexProviderRewireConcurrentStash(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	home, read := codexHome(t, "", "model = \"own-original\"\n")
	a := codex(home)
	setPort(t, 3591)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3592)
	a.Beside = nil
	f := a.Field("model")
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
	if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.value"] != "concurrent" || !strings.Contains(read(), "127.0.0.1:3592") {
		t.Fatalf("rewire: %v %v %s", moved, errs, read())
	}
	if err := a.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read(), "own-original") {
		t.Fatalf("original lost: %s", read())
	}
}
