package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
	"gopkg.in/yaml.v3"
)

func TestHermesWritesModelVisionAndKeepsOverrides(t *testing.T) {
	home := t.TempDir()
	for k, v := range map[string]string{"HOME": home, "USERPROFILE": home, "HERMES_HOME": "", "XDG_CONFIG_HOME": filepath.Join(home, ".config"), "XDG_CACHE_HOME": filepath.Join(home, ".cache")} {
		t.Setenv(k, v)
	}
	if err := provider.Save(provider.Provider{ID: "probe", Name: "Probe", Chat: "https://example.invalid/v1", Key: "fixture", Models: []string{"vision", "text", "auto"}}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveLive("probe", "https://example.invalid/v1", []catalog.Model{{ID: "vision", Images: true, ImageInput: imageInputBool(true)}, {ID: "text", ImageInput: imageInputBool(false)}, {ID: "auto", ImageInput: imageInputBool(false)}}); err != nil {
		t.Fatal(err)
	}
	visionOff(t)
	dir := appdir.HermesHomeIn(home)
	path := filepath.Join(dir, "config.yaml")
	bot := filepath.Join(dir, "profiles", "bot", "config.yaml")
	for _, p := range []string{path, bot} {
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("model:\n  provider: own\n  default: mine\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	a := hermes(home)
	if err := a.Field("model").Set("magpie/probe/vision"); err != nil {
		t.Fatal(err)
	}
	read := func(p string) map[string]any {
		t.Helper()
		var doc map[string]any
		if err := yaml.Unmarshal([]byte(readFile(p)), &doc); err != nil {
			t.Fatal(err)
		}
		ps, _ := doc["providers"].(map[string]any)
		mp, _ := ps["magpie"].(map[string]any)
		ms, ok := mp["models"].(map[string]any)
		if !ok {
			t.Fatalf("Hermes requires model capability mapping, got %T: %s", mp["models"], readFile(p))
		}
		return ms
	}
	for _, p := range []string{path, bot} {
		ms := read(p)
		entry, _ := ms["probe/vision"].(map[string]any)
		if entry["supports_vision"] != true {
			t.Fatalf("vision not advertised in %s: %#v", p, entry)
		}
		if ms["probe/text"].(map[string]any)["supports_vision"] != false {
			t.Fatal("text model advertised as vision")
		}
		raw := readFile(p)
		var doc map[string]any
		if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
			t.Fatal(err)
		}
		entry["supports_vision"] = false
		entry["context_length"] = 12345
		text, _ := ms["probe/text"].(map[string]any)
		delete(text, "supports_vision")
		text["vision"] = false
		ms["gone/model"] = map[string]any{"supports_vision": true}
		doc["providers"].(map[string]any)["magpie"].(map[string]any)["models"] = ms
		b, err := yaml.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := settings.Load()
	s.Vision = "probe/vision"
	if err := settings.Save(s); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if err := a.Sync(); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{path, bot} {
		ms := read(p)
		if ms["probe/auto"].(map[string]any)["supports_vision"] != true {
			t.Fatal("generated capability did not follow gateway image description")
		}
		entry := ms["probe/vision"].(map[string]any)
		text := ms["probe/text"].(map[string]any)
		if entry["supports_vision"] != false || entry["context_length"] != 12345 || text["vision"] != false || text["supports_vision"] != nil || ms["gone/model"] != nil {
			t.Fatalf("user model settings or catalog membership changed in %s: %#v", p, ms)
		}
	}
	if err := a.Unwire(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{path, bot} {
		var doc map[string]any
		if err := yaml.Unmarshal([]byte(readFile(p)), &doc); err != nil {
			t.Fatal(err)
		}
		ps, _ := doc["providers"].(map[string]any)
		if ps["magpie"] != nil || stashLoad()["hermes:"+p+":vision"] != "" {
			t.Fatalf("disconnect left provider or generated capability state at %s", p)
		}
	}

}
