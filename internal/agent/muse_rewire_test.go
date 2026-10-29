package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMuseRewireConcurrentStash(t *testing.T) {
	home := syncHome(t)
	cfg := filepath.Join(home, ".config")
	path := filepath.Join(cfg, "muse", "settings.json")
	writeFile(t, path, `{"schema_version":1,"model":"own-original","endpoint_transport":{"base_url":"https://own.example"}}`)
	a := muse(cfg)
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
	if b := readFile(path); !strings.Contains(b, "127.0.0.1:3592") {
		t.Fatalf("old gateway: %s", b)
	}
	if err := a.Apply("model", ""); err != nil {
		t.Fatal(err)
	}
	if b := readFile(path); !strings.Contains(b, "own-original") || !strings.Contains(b, "https://own.example") {
		t.Fatalf("original lost: %s", b)
	}
}
