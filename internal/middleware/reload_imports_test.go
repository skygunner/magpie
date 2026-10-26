package middleware

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/plugin"
)

// A middleware's module can import files beside it (jsmod.Compile reads
// them), but only the entry's own time and size were stamped, so an edit
// to an imported file never reloaded the middleware: the old code kept
// running until the entry or plugins.json changed (#3723). The files the
// module imports are stamped too, so rewriting one reloads.
func TestReloadOnImportedChange(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"names","magpie":{"middleware":"./index.js"}}`), 0o644)
	os.WriteFile(filepath.Join(dir, "names.js"), []byte(`export const names = {"m1": "one"};`), 0o644)
	os.WriteFile(filepath.Join(dir, "index.js"), []byte(`
import { names } from "./names.js";
export function onRequest(b) { b.v = names[b.model]; return b; }`), 0o644)
	entry := plugin.Entry{Spec: dir}
	if f, _ := plugin.Middleware(dir); f == "" {
		t.Fatal("no middleware in the package")
	}
	use(t, entry)
	run := func() string {
		r := Begin(info())
		defer r.End()
		b, _ := r.Request([]byte(`{"model":"m1"}`))
		return string(b)
	}
	if got := run(); got != `{"model":"m1","v":"one"}` {
		t.Fatal(got)
	}
	// only the imported file changes: a later time, the same size
	imported := filepath.Join(dir, "names.js")
	os.WriteFile(imported, []byte(`export const names = {"m1": "two"};`), 0o644)
	later := time.Now().Add(time.Second)
	os.Chtimes(imported, later, later)
	nextCheck.Store(0)
	if got := run(); got != `{"model":"m1","v":"two"}` {
		t.Fatal("an imported change didn't reload:", got)
	}
}
