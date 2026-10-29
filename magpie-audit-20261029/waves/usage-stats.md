# Usage accounting, cost math, request ledger and stats - audit

Snapshot: `/tmp/magpie-audit-Pzn4fNp7/src` (release tree, read-only). Lane: `internal/usage`, `internal/usageapi`, `internal/stats`, `internal/gateway/{usage_api,usage_ingest}.go`.

## Coverage

- internal/usage/usage.go 1-1224 - Record/Append/openLog/Load/LastSeen/CostAt/Totals.addFrom/summarizeFrom/DecodeOf/FormatCost.
- internal/usage/ledger.go 1-1133 - Filter.keepsRecord, LedgerOf/ledgerMatched/Pricer/NewPricer, WriteCSV, timeline/bucketIndex, addRow/plus, Breakdown, LedgerSeries, Visit, NewCoster.
- internal/usage/request_page.go 55-375, 390-510, 511-527, 596-840, 940-1050, 1040-1270, 1868-2208, 2266-2328 - packedRow/add/row, chunkRows/rowsOf, pageKey, readRequestState, queryPageOf, readRequestBlocks, requestIndex (adopt/keep/save/trim), buildRequestBlocksIn, pageFromLedgerAt, prefetchEach. Unread: 1270-1868, 2208-2266.
- internal/usage/request_cells.go 1-382 - pageCell/chunkPart/partKey, cellPass, part, partsOf, newest, sigs.
- internal/usage/request_snapshot.go 1-299 - pack/encode/unpackReusing/keptStrings.
- internal/usage/request_pairs.go 1-139 - pairMemo/memoOf/apply.
- internal/usage/match_chunks.go 1-119 - matchHints/hintsOf/namesAny.
- internal/usage/log_index.go 1-373 - readLogSnapshot/visit/writable/freeze/settledLogStamp/noteLogAppend.
- internal/usage/log_file.go 1-34 - sameLogFile/logChangeStamp.
- internal/usage/reported.go 1-147 - catchUpReported/IngestReported.
- internal/usage/cleared.go 1-110 - readCleared/hides/clearedSince.
- internal/usage/summary_cache.go 1-148 - topSummary/topOf/cloneSummary/usagePriceCacheKey.
- internal/usage/block_price.go 1-281 - pricedAs/blockPricer/as/block/priceBlock.
- internal/usage/request_disk.go 1-451 - diskEntry/readDiskEntry/readDiskEntries/fromDisk/toDisk/onDisk/diskEntryPath.
- internal/usage/request_disk_map_unix.go 1-47 - mapDiskFile/read/readMapped.
- internal/usage/request_spill.go 1-115 - archiveSpill put/hand/done/pagesRank.
- internal/usage/direct.go 1-135 - Direct/directRows/direct.
- internal/usage/shared.go 1-300 - Computer/OwnDays/KeepShared/sharedSources/sharedRecords/DropShared.
- internal/usage/gateway_sessions.go 1-419 - GatewaySessions/gatewaySessionFrom/gatewayProjection/buildGatewaySessions/gatewayDays/GatewayModel.addRecord.
- internal/usage/vias_cache.go 1-116 - ViaIndex.Of/Len/window, cachedVias, logSnapshot.complete.
- internal/usage/heatmap.go 1-188 - heatmapSince/heatDays/heatOf/heatmapFrom.
- internal/usage/window_holds.go 1-342 - WithWindowHolds/holdsBounds/dryRuns/windowHolds/apiPricer.
- internal/usage/observe.go 1-161 - Observe/ServeObservers/observeLedger.
- internal/usage/codex_used.go 1-125 - codexUsed/codexChoice/codexUsedNew/mostRecent.
- internal/usage/client_agents.go 1-63 - readClientAgents/reportedAs.
- internal/usage/probes.go 1-149 - probeRow/probeChunk/probeProvider/sitesOf.
- internal/usage/served.go 1-266 - bareModel/Swapped/ListedAs/RelayRenamed/ServingName/GroupRouted.
- internal/usage/purpose.go 1-54 - purposeKinds/PurposeOf.
- internal/usage/period_range.go 1-107 - RangeOf/LastOf/Last/days/IsRange/Known/Until/shown.
- internal/usage/zone.go 1-62 - Zone/sameClock/loc.
- internal/usageapi/usageapi.go 1-467 - State/PageIn/PageSessions/stateOf/PeriodOf/FilterOf/viaOf/LedgerPage/OwnPage.
- internal/stats/stats.go 1-174 - Run/Send/Off/installID/osName.
- internal/stats/usage.go 1-109 - readUsage/providerLabel/modelLabel/agentLabel.
- internal/gateway/usage_api.go 1-82 - heldKey/heldRefused/refuseHeld/usageSummary/usageRequests.
- internal/gateway/usage_ingest.go 1-377 - usageIngest/parseIngest/parseReported/ingest limits.
- internal/catalog/catalog.go 250-380 (supporting, out of lane) - Price.Times/At/OneHour/Cost/CostSplit/tier parsing.

