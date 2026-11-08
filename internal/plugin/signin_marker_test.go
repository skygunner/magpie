package plugin

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// #5264: a marker's name must not appear while its contents are still being
// written, or the cancellation test can read an empty error name.
func TestSignInMarkerAppearsComplete(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fixture, err := filepath.Abs("testdata/signinend/index.js")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, node, "--input-type=module", "-e", signInMarkerJS, fixture, t.TempDir())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("marker publication: %v\n%s", err, out)
	}
}

const signInMarkerJS = `
import assert from 'node:assert/strict'
import fs from 'node:fs'
import { syncBuiltinESMExports } from 'node:module'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'

const [fixture, home] = process.argv.slice(1)
const target = join(home, 'fetch-aborted')
process.env.HOME = home
process.env.FAKE_BASE = 'https://example.invalid'
globalThis.fetch = async () => { throw new DOMException('cancelled', 'AbortError') }
const original = fs.writeFileSync
fs.writeFileSync = (path, data, ...options) => {
  const fd = fs.openSync(path, 'w')
  try {
    assert.equal(fs.existsSync(target), false, 'marker appeared before its contents were written; actual contents: ' + JSON.stringify(fs.existsSync(target) ? fs.readFileSync(target, 'utf8') : null))
    original(fd, data, ...options)
  } finally {
    fs.closeSync(fd)
  }
}
syncBuiltinESMExports()
const { SignInEndPlugin } = await import(pathToFileURL(fixture))
const plugin = await SignInEndPlugin()
const authorization = await plugin.auth.methods[7].authorize()
await authorization.callback()
fs.writeFileSync = original
syncBuiltinESMExports()
assert.equal(fs.readFileSync(target, 'utf8'), 'AbortError')
`
