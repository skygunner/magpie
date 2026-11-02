## Coverage

- `internal/provider/models.go` 1-520, 521-1100, 1101-1480, 1480-1700, 2980-3173: `Available`/`available`, `live`/`lists`/`liveAll`, `Fetch`/`Refetch`/`RefetchTaking`/`refetch`, `liveIDs`, `dropGonePicks`/`relistPicks`, `fetch`/`fetchOne`/`List`, `FetchNew`/`newDue`/`fetchDue`, `fixV1`, `fetchPerKey`/`keepDeadSides`, `planModels`, `Exposed`/`Picks`/`withNewlyListed`, `Known`/`Efforts`/`Thinks`, `SetModelOutputs`/`DropModelOutput`, `SetContext`/`DropContext`.
- `internal/provider/modelprefs.go` 1-520, 521-900, 901-1285 (whole file): `splitRef`, `setModelName`, `SetModelPrice`/`PriceKey`/`DropModelPrice`, `setModelEfforts`/`keptEfforts`, `setModelReasonless`, `setModelImage`/`ApplyImage`/`vendorSees`, `setModelAPI`/`ModelAPI`, `SetModelPrefs`, `setModelSame`, `renameModelPrefs`, `Labels`, `serves`/`ServesModel`, `UpstreamName*`, `SetUpstreamName`/`DropUpstreamName`/`wireKey`, `asksOwnBackend`.
- `internal/provider/codex_models.go` 1-500: `codexModels`, `CodexLiveList`, `codexVersion`/`newerVersion`, `SawCodexClient`/`seenVersion`, `codexExecutable`, `parseCodexModels`, `Account.Lists`/`Levels`/`Tiers`, `codexPoolModels`/`levelRank`, `codexFetchSaved`, `CodexListed`/`CodexCatalog`/`CodexNativeHidden`/`CodexOrder`.
- `internal/provider/presets.go` 1-60, 660-758: `Kind`/`PresetDef`/`Region`, `Presets`, `Preset`/`builtinPreset`, `FromPreset`, `atRegionOf`, `IconForCatalog` (the literal preset table at 61-659 was not read).
- `internal/provider/migrate.go` 1-1250 (whole file): `Move`, `Adopt`, `move`, `commitMove`, `putBack`, `handBack`, `tidyMoved`, `keepOrder`, `MoveRetiring`, `HandOver`, `pluginReady`, `keepMovedCurrent`, `KeepRetiringMoved`/`nextRetiringRun`, `lockMoves`, `setMigration`/`readMigrations`/`MigrationOf`/`Moved`, `MoveCandidates`, `movedAgent`, `RemovePlugin`, `SetPluginOff`, `renewedSince`, `signinHash`, `installPlugin`.
- `internal/provider/migrate_side.go` 1-120: `init` movers for `devin` and `grok` (`out`/`back`).
- `internal/provider/migrate_kiro.go` 1-180 (whole file): `init` mover (`out`, `back`, `take`, `give`, `settle`), `kiroOut`, `setKiroKey`, `kiroForgetHome`.
- `internal/provider/migrate_mimo.go` 1-88 (whole file): `init` mover `out`/`back`, `idText`.
- `internal/provider/migrate_qoder.go` 1-101 (whole file): `init`, `qoderMover` (`out`/`back`), `num`.
- `internal/provider/migrate_workbuddy.go` 1-83 (whole file): `init` movers for `wbCN`/`wbAI`, `out`/`back`, `msOf`.
- `internal/provider/migrate_zcode.go` 1-103 (whole file): `init` mover, `out`/`back`, `zcodeOut`.
- `internal/provider/migrate_factory.go` 1-108 (whole file): `init` mover, `out`/`back`, `factoryKeyBack`.
- `internal/provider/migrate_zed.go` 1-103 (whole file): `init` mover, `out`/`back`, `zedRefresh`, `backInto`, `jsonText`, `str`.
- `internal/provider/plugins.go` 1-831 (whole file): `PluginID`/`customIDs`/`subscriptionID`, `IsPlugin`/`PluginProvider`/`PluginOf`, `pluginAccounts`, `pluginProtocol`/`copilotAPI`/`pluginModel`, `PluginTiers`/`pluginTierOn`, `pluginCatalog`/`pluginPrice`, `pluginAccountCatalog`/`pluginServedCatalog`, `PluginListError`/`ListError`, `pluginLists`, `pluginProvider(Serving)`, `pluginAPIs`, `Origin`, `pluginFetch`, `modelOnly401`/`notePluginSignIn`/`notePluginSaid`, `replaceModel`/`withModel`, `Do`.
- `internal/provider/google.go` 1171-1250: `googleAccount.models`, `antigravityList`, `fallbackModels`, start of `quota` (the account/refresh paths before 1171 belong to another lane).
- `internal/provider/vertex.go` 244-430: `vertexTokens`/`vertexToken`, `forgetVertexTokens`, `vertexAccessToken`, `vertexCredentialsFile`, `gcloudDir`, `vertexEnvPath`, `homePath`.
- Supporting reads outside the assigned files, used only to judge callers and guards: `internal/provider/hold.go` 100-230 (`Changed`/`heldOf`/`StampNow`), `provider.go` 500-640 (`load`/`read`/`store`/`FileError`), 1029-1090 (`Add`), 1180-1300 (`TuckAccount`/`QuietAccount`/`ShowAccount`/`Delete`), `internal/settings/settings.go` 1210-1290 (`Load`/`LoadChecked`/`Shared`), 1424-1500 (`Save`, `SetGatewayConversations`, `EditCheckinOff`, `EditUsageAliases`), 1700-1760 (`save` tail), `qoder.go` 18-75 and `factory.go` 38-107 (mover mutex protocol), `logins.go` 211-255 and `side_logins.go` 34-47 (logins read/write locking).

