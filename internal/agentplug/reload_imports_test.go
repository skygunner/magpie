package agentplug

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/settings"
)

// use makes plugins.json list one agent plugin in a folder of its own,
// with files beside its entry, and loads it afresh.
func use(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(settings.Dir(), 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(plugin.List{Plugins: []plugin.Entry{{Spec: dir}}})
	if err := os.WriteFile(filepath.Join(settings.Dir(), "plugins.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	Reload()
	t.Cleanup(func() {
		os.Remove(filepath.Join(settings.Dir(), "plugins.json"))
		Reload()
	})
}

// An agent module can import files beside it (jsmod.Compile reads them),
// but only the entry's own time and size were stamped, so an edit to an
// imported file never reloaded the agent: the old name kept showing until
// the entry or plugins.json changed (#3723). The files the module imports
// are stamped too, so rewriting one reloads.
func TestReloadOnImportedChange(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"t","magpie":{"agent":"./agent.js"}}`), 0o644)
	os.WriteFile(filepath.Join(dir, "name.js"), []byte(`export const name = "Before";`), 0o644)
	os.WriteFile(filepath.Join(dir, "agent.js"), []byte(`
import { name } from "./name.js";
export const agent = { id: "t", name, bin: "t", config: "~/.t.yml", model: "model" };
export function connect() { return { k: "v" }; }`), 0o644)
	use(t, dir)

	list, errs := List()
	if len(list) != 1 || list[0].Desc.Name != "Before" {
		t.Fatalf("listed %+v %v", list, errs)
	}
	// only the imported file changes: a later time, the same size
	imported := filepath.Join(dir, "name.js")
	os.WriteFile(imported, []byte(`export const name = "After";`), 0o644)
	later := time.Now().Add(time.Second)
	os.Chtimes(imported, later, later)
	nextCheck.Store(0)
	list, errs = List()
	if len(list) != 1 || list[0].Desc.Name != "After" {
		t.Fatalf("an imported change didn't reload: %+v %v", list, errs)
	}
}
