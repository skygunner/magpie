package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/provider"
)

func failFinalDisconnect(t *testing.T, a *Agent) {
	t.Helper()
	path, held := stashPath(), stashPath()+".held-rereview"
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
		t.Fatalf("failure not injected: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(held, path); err != nil {
		t.Fatal(err)
	}
}

func TestPickAfterPendingDisconnect(t *testing.T) {
	home, _ := codexHome(t, "", "model = \"own-original\"\n")
	if err := provider.Save(provider.Provider{ID: "fake", Name: "Fake", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"m1", "m2"}}); err != nil {
		t.Fatal(err)
	}
	a := codex(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	failFinalDisconnect(t, a)
	a = codex(home)
	if err := a.Pick("model", "fake/m2"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ConnectHow(); err != nil {
		t.Fatal(err)
	}
	if got := a.Field("model").Get(); got != "fake/m2" {
		t.Fatalf("new pick lost: %q", got)
	}
}

func TestClaudeCheckpointFailureKeepsUserChanges(t *testing.T) {
	home, _ := codexHome(t, "", "")
	path := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, path, `{"model":"own-original"}`)
	a := claude(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	stashFile, held := stashPath(), stashPath()+".held-checkpoint"
	injected := false
	err := edit.Filtered(func(p string, data []byte) []byte {
		var m map[string]string
		var pending struct {
			Unwired bool `json:"unwired"`
		}
		if p == stashFile && json.Unmarshal(data, &m) == nil && json.Unmarshal([]byte(m[a.ID+".disconnect.pending"]), &pending) == nil && pending.Unwired && !injected {
			injected = true
			if err := os.Rename(stashFile, held); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(stashFile, 0700); err != nil {
				t.Fatal(err)
			}
		}
		return data
	}, a.Disconnect)
	if !injected || err == nil {
		t.Fatalf("checkpoint failure not injected: %v", err)
	}
	if err := os.Remove(stashFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(held, stashFile); err != nil {
		t.Fatal(err)
	}
	if err := edit.SetJSON(path, edit.KV{Path: "model", Value: "user-after-crash"}); err != nil {
		t.Fatal(err)
	}
	if err := claude(home).Disconnect(); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "user-after-crash" {
		t.Fatalf("user change lost: %q", got)
	}
}

func TestClaudeAfterUnwireKeepsUserModel(t *testing.T) {
	home, _ := codexHome(t, "", "")
	path := filepath.Join(home, ".claude", "settings.json")
	original := `{"model":"own-original","env":{"ANTHROPIC_BASE_URL":"https://original.example","ANTHROPIC_AUTH_TOKEN":"original-token"}}`
	writeFile(t, path, original)
	a := claude(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	stashFile, held := stashPath(), stashPath()+".held-after-unwire"
	injected := false
	err := edit.Filtered(func(p string, data []byte) []byte {
		var m map[string]string
		var pending struct {
			Unwired bool `json:"unwired"`
		}
		if p == stashFile && json.Unmarshal(data, &m) == nil && json.Unmarshal([]byte(m[a.ID+".disconnect.pending"]), &pending) == nil && pending.Unwired && !injected {
			injected = true
			if err := os.Rename(stashFile, held); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(stashFile, 0700); err != nil {
				t.Fatal(err)
			}
		}
		return data
	}, a.Disconnect)
	if !injected || err == nil {
		t.Fatalf("post-unwire failure not injected: %v", err)
	}
	if err := os.Remove(stashFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(held, stashFile); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); got != "https://original.example" {
		t.Fatalf("interruption was not after unwire: %q", got)
	}
	if err := provider.Save(provider.Provider{ID: "fake", Name: "Fake", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"m1", "m2"}}); err != nil {
		t.Fatal(err)
	}
	if err := edit.SetJSON(path, edit.KV{Path: "model", Value: "fake/m2"}); err != nil {
		t.Fatal(err)
	}
	if err := claude(home).Disconnect(); err == nil {
		t.Fatal("retry overwrote or ignored the user model")
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); got != "https://original.example" {
		t.Fatalf("endpoint lost: %q", got)
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_AUTH_TOKEN"); got != "original-token" {
		t.Fatalf("token lost: %q", got)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "fake/m2" {
		t.Fatalf("user model lost: %q", got)
	}
	if _, err := claude(home).ConnectHow(); err == nil {
		t.Fatal("connect resolved the conflict by changing the user model")
	}
	if got, _ := edit.GetJSON(path, "model"); got != "fake/m2" {
		t.Fatalf("selection lost: %q", got)
	}
}

func TestWorkBuddyStartedRetryDoesNotPanic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	if err := provider.Save(provider.Provider{ID: "fake", Name: "Fake", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"m1"}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".workbuddy", "models.json")
	writeFile(t, path, `[]`)
	a := workbuddy(home)
	if err := a.Apply("provider", "magpie"); err != nil {
		t.Fatal(err)
	}
	a.Unwire = func() error { return errors.New("interrupted") }
	if err := a.Disconnect(); err == nil {
		t.Fatal("disconnect did not stop at unwire")
	}
	if err := workbuddy(home).Disconnect(); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeProxyStartedKeepsNativeModel(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	fakeMagpieAt(t, map[string]bool{corpProxy: true})
	home, _ := codexHome(t, "", "")
	path := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, path, `{"model":"own-original","env":{"ANTHROPIC_BASE_URL":"https://original.example","ANTHROPIC_AUTH_TOKEN":"original-token"}}`)
	a := claude(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	if err := edit.SetJSON(path, edit.KV{Path: "env.ANTHROPIC_BASE_URL", Value: "https://ai.corp.example/magpie"}); err != nil {
		t.Fatal(err)
	}
	if !surelyMagpie(corpProxy, here(home).gw()) {
		t.Fatal("proxy was not recognized as routed")
	}
	a.Unwire = func() error { return errors.New("interrupted") }
	if err := a.Disconnect(); err == nil {
		t.Fatal("disconnect did not stop at unwire")
	}
	if err := edit.SetJSON(path, edit.KV{Path: "model", Value: "user-native"}); err != nil {
		t.Fatal(err)
	}
	if err := claude(home).Disconnect(); err == nil {
		got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL")
		t.Fatalf("proxy retry changed native model at %q", got)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "user-native" {
		t.Fatalf("native model lost: %q", got)
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); got != "https://ai.corp.example/magpie" {
		t.Fatalf("proxy lost: %q", got)
	}
	if err := edit.DelJSON(path, "model"); err != nil {
		t.Fatal(err)
	}
	if err := claude(home).Disconnect(); err == nil {
		t.Fatal("retry ignored the removed model")
	}
	if got, _ := edit.GetJSON(path, "model"); got != "" {
		t.Fatalf("removed model restored: %q", got)
	}
}

func TestClaudeStartedNativeModelStays(t *testing.T) {
	home, _ := codexHome(t, "", "")
	path := filepath.Join(home, ".claude", "settings.json")
	original := `{"model":"own-original","env":{"ANTHROPIC_BASE_URL":"https://original.example","ANTHROPIC_AUTH_TOKEN":"original-token"}}`
	writeFile(t, path, original)
	a := claude(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	stashFile, held := stashPath(), stashPath()+".held-before-unwire"
	seen, injected := false, false
	err := edit.Filtered(func(p string, data []byte) []byte {
		var m map[string]string
		var pending struct {
			Started bool   `json:"started"`
			Model   string `json:"model"`
		}
		if p == stashFile && json.Unmarshal(data, &m) == nil && json.Unmarshal([]byte(m[a.ID+".disconnect.pending"]), &pending) == nil && pending.Started && pending.Model != "" {
			if seen && !injected {
				injected = true
				if err := os.Rename(stashFile, held); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(stashFile, 0700); err != nil {
					t.Fatal(err)
				}
			}
			seen = true
		}
		return data
	}, a.Disconnect)
	if !injected || err == nil {
		t.Fatalf("started boundary not injected: %v", err)
	}
	if err := os.Remove(stashFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(held, stashFile); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); !strings.Contains(got, "127.0.0.1") {
		t.Fatalf("interruption was not before unwire: %q", got)
	}
	if err := edit.SetJSON(path, edit.KV{Path: "model", Value: "user-native"}); err != nil {
		t.Fatal(err)
	}
	if err := claude(home).Disconnect(); err == nil {
		got, _ := edit.GetJSON(path, "model")
		t.Fatalf("retry changed the native model to %q", got)
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); !strings.Contains(got, "127.0.0.1") {
		t.Fatalf("endpoint changed: %q", got)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "user-native" {
		t.Fatalf("native model lost: %q", got)
	}
	if _, err := claude(home).ConnectHow(); err == nil {
		t.Fatal("connect resolved the conflict")
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); !strings.Contains(got, "127.0.0.1") {
		t.Fatalf("endpoint changed by connect: %q", got)
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_AUTH_TOKEN"); got == "" {
		t.Fatal("token lost")
	}
	if got, _ := edit.GetJSON(path, "model"); got != "user-native" {
		t.Fatalf("selection lost: %q", got)
	}
}

func TestClaudeStartedFailureKeepsUserModel(t *testing.T) {
	home, _ := codexHome(t, "", "")
	path := filepath.Join(home, ".claude", "settings.json")
	original := `{"model":"own-original","env":{"ANTHROPIC_BASE_URL":"https://original.example","ANTHROPIC_AUTH_TOKEN":"original-token"}}`
	writeFile(t, path, original)
	a := claude(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	stashFile, held := stashPath(), stashPath()+".held-started"
	injected := false
	err := edit.Filtered(func(p string, data []byte) []byte {
		var m map[string]string
		var pending struct {
			Started bool `json:"started"`
		}
		if p == stashFile && json.Unmarshal(data, &m) == nil && json.Unmarshal([]byte(m[a.ID+".disconnect.pending"]), &pending) == nil && pending.Started && !injected {
			injected = true
			if err := os.Rename(stashFile, held); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(stashFile, 0700); err != nil {
				t.Fatal(err)
			}
		}
		return data
	}, a.Disconnect)
	if !injected || err == nil {
		t.Fatalf("started failure not injected: %v", err)
	}
	if err := os.Remove(stashFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(held, stashFile); err != nil {
		t.Fatal(err)
	}
	if err := edit.SetJSON(path, edit.KV{Path: "model", Value: "fake/m1"}); err != nil {
		t.Fatal(err)
	}
	if err := claude(home).Disconnect(); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); got != "https://original.example" {
		t.Fatalf("endpoint lost: %q", got)
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_AUTH_TOKEN"); got != "original-token" {
		t.Fatalf("token lost: %q", got)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "own-original" {
		t.Fatalf("original model lost: %q", got)
	}
	if _, err := claude(home).ConnectHow(); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "fake/m1" {
		t.Fatalf("selection lost: %q", got)
	}
}

func TestClaudeMarkerBeforeUnwireRestoresOriginal(t *testing.T) {
	home, _ := codexHome(t, "", "")
	if err := provider.Save(provider.Provider{ID: "fake", Name: "Fake", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"m1", "m2"}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".claude", "settings.json")
	original := `{"model":"own-original","env":{"ANTHROPIC_BASE_URL":"https://original.example","ANTHROPIC_AUTH_TOKEN":"original-token"}}`
	writeFile(t, path, original)
	a := claude(home)
	if err := a.Apply("model", "fake/m2"); err != nil {
		t.Fatal(err)
	}
	stashFile, held := stashPath(), stashPath()+".held-marker"
	injected := false
	err := edit.Filtered(func(p string, data []byte) []byte {
		var m map[string]string
		var pending struct {
			Unwired bool `json:"unwired"`
		}
		if p == stashFile && json.Unmarshal(data, &m) == nil && json.Unmarshal([]byte(m[a.ID+".disconnect.pending"]), &pending) == nil && pending.Unwired && !injected {
			injected = true
			if err := os.Rename(stashFile, held); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(stashFile, 0700); err != nil {
				t.Fatal(err)
			}
		}
		return data
	}, a.Disconnect)
	if !injected || err == nil {
		t.Fatalf("marker failure not injected: %v", err)
	}
	if err := os.Remove(stashFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(held, stashFile); err != nil {
		t.Fatal(err)
	}
	if err := claude(home).Disconnect(); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); got != "https://original.example" {
		t.Fatalf("endpoint lost: %q", got)
	}
	if got, _ := edit.GetJSON(path, "env.ANTHROPIC_AUTH_TOKEN"); got != "original-token" {
		t.Fatalf("token lost: %q", got)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "own-original" {
		t.Fatalf("model lost: %q", got)
	}
	if _, err := claude(home).ConnectHow(); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "fake/m2" {
		t.Fatalf("prior selection lost: %q", got)
	}
}

func TestClaudeStartMarkerCrashRestoresPriorModel(t *testing.T) {
	home, _ := codexHome(t, "", "")
	path := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, path, `{"model":"own-original"}`)
	a := claude(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	if err := stash(map[string]string{a.ID + ".disconnect.pending": `{"unwired":true}`}); err != nil {
		t.Fatal(err)
	}
	if err := edit.SetJSON(path, edit.KV{Path: "model", Value: "user-after-crash"}); err != nil {
		t.Fatal(err)
	}
	if err := claude(home).Disconnect(); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "user-after-crash" {
		t.Fatalf("user change lost: %q", got)
	}
	if _, err := claude(home).ConnectHow(); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "fake/m1" {
		t.Fatalf("prior model lost: %q", got)
	}
}

func TestClaudePendingDisconnectKeepsUserChanges(t *testing.T) {
	home, _ := codexHome(t, "", "")
	path := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, path, `{"model":"own-original"}`)
	a := claude(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	failFinalDisconnect(t, a)
	if err := edit.SetJSON(path, edit.KV{Path: "model", Value: "fake/m1"}); err != nil {
		t.Fatal(err)
	}
	if err := claude(home).Disconnect(); err != nil {
		t.Fatal(err)
	}
	if got, _ := edit.GetJSON(path, "model"); got != "own-original" {
		t.Fatalf("original model lost: %q", got)
	}
}
