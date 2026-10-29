package agent

import (
	"errors"
	"testing"

	"github.com/yetone/magpie/internal/edit"
)

func TestAnteStartedDisconnectKeepsChangedModel(t *testing.T) {
	for _, model := range []string{"user-native", ""} {
		t.Run("model="+model, func(t *testing.T) { testAnteStartedDisconnectKeepsChangedModel(t, model) })
	}
}

func testAnteStartedDisconnectKeepsChangedModel(t *testing.T, model string) {
	home, _, settings := anteHome(t)
	a := ante(home)
	if err := a.Apply("model", "deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	a.Unwire = func() error { return errors.New("interrupt before unwire") }
	if err := a.Disconnect(); err == nil {
		t.Fatal("disconnect did not interrupt")
	}
	pending, err := peekStash(a.ID + ".disconnect.pending")
	if err != nil || pending == "" {
		t.Fatalf("no durable interruption marker: %v", err)
	}
	if got := anteRead(t, settings); got["provider"] != magpieID || got["model"] != "deepseek/pro" {
		t.Fatalf("interruption was not before unwire: %v", got)
	}
	if err := edit.SetJSON(settings, edit.KV{Path: "model", Value: model}); err != nil {
		t.Fatal(err)
	}
	if err := ante(home).Disconnect(); err == nil {
		t.Fatal("retry did not report the model conflict")
	}
	got := anteRead(t, settings)
	if got["model"] != model || got["provider"] != magpieID {
		t.Fatalf("retry overwrote user's selection: %v", got)
	}
}
