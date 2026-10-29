package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/edit"
)

func TestPiFamilyBackupWriteFailurePreservesFiles(t *testing.T) {
	for _, tc := range []struct {
		name      string
		makeAgent func(string) *Agent
		dir       string
	}{
		{"pi", pi, ".pi/agent"}, {"omo", omo, ".omo/agent"}, {"prime", primeAgent, ".prime/agent"}, {"mothx", mothx, ".mothx"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PI_CODING_AGENT_DIR", "")
			t.Setenv("OMO_CODING_AGENT_DIR", "")
			t.Setenv("SENPI_CODING_AGENT_DIR", "")
			t.Setenv("PRIME_AGENT_CODING_AGENT_DIR", "")
			home := syncHome(t)
			a := tc.makeAgent(home)
			path := filepath.Join(home, tc.dir, "settings.json")
			original := `{"defaultProvider":"own","defaultModel":"native"}`
			writeFile(t, path, original)
			backup := path + ".before-magpie"
			if err := os.Mkdir(backup, 0700); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(backup, "kept"), "earlier")
			if err := a.Apply("model", "magpie/relay/glm-4.6"); err == nil {
				t.Fatal("connect succeeded despite backup failure")
			}
			if got := readFile(path); got != original {
				t.Fatalf("settings overwritten: %s", got)
			}
			if got := readFile(filepath.Join(backup, "kept")); got != "earlier" {
				t.Fatalf("existing backup lost: %s", got)
			}
		})
	}
}

func TestPiBackupRetainedWhenRestoreReadFails(t *testing.T) {
	home := syncHome(t)
	dir := filepath.Join(home, ".pi", "agent")
	path := filepath.Join(dir, "settings.json")
	original := `{"defaultProvider":"own","defaultModel":"native"}`
	writeFile(t, path, original)
	a := pi(home)
	if err := a.Apply("model", "magpie/relay/glm-4.6"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := a.Restore(); err == nil {
		t.Fatal("restore accepted unreadable config")
	}
	if got := readFile(path + ".before-magpie"); got != original {
		t.Fatalf("backup lost: %s", got)
	}
}

func TestPiBackupRefusesDifferentExistingCopy(t *testing.T) {
	home := syncHome(t)
	path := filepath.Join(home, ".pi", "agent", "settings.json")
	original := `{"defaultProvider":"own","defaultModel":"native"}`
	writeFile(t, path, original)
	writeFile(t, path+".before-magpie", "older user's backup")
	if err := pi(home).Apply("model", "magpie/relay/glm-4.6"); err == nil {
		t.Fatal("different backup overwritten")
	}
	if got := readFile(path + ".before-magpie"); got != "older user's backup" {
		t.Fatalf("backup lost: %s", got)
	}
	if got := readFile(path); got != original {
		t.Fatalf("config changed: %s", got)
	}
}

func TestPiBackupRetryKeepsPartialCopy(t *testing.T) {
	home := syncHome(t)
	dir := filepath.Join(home, ".pi", "agent")
	settings, models := filepath.Join(dir, "settings.json"), filepath.Join(dir, "models.json")
	original := `{"defaultProvider":"own","defaultModel":"native"}`
	writeFile(t, settings, original)
	if err := os.Mkdir(models, 0700); err != nil {
		t.Fatal(err)
	}
	a := pi(home)
	if err := a.Apply("model", "magpie/relay/glm-4.6"); err == nil {
		t.Fatal("read failure accepted")
	}
	if got := readFile(settings + ".before-magpie"); got != original {
		t.Fatalf("partial backup lost: %s", got)
	}
	if err := os.Remove(models); err != nil {
		t.Fatal(err)
	}
	if err := a.Apply("model", "magpie/relay/glm-4.6"); err != nil {
		t.Fatal(err)
	}
	if got := readFile(settings + ".before-magpie"); got != original {
		t.Fatalf("retry changed backup: %s", got)
	}
}

func TestPiBackupFailurePreservesEarlierBackup(t *testing.T) {
	home := syncHome(t)
	dir := filepath.Join(home, ".pi", "agent")
	settings := filepath.Join(dir, "settings.json")
	models := filepath.Join(dir, "models.json")
	original := `{"defaultProvider":"own","defaultModel":"native"}`
	writeFile(t, settings, original)
	if err := os.MkdirAll(models, 0755); err != nil {
		t.Fatal(err)
	}
	backup := models + ".before-magpie"
	writeFile(t, backup, "earlier backup")
	a := pi(home)
	if err := a.Apply("model", "magpie/relay/glm-4.6"); err == nil {
		t.Fatal("connect succeeded when models file could not be read")
	}
	if got := readFile(backup); got != "earlier backup" {
		t.Fatalf("earlier backup lost: %q", got)
	}
	if got := readFile(settings); got != original {
		t.Fatalf("settings overwritten: %s", got)
	}
}

func TestDisconnectRetriesAppliedCleanupFailure(t *testing.T) {
	home, _ := codexHome(t, "", "model = \"own-original\"\n")
	a := codex(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	path, held := appliedPath(), appliedPath()+".held-by-test"
	injected := false
	err := edit.Filtered(func(p string, data []byte) []byte {
		if p == path && !injected {
			injected = true
			if err := os.Rename(path, held); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
		}
		return data
	}, a.Disconnect)
	if !injected || err == nil {
		t.Fatalf("applied cleanup failure not reported: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(held, path); err != nil {
		t.Fatal(err)
	}
	if err := codex(home).Disconnect(); err != nil {
		t.Fatal(err)
	}
	if _, ok := appliedLoad()[a.ID]; ok {
		t.Fatal("applied state remains after retry")
	}
	if _, err := codex(home).ConnectHow(); err != nil {
		t.Fatal(err)
	}
	if got := codex(home).Field("model").Get(); !strings.Contains(got, "fake/m1") {
		t.Fatalf("last model lost: %q", got)
	}
}

func TestDisconnectRetriesFinalStashFailure(t *testing.T) {
	for _, connectDirectly := range []bool{false, true} {
		t.Run(fmt.Sprintf("connectDirectly=%v", connectDirectly), func(t *testing.T) {
			testDisconnectRetriesFinalStashFailure(t, connectDirectly)
		})
	}
}

func testDisconnectRetriesFinalStashFailure(t *testing.T, connectDirectly bool) {
	home, _ := codexHome(t, "", "model = \"own-original\"\n")
	a := codex(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	path := stashPath()
	held := path + ".held-by-test"
	injected := false
	err := edit.Filtered(func(p string, data []byte) []byte {
		var m map[string]string
		if p == path && json.Unmarshal(data, &m) == nil && m[a.ID+".reconnect"] != "" && !injected {
			injected = true
			if err := os.Rename(path, held); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
		}
		return data
	}, a.Disconnect)
	if !injected || err == nil {
		t.Fatalf("final stash failure not reported: injected=%v err=%v", injected, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(held, path); err != nil {
		t.Fatal(err)
	}
	a = codex(home)
	if connectDirectly {
		if _, err := a.ConnectHow(); err != nil {
			t.Fatal(err)
		}
		if got := a.Field("model").Get(); !strings.Contains(got, "fake/m1") {
			t.Fatalf("last model not reconnected: %q", got)
		}
	}
	if err := a.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if _, ok := appliedLoad()[a.ID]; ok {
		t.Fatal("applied state remains after retry")
	}
	if _, err := a.ConnectHow(); err != nil {
		t.Fatal(err)
	}
	if got := a.Field("model").Get(); !strings.Contains(got, "fake/m1") {
		t.Fatalf("last model not reconnected: %q", got)
	}
}
