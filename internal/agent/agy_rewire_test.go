package agent

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestAgyRewireConcurrentStash(t *testing.T) {
	home := syncHome(t)
	path := filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
	writeFile(t, path, `{"model":"own-original","modelProvider":"google"}`)
	a := agy(home)
	setPort(t, 3591)
	if err := a.Apply("model", "magpie/relay/glm-4.6"); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3592)
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
	if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.value"] != "concurrent" {
		t.Fatalf("rewire: %v %v", moved, errs)
	}
	if err := a.Apply("model", ""); err != nil {
		t.Fatal(err)
	}
	if got := f.Get(); got != "own-original" {
		t.Fatalf("original lost: %q", got)
	}
}
