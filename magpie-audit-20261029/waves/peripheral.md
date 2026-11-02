# Peripheral packages audit (v0.1.957 snapshot)

Scope: internal/{tui,cli,appdir,settings,catalog,omarchy,imagemcp,redact,pii,maskrec,codexcat,shortcut,tune,source,lastgood,upstream,claudecode,mcpauth}.
Tree: /tmp/magpie-audit-Pzn4fNp7/src (read-only).

## Coverage
- internal/appdir/appdir.go 1-278 — Portable, Resolve, Passed, shared, hasFiles, Beside, within, Config, Redirected, Cache, WebView, SystemCache.
- internal/appdir/env.go 1-126 — homeVar, rooted, Home, mustHome, pathVars, usable, LookupEnv, Getenv, CheckEnv.
- internal/appdir/codex.go 1-112 — CodexHome(In), CodexHomes(In), codexHomeAt, sameDir.
- internal/appdir/claude.go 1-29 — ClaudeDir(In).
- internal/appdir/hermes.go 1-46 — HermesHome(In), HermesDefaultIn, hermesDefault.
- internal/appdir/zcode.go 1-63 — ZCodeHome(On), ZCodeDataIn.
- internal/settings/settings.go 1180-1400 — Path/Dir, lockFile, Load, LoadChecked, Shared, sharedFor, load(Checked), parse, validSettings; 1400-1600 — GatewayAddr/SavedAddr, Save, SetGatewayConversations, EditCheckinOff, EditUsageAliases, Set/UseRedactModel*, save validation head; 1600-1770 — save tail (validation, consent re-read, lastgood.Keep, marshal, chmod, WriteAtomic path).
- internal/settings/accounts.go 1-42 — AccountDiscoveryOff(In), noAccountsEnv.
- internal/settings/allowips.go 1-64 — SplitAllowIPs, ParseAllowIP, CleanAllowIPs.
- internal/settings/clone.go 1-109 — clone, deepCopy, plain.
- internal/settings/masking.go 1-60 — redactActions, redactAllow.
- internal/settings/migrate.go 1-70 — Migrate, copyTree.
- internal/settings/origins.go 1-88 — CleanOrigins, cleanOrigin, extensionIDs, extensionOrigin.
- internal/settings/otel.go 1-58 — OTel.Check, header/value validation.
- internal/catalog/catalog.go 200-360 — Price, Tier, UnmarshalJSON, OneHourFor, Times, Same, At, OneHour, Cost, CostSplit; 500-790 — loadGen, Source, Stale, Sync/SyncIfMissing, syncLock; 790-1000 — PriceOf, PricedBy, priceIndexOf, canonicallyPriced, APIOf, Provider; 1290-1345 — knownModel, unmarked, ListedBy.
- internal/catalog/fresh.go 1-75 — KeepFresh, Missing, fetch.
- internal/catalog/labs.go 60-125 — syncLabs (temp+rename), labsMissing.
- internal/omarchy/omarchy.go 1-360 — Detect, ThemeDir, Parse, parseColor, color.over/css, Vars, themeFiles, deskFiles, stampOf, Current, fontName, hyprInt; 386-405 re-read.
- internal/omarchy/hypr.go 1-206 — Hyprland, hyprMonitor.size, PanelAt/panelAt, PlacePanel/panelRule, ReportClicks/StopClicks, WatchClicks, ClickedOutside, outside, regexpQuote, hyprJSON, hyprctl.
- internal/omarchy/widget.go 1-120 — WidgetOn, AddWidget, KeepWidget, widget, RemoveWidget, run.
- internal/maskrec/maskrec.go 1-511 — Hold, settle, Clear, merged, files, prune, read, at, withSnips, Restored, SentOf, Filter.takes, List, Get, OfRoute.
- internal/claudecode/claudecode.go 110-297 — Platform, Release, Latest/latest, get, Install, download, progressWriter, Remove.
- internal/source/source.go 1-335 — chinaFirst, ChinaFirst, modelMirror, Enabled, npmBase/githubBase, URLs, faithful, distinct, Do, bare, mirrorable, DoOfficial, DoFaithful, do, candidateContext, retryable.
- internal/lastgood/lastgood.go 1-276 — JSON, Bak, Read, Fallback, describe, Keep, Into, KeepUnparsed, KeepBad, keptAlready, globEscape, note/Noted/Recovered/Reset.
- internal/imagemcp/imagemcp.go 1-700 — schemas, server.start, await, stillGoing, ToolTimeout, callLongest, Run, send, serve, handle, generate, project, root, filePath, reference, Target/target, folderFor, extOf.
- internal/imagemcp/wsl.go 1-120 — options, wslSide.local, wslSide.shown, shownPath.
- internal/pii/fetch.go 120-290 — ensureServe, upgrade, removeOld, installServe, unzipTo, fetch, get.
- internal/pii/cache.go 40-140 — loadCache, saveSoon, saveCache, encodeCache.
- internal/shortcut/shortcut.go 1-129 — Ensure, ensure, remember, aside, Target, Stale.
- internal/shortcut/shortcut_windows.go 1-123 — registerAppPath, registeredAppPath, programs, write (COM shell link).
- internal/shortcut/shortcut_other.go 1-11.
- internal/cli/backup_cmd.go 1-170 — backupCmd, restoreCmd, passphrase, stdin, secret.
- internal/cli/quota_wait.go 1-334 — resolveQuotaTarget, readable, quotaWait.run, roundWait, sleepCtx, quotaWaitCmd.
- internal/codexcat/codexcat.go 36-200 — DefaultEffort, NewSessionEffort, ModelDefault, Catalog(At/Used), FitOwn call path, model struct tags; 300-400 — owned/keepTrue, Keep, datedSuffix, agentsEffort.
- internal/redact/redact.go 120-215 — rule struct, rules table (PRIVATE_KEY, API_KEY×7, TOKEN/JWT, PASSWORD URL/CJK, SECRET fields).
- internal/redact/rules.go 1-143 — Rule, RuleKind, KindName, CheckRules, compile, compiled, customRules.
- internal/redact/shape.go 1-404 — keyHead, splitName, shapeOf, shaped, restoreShapes, restoreAddr, addrByte, shapeTail, hasShapes, outsideShapes, standInOf, ipStandIns.
- internal/upstream/upstream.go 1-120 — Vendors, VendorOf, hostOf, Status/Part/Incident.
- COVERAGE: read 38 files, 44950 non-test LOC in scope; skipped 87 (tui/*.go 9 files, cli/* beyond backup_cmd+quota_wait, pii/guard/*, redact/{dateword,fieldvalue,ids,json,known,model,more,phone,scrub,writer,zhname}.go, tune, mcpauth, catalog/live.go+public+gemini_list, omarchy/niri.go — budget; the largest request-path files were only grep-surveyed for swallowed errors, defer-in-loop and goroutine patterns). Files listed as read may be read in ranges for files over 800 lines.

## Findings

### F1 [P2] models.dev cache is rewritten in place, so an interrupted sync leaves magpie with no models for up to a day
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/catalog/catalog.go:780
- code:
```go
	p := CachePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return err
	}
```
- trigger: a catalog sync (hourly KeepFresh, `magpie sync`, or Missing() after an unpriced model) is interrupted after os.WriteFile has truncated the ~60 MB models.json — process killed, power loss (the #1505 zero-filled-file crash), or the disk filling mid-write.
- effect: models.json is left empty or truncated. loadGen (catalog.go:563-570) `continue`s past a file that does not unmarshal, so mdev stays nil: every provider's model list is empty, PriceOf/PricedBy price nothing (usage shows $0), and Provider(id) lists nothing. Stale() (catalog.go:793-800) only compares ModTime, so the just-truncated file is not stale, and SyncIfMissing (catalog.go:752) returns nil because Source() finds the file — the empty catalog stands until the 24 h check or a manual `magpie sync`.
- why: the cache is the only copy of the model list and is written non-atomically, while its sibling labs.json is written by temp+rename (labs.go:92-112); nothing validates what is read back.
- fix: write the cache with steady.WriteFile (or labs.go's temp+rename), and treat a cache that does not unmarshal into a non-empty map as stale in Stale().
- evidence: `rg -n --no-heading "models\.json|unexpected payload|len\(probe\)" internal/catalog internal/plugin` → 6 hits; the only payload validation is on the downloaded body (`catalog.go:773: if err := json.Unmarshal(b, &probe); err != nil || len(probe) == 0 {`), none on the cached file. verified-by: read+grep
- confidence: high

### F2 [P2] image MCP save path numbers names forever when Stat fails for any reason but "not exist"
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/imagemcp/imagemcp.go:677
- code:
```go
	out := filepath.Join(dir, base+e)
	for k := 2; ; k++ {
		if _, err := os.Stat(out); errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		out = filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, k, e))
	}
```
- trigger: `generate_image` (or `generate_video`) with `path` naming a file inside a directory this process cannot stat — e.g. `assets/hero.png` in a root-owned 0700 folder the user created with sudo once, an ACL-denied Windows folder, or a path on a disconnected network share: Stat returns EACCES/EPERM/EIO instead of ErrNotExist.
- effect: the job goroutine (imagemcp.go:158-162, no context and no deadline on `run`) spins for the life of the MCP server building hero-2.png, hero-3.png, … at full CPU, while every `tools/call` answers "still being made" (await's 20 s limit) and the image the user already paid for is never written.
- why: "never written over" is implemented as "keep numbering until Stat says ErrNotExist", with no bound and no branch for any other error, so a failure that will never clear is retried forever.
- fix: `if _, err := os.Stat(out); errors.Is(err, os.ErrNotExist) { return out, nil } else if err != nil { return "", err }`.
- evidence: `rg -n "os.Stat\(out\)|errors.Is\(err, os.ErrNotExist\)" internal/imagemcp/imagemcp.go` → single hit `677:` with no error branch, and the loop head `676: for k := 2; ; k++ {` has no counter bound. verified-by: read+grep
- confidence: high

### F3 [P3] masking-record store keeps its one mutex across file writes and whole-file reads, so the request path waits on the Privacy page
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/maskrec/maskrec.go:218
- code:
```go
	path := filepath.Join(dir, r.Time.Local().Format(dayForm)+".jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	before, _ := f.Seek(0, 2)
	_, err = f.Write(append(b, '\n'))
	f.Close()
```
- trigger: a request whose masking record settles (settle, called from Hold's `done`) while the Privacy page's List is running: List takes the same `store.mu` (maskrec.go:440) and reads every day file as it goes (`read(fs[i].path, ...)` at 452), up to 30 files / 100 MB; settle takes it at 200 with `defer store.mu.Unlock()` and holds it across MkdirAll, OpenFile, Write, Close and possibly prune (ReadDir + Stat/Remove of up to 30 files).
- effect: the two serialize. While a large record set is listed, every concurrent request's Hold (maskrec.go:172) blocks for the duration of that read, and a slow or network-mounted config dir stretches each settle by the file open+append; the GUI's own status/list calls stall behind request settles in the same way.
- why: one mutex protects both the in-memory slices and the on-disk work, so I/O latency becomes request latency; SentOf already shows the safe pattern (copy under the lock, decode outside, maskrec.go:395-406).
- fix: marshal and append the record before taking store.mu (or copy the needed state out and write after unlocking), and have List read the day files unlocked.
- evidence: `rg -n --no-heading "defer " internal/maskrec --glob '!*_test.go'` → `internal/maskrec/maskrec.go:201:	defer store.mu.Unlock()` (settle) and `441:` (List); `sed -n '198,236p'` and `sed -n '440,467p'` show the file I/O inside both critical sections. verified-by: read+grep
- confidence: medium

### F4 [P3] saving settings.json drops every key this build does not model
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/settings/settings.go:1754
- code:
```go
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
```
- trigger: a settings.json that holds a key this release does not know — written by a newer magpie or a fork sharing the same config dir, or added by hand — followed by any save here (toggling anything in the GUI, `SetRedactModel`, `UseRedactModel`, `EditUsageAliases`, …).
- effect: the unknown key is removed from the file with no copy of what it held; going back to the build that wrote it, that setting is at its default. lastgood.Keep (settings.go:1748) keeps a .bak only to recover from an unreadable file, so it does not preserve the lost key either.
- why: the file is rewritten as a marshalling of the Settings struct instead of being the on-disk JSON with the changed keys merged in, so forward-unknown keys cannot survive a round trip.
- fix: keep unknown top-level keys (`Extra map[string]json.RawMessage` with custom Marshal/Unmarshal, or read the file into a map, overlay the struct's marshal, re-marshal).
- evidence: `rg -n --no-heading "MarshalJSON|UnmarshalJSON|Extra |extra |unknown key" internal/settings/*.go --glob '!*_test.go'` → no matches (the run printed only the following `type Settings struct` section), and save() at 1754 marshals `s` alone. verified-by: read+grep
- confidence: high (mechanism); the trigger needs a build that wrote a newer key

### F5 [P3] omarchy.Current refreshes the theme inside its mutex, so a theme switch stalls every look for seconds
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/omarchy/omarchy.go:390
- code:
```go
	if d != desk || cached.Stamp == "" {
		font = fontName()
		rounding, border = hyprInt("decoration:rounding", 0), hyprInt("general:border_size", 2)
		desk = d
	}
	colors, _ := os.ReadFile(filepath.Join(dir, "colors.toml"))
```
- trigger: Hyprland reloads and rewrites ~/.config/hypr/* and fontconfig, which changes deskFiles()' stamp; the next Current() call (GUI theme request, `magpie gui` page load, a second caller at the same moment) enters the refresh branch with `mu` held (locked at omarchy.go:380, released by defer at 381).
- effect: the refresh runs omarchy-font-current (2 s timeout, fontName) plus two `hyprctl getoption` calls (1 s timeout each, and the code's own note at omarchy.go:334-336 says hyprctl asked during a reload "can stall for seconds"): the first caller's theme arrives up to ~4 s late and every other caller blocks on the mutex for that whole time even though the cached theme would have answered.
- why: a single mutex guards both the cache and the refresh work, so a refresh that blocks on subprocesses blocks all readers instead of only the one refreshing.
- fix: read the stamp and do the subprocess work outside the lock, then re-take it and store the result only if the stamp is unchanged.
- evidence: `sed -n '380,405p' internal/omarchy/omarchy.go` → `mu.Lock()` at 380 with `defer mu.Unlock()`, fontName()/hyprInt() at 391-392 inside; comment at 334-336 quotes hyprctl stalling "for seconds". verified-by: read
- confidence: medium (magnitude not measured on a live Hyprland)

## Checked, no defect
- appdir.Resolve/Passed/shared (appdir.go:91-138): a `data` folder in a shared folder is only taken when it already holds magpie's files and the installed folder does not, so the #989 "empty magpie" case cannot recur; Resolve is decided once from the original executable.
- appdir.LookupEnv/usable (env.go:84-107): a relative XDG/agent folder variable reads as unset rather than writing under the working folder, and Home() rejects a relative or missing home instead of joining onto "".
- settings.LoadChecked + parse + save (settings.go:1290-1354, 1721-1750): an unreadable settings.json is an error, not defaults; a file that does not parse falls back to lastgood's .bak and is copied aside by lastgood.Keep before being written over; save refuses to touch a file it could not read.
- catalog.Price.CostSplit/At (catalog.go:307-342): cacheWrite1h is clamped into cacheWrite, the tier is chosen once for the whole request by input+cacheRead+cacheWrite, and division is by 1e6 per million tokens — no unit mix-up in the cost path.
- redact rules table (redact.go:151-197): the shapes I spot-checked (private keys, sk-/ghp_/glpat-/AIza/xox/SG./JWT, URL passwords, .env-style names, CJK-named secrets) are bounded and case-correct; source.mirrorable (source.go:202-227) keeps Authorization/Cookie/API-key headers and token-ish query parameters off mirrors, and bare() drops a GitHub token before a mirror is asked.
- source.do/candidateContext (source.go:245-329): a retried request keeps its body only when GetBody exists, the mirror attempt has half the remaining budget, and the response body's context survives the stopped timer.
- pii.fetch/installServe (fetch.go:181-276): size and sha256 are both checked before rename, a resumed `.part` is verified again, and zip entries are written by filepath.Base so no entry escapes serveDir.
- shortcut.write (shortcut_windows.go:92-121): the shell link is reused only when Stale() says it already opens this exe (case-insensitive compare), and COM is initialised/released with the RUnlockOSThread pairing intact.
- lastgood.Keep/KeepBad (lastgood.go:112-218): a file that is there but invalid is copied aside before any write, and keptAlready stops the same bad bytes being stored again for ever.

## Open questions
- Does Hyprland drop `hyprctl eval` window rules named "magpie-panel" on a config reload, as panelRule's comment (hypr.go:90-92) claims? If not, every panel open adds one rule to a list with no removal, unbounded over a session. Needs a live Hyprland.
- Does Codex's models.json ever carry top-level keys besides "models"? codexcat.Keep re-marshals through its `list` struct ({models} only, codexcat.go:329-331, 375), so any other top-level key is dropped whenever a user-added entry key is kept.
