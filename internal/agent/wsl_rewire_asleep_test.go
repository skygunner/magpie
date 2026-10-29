package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWSLAsleepCodexJoinedRewire(t *testing.T) {
	codexHome(t, "", "")
	root := t.TempDir()
	d := distro{Name: "Stopped", Root: root, Home: "/home/me", Running: true, Mirrored: true, Has: map[string]bool{"dir:.codex": true}}
	dir := filepath.Join(root, "home", "me", ".codex")
	path := filepath.Join(dir, "config.toml")
	writeFile(t, path, "model = \"gpt-5.1\"\n")
	writeFile(t, filepath.Join(dir, "auth.json"), `{"auth_mode":"chatgpt","tokens":{"access_token":"at","refresh_token":"rt","id_token":"x.e30.x","account_id":"acc"}}`)
	fakeWSL(t, "Stopped\r\n", "", nil, nil)
	wsl.seen = map[string]*distro{"Stopped": {Name: "Stopped", Values: map[string]string{}}}
	live := wslAgentsOf([]distro{d})[0]
	setPort(t, 3591)
	if _, err := live.ConnectHow(); err != nil {
		t.Fatal(err)
	}
	if live.Joined == nil || !live.Joined() {
		t.Fatal("Codex did not join")
	}
	wslRemember("Stopped", "model", live.Field("model").Get())
	d.Running = false
	sleeping := wslAgentsOf([]distro{d})[0]
	setPort(t, 3592)
	moved, errs := setAgain([]*Agent{sleeping})
	if len(errs) != 0 || len(moved) != 1 {
		t.Fatalf("rewire: %v %v", moved, errs)
	}
	if b := readFile(path); !strings.Contains(b, "127.0.0.1:3592") || !strings.Contains(b, "gpt-5.1") {
		t.Fatalf("old gateway or model lost: %s", b)
	}
}

func TestWSLAsleepCodexRewire(t *testing.T) {
	codexHome(t, "", "")
	root := t.TempDir()
	d := distro{Name: "Stopped", Root: root, Home: "/home/me", Running: true, Mirrored: true, Has: map[string]bool{"dir:.codex": true}}
	path := filepath.Join(root, "home", "me", ".codex", "config.toml")
	writeFile(t, path, "model = \"own-original\"\n")
	fakeWSL(t, "Stopped\r\n", "", nil, nil)
	wsl.seen = map[string]*distro{"Stopped": {Name: "Stopped", Values: map[string]string{}}}
	live := wslAgentsOf([]distro{d})[0]
	setPort(t, 3591)
	if err := live.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	wslRemember("Stopped", "model", live.Field("model").Get())
	if got := wslLastSeen("Stopped", "model"); got != "fake/m1" {
		t.Fatalf("saved model %q", got)
	}
	d.Running = false
	sleeping := wslAgentsOf([]distro{d})[0]
	oldRun := wslRun
	wslRun = func(timeout time.Duration, args ...string) ([]byte, error) {
		done := make(chan error, 1)
		go func() { done <- stash(map[string]string{"other.value": "concurrent"}) }()
		select {
		case err := <-done:
			if err != nil {
				return nil, err
			}
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("unrelated stash blocked")
		}
		return oldRun(timeout, args...)
	}
	setPort(t, 3592)
	moved, errs := setAgain([]*Agent{sleeping})
	if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.value"] != "concurrent" {
		t.Fatalf("rewire: %v %v stash=%q model=%q", moved, errs, stashLoad()["other.value"], sleeping.Field("model").Get())
	}
	if b := readFile(path); !strings.Contains(b, "127.0.0.1:3592") {
		t.Fatalf("old gateway: %s", b)
	}
}

func TestWSLAsleepOmpProfileRewire(t *testing.T) {
	syncHome(t)
	root := t.TempDir()
	d := distro{Name: "Stopped", Root: root, Home: "/home/me", Running: true, Mirrored: true, Has: map[string]bool{"dir:.omp": true}, OmpProfiles: []string{"work"}}
	path := filepath.Join(root, "home", "me", ".omp", "profiles", "work", "agent", "config.yml")
	writeFile(t, path, "modelRoles:\n  default: own/original\n")
	fakeWSL(t, "Stopped\r\n", "", nil, nil)
	wsl.seen = map[string]*distro{"Stopped": {Name: "Stopped", Values: map[string]string{}}}
	live := wslAgentsOf([]distro{d})[1]
	setPort(t, 3591)
	if err := live.Apply("model", "magpie/relay/glm-4.6"); err != nil {
		t.Fatal(err)
	}
	wslRemember("Stopped", "omp#work.model", live.Field("model").Get())
	d.Running = false
	sleeping := wslAgentsOf([]distro{d})[1]
	oldRun := wslRun
	wslRun = func(timeout time.Duration, args ...string) ([]byte, error) {
		done := make(chan error, 1)
		go func() { done <- stash(map[string]string{"other.value": "concurrent"}) }()
		select {
		case err := <-done:
			if err != nil {
				return nil, err
			}
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("unrelated stash blocked")
		}
		return oldRun(timeout, args...)
	}
	setPort(t, 3592)
	moved, errs := setAgain([]*Agent{sleeping})
	if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.value"] != "concurrent" {
		t.Fatalf("rewire %v %v", moved, errs)
	}
	if body := readFile(filepath.Join(filepath.Dir(path), "models.yml")); !strings.Contains(body, "127.0.0.1:3592") {
		t.Fatalf("old gateway: %s", body)
	}
	if err := sleeping.Apply("model", ""); err != nil {
		t.Fatal(err)
	}
	if got := readFile(path); !strings.Contains(got, "own/original") {
		t.Fatalf("original lost: %s", got)
	}
}
