package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClineRewireRejectsChangedExtensionState(t *testing.T) {
	home := syncHome(t)
	data := filepath.Join(home, ".cline", "data")
	state := filepath.Join(data, "globalState.json")
	writeFile(t, state, `{"openAiBaseUrl":"https://original.example"}`)
	writeFile(t, filepath.Join(data, "secrets.json"), `{"openAiApiKey":"original-secret"}`)
	a := cline(home)
	setPort(t, 3591)
	if err := a.Apply("model", "magpie/relay/glm-4.6"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, state, `{"openAiBaseUrl":"https://changed.example"}`)
	setPort(t, 3592)
	moved, errs := setAgain([]*Agent{a})
	if len(moved) != 0 || len(errs) == 0 {
		t.Fatalf("changed state accepted: %v %v", moved, errs)
	}
	if got := readFile(state); !strings.Contains(got, "https://changed.example") {
		t.Fatalf("user change lost: %s", got)
	}
}

func TestClineRewirePreservesExtensionState(t *testing.T) {
	home := syncHome(t)
	data := filepath.Join(home, ".cline", "data")
	path := filepath.Join(data, "settings", "providers.json")
	state := filepath.Join(data, "globalState.json")
	secret := filepath.Join(data, "secrets.json")
	writeFile(t, path, `{"version":1,"lastUsedProvider":"anthropic","providers":{"anthropic":{"settings":{"model":"mine"}}}}`)
	writeFile(t, state, `{"openAiBaseUrl":"https://original.example"}`)
	writeFile(t, secret, `{"openAiApiKey":"original-secret"}`)
	a := cline(home)
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
	if b := readFile(state); !strings.Contains(b, "127.0.0.1:3592") {
		t.Fatalf("state still on old gateway: %s", b)
	}
	if err := a.Apply("model", ""); err != nil {
		t.Fatal(err)
	}
	if got := f.Get(); got != "mine" {
		t.Fatalf("model lost: %q", got)
	}
	if b := readFile(secret); b != `{"openAiApiKey":"original-secret"}` {
		t.Fatalf("secret lost: %s", b)
	}
}
