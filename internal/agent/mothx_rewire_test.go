package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMothxRewireConcurrentStash(t *testing.T) {
	home := syncHome(t)
	path := filepath.Join(home, ".mothx", "settings.json")
	original := mothxDefault(home)
	writeFile(t, path, original)
	a := mothx(home)
	setPort(t, 3591)
	if err := a.Connect(); err != nil {
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
	if b := readFile(path); !strings.Contains(b, "127.0.0.1:3592") {
		t.Fatalf("old gateway: %s", b)
	}
	if err := a.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(path); got != original {
		t.Fatalf("original not restored: %s", got)
	}
}
