package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/edit"
)

func TestStepCodeDisconnectWithoutWiringState(t *testing.T) {
	home, _ := museHome(t)
	path := filepath.Join(home, ".stepcode", "config.toml")
	models := filepath.Join(home, ".stepcode", "models.json")
	stepcodeWriteTestFile(t, path, "defaultProvider = \"magpie\"\ndefaultModel = \"deepseek/pro\"\nuserOption = true\n")
	stepcodeWriteTestFile(t, models, `{"providers":{"magpie":{"name":"user-provider"},"native":{"name":"keep"}}}`)
	a := stepcode(home)
	if !a.Wired() {
		t.Fatal("fixture must be wired")
	}
	beforeConfig, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	beforeModels, err := os.ReadFile(models)
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		err := a.Disconnect()
		if err == nil || !strings.Contains(err.Error(), "back up") || !strings.Contains(err.Error(), "stepcode.disconnect.pending") {
			t.Fatalf("attempt %d: expected actionable error, got %v (wired=%v)", attempt, err, a.Wired())
		}
		if !a.Wired() {
			t.Fatal("failed disconnect changed wiring")
		}
		config, err := os.ReadFile(path)
		if err != nil || string(config) != string(beforeConfig) {
			t.Fatalf("config changed: %v", err)
		}
		currentModels, err := os.ReadFile(models)
		if err != nil || string(currentModels) != string(beforeModels) {
			t.Fatalf("models changed: %v", err)
		}
	}
	cfg, err := readStepcodeTOML(path)
	if err != nil {
		t.Fatal(err)
	}
	if stepcodeValue(cfg.Model) != "deepseek/pro" || stepcodeValue(cfg.Provider) != "magpie" {
		t.Fatalf("user configuration changed: %+v", cfg)
	}
	got, ok := edit.GetJSON(models, "providers.magpie.name")
	if !ok || got != "user-provider" {
		t.Fatal("user provider changed")
	}
	if pending, err := peekStash("stepcode.disconnect.pending"); err != nil || pending == "" {
		t.Fatalf("pending: %q, %v", pending, err)
	}
}

func TestStepCodeLegacyNoWiringStartedRecovery(t *testing.T) {
	home, _ := museHome(t)
	path := filepath.Join(home, ".stepcode", "config.toml")
	models := filepath.Join(home, ".stepcode", "models.json")
	stepcodeWriteTestFile(t, path, "defaultProvider = \"magpie\"\ndefaultModel = \"deepseek/pro\"\n")
	stepcodeWriteTestFile(t, models, `{"providers":{"magpie":{"name":"user-provider"}}}`)
	snapshot, err := stepcodePlan(here(home), path, models)
	if err != nil {
		t.Fatal(err)
	}
	marker, err := json.Marshal(map[string]any{"started": true, "values": map[string]string{"model": "magpie/deepseek/pro"}, "snapshot": snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if err := stash(map[string]string{"stepcode.disconnect.pending": string(marker)}); err != nil {
		t.Fatal(err)
	}
	a := stepcode(home)
	if !a.Wired() {
		t.Fatal("fixture must be wired")
	}
	if err := a.Disconnect(); err == nil || !strings.Contains(err.Error(), "stepcode.disconnect.pending") {
		t.Fatalf("missing recovery error: %v", err)
	}
	if !a.Wired() {
		t.Fatal("failed recovery changed wiring")
	}
	cfg, err := readStepcodeTOML(path)
	if err != nil || stepcodeValue(cfg.Provider) != "magpie" || stepcodeValue(cfg.Model) != "deepseek/pro" {
		t.Fatalf("configuration changed: %+v %v", cfg, err)
	}
	if got, ok := edit.GetJSON(models, "providers.magpie.name"); !ok || got != "user-provider" {
		t.Fatal("provider changed")
	}
	if got, err := peekStash("stepcode.disconnect.pending"); err != nil || got != string(marker) {
		t.Fatalf("recovery marker changed: %q %v", got, err)
	}
}

func TestStepCodeRecoveryEditDuringLockWait(t *testing.T) {
	home, _ := museHome(t)
	path := filepath.Join(home, ".stepcode", "config.toml")
	models := filepath.Join(home, ".stepcode", "models.json")
	stepcodeWriteTestFile(t, path, "defaultProvider = \"native\"\ndefaultModel = \"original\"\n")
	a := stepcode(home)
	if err := a.Apply("model", "magpie/deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	snapshot, err := stepcodePlan(here(home), path, models)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(map[string]any{"started": true, "snapshot": snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if err := stash(map[string]string{"stepcode.disconnect.pending": string(b)}); err != nil {
		t.Fatal(err)
	}
	reached := make(chan struct{})
	old := stepcodeRecoveryLock
	stepcodeRecoveryLock = func() (func(), error) { close(reached); return stashLock() }
	t.Cleanup(func() { stepcodeRecoveryLock = old })
	unlock, err := stashLock()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- stepcodeRecover(here(home), path, models, &snapshot) }()
	select {
	case <-reached:
	case <-time.After(10 * time.Second):
		unlock()
		t.Fatal("recovery did not reach lock")
	}
	if err := edit.SetTOMLTopPreserving(path, edit.KV{Path: "defaultModel", Value: "user-new"}); err != nil {
		unlock()
		t.Fatal(err)
	}
	unlock()
	select {
	case err := <-done:
		if err == nil {
			t.Error("expected conflict")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("recovery did not finish")
	}
	cfg, err := readStepcodeTOML(path)
	if err != nil || stepcodeValue(cfg.Model) != "user-new" {
		t.Fatalf("user model overwritten: %+v %v", cfg, err)
	}
}