COVERAGE: read 36 files, 12,539 non-test LOC in scope; skipped 10 (codex_identity.go is 812 lines, over the full-read budget and reached only by grep; session_provider.go, window_holds_history.go, codex_actors.go, request_build_share.go, log_change_darwin.go, log_change_linux.go, log_change_other.go, log_change_windows.go, request_disk_map_other.go are helpers off the accounting/page path; request_page.go 1270-1868 and 2208-2266 were not read).

## Findings

### F1 [P2] An unbounded `?period=` makes one Requests request allocate ~0.5M chart points and ~3M maps
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usage/ledger.go:639
- code:
```go
	if d := p.Last(); d > 0 {
		since = now.Add(-d)
		if d <= 48*time.Hour {
			bucket = "hour"
			for h := 0; h <= int(d.Hours()); h++ {
				t := since.Add(time.Duration(h) * time.Hour)
				pts = append(pts, Point{Label: t.Format("15"), Time: t})
			}
		} else {
			n = int(d.Hours())/24 + 1
```
- trigger: one loopback request, no key needed (`usageRequests` only checks `local(r) || sharedWith(r)`, usage_api.go:67-70): `GET /v1/magpie/usage/requests?period=0001-01-01..9999-12-31&limit=1`. `usageapi.PeriodOf` accepts it because `Period.Known()` accepts any parseable date range (usageapi.go:197-198, period_range.go:80-86); a range past 60 days becomes one weekly point per 7 days. `period=last:2562047h` (the longest `time.ParseDuration` accepts) gives 106,752 daily points instead.
- effect: `buildRequestBlocksIn` runs `out.Series = make([]SeriesPoint, len(base))` (request_page.go:2065) and then allocates 1 outer + 5 dimension maps for each point (`SeriesPoint{..., By: map[string]map[string]Part{}}` plus `out.Series[i].By[d] = map[string]Part{}`) - 521,723 points and 3.13M maps, ~250-350 MB, on an empty log, and the same `base` is built again and discarded by `newCellPass` (request_cells.go:143). The finished page is then kept in `requestCache.pages` (up to 16 entries, request_page.go:815-818), so the memory stays after the request returns; two such requests at once (a second waits on the flight) or a few distinct periods wedge the gateway process for seconds and can OOM it. The ledger path has the same shape at ledger.go:1074.
- why: the chart's cost is linear in the period's calendar length, but nothing bounds the period: `Limit` is capped at 500 and `offset` clamped, while the period string is taken from the query and only tested for parseability. The answer's data (rows) stays tiny; the allocation is pure derived state.
- fix: reject/clamp periods in `usageapi.PeriodOf` (dates and `last:` alike) to the 53-week heatmap horizon, or cap `timeline`'s `n` and fold older days into the first bucket.
- evidence: `grep -rn -e maxPoints -e MaxPeriod -e maxSeries -e 'len(base)' -e 'Series\[:' internal/usage internal/usageapi internal/gateway | grep -v _test.go` -> only `request_page.go:2065	out.Series = make([]SeriesPoint, len(base))` and `ledger.go:1074	pts = make([]SeriesPoint, len(base))`, no cap; `grep -n 'if p := usage.Period(s); p.Known()' internal/usageapi/usageapi.go` -> `198:	if p := usage.Period(s); p.Known() {`. verified-by: read|grep
- confidence: high

