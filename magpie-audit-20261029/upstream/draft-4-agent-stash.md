## Symptom

When Magpie connects an agent it saves the original configuration values in `stash.json`, so Disconnect can put them back. Two failure modes lose those originals:

1. **Restore-before-write**: Claude and Codex `unroute` removed saved values from the stash *before* the corresponding config write succeeded. If the write failed (read-only file, failed rename, full disk), the originals were gone and a retry could not restore them.
2. **Malformed stash replacement**: a truncated, `null`, or JSON-array stash file was decoded as empty or partial, then written back as the active stash, dropping every other agent's saved originals.

## Root cause (base `926ef769ffd102e0a06f4cd7b0d583782d254d40`)

- `internal/agent/claude.go:1440` and `internal/agent/codex.go:1266` called `unstash` (delete + commit) before `edit.SetJSON` / `edit.SetTOMLTop`.
- `internal/agent/stash.go:69` ignored JSON unmarshal errors; `stashWrite` then replaced the active file with the incomplete map.
- `internal/agent/stash.go:27` used only a process-local mutex, so two Magpie processes sharing the config folder could overwrite each other's saved values.

## Fix

Branch `fix/wip-agent-stash` on `skygunner/magpie`, commit `9b83ba670`.

- Claude and Codex now `peekStash` every original, write config first, then `forgetIfUnchanged(key, exactValue)` only after the corresponding write succeeds. Any write error returns immediately and leaves unconsumed originals for retry; concurrent stash keys written during the failure window survive.
- Stash mutations now take a cross-process file lock for the whole read-modify-write.
- Malformed stash JSON is rejected at the read boundary and the active file is left unchanged.
- StepCode recovery uses a versioned write-ahead journal; a missing wiring record returns an actionable error instead of reporting success while still routed through Magpie.

### stash: reject malformed file instead of replacing it

```diff
@@ -48,9 +132,5 @@ func stashWrite(m map[string]string) error {
 		return err
 	}
-	// stashLoad reads a file that doesn't unmarshal into the stash (a JSON
-	// array too, not only one cut short) as empty: it is copied aside, and
-	// one that can't be copied isn't written over
-	if err := lastgood.KeepUnparsed(p, lastgood.Into[map[string]string]); err != nil {
-		log.Printf("stash.json wasn't written over: %v", err)
+	if _, err := stepcodeReadStash(); err != nil {
 		return err
 	}
@@ -62,11 +142,45 @@ func stashWrite(m map[string]string) error {
 		return err
 	}
```

### stash: cross-process lock around read-modify-write

```diff
@@ -22,11 +23,48 @@ import (
 // two changes at once, or one cut short, never leave it half there, which
 // the next change read as no stash and wrote back over every agent's. A
-// file that doesn't parse (edited by hand, or half written by an older
-// magpie) is copied aside before it is written over, so what it held can
-// still be got back.
+// file that doesn't parse is left untouched; a change reports the error.
+//
+// The gateway and a magpie run from a terminal share the config folder, so
+// the change is one at a time across them too: each takes the lock on
+// stash.json.lock while it reads, changes and writes. A change that can't
+// be made — another magpie holding the lock past the wait, or a folder
+// that can't be written — says so instead of reporting a value kept: the
+// caller goes on to write magpie's own value over the user's, and a stash
+// that wasn't written has nothing to put back.
 var stashMu sync.Mutex
 
+// During rewire, writes that would change saved originals fail closed.
+var rewireStash *bool
+var rewireWiredAt map[string]string
+
+type rewireGuard struct {
+	blocked bool
+	wiredAt map[string]string
+}
+
+// stashWait is how long a change waits for another magpie to let go of the
+// stash; a test shortens it.
+var stashWait = filelock.Wait
+
+// errStashLock says a change went unwritten because another magpie kept the
+// stash: what it saved would have been written over.
+var errStashLock = errors.New("another magpie held the stash; nothing was saved")
+
 func stashPath() string { return filepath.Join(filepath.Dir(provider.Path()), "stash.json") }
 
+// stashLock takes the stash's lock for this magpie and the one on its lock
+// file, and gives back the way to let both go. A file system that can't
+// lock says so too: two magpies would otherwise write over each other's
+// saved values, so the change fails rather than risk them.
+func stashLock() (func(), error) {
+	stashMu.Lock()
+	unlock, held := filelock.Lock(stashPath()+".lock", stashWait)
+	if !held {
+		stashMu.Unlock()
```

### Claude: peek originals, write, then forget by value

