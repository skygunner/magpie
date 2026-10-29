package agent

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/edit"
)

func TestStepCodeStartedDisconnectKeepsChangedModel(t *testing.T) {
	for _, model := range []string{"native-model", ""} {
		t.Run("model="+model, func(t *testing.T) {
			home, _ := museHome(t)
			path := filepath.Join(home, ".stepcode", "config.toml")
			stepcodeWriteTestFile(t, path, "defaultProvider = \"native\"\ndefaultModel = \"original\"\nenabledModels = [\"native/one\"]\n")
			a := stepcode(home)
			if err := a.Apply("model", "magpie/deepseek/pro"); err != nil {
				t.Fatal(err)
			}
			a.Unwire = func() error { return errors.New("interrupt before unwire") }
			if err := a.Disconnect(); err == nil {
				t.Fatal("disconnect did not interrupt")
			}
			pending, err := peekStash(a.ID + ".disconnect.pending")
			if err != nil || pending == "" {
				t.Fatalf("missing marker: %v", err)
			}
			cfg, err := readStepcodeTOML(path)
			if err != nil || stepcodeValue(cfg.Provider) != magpieID || stepcodeValue(cfg.Model) != "deepseek/pro" {
				t.Fatalf("wrong interruption: %+v %v", cfg, err)
			}
			if model == "" {
				err = edit.DelTOMLTop(path, "defaultModel")
			} else {
				err = edit.SetTOMLTopPreserving(path, edit.KV{Path: "defaultModel", Value: model})
			}
			if err != nil {
				t.Fatal(err)
			}
			err = stepcode(home).Disconnect()
			cfg, readErr := readStepcodeTOML(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if stepcodeValue(cfg.Model) != model || (model == "" && cfg.Model != nil) {
				t.Fatalf("user model overwritten: %q, retry error %v", stepcodeValue(cfg.Model), err)
			}
		})
	}
}
