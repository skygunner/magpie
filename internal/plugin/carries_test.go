package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCarries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	// test 1: package.json with carries
	target := filepath.Join(home, "config", "magpie", "plugins", "node_modules", "test-plugin")
	os.MkdirAll(target, 0o755)

	os.WriteFile(filepath.Join(target, "package.json"), []byte(`{
  "name": "test-plugin",
  "magpie": {
    "carries": ["Provider A", "Provider B"]
  }
}`), 0o644)

	pluginsDir := filepath.Join(home, "config", "magpie", "plugins")
	os.MkdirAll(pluginsDir, 0o755)
	os.Symlink(target, filepath.Join(pluginsDir, "test-plugin"))

	carries := Carries("test-plugin")
	if len(carries) != 2 || carries[0] != "Provider A" || carries[1] != "Provider B" {
		t.Errorf("carries = %v, want [Provider A, Provider B]", carries)
	}
}

// TestCarriesLimits tests the 8 names limit and 48 rune limit per name.
func TestCarriesLimits(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	target := filepath.Join(home, "config", "magpie", "plugins", "node_modules", "test-plugin2")
	os.MkdirAll(target, 0o755)

	// Create 10 names, each 50 runes (should be limited to 8 names of 48 runes each)
	// Make a name with 50 runes
	longName50 := ""
	for i := 0; i < 50; i++ {
		longName50 += "a"
	}

	carries := make([]string, 10)
	for i := 0; i < 10; i++ {
		if i < 8 {
			carries[i] = "Provider " + string(rune('A'+i)) // 9 runes each
		} else {
			carries[i] = longName50 // 50 runes
		}
	}

	// Build JSON manually
	json := `{"name": "test-plugin2", "magpie": {"carries": [`
	for i, c := range carries {
		if i > 0 {
			json += ", "
		}
		json += `"` + c + `"`
	}
	json += `]}}`

	os.WriteFile(filepath.Join(target, "package.json"), []byte(json), 0o644)

	pluginsDir := filepath.Join(home, "config", "magpie", "plugins")
	os.MkdirAll(pluginsDir, 0o755)
	os.Symlink(target, filepath.Join(pluginsDir, "test-plugin2"))

	result := Carries("test-plugin2")

	// Should have at most 8 names
	if len(result) > 8 {
		t.Errorf("expected at most 8 names, got %d: %v", len(result), result)
	}

	// Each name should be at most 48 runes
	for _, name := range result {
		if len([]rune(name)) > 48 {
			t.Errorf("name %q has %d runes, want <= 48", name, len([]rune(name)))
		}
	}
}

// TestCarriesEmpty tests various edge cases.
func TestCarriesEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	// test with no magpie section
	target := filepath.Join(home, "config", "magpie", "plugins", "node_modules", "test-plugin3")
	os.MkdirAll(target, 0o755)
	os.WriteFile(filepath.Join(target, "package.json"), []byte(`{"name": "test-plugin3"}`), 0o644)

	pluginsDir := filepath.Join(home, "config", "magpie", "plugins")
	os.MkdirAll(pluginsDir, 0o755)
	os.Symlink(target, filepath.Join(pluginsDir, "test-plugin3"))

	carries := Carries("test-plugin3")
	if len(carries) != 0 {
		t.Errorf("expected empty carries, got %v", carries)
	}

	// test with empty carries array
	target2 := filepath.Join(home, "config", "magpie", "plugins", "node_modules", "test-plugin4")
	os.MkdirAll(target2, 0o755)
	os.WriteFile(filepath.Join(target2, "package.json"), []byte(`{"name": "test-plugin4", "magpie": {"carries": []}}`), 0o644)
	os.Symlink(target2, filepath.Join(pluginsDir, "test-plugin4"))

	carries2 := Carries("test-plugin4")
	if len(carries2) != 0 {
		t.Errorf("expected empty carries, got %v", carries2)
	}

	// test with non-existent plugin
	carries3 := Carries("nonexistent")
	if len(carries3) != 0 {
		t.Errorf("expected empty carries for nonexistent, got %v", carries3)
	}
}
