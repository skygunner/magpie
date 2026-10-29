package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOmpProfileRewireConcurrentStash(t *testing.T) {
	home := syncHome(t)
	dir := ompProfileAt(t, home, "work", "own/original")
	a := ompProfiles(home)[0]
	setPort(t, 3591)
	if err := a.Apply("model", "magpie/relay/glm-4.6"); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3592)
	f := a.Field("model")
	original := f.setRewire
	f.setRewire = func(g *rewireGuard, v string) error {
		done := make(chan error, 1)
		go func() { done <- stash(map[string]string{"other.value": "concurrent"}) }()
		select {
		case err := <-done:
			if err != nil {
				return err
			}
		case <-time.After(5 * time.Second):
			t.Fatal("unrelated write blocked")
		}
		return original(g, v)
	}
	moved, errs := setAgain([]*Agent{a})
	if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.value"] != "concurrent" {
		t.Fatalf("rewire: %v %v", moved, errs)
	}
	b, err := os.ReadFile(filepath.Join(dir, "models.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "127.0.0.1:3592") {
		t.Fatalf("old gateway: %s", b)
	}
	if err := a.Apply("model", ""); err != nil {
		t.Fatal(err)
	}
	if got := a.Field("model").Get(); got != "own/original" {
		t.Fatalf("original lost: %q", got)
	}
}
