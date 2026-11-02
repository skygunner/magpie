## Coverage
- `internal/gateway/fastjson.go`: 1-700, 701-1591 (whole file) - decodeJSON/decodeJSONIn/decodeJSONView, holdBody/inHeldBody/heldBodies, fillJSON/fillJSONIn, longBody/validMemo/jsonValid/jsonCheck, skipJSONString/skipJSONValue/eachMember/eachMemberAt/eachElement, unquoteJSON/keyName, marshalRaw/marshalBody/marshalItems/appendRawJSON, getIn.
- `internal/gateway/ir.go`: 517-560 - collector struct, last/open/grow/flush text accumulation.
- `internal/gateway/chattidy.go`: 255-355 - per-chunk chatTidy rewrite path (delta/choices/chunk re-marshal), withoutEmptyNames.
- `internal/gateway/gateway.go`: 740-870, 1185-1325, 438 - route table, `models`/`model` handlers, codexTurns map.
- `internal/gateway/request_bounds.go`: 1-260 - requestBody/readRequestBody, readBoundedRequestBody, decodedBody, pooledZstd, readSized.
- `internal/gateway/body_room.go`: 1-200 (whole file) - bodyRoomT.take/letIn/give, bodyRoomSize.
- `internal/gateway/holdcatalog.go`: 1-25 - holdCatalog (provider.Hold + filememo.Hold per request).
- `internal/gateway/autoref.go`: 1-80 - AutoPrefix/CanonicalRef/autoRefsOn (the id change behind the failing benchmark below).
- `internal/usage/request_page.go`: 560-710, 1420-1540 - readRequestState, statKey/recordHash, requestCache bounds, matchedLocal/matchedBlocks.
- `internal/usage/request_pairs.go`: 1-139 (whole file) - pairSerials/unnamedSerial, pairRowsMemo, memoOf, pairMemo.apply.
- `internal/usage/log_index.go`: 98-130 - readLogSnapshot and its hash fallback.
- `internal/sessions/sessions.go`: 690-800, 1181-1220 of calls.go - rolloutName, codexFiles, warmListing, listCodexFiles, codexFilesIn/walkLinked, CallSourcesAndCodexIdentities.
- `internal/sessions/quiet.go`: 40-70 - Ask/quiet/readSlot.
- `internal/sessions/one_listing.go`: 72-97 - heldFiles per-request listing memo.
- `internal/sessions/call_cache.go`: 1-130 - callCache/callCounts/callOrder bounds, pruneCalls.
- `internal/gui/api.go`: 2440-2480 - `held` middleware (provider.Hold, filememo.Hold, sessions.Ask, tightenGC); route list via grep.
- `internal/provider/group.go`: 1000-1080 - buildGroupEntries (read as context for the 18.1 ms/op figure; not my lane).
- `internal/provider/autoref.go` id semantics confirmed against `/v1/models` output from `BenchmarkHeldKeyListsModels`.
- Benchmarks run: `go test -run '^$' -bench . -benchtime=100ms -count=1 ./internal/gateway ./internal/usage ./internal/sessions ./internal/provider ./internal/plugin ./internal/redact ./internal/middleware` (output kept in `perf-hot.bench.txt`; gateway exited 1, see Open questions).
COVERAGE: read 18 files, 23714 non-test LOC in scope; skipped 4 (provider/models.go, the gui/api.go handlers past :2480, usage/ledger.go+usage.go, sessions/stats.go+calls.go bodies - no benchmark in this run pointed at a hot path inside them, and their entry points are covered by the page-path findings below).

## Findings

