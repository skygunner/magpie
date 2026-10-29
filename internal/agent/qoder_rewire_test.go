package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestQoderRewireRejectsReplacingOriginal(t *testing.T) {
	home := syncHome(t)
	path := filepath.Join(home, ".qoder", "settings.json")
	writeFile(t, path, `{"model":{"name":"own/original"}}`)
	a := qoder(home)
	if err := a.Apply("model", "magpie/relay/glm-4.6"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, `{"model":{"name":"other/new"}}`)
	moved, errs := setAgain([]*Agent{a})
	if len(moved) != 0 || len(errs) == 0 {
		t.Fatalf("conflict accepted: %v %v", moved, errs)
	}
	if got := a.Field("model").Get(); got != "other/new" {
		t.Fatalf("user model lost: %q", got)
	}
	if got := stashLoad()["qoder:"+path+":model"]; got != "own/original" {
		t.Fatalf("original lost: %q", got)
	}
}

func TestQoderRewireConcurrentStash(t *testing.T) {
	for _, build := range []qoderBuild{qoderGlobal, qoderChina} {
		t.Run(build.id, func(t *testing.T) {
			home := syncHome(t)
			path := filepath.Join(home, build.dir, "settings.json")
			writeFile(t, path, `{"model":{"name":"own/original"}}`)
			a := qoderSite(home, build)
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
			if got := a.Field("model").Get(); got != "own/original" {
				t.Fatalf("original lost: %q", got)
			}
		})
	}
}
