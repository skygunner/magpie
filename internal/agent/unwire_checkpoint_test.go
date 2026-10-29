package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/edit"
)

func failUnwiredCheckpoint(t *testing.T, a *Agent) {
	t.Helper()
	path, held := stashPath(), stashPath()+".checkpoint-held"
	injected := false
	err := edit.Filtered(func(p string, data []byte) []byte {
		var m map[string]string
		var pending struct {
			Unwired bool `json:"unwired"`
		}
		if p == path && json.Unmarshal(data, &m) == nil && json.Unmarshal([]byte(m[a.ID+".disconnect.pending"]), &pending) == nil && pending.Unwired && !injected {
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
		t.Fatalf("checkpoint failure not injected: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(held, path); err != nil {
		t.Fatal(err)
	}
}

func TestStepCodeLegacyStartedMarkerNeedsManualRecovery(t *testing.T) {
	home, _ := museHome(t)
	path := filepath.Join(home, ".stepcode", "config.toml")
	stepcodeWriteTestFile(t, path, "defaultProvider = \"native\"\ndefaultModel = \"original\"\n")
	a := stepcode(home)
	if err := a.Apply("model", "magpie/deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	a.Unwire = func() error { return errors.New("interrupt") }
	if err := a.Disconnect(); err == nil {
		t.Fatal("missing interruption")
	}
	key := a.ID + ".disconnect.pending"
	pending, err := peekStash(key)
	if err != nil {
		t.Fatal(err)
	}
	var marker map[string]json.RawMessage
	if err := json.Unmarshal([]byte(pending), &marker); err != nil {
		t.Fatal(err)
	}
	delete(marker, "snapshot")
	b, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := stash(map[string]string{key: string(b)}); err != nil {
		t.Fatal(err)
	}
	if err := edit.SetTOMLTopPreserving(path, edit.KV{Path: "defaultModel", Value: "user-native"}); err != nil {
		t.Fatal(err)
	}
	err = stepcode(home).Disconnect()
	if err == nil || !strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), "Back up") {
		t.Fatalf("missing recovery instructions: %v", err)
	}
	cfg, err := readStepcodeTOML(path)
	if err != nil {
		t.Fatal(err)
	}
	if stepcodeValue(cfg.Model) != "user-native" {
		t.Fatal("user model overwritten")
	}
}

func TestStepCodeCheckpointRetry(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy=%v", legacy), func(t *testing.T) { stepCodeCheckpointRetry(t, legacy) })
	}
}

func stepCodeCheckpointRetry(t *testing.T, legacy bool) {
	home, _ := museHome(t)
	path := filepath.Join(home, ".stepcode", "config.toml")
	stepcodeWriteTestFile(t, path, "defaultProvider = \"native\"\ndefaultModel = \"original\"\n")
	a := stepcode(home)
	if err := a.Apply("model", "magpie/deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	failUnwiredCheckpoint(t, a)
	if legacy {
		key := a.ID + ".disconnect.pending"
		pending, err := peekStash(key)
		if err != nil {
			t.Fatal(err)
		}
		var marker map[string]json.RawMessage
		if err := json.Unmarshal([]byte(pending), &marker); err != nil {
			t.Fatal(err)
		}
		delete(marker, "snapshot")
		b, err := json.Marshal(marker)
		if err != nil {
			t.Fatal(err)
		}
		if err := stash(map[string]string{key: string(b)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := stepcode(home).Disconnect(); err != nil {
		t.Fatal(err)
	}
	cfg, err := readStepcodeTOML(path)
	if err != nil {
		t.Fatal(err)
	}
	if stepcodeValue(cfg.Provider) != "native" || stepcodeValue(cfg.Model) != "original" {
		t.Fatal("original settings not restored")
	}
}

func TestGeminiCheckpointRetry(t *testing.T) {
	home, _ := museHome(t)
	path := filepath.Join(home, ".gemini", "settings.json")
	writeFile(t, path, `{"model":{"name":"native-original"}}`)
	a := gemini(home)
	if err := a.Apply("model", "deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	failUnwiredCheckpoint(t, a)
	if err := gemini(home).Disconnect(); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "model.name"); got != "native-original" {
		t.Fatalf("model: %q", got)
	}
}

func TestPluginCheckpointRetry(t *testing.T) {
	home := pluggedHome(t, `export default { agent: { id: "jot", config: "$HOME/.jot/settings.json", model: "llm.model" }, async connect({ gateway }) { return { "llm.endpoint": { url: gateway.v1, key: gateway.key }, "llm.window": 100 }; } };`)
	path := filepath.Join(home, ".jot", "settings.json")
	writeFile(t, path, `{"llm":{"model":"native-original","window":50}}`)
	a, err := Find("jot")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Apply("model", "magpie/deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	failUnwiredCheckpoint(t, a)
	a, err = Find("jot")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "llm.model"); got != "native-original" {
		t.Fatalf("model: %q", got)
	}
	if got, _ := edit.GetJSON(path, "llm.window"); got != "50" {
		t.Fatalf("window: %q", got)
	}
}
