package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAnteRewireConcurrentStash(t *testing.T) {
	home := syncHome(t)
	dir := filepath.Join(home, ".ante")
	settings := filepath.Join(dir, "settings.json")
	writeFile(t, settings, `{"provider":"own","model":"own-original"}`)
	a := ante(home)
	setPort(t, 3591)
	if err := a.Apply("provider", "magpie"); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3592)
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
	if b := readFile(filepath.Join(dir, "catalog.json")); !strings.Contains(b, "127.0.0.1:3592") {
		t.Fatalf("old gateway: %s", b)
	}
	if err := a.Apply("provider", ""); err != nil {
		t.Fatal(err)
	}
	if b := readFile(settings); !strings.Contains(b, "own-original") || !strings.Contains(b, `"own"`) {
		t.Fatalf("original lost: %s", b)
	}
}