### F1 [P2] Every Requests-page rebuild re-walks all Codex homes and lstats every rollout; the listing cache exists only for the warm-up
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/sessions/sessions.go:705
- code:
```go
func codexFiles() []file {
	if !quiet() {
		return heldFiles("codex", listCodexFiles)
	}
	warmListing.Lock()
	defer warmListing.Unlock()
	if warmListing.files != nil && time.Since(warmListing.at) < warmListingFor {
```
- trigger: a GUI poll of `/api/usage/requests` (or `/api/sessions`) on a machine with ~12k rollouts across several Codex homes: `readRequestState` -> `sessions.CallSourcesAndCodexIdentities` -> `callSourcesWith` -> `codexFiles`. `quiet()` is `warming.Load() > 0 && asking.Load() == 0` (sessions/quiet.go:52) and the `held` middleware calls `sessions.Ask()` for every `/api/usage` and `/api/sessions` request (gui/api.go:2460), so `quiet()` is false for exactly the requests that do this work; `heldFiles` only answers from its memo while a hold is open (one_listing.go:74: `if held.n == 0 { return list() }`), so the walk runs again for the next poll.
- effect: each rebuild walks `sessions` and `archived_sessions` of every Codex home with `filepath.WalkDir`, lstat-ing every rollout and allocating a `file` per entry. The same file's own measurement of that walk: "each listed every rollout again within a few seconds, ~12k lstat()s and ~45 MB a time" (sessions.go:719-723). That cost is paid before any parsing, on every poll, and it is what the page's change key is computed from - so it cannot be skipped or cached by the key it produces.
- why: the invariant "a listing used to decide whether anything changed is cheap" is broken by scoping the only cross-request listing cache to `quiet()` - the warm-up - while the page readers (the ones that poll) are precisely the callers excluded from it.
- fix: let `codexFiles` read `warmListing` outside `quiet()` too, with a short TTL (`warmListingFor`, 10 s), instead of only during warm-up.
- evidence: `rg -n 'func quiet' internal/sessions/quiet.go` -> `func quiet() bool { return warming.Load() > 0 && asking.Load() == 0 }`; `rg -n 'codexFiles|warmListing' internal/sessions/sessions.go` -> 705/707/713-716 as quoted, comment at 719-723 claiming 12k lstat()s and ~45 MB per walk; `grep -n 'sessions.Ask()' internal/gui/api.go` -> 2460 (`/api/sessions` and `/api/usage`). verified-by: read
- confidence: high (mechanism); the 12k/45 MB magnitudes are the tree's own measurement, not re-measured here.

### F2 [P3] The Requests page's change key is built by JSON-marshalling the whole price tables and globbing desktop folders on every rebuild
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usage/request_page.go:649
- code:
```go
	prices, _ := json.Marshal(cfg.ModelPrices)
	wires, _ := json.Marshal(cfg.ModelWires)
	meta := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s", ids, renamed, prices, wires, statKey(provider.Path()), statKey(catalog.CachePath()), clientAgentsKey(), clearedKey())
	h := sha256.New()
	for _, current := range provider.All() {
		fmt.Fprintf(h, "%q:%q:%v:%t:%t:%t;", current.ID, current.Where(), current.Catalogs(), current.IsPlugin(), current.Account == nil, current.Key != "")
		fmt.Fprint(h, statKey(catalog.LivePath(current.ID)))
	}
	for _, root := range sessions.DesktopDataDirs() {
```
- trigger: any rebuild of the requests page (`QueryPage` -> `readRequestState`, request_page.go:638), i.e. every poll that arrives with a new page key, on a settings file with a large `ModelPrices` table (the per-model override map) and on machines whose Claude-desktop folders hold many `local_*.json` session files.
- effect: two full encodings of the price tables into fresh byte slices that then enter a `fmt.Sprintf` (a third copy of both), one `os.Stat` per provider plus one per provider catalog file (`catalog.LivePath`), and, per desktop root and per kind, a `sessions.SessionGlob(root, kind, "*", "*", "local_*.json")` - a two-level directory scan plus a stat per match - all to derive one hash. Nothing here is measured by a benchmark, so the magnitude is a fresh allocation of the price table plus one directory scan per root per poll.
- why: the change key is meant to be a cheap fingerprint of what the page read; marshalling a settings table and scanning directories to fingerprint it makes the key itself a full-file/whole-table pass, and the glob result is never reused by the rebuild that follows it.
- fix: hash the settings' own change key (as `provider.UsageRenames().Key()` already does for renames) instead of `json.Marshal(ModelPrices/ModelWires)`, and reuse the globbed path list from the same call rather than recomputing it.
- evidence: `grep -n 'prices, _ := json.Marshal\|DesktopDataDirs()\|for _, current := range provider.All()' internal/usage/request_page.go` -> 649, 650, 653, 657; the `SessionGlob` call sits at 659 inside that loop. verified-by: read
- confidence: medium (the work is certain; no benchmark in the tree measures `readRequestState`).

