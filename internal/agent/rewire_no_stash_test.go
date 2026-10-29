package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNoStashAgentsRewireConcurrentStash(t *testing.T) {
	for _, kind := range []string{"crush", "goose", "fx", "commandcode", "reasonix", "codebuddy", "workbuddy", "workbuddy-ai"} {
		t.Run(kind, func(t *testing.T) {
			home := syncHome(t)
			cfg := filepath.Join(home, ".config")
			var a *Agent
			switch kind {
			case "crush":
				a = crush(home, cfg)
			case "goose":
				a = goose(home, cfg)
			case "fx":
				a = fx(home)
			case "commandcode":
				a = commandCode(home)
			case "reasonix":
				a = reasonix(home)
			case "codebuddy":
				a = codebuddy(home)
			case "workbuddy":
				a = workbuddy(home)
			case "workbuddy-ai":
				a = workbuddyAI(home)
			}
			setPort(t, 3591)
			field, value := "model", "magpie/relay/glm-4.6"
			if kind == "workbuddy" || kind == "workbuddy-ai" {
				field, value = "provider", "magpie"
			}
			if err := a.Apply(field, value); err != nil {
				t.Fatal(err)
			}
			setPort(t, 3592)
			f := a.Field(field)
			set := f.Set
			f.Set = func(v string) error {
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
				return set(v)
			}
			moved, errs := setAgain([]*Agent{a})
			if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.value"] != "concurrent" {
				t.Fatalf("rewire: %v %v", moved, errs)
			}
			if a.Check != nil && a.Check() != "" {
				t.Fatalf("bad wiring: %s", a.Check())
			}
			if got := f.Get(); !strings.HasPrefix(got, "magpie") {
				t.Fatalf("model lost: %q", got)
			}
		})
	}
}