### F2 [P2] A rolling period's Requests page is cached for the whole day, so `last:24h` freezes
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usage/request_page.go:743
- code:
```go
	q := pageKey{p, f, offset, limit, now.Format(time.DateOnly)}
	if page, ok := idx.pages[q]; ok {
		idx.Unlock()
		return page
	}
```
- trigger: `GET /v1/magpie/usage/requests?period=last:24h` at 09:00, then the same request at 21:00 while no usage.jsonl append and no session-file change happened in between (`adopt` clears `pages` only when `st.key` changes, request_page.go:1079).
- effect: the 21:00 answer is the 09:00 page verbatim: rows, `Total`, `Sum`, `Through/Direct/Reported` and the hourly `Series` still include calls older than 21:00-24h (e.g. a call at 08:30 the previous day) and stop 12 hours short of "now". The page is stale until midnight (the key's only clock input is `now.Format(time.DateOnly)`), the next log/session write, or a restart. Presets (Today/Week/Month/All) are unaffected because their start really is fixed for the day.
- why: the cache identity omits the window's start. The same file already treats rolling windows as time-dependent everywhere else - `cache` for the cell parts is skipped when `p.Last() != 0` (request_page.go:779) and `topSummary` refuses to cache rolling summaries "whose start moves with the clock though the log hasn't changed" (summary_cache.go:77) - so `idx.pages` is the one cache that treats `p.Last() > 0` as invariant.
- fix: do not reuse `idx.pages`/`flights` when `p.Last() > 0`, or include `p.Since(now)` in `pageKey`.
- evidence: `sed -n '511,519p' internal/usage/request_page.go` -> pageKey holds only `Period, Filter, Offset, Limit, Date`; `grep -n 'p.Last() == 0' internal/usage/request_page.go` -> `779:	} else if p.Last() == 0 {`; `grep -n 'Format("2006-01-02 MST")' internal/usage/request_page.go` -> `668:` (the request-state key's only clock input); `grep -rln 'last:' internal/usage --include='*_test.go'` -> `internal/usage/period_last_test.go` only (parsing), no page-cache test. verified-by: read|grep
- confidence: high

### F3 [P2] `LastSeen` holds the global usage-log mutex across a per-agent 256 KiB file scan
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usage/usage.go:1204
- code:
```go
	mu.Lock()
	defer mu.Unlock()
	f, err := os.Open(Path())
	if err != nil {
		return last
	}
	defer f.Close()
	const tail = 256 << 10
	if st, err := f.Stat(); err == nil && st.Size() > tail {
		f.Seek(st.Size()-tail, 0)
	}
```
- trigger: any Agents/manage-page render that calls `agent.Applied` - it calls `lastSeen(a.ID)` per agent (internal/agent/applied.go:226 and 236) while the gateway serves calls; `mu` is the same mutex `Append` takes for every served call (usage.go:277).
- effect: N agents = N sequential open/seek/read/JSON-decode passes over the last 256 KiB of usage.jsonl, every one of them under `mu`. On a 200 MB log that is ~256 KiB and up to ~1,000 `json.Unmarshal`s per agent (a few ms each on a local disk, hundreds on a network home), so a 20-agent render stalls `usage.Append` - and therefore the accounting write of every in-flight call - for tens of milliseconds per render; concurrent renders queue behind each other and behind appends.
- why: `mu` exists to serialize writers (`Append`, `IngestReported`), but `LastSeen` only reads an append-only file whose partial trailing line it already tolerates (`json.Unmarshal` failures are skipped), so it needs no mutual exclusion. Holding the lock across `os.Open`/`Seek`/`Scan` turns a read into a writer-blocking critical section.
- fix: drop `mu` in `LastSeen` (or read the tail first and take the lock only for the `seen` lookup).
- evidence: `grep -n -e 'func LastSeen(' -e 'const tail = 256 << 10' internal/usage/usage.go` -> `1199`, `1211`; `grep -rn 'lastSeen(' internal/agent` -> `internal/agent/applied.go:226`, `:236`. verified-by: read|grep
- confidence: high

### F4 [P3] A held gateway key's Account filter lists only the accounts on the page being served
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usageapi/usageapi.go:463
- code:
```go
	accounts := map[string]bool{}
	for _, r := range l.Rows {
		accounts[r.Account()] = true
	}
	l.Accounts = slices.DeleteFunc(l.Accounts, func(a Account) bool { return !accounts[a.ID] })
```
- trigger: a shared magpie and a gateway key held to a budget or model list (so `heldKey` is true, gateway/usage_api.go:29-31) whose calls in the period used two subscription accounts, with the second account's only calls beyond row 500; `GET /v1/magpie/usage/requests?offset=0&limit=100`.
- effect: `l.Accounts` is the key-filtered, whole-period list built from the page's cells (`request_page.go:2051`, `out.Accounts = callerGroups(accounts)`), but `OwnPage` re-cuts it with the accounts seen in `l.Rows` - at most `limit` rows of the current page. The Account filter therefore changes as the user pages through their own calls: an account used only later in the period never appears, and an account selected from page 1 disappears when the user pages to 2, while the rows still exist under it.
- why: it conflates "accounts these 100 rows used" with "accounts this caller's calls in the period used". The per-key cut the comment intends is already made by `f.CallerKey = who.KeyID` (gateway/usage_api.go:80) before the page is built, so the whole-period `l.Accounts` is already free of other keys' accounts.
- fix: keep `l.Accounts` as built for the callerKey filter and do not re-cut it by page rows.
- evidence: `grep -n -e 'func OwnPage(' -e 'accounts\[r.Account()\] = true' -e 'l.Accounts = slices.DeleteFunc' internal/usageapi/usageapi.go` -> `444`, `463`, `465`; `grep -n 'f.CallerKey = who.KeyID' internal/gateway/usage_api.go` -> `80`; page build `grep -n 'out.Accounts = callerGroups' internal/usage/request_page.go` -> `2053`. verified-by: read|grep
- confidence: medium

### F5 [P3] Calls with no input/output tokens are priced as free and never counted as unpriced
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usage/block_price.go:274
- code:
```go
		p.Cost = 0
		if got.price != nil && p.Tokens[0]+p.Tokens[1] > 0 {
			p.Flags |= 1
			p.Cost = got.price.CostSplit(int(p.Tokens[0]), int(p.Tokens[1]), int(p.Tokens[2]), int(p.Tokens[3]), int(p.Tokens[5]))
		}
```
- trigger: a row whose only tokens are cache traffic - e.g. a call the client cancelled or the vendor failed after the prompt was written to cache (`in=out=0`, `cache_write=20000`), or a reported record `{"rid":"r1","agent":"x","provider":"anthropic","model":"claude-sonnet-4-5","t":"...","status":200,"in":0,"out":0,"cache_read":100000,"cost":0.03}` posted to `POST /v1/magpie/usage/requests`.
- effect: `p.Cost` stays 0 and `Priced` stays false, and both totals paths skip the row before pricing: `Totals.addFrom` returns at `if r.Input+r.Output == 0` (usage.go:759) before the `Unpriced` counter (usage.go:763), and `ledgerMatched` prices only when `r.Input+r.Output > 0` (ledger.go:452). So the cache read/write cost is missing from the page's Cost, the ledger's Sum and the CSV, and - worse for a reader - the page does not report the call as unpriced either; a reported call's stated `cost` survives on the row (`OwnCost` is restored at request_page.go:324-330) but is in no total, so per-row and period figures disagree.
- why: `in+out > 0` is used as a proxy for "this call has a billable cost", but cache reads and cache writes are billable on their own and `CostSplit` already prices them.
- fix: price when `in+out+cache_read+cache_write > 0` and count `Unpriced` on that same test in addFrom/addRow/ledgerMatched.
- evidence: `grep -n -e 'if p.Flags&rowOwnCost != 0' -e 'if got.price != nil' -e 'CostSplit(int(p.Tokens' internal/usage/block_price.go` -> `267`, `275`, `277`; `grep -n 'if r.Input+r.Output == 0 {' internal/usage/usage.go` -> `759`; `grep -n 'if pr != nil && r.Input+r.Output > 0' internal/usage/ledger.go` -> `452`. verified-by: read|grep
- confidence: medium

## Checked, no defect

- `blockPricer.priceBlock`'s packed token order (`Tokens[0]=in, [1]=out, [2]=cacheRead, [3]=cacheWrite, [4]=reasoning, [5]=cacheWrite1h`) matches `rowChunk.row` (request_page.go:295) and the `CostSplit(in, out, cacheRead, cacheWrite, cacheWrite1h)` argument order in catalog.go:336-341; the per-million division is `float64` and cannot overflow, and 1h writes are clamped to `cacheWrite`.
- `timedRef.newer` (request_page.go:370-381) breaks ties by time, then gateway-vs-session chunk, then path, then order, so two distinct rows are strictly ordered and an offset walk cannot list a row twice or skip one; `newest`'s early break is only taken when the chunk's `Latest` is older than the oldest row already kept.
- `readLogSnapshot` (log_index.go:118-215) resumes only when the inode, size growth and the prefix hash all agree; a truncation, replacement or rewrite falls back to a full parse, and a partial trailing line is re-read on the next pass instead of being half-parsed.
- `IngestReported` (reported.go:98-146) holds `reported.ingest` across both log scans and the write, re-scans under `mu`, dedupes against the log itself and against the batch, and leaves a partially written batch to be re-read (and deduped) after a failed write.
- `prefetchEach` (request_page.go:2273-2328) is consumed exactly once per wanted source by its single caller (request_page.go:985, whose loop reads each wanted source once), so the producer's `window` tokens are all taken and the producer/workers exit; an unwanted source uses the serial `alone` path and holds no token.
- `requestDisk` entries are length-bounded (256 MiB), CRC-32 checked, and read through `readMapped`, which converts a fault from a file cut short under the mapping into `false` rather than a panic (request_disk.go:149-180, 244-262, 304-318; request_disk_map_unix.go:19-45).
- `matchHints`/`chunkPart` reuse is keyed by chunk serial plus a signature over the pairings, actors and compactions (request_cells.go:266-338), so a chunk rebuilt or re-paired is re-read instead of serving stale cells.

## Open questions

- Does any real vendor reply carry cache tokens with `in=out=0` (the trigger for F5's gateway case)? That needs a captured 4xx/5xx or cancelled-stream response from Anthropic/OpenAI whose usage block names cache reads/writes; not run here (read-only, no network).
- How large is F1's blast radius on a shared gateway: does `sharedWith(r)` admit any enabled key (making the request remote), or only loopback plus the managed key? Deciding it needs a running shared instance and `access.Caller` state; not run here.