COVERAGE: read 16 files, 7072 non-test LOC read of 11437 in scope; skipped 4365 (preset literal table in presets.go, codex_models.go tail 501-778, models.go 1701-2979, google.go 1-1170 and 1251-1683, vertex.go 1-243 and 431-640, migrate_side.go 121-273 - none of the skipped regions holds catalog-merge or preference logic this lane owns).

## Findings

### F1 [P2] Two model-preference writes at once lose one of them, with the whole settings file
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/modelprefs.go:90
- code:
```go
	if name == "" {
		delete(s.ModelNames, key)
	} else {
		if s.ModelNames == nil {
			s.ModelNames = map[string]string{}
		}
		s.ModelNames[key] = name
	}
	if err := settings.Save(s); err != nil {
```
- trigger: two preference writes in flight in the same `magpie serve`: the GUI's provider Save (`internal/gui/providers.go:1715` `SetModelPrefs`) or a `magpie model name/price/efforts` run next to a GUI save of another setting. Each takes `s := settings.Load()` and later `settings.Save(s)`, with no lock spanning the two.
- effect: whichever Save lands second writes its own snapshot whole (settings.go:1754 marshals `s` and replaces the file; `save` only re-reads `GatewayConversations`), so the other change is gone, and so is every other setting changed in that window -- theme, prices, wire names, privacy consent -- silently and with a success reply. The user sees a setting revert after a restart or a redraw.
- why: the read-modify-write is not atomic. `settings.Load` releases `fileMu` before returning and `settings.Save` takes it only for the write, so the mutex the package added for exactly this (`EditUsageAliases`: "a save elsewhere from an older snapshot isn't undone") is bypassed by all of modelprefs.go.
- fix: give the settings package one `Edit(func(*Settings) error) error` that does load, mutation and save under `lockFile`, and call it from `setModelName`, `SetModelPrice`, `setModelEfforts`, `setModelReasonless`, `setModelImage`, `setModelAPI`, `setModelSame`, `SetUpstreamName`, `DropUpstreamName`.
- evidence: `grep -n "Lock()" internal/provider/modelprefs.go` -> `NO LOCK in modelprefs.go`; `nl -ba internal/settings/settings.go | sed -n '1424,1428p'` -> `func Save(s Settings) error { defer lockFile()(); return save(s, false) }`. verified-by: read|grep
- confidence: high