## Checked, no defect
- `request_bounds.go` + `body_room.go`: a request body is capped at 512 MB decoded (`readBoundedRequestBody`), rejected before reading when `Content-Length` exceeds it, and charged to a global memory room keyed by context - `take` is called twice (on the declared length, then on the bytes read) and `b.by[ctx]` makes the second call add only the excess, so a body is never double-counted; a request whose context never ends takes nothing (`ctx.Done() == nil`), so the room map cannot leak. `readSized` avoids io.ReadAll's two copies.
- `fastjson.go`: the per-request decode path has no per-call compilation of anything - field tables are cached per `reflect.Type` (`fastFieldsByType`), the validity memo is 8 entries keyed by `{len, maphash}` under a process-private seed, and held bodies are capped at 64 (`heldBodiesMost`) with a documented copy fallback, so neither cache grows with attacker input. `decodeJSON` falls back to `json.Unmarshal` for every case it does not model, which keeps the semantics.
- `fastjson.go` stream/body builders (`marshalBody`/`marshalItems`, `appendCompactJSON`): no `+=` string growth; lists are written item by item into a buffer sized from the items (`bodyKept` bounds what is retained), and the text collector in `ir.go:517-560` accumulates deltas in a `strings.Builder` that is flushed into the part, so reply accumulation is linear (locked by `TestCollectorTextLinear`).
- `chattidy.go` per-chunk path: the three `json.Marshal` calls (delta, choices, chunk) only run when the tidy actually changed something - `if !changed && !usageMended { return line }` returns the upstream line untouched - so an ordinary streamed chunk costs a decode of the chunk and of its delta, no re-encode.
- `sessions/call_cache.go`: `callCache`/`callOrder`/`callCounts` are bounded (`maxKeptCalls = 131072`, `maxKeptFiles = 64`) and pruned against the live file list on every `pruneCalls`, so the per-path cache is not an unbounded growth vector.
- `usage/request_pairs.go`: pairing results are memoised by chunk serials plus the window (`pairRowsMemo`), and the memo applies only on freshly cloned chunks - the O(rows) `pairRows` cost is paid once per changed chunk set, not per poll (177 ms/116 MB for a 2000-file/200k-row fixture, `BenchmarkPairRows`, is that one-off cost).
- `usage/request_page.go` `matchedBlocks`: the row-by-row scan is guarded by per-chunk hints (`hintsOf`, `namesAny`, `hints[n].latest`) and reuses pooled scratch maps/slices (`matchScratchFor`, `mapFor`), so a page is not O(chunks x rows) over unrelated history.
- `gateway.go` `models` handler (:1185): builds the list per request with no per-catalog memo, but every pass (`provider.Labels`, `codexTiers`, `priceOf`, `modelObject`) is linear in the catalog and the catalog itself is held once per request (`holdCatalog`), so no request triggers a per-model provider lookup (`EntryPricer` is built once for the list, not once per model).
- Tickers on long-lived paths all carry `defer tick.Stop()` (`hub_mcp.go:211-212`, `patience.go:188-189`, `fallback.go:2619-2620/2664-2665`, `gateway.go:6861-6862/6896-6897`, `sessions.go:1222-1223`, `quiet.go:61-62`), and the keepalive loops exit on context done, so no ticker/goroutine leak was found on the request path.

## Open questions
- `BenchmarkHeldKeyListsModels` fails at HEAD on all three sub-benchmarks; the failing assertion is on the model id the list returns, not on cost: `go test -run '^$' -bench '^BenchmarkHeldKeyListsModels$' -benchtime=3x -count=1 -v ./internal/gateway` -> `--- FAIL: BenchmarkHeldKeyListsModels/models`, `/accounts`, `/free`, message `key_models_list_test.go:255: 200 {"data":[...{"id":"auto/model-300"...` while the test wants `"group/auto-model-499"`. That matches `internal/provider/autoref.go:1-20` (found groups are listed by their public id `auto/<model>`; `group/auto-<slug>` is the internal id), so this looks like a stale expectation in the benchmark, not a wrong result - but it is the only benchmark that would have measured `/v1/models` with a 2x500-model catalog, so that cost is still unmeasured.
- Unmeasured here: `/v1/models` per-request object building on a large catalog (`BenchmarkGroupEntries` measures only `buildGroupEntries`, 18.1 ms/op on 30 providers x 60 models, which the request path amortises through `provider.Hold`); `readRequestState`'s real cost with a large `ModelPrices` table (F2) and the walk in F1 on a 12k-rollout history. All three need a timed run against a prepared history to rank.
- `readLogSnapshot` (usage/log_index.go:112) falls back to `logRecordHash(path, old.off)` - re-hashing the whole already-indexed log - whenever `logChangeStamp(info) == ""` or the previous snapshot was unsettled; on a 100 MB+ gateway log that is a full re-read per poll. Whether that branch is reachable on Linux/macOS at HEAD (it hangs on the platform change stamp being available) needs a runtime experiment.
