package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSnowRewireConcurrentStash(t *testing.T) {
	home := syncHome(t)
	dir := filepath.Join(home, ".snow")
	writeFile(t, filepath.Join(dir, "active_profile"), "work\n")
	original := `{"snowcfg":{"advancedModel":"own-original","apiBaseUrl":"https://own.example"}}`
	writeFile(t, filepath.Join(dir, "profiles", "work.json"), original)
	a := snow(home)
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
	if b := readFile(filepath.Join(dir, "profiles", "magpie.json")); !strings.Contains(b, "127.0.0.1:3592") {
		t.Fatalf("old gateway: %s", b)
	}
	if err := a.Apply("model", ""); err != nil {
		t.Fatal(err)
	}
	if b := readFile(filepath.Join(dir, "profiles", "work.json")); b != original {
		t.Fatalf("original lost: %s", b)
	}
}
