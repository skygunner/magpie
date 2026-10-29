package agent

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/edit"
)

func TestGeminiStartedDisconnectKeepsChangedModel(t *testing.T) {
	home, _ := museHome(t)
	path := filepath.Join(home, ".gemini", "settings.json")
	writeFile(t, path, `{"model":{"name":"native-original"}}`)
	a := gemini(home)
	if err := a.Apply("model", "deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	a.Unwire = func() error { return errors.New("interrupt before unwire") }
	if err := a.Disconnect(); err == nil {
		t.Fatal("disconnect did not interrupt")
	}
	if err := edit.SetJSON(path, edit.KV{Path: "model.name", Value: "native-new"}); err != nil {
		t.Fatal(err)
	}
	if err := gemini(home).Disconnect(); err == nil {
		t.Fatal("missing conflict")
	}
	if got, _ := edit.GetJSON(path, "model.name"); got != "native-new" {
		t.Fatalf("model overwritten: %q", got)
	}
}

func TestPluggedStartedDisconnectKeepsChangedSetting(t *testing.T) {
	home := pluggedHome(t, `export default {
 agent: { id: "jot", config: "$HOME/.jot/settings.json", model: "llm.model" },
 async connect({ gateway }) { return { "llm.endpoint": { url: gateway.v1, key: gateway.key }, "llm.window": 100 }; }
};`)
	path := filepath.Join(home, ".jot", "settings.json")
	writeFile(t, path, `{"llm":{"model":"native-original","window":50}}`)
	a, err := Find("jot")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Apply("model", "magpie/deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	a.Unwire = func() error { return errors.New("interrupt before unwire") }
	if err := a.Disconnect(); err == nil {
		t.Fatal("disconnect did not interrupt")
	}
	if err := edit.SetJSON(path, edit.KV{Path: "llm.window", Value: 200}); err != nil {
		t.Fatal(err)
	}
	a, err = Find("jot")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Disconnect(); err == nil {
		t.Fatal("missing conflict")
	}
	if got, _ := edit.GetJSON(path, "llm.window"); got != "200" {
		t.Fatalf("setting overwritten: %q", got)
	}
}