```diff
@@ -1369,24 +1402,70 @@ func claudeIn(at place) *Agent {
 	// was wired in; it answers the model Claude Code was on then, for
 	// Unwire to go back to
-	unroute := func() (string, error) {
-		if err := dropOutput(); err != nil {
+	unroute := func(g *rewireGuard) (string, error) {
+		if err := dropOutput(g); err != nil {
 			return "", err
 		}
-		if err := dropCompact(); err != nil {
+		if err := dropCompact(g); err != nil {
 			return "", err
 		}
 		// the window and capabilities stay as Claude Desktop's Code tab
 		// needs them while Desktop is on magpie (#1458)
-		if err := desktopOnly(); err != nil {
+		if err := desktopOnly(g); err != nil {
 			return "", err
 		}
-		if err := dropPicker(); err != nil {
+		if err := dropPicker(g); err != nil {
 			return "", err
 		}
-		if err := putClouds(); err != nil {
+		if err := putClouds(g); err != nil {
 			return "", err
 		}
 		if !routed() {
-			return "", nil
+			was, err := peekStash(at.key("claude.model"))
+			if err != nil {
+				return "", err
+			}
+			var back []edit.KV
+			kept := map[string]string{}
+			for _, k := range claudeOwnEnv {
+				v, err := peekStash(at.key("claude.env." + k))
+				if err != nil {
+					return "", err
+				}
+				kept[at.key("claude.env."+k)] = v
+				if v != "" && rawEnv(k) == "" {
+					back = append(back, edit.KV{Path: "env." + k, Value: v})
+				}
+			}
+			u, err := peekStash(at.key("claude.base_url"))
+			if err != nil {
+				return "", err
+			}
+			if u != "" && rawEnv("ANTHROPIC_BASE_URL") == "" {
+				back = append(back, edit.KV{Path: "env.ANTHROPIC_BASE_URL", Value: u})
+			}
+			t, err := peekStash(at.key("claude.auth_token"))
+			if err != nil {
+				return "", err
+			}
+			if t != "" && rawEnv("ANTHROPIC_AUTH_TOKEN") == "" {
+				back = append(back, edit.KV{Path: "env.ANTHROPIC_AUTH_TOKEN", Value: t})
+			}
+			if len(back) > 0 {
+				if err := edit.SetJSON(path, back...); err != nil {
+					return "", err
+				}
```

### Codex: peek originals, write, then forget by value

```diff
@@ -1241,5 +1253,45 @@ func codexIn(at place) *Agent {
 		}
 		if !routed() {
-			return "", nil
+			was, err := peekStash(at.key("codex.model"))
+			if err != nil {
+				return "", err
+			}
+			var back []edit.KV
+			p, err := peekStash(at.key("codex.provider"))
+			if err != nil {
+				return "", err
+			}
+			if p != "" && p != magpieID && get("model_provider") == "" {
+				back = append(back, edit.KV{Path: "model_provider", Value: p})
+			}
+			c, err := peekStash(at.key("codex.catalog"))
+			if err != nil {
+				return "", err
+			}
+			if c != "" && !ownCatalog(c) && !deskCatalog(c) && !theirListGone(c) && (get("model_catalog_json") == "" || deskCatalog(get("model_catalog_json"))) {
+				back = append(back, edit.KV{Path: "model_catalog_json", Value: c})
+			}
+			e, err := peekStash(at.key("codex.effort"))
+			if err != nil {
+				return "", err
+			}
+			if e != "" && get("model_reasoning_effort") == "" {
+				back = append(back, edit.KV{Path: "model_reasoning_effort", Value: e})
+			}
+			if len(back) > 0 {
+				if err := edit.SetTOMLTop(path, back...); err != nil {
+					return "", err
+				}
+			}
+			forgetIfUnchanged(at.key("codex.provider"), p)
+			forgetIfUnchanged(at.key("codex.catalog"), c)
+			forgetIfUnchanged(at.key("codex.effort"), e)
```

## Validation

- `TestRestoreFailureKeepsOriginalsForRetry` injects a config-write failure for Claude model/env/base URL/token and Codex model/provider/catalog/effort, writes a concurrent stash key, then retries: originals restored, concurrent key preserved.
- Malformed truncated, `null`, and array stash files are rejected with the active file unchanged.
- Targeted `-race` run: 469 passed. Earlier full agent `-race` baseline: 1726 passed.
- `go build -tags nogui ./...`, `go vet -tags nogui ./...`, `gofmt -l`, `git diff --check`: pass.
- Independent gate review: APPROVE.

## Limitations

- An external editor does not take the stash lock; an edit after recovery's comparison and before its write can still be overwritten. Not reproduced.
- Process crash, power loss, and hardlink in-place write durability were not verified.
- Full repository suite is not claimed green: six baseline failures in this environment, unrelated to this change.
- Real Windows/WSL behavior not tested.
- The complete diff is ~458 KB across 114 files, over GitHub's 65,536-character body limit; the excerpts above are the safety-critical paths, and the full diff is on the branch.

## Branch

https://github.com/skygunner/magpie/tree/fix/wip-agent-stash

## Delivered

- issue: https://github.com/yetone/magpie/issues/4258
- branch: https://github.com/skygunner/magpie/tree/fix/4258-agent-stash-durability
- head commit: https://github.com/skygunner/magpie/commit/9b83ba67049afc9f9cf96c339c8c46a4f29dc0bd
- comment: https://github.com/yetone/magpie/issues/4258#issuecomment-6494205053