### F2 [P2] A model refresh rewrites providers.json from its own snapshot and reverts edits made meanwhile
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/models.go:334
- code:
```go
	if len(dropped) > 0 {
		q.Models = kept
		for _, m := range dropped {
			if !slices.Contains(q.Delisted, m) {
				q.Delisted = append(q.Delisted, m)
			}
		}
	}
	return dropped, store(f)
```
- trigger: `p.Refetch`/`Fetch` for provider A (the GUI's Refresh or the Save path at gui/providers.go:1728, both on a request goroutine) overlapping any other write to providers.json: `Delete("B")`, `Add(c)` from another tab, a key change, a rename, or start-up's background `FetchNew`. `dropGonePicks` reads the file at models.go:305, mutates one provider, and `store(f)` replaces the file whole (provider.go:604-640). `fixV1` (models.go:1181-1198) does the same from inside a fetch.
- effect: the later writer's snapshot wins, so an edit made between the read and the store is lost. Deleting a provider does not stick (it and its API key come back), or a provider added with its key `Add` had just written disappears from providers.json; a rotated key reverts to the old one. No error is reported to either action.
- why: 47 call sites do read-modify-write on providers.json with no package-level lock (`store` is only atomic against a torn read, not against a stale read), and a background fetch writes the user's configuration file.
- fix: serialise providers.json read-modify-write under one package mutex (as `served.go:110` does for its own file), or make these two paths patch the single provider inside `store` from a fresh read.
- evidence: `grep -rn "providersMu\|providers.json.*lock\|lockProviders" internal/provider/*.go` -> `NO providers.json lock`; `grep -rn "store(f)" --include=*.go internal/provider | wc -l` -> `47`. verified-by: read|grep
- confidence: high

### F3 [P3] Codex's pooled list drops the levels a second account lists when the signed-in account lists none
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/codex_models.go:396
- code:
```go
			if i < 0 {
				o.Efforts = slices.Clone(o.Efforts)
				out = slices.Insert(out, next, o)
				next++
				continue
			}
			next = max(next, i+1)
			if len(out[i].Efforts) == 0 {
				continue
			}
```
- trigger: a Codex provider signed in to account P (whose saved list has model M with no reasoning levels) plus one more account switched on whose saved list has M with levels (the case #520 names, one plan listing fewer levels than another), then `CodexListed`/`CodexCatalog` builds the provider's list.
- effect: M is offered with `Efforts` empty, so the picker and the gateway's `/v1/models` say it takes no effort and agents cannot ask for `high`/`xhigh` on it, while the account that lists those levels would accept them. Only a model absent from the first list gets the union (the `i < 0` branch above).
- why: the union promised by the function's own doc ("each model takes the reasoning levels any of them gives it") is applied only to models that already have levels, so a base list that says nothing about a model suppresses every other list's word.
- fix: replace the `continue` with `out[i].Efforts = slices.Clone(o.Efforts)` when the base list has none, then merge as below.
- evidence: `nl -ba internal/provider/codex_models.go | sed -n '384,406p'` shows the guard at 396-398 ahead of every merge; the doc at 366-377 says the levels are "what its accounts together take". verified-by: read
- confidence: medium

### F4 [P3] The Codex model list is truncated at 4 MiB where the same endpoint is read to 16 MiB, and the read error is thrown away
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/codex_models.go:51
- code:
```go
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ChatGPT models: %s", resp.Status)
	}
	saveCodexPrompts(b)
	ms := parseCodexModels(b)
	if len(ms) == 0 {
		return nil, errors.New("ChatGPT listed no Codex models")
	}
```
- trigger: `GET {CodexBase}/models` answering a body above 4 MiB (the same URL `CodexLiveList` reads at `16<<20`, line 89, for the base instructions it carries), or any error after the 200 that cuts the body short (a reset proxy, a cancelled stream).
- effect: the body handed to `parseCodexModels` is truncated, JSON decoding fails, and the provider reports "ChatGPT listed no Codex models" -- the wrong cause, no partial list, and `codexFetchSaved` silently keeps the previous list while the active account falls back to magpie's short default list.
- why: two readers of one endpoint disagree on its size, and the only signal that the body was not read whole (`io.ReadAll`'s error) is discarded before the parse, so truncation and an empty list are the same state.
- fix: read with `16<<20` as `CodexLiveList` does and return the `io.ReadAll` error instead of `_`.
- evidence: `nl -ba internal/provider/codex_models.go | sed -n '51,59p'` (4<<20, `b, _ :=`) and `sed -n '89p'` (`io.LimitReader(resp.Body, 16<<20)`). verified-by: read|grep
- confidence: high

## Checked, no defect

- `dropGonePicks`/`relistPicks` (models.go:304-362): only a pick the *previous fetched list* had and the new one lacks moves to `Delisted`, so a hand-typed pick survives every refresh; a fetch that fails or answers zero models returns before the drop (models.go:266-278), and a list that shrinks for a while gives its picks back on the next fetch.
- `refetch`'s four paths (models.go:241-287): an account, a decision-only provider and a plugin each take their own branch; only the built-in list path touches `Delisted`, and the first fetch after an upgrade (`before` empty) is limited to relisting, never dropping.
- `handBack`/`commitMove` ordering (migrate.go:915-1000, 800-850): plugin sign-ins are written into logins.json before `plugin.Take` removes them, `renewedSince` refuses to commit when the built-in refreshed mid-move (a spent refresh token), and a failed `setMigration` gives `Kept` back through `mv.give`.
- `setMigration` (migrate.go:445-470): clones `Accounts`, each `Was` map and `Backup` before `f` runs, so the map shared with readers through `filememo` is never mutated, and `Changed()` runs after the write to drop the held copy.
- `Move`/`putBack`/`MoveRetiring` state machine (migrate.go:521-620, 824-845, 1040-1065): `MovePlugin` is the only state `Moved` accepts, a cut-short `moving` is put back before retrying, `failed` respects `retryDue` (10 min for retired built-ins), and `removed` is never adopted without the user's own install.
- `presets.go` lookup (697-750): `Preset` returns a pointer into the package table but every caller only reads it, and `FromPreset`/`atRegionOf` write to the new `Provider`, so no request mutates the shared table.
- `plugins.go` ownership gate (78-153, 254-360): `PluginID` is total and deterministic (`PluginServes` first, then the `-plugin` suffix for ids a preset, built-in or user's provider owns), `PluginOf` scans held plugins only, and `pluginCatalog`/`pluginAccountCatalog` run `slices.DeleteFunc` on freshly built slices, so no model slice is shared and mutated.
- `vertexAccessToken` (vertex.go:285-345): the cache is keyed by credentials file plus impersonation, the mint is serialised by a one-slot channel per key, `ctx.Done()` breaks a waiter out, and a token whose file stamp or expiry lead changed is minted again.

## Open questions

- How large is a real `GET /chatgpt/backend-api/codex/models` answer for a Pro account with base instructions (above or below 4 MiB)? That decides whether F4 fires in the field or only on a broken read; it needs a signed-in ChatGPT account, which this read-only snapshot cannot exercise.
