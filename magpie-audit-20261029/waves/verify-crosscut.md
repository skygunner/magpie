# Cross-lane verification — concurrency + perf-hot + security-input (release v0.1.1957)

Source: the release tree is tag `v0.1.1957` of the checked-out repo (quotes match this tree at
every cited line; the removed /tmp snapshot was re-materialised from that tag into /tmp/magpie-audit-Pzn4fNp7/src).
The three input files were deleted from /tmp by something outside this task after all nine findings had been
read in full; their content is reproduced from that read. Read-only: no repo file was changed.

## Verdicts

concurrency/F1 DOWNGRADED P3 - load-compare-store on `searchRests` is real, but both loads must land in the few-microsecond window between the two refusals of one key, and the cost of losing is one 30 s retry cycle, not "~120 more refused requests an hour".
concurrency/F2 REJECTED - the quote is exact, but the blocking `h.w.Write(b)` only runs when `h.passing` (fallback.go:1996), and `started()` (fallback.go:2761) counts `h.passing`, so the deadline branch the finding says is starved (fallback.go:2692-2699) would not set `h.slow` or call `h.stop()` in that state anyway; the mutex is the write lock by design and nothing is lost.
concurrency/F3 CONFIRMED P3 - gui/backend.go:96-100 holds `gatewayMu` across `serveGatewayLocked`'s `ServedBy` (700 ms HTTP) and `Taken` (500 ms dial), so a user restart/take-over waits up to ~1.2 s per tick.
perf-hot/F1 CONFIRMED P2 - `codexFiles` goes through `heldFiles` (one walk per request, memo only inside a hold) whenever `quiet()` is false, and the `held` middleware makes it false for every `/api/usage` poll, so `readRequestState`'s change key re-walks all Codex homes on every poll.
perf-hot/F2 CONFIRMED P3 - two `json.Marshal` of the price tables, a third copy through `fmt.Sprintf`, one stat pair per provider and a `SessionGlob` per desktop root/kind, all before any cache lookup; cost is data-dependent but the work is per poll.
security-input/F1 CONFIRMED P2 - every request to a ruled group inserts one `turnRules` entry keyed by group+session+first words, and eviction only fires above 4096 *for entries older than 24 h*, so the map and the per-request full scan under `turnRules.Lock()` are both caller-sized.
security-input/F2 CONFIRMED P3 - `webGuard`'s sign-in POST compares the key with no failure bucket at all while the page port can be bound to 0.0.0.0 (`web-page.json` mode `lan`) and serves the whole `/api`.
security-input/F3 CONFIRMED P3 - `ownCallback` returns true before `refuseNotAllowed` and before the key check whenever `forwarding()`, so during a hand-over the allow list and the key are both skipped for `POST /_magpie/claude-mcp/<token>`, which `forwardCallback` relays over the predecessor's socket.
security-input/F4 CONFIRMED P3 - `.local` is trusted without resolving (so it is dialled direct, bypassing the user's proxy) and any unseen name costs up to `lanWait` = 400 ms on the caller's goroutine before a 256-entry wholesale cache clear; the image guard's public-address requirement narrows the direct-dial part to proxy bypass.

## Corrected findings

### perf-hot/F1 [P2] Every Requests-page rebuild re-walks all Codex homes and lstats every rollout; the listing cache exists only for the warm-up

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/sessions/sessions.go:705 (code block 705-716; line ref was correct)
- code:
```go
func codexFiles() []file {
	if !quiet() {
		return heldFiles("codex", listCodexFiles)
	}
	warmListing.Lock()
	defer warmListing.Unlock()
	if warmListing.files != nil && time.Since(warmListing.at) < warmListingFor {
		return slices.Clone(warmListing.files)
	}
	files := listCodexFiles()
	warmListing.at, warmListing.files = time.Now(), slices.Clone(files)
	return files
}
```
- trigger: `GET /api/usage/requests` (gui/usage.go:179) -> `usage.QueryPage` -> `queryPageOf` -> `readRequestState` (usage/request_page.go:638-639) -> `sessions.CallSourcesAndCodexIdentities` (sessions/calls.go:1187-1188) -> `codexFiles`. The `held` middleware (gui/api.go:2456-2461) runs `sessions.Ask()` for every `/api/usage` path, so `quiet()` (`warming.Load() > 0 && asking.Load() == 0`, sessions/quiet.go:52) is false for exactly the polling requests. `heldFiles` answers from its memo only while a listing hold is open (`if held.n == 0 { return list() }`, sessions/one_listing.go:74), and the middleware opens none, so each poll falls through to `listCodexFiles` -> `codexFilesIn` -> `walkLinked` (filepath.WalkDir lstat of every rollout under `sessions` and `archived_sessions` of every Codex home).
- effect: one full walk of every Codex home per `/api/usage/requests` poll, before any parsing. `readRequestState` runs the walk even when the page answer itself is cached: it is called first, and the local page cache (`idx.pages[q]`) is consulted afterwards (request_page.go:736-745). The price is the one the file itself records for the same walk: "~12k lstat()s and ~45 MB a time" (sessions.go:718-723, #3493). The Requests page can therefore never be cheaper than one full listing of local history per poll.
- why: the only cross-request listing cache (`warmListing`, 10 s TTL) is scoped to `quiet()`, i.e. to the warm-up, while the readers that poll are the callers excluded from it; the per-request memo (`heldFiles`) is scoped to a hold the `/api/usage` path never opens. Freshness is the stated reason for the scope, but the walk is also the page's change key, so the freshness it buys is paid in full on every poll rather than amortised.
- fix: let `codexFiles` use the 10 s `warmListing` memo outside `quiet()` too (the TTL is what bounds staleness for the page's change key), or open a listing hold around the `/api/usage` read so `heldFiles` covers the request.
- evidence: read of sessions.go:705-731 (`quiet()` branch, `warmListingFor = 10 * time.Second` at 731, cost comment 718-723); `sed -n '1,60p' internal/sessions/one_listing.go` (`held.n`, `OneListing`, `ShareListing`, `heldFiles` memo since `held.since`); `sed -n '2440,2470p' internal/gui/api.go` (`sessions.Ask()` deferred for `/api/sessions` and `/api/usage`); `sed -n '730,752p' internal/usage/request_page.go` (`st := readRequestState(now)` before the page-cache lookup). verified-by: read
- confidence: high on the mechanism and on the per-poll walk; the 12k/45 MB are the tree's own measurement, not re-measured here.

### security-input/F1 [P2] Routing-turn table is unbounded for a day and re-scanned on every request

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/rules.go:348 (insert 348, eviction 349-353; line ref was correct)
- code:
```go
	turnRules.m[key] = turnRule{turn: turn, use: hit.Use, n: hit.N, at: now, input: input, intent: q.Intent, effort: hit.Pick, wanted: hit.Wanted, answered: answered, size: size, hour: hour}
	if len(turnRules.m) > 4096 {
		for k, tr := range turnRules.m {
			if now.Sub(tr.at) > stickKeep {
				delete(turnRules.m, k)
			}
		}
	}
	return hit
```
- trigger: a group with rules (`ruleFor` returns early for `!g.Ruled()`, rules.go:178-180) reachable by any caller the gateway answers (a local agent, or any enabled key while shared). Send N requests with a fresh `X-Magpie-Session` and a distinct first user message each: `ruleKey` is `group|conversation|firstWords` (rules.go:132-135), so every request is a new map key and line 348 inserts a new entry.
- effect: entries younger than `stickKeep` (24 h, affinity.go:71) are never deleted, so the map grows one entry (~250-300 B with the key string) per distinct session/first-message for a whole day, with no ceiling. Past 4096 entries every request walks the entire map at 349-353 while holding `turnRules.Lock()` — the same mutex taken at 192, 256 and 430 — so a caller that creates 200k sessions makes every later request, from anyone, iterate 200k entries under one global lock, and keeps ~50 MB alive for the day.
- why: the cap is a threshold for a scan, not a bound: nothing evicts by count, nothing evicts before insert, and the eviction predicate (`older than 24 h`) cannot remove anything a burst just added. The cleanup is itself on the request path under the hot lock, so a caller-sized table converts into caller-sized latency for every other caller.
- fix: bound by count — on insert, when `len(turnRules.m) >= cap`, delete oldest entries (the map already carries `at`) or the inserted key's own predecessor, without a whole-map walk per request; a periodic sweep outside the request path is the alternative.
- evidence: `rg -n 'turnRules\.' internal/gateway` (no _test) -> 125 declaration (`var turnRules = struct{ sync.Mutex; m map[string]turnRule }`), 192/256/342/430 locks, 348 insert, 349-353 the only delete; inserts also happen at 273 and 279 (the within-turn path) with no cap check at all, which widens, not narrows, the growth path; `rg -n 'stickKeep =' internal/gateway/affinity.go` -> `71: stickKeep = 24 * time.Hour`; `sed -n '103,123p' internal/gateway/rules.go` (struct fields, 5 strings + 4 times/ints). verified-by: read+grep
- confidence: high on growth and on the O(n)-under-lock scan; the per-entry ~250-300 B is arithmetic on the struct, not measured.

### concurrency/F3 [P3] The GUI's gateway watchdog holds the process-wide gateway mutex across a dial and an HTTP probe

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/backend.go:96 (block 96-100; line ref was correct)
- code:
```go
		gatewayMu.Lock()
		if ctx.Err() == nil && served.Load() == nil && !gateway.Running() {
			serveGatewayLocked()
		}
		gatewayMu.Unlock()
```
- trigger: this process serves no gateway (`served.Load() == nil`) and the gateway port is held by a program that does not answer as magpie, or by a magpie too busy to answer in time. Every 15 s (`gatewayWatch`) the watchdog enters `serveGatewayLocked`, which (outside hand-over, backend.go:132) calls `gateway.ServedBy()` (`&http.Client{Timeout: 700ms}` GET, gateway.go:327-329) and then `gateway.Taken()` (`net.DialTimeout` 500 ms, gateway.go:351-352) while `gatewayMu` is held. A user's restart (`gui/gateway_takeover.go:82-99`) or a port change (`gui/port.go:123` -> `serveGateway`) takes the same mutex.
- effect: the restart/take-over call waits behind up to ~1.2 s of probe I/O it does not depend on, once per watchdog tick that overlaps it. The GUI action is delayed, not wrong; with no squatter the probes return in microseconds and nothing is noticed.
- why: the mutex protects in-process state (`served`, `servedRun`) but is held across outbound network I/O to a peer that may be slow or hostile, so an unrelated control path inherits that peer's latency.
- fix: check `served`/`gateway.Running()` under `gatewayMu`, release it, probe, then re-take the lock and install the server only if it is still needed.
- evidence: read of gui/backend.go:87-167 (`watchGateway` block at 96-100; `serveGateway`/`serveGatewayLocked` 121-167; `ServedBy`+`Taken` calls at 133 and 143); `rg -n 'func ServedBy' -A20 internal/gateway/gateway.go` (700 ms client) and `func Taken` (500 ms dial); `rg -n 'serveGateway\(\)|serveGatewayLocked\(\)|gatewayMu' internal/gui` -> backend.go:43/96/122, gateway_takeover.go:65/82/99, port.go:123. verified-by: read+grep
- confidence: high on the lock/IO overlap; the exposure is a user-visible delay of ~1.2 s at worst, hence P3.

### perf-hot/F2 [P3] The Requests page's change key is built by JSON-marshalling the whole price tables and globbing desktop folders on every rebuild

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usage/request_page.go:649 (block 649-661; line ref was correct)
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
		for _, kind := range []string{"local-agent-mode-sessions", "claude-code-sessions"} {
			files, _ := sessions.SessionGlob(filepath.Join(root, kind, "*", "*", "local_*.json"))
```
- trigger: any `/api/usage/requests` read: `queryPageOf` calls `readRequestState` before it looks at its own page cache (request_page.go:736-745), so the key is rebuilt on every poll, cached page or not.
- effect: two fresh byte slices from the price tables and a third copy inside `fmt.Sprintf` (`%s` of a `[]byte` allocates), one `os.Stat` pair per provider (`provider.All()` plus `catalog.LivePath(current.ID)`), and one `SessionGlob` (two-level directory scan plus a stat per match) per desktop root per kind, all to derive one hash. The table cost scales with `Settings.ModelPrices` (usually small or nil) and the glob cost with the number of Claude-desktop session files; neither is benchmarked in the tree, so this is per-poll work of data-dependent size, not a measured regression.
- why: the change key is meant to be a cheap fingerprint of what the page read, but it marshals a settings table and scans directories itself, and the glob result is thrown away even though the rebuild that follows needs the same paths.
- fix: hash the settings' own change key (`provider.UsageRenames().Key()` already does this for renames) instead of `json.Marshal(ModelPrices/ModelWires)`, and reuse the globbed path list from the same call rather than recomputing it.
- evidence: `rg -n 'prices, _ := json.Marshal|DesktopDataDirs()|SessionGlob' internal/usage` -> request_page.go:649, 650, 657, 659 (and session_provider.go:47-50 for the same glob); read of request_page.go:638-700 (the whole key build) and 730-745 (`readRequestState` before the page-cache lookup). verified-by: read
- confidence: medium-high on the work performed per poll; magnitude unmeasured, and `ModelPrices` is empty for most installs.

### security-input/F2 [P3] Page key has no failure accounting: unlimited guesses against the one credential for every setting

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/web.go:238 (block 238-247; line ref was correct)
- code:
```go
		if r.URL.Path == webSignInPath && r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(rw, r.Body, 4096)
			back := localPath(r.PostFormValue("back"))
			if !same(r.PostFormValue("key")) {
				webTurnedAway(rw, r, back, "That key is not this magpie web's.")
				return
			}
			signIn(rw, r, back)
			return
		}
```
- trigger: the page port reachable off-host — `web-page.json` mode `lan` binds `0.0.0.0` (gui/apppage.go:62-69) — or any tunnel/forward to it. Then repeated `POST /_magpie/sign-in` with `back=/` and a guessed `key`. `same` is `subtle.ConstantTimeCompare` (web.go:201-202) and nothing counts failures; a wrong guess is answered by `webTurnedAway` (401 with a form explaining why), a right one sets the cookie and the whole `/api` is open (`Handler(webHost{quit}, startBackend())` is what the page port serves, web.go:108; everything but gateway client paths stays behind the page key, web.go:421-436).
- effect: guesses are unlimited at HTTP speed and stateless per peer. The key is unguessable when random (`NewWebKey` = 32 hex chars), but a user-set `MAGPIE_WEB_KEY` may be as short as 16 characters of `[A-Za-z0-9-._~]` (`webKeyMin = 16`, web.go:147-149) — for a human-chosen key of that length, offline-speed guessing is feasible, and a hit yields every provider credential, every gateway key and every setting.
- why: the page key is the only credential for the whole admin API, and its check has constant-time comparison but no bucket, while the gateway refuses to stay an authentication oracle once failures pile up (`tooManyAuthFails`, gateway/auththrottle.go:37, used at lan.go:502-508). The page port, which is the one meant to be exposed (`lan` mode), has no such ceiling.
- fix: reuse the gateway's failed-auth bucket keyed by peer address in `webGuard`, answering 429 without comparing once over.
- evidence: read of gui/web.go:201-280 (`same`, sign-in branch at 238, no counter anywhere in `webGuard`); `rg -n 'throttle|rate.?limit|tooManyAuthFails|Retry-After' internal/gui --glob '!*_test.go'` -> no Go matches (only assets/app.js and i18n.js UI strings); `rg -n 'webKeyMin' internal/gui/web.go` -> 149; read of gui/apppage.go:53-75 (`lan` -> `0.0.0.0`) and gui/web.go:400-440 (`onePort` sends everything non-client to the key-guarded page). verified-by: read+grep
- confidence: medium (the missing bucket is certain; exploitability rests on a weak user-chosen key).

### security-input/F3 [P3] Allow list and key requirement are skipped on the Claude-helper path while a predecessor is set

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/lan.go:398 (ownCallback 398-427; forwarding branch 410-412; line ref was correct)
- code:
```go
func ownCallback(r *http.Request) bool {
	if r.Method != http.MethodPost || r.URL.RawPath != "" {
		return false
	}
	token, ok := strings.CutPrefix(r.URL.Path, claudeMCPPath)
	if !ok || token == "" {
		return false
	}
	// while the magpie this one took the gateway over from finishes, a
	// token this one doesn't have may be one of that one's runs: it is
	// passed on there (forwardCallback), and that magpie judges it as it
	// would its own
	if forwarding() {
		return true
	}
```
- trigger: `ForwardCallbacks` has set `predecessorCallbacks.sock` (hand-over, handover.go:301-303; cleared only when the socket disappears, 313-317), and this gateway accepts non-loopback connections (shared, or `MAGPIE_ADDR` on an interface address). Then any peer may `POST /_magpie/claude-mcp/<any non-empty token>`, even one the allow list excludes: `lanGuard` calls `ownCallback` first and returns straight into the handler (lan.go:316-319), before `refuseNotAllowed` (328), before the `remote && !remoteKeyed()` refusal (337-349), and before `identifyCaller` (353).
- effect: for the life of the hand-over window the allow list and the key requirement are both bypassed for one route family; each such POST is relayed over the predecessor's unix socket (`forwardCallback`, handover.go:307-341). The predecessor still judges tokens, so the observable is a token oracle from excluded addresses ("unknown or expired Claude run" 404 vs a tool result) plus unauthenticated work on the predecessor's single-threaded callback socket.
- why: the exemption exists because the helper's token is unknown to this process, but it is applied before the address check, which has nothing to do with tokens — the reason for skipping the key does not justify skipping the allow list.
- fix: run `refuseNotAllowed` before returning true for a forwarded callback, keeping only the key requirement skipped.
- evidence: read of lan.go:314-363 (order: ownCallback at 316 -> takePagePort/fromWSL -> refuseNotAllowed at 328 -> remote/rebound -> identifyCaller at 353) and 386-430 (`ownCallback`, `forwarding()` at 410); read of handover.go:295-348 (`ForwardCallbacks`, `forwardCallback`'s unix dial and the 404 error handler, `forwarding`). verified-by: read
- confidence: high on the path and the ordering; medium on impact, because the predecessor re-checks tokens.

### security-input/F4 [P3] "Is this host on my network?" costs a blocking DNS lookup on the request path, and `.local` is trusted without resolving

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/netproxy/lan.go:57 (`.local` shortcut 57-59, lookup 66-69, wholesale clear 75-77; line ref was correct)
- code:
```go
	if strings.HasSuffix(host, ".local") {
		return true
	}
	lanCache.Lock()
	s, ok := lanCache.m[host]
	lanCache.Unlock()
	if ok && time.Since(s.at) < lanKeep {
		return s.lan
	}
	ctx, cancel := context.WithTimeout(context.Background(), lanWait)
	addrs, err := lanLookup(ctx, host)
	cancel()
```
- trigger: a request whose content names hosts magpie then fetches — image URLs in a caller's body (`publicTransport.RoundTrip` at image_fetch_guard.go:195 asks `netproxy.Func` at 205) — with names never seen in the last minute. `netproxy.Func` calls `onLAN` (netproxy.go:71), so each unseen non-`.local` name waits up to `lanWait` (400 ms, lan.go:30) on the caller's goroutine, and a `*.local` name takes the early `return true` and is dialled direct (`netproxy.Func` returns a nil proxy when `onLAN`, netproxy.go:70-72).
- effect: attacker-chosen image hosts add up to 400 ms of resolve each on the request's own goroutine, and the cache is cleared wholesale at 256 entries (75-77), so a handful of images flushes it and every later name pays again; the user's resolver also sees the caller's names. A `.local` name is never resolved for the proxy decision and is fetched direct, bypassing the user's egress proxy. The `.local` risk is bounded by the image guard itself: it resolves the name and requires every address to be public/allowed (image_fetch_guard.go:207-232), so a private answer is refused — what remains is proxy bypass and the caller's chosen name reaching magpie's direct dial rather than the proxy.
- why: the trust decision "this is the user's own network" is made from the name alone for `.local`, and from a synchronous resolver call inside the request path otherwise, so both the decision and its cost are influenced by data in the request body.
- fix: resolve `.local` like any other name and require every answer to be private/link-local (and keep the cache bounded per entry rather than clearing it), so an unseen name cannot stall a request for 400 ms.
- evidence: read of netproxy/lan.go:1-88 in full (`lanWait = 400 ms` at 30, `.local` at 57, lookup at 66-69, clear at 76); read of netproxy/netproxy.go:60-75 (`Func`: `onLAN` -> nil proxy); read of image_fetch_guard.go:196-235 (`netproxy.Func` at 205, addresses resolved at 207-215, `allAllowed` refusal 230-232 with `allowFor` at 87 and `allAllowed` at 294). verified-by: read+grep
- confidence: high on the code; medium on severity, since the fetch guard still requires a public answer.

### concurrency/F1 [P3] `restSearchKey` keeps the wrong rest: read-then-store on a `sync.Map` can shorten an hour's rest to 30 seconds

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/search_keys.go:82 (function 80-86; line ref was correct; downgraded from P2)
- code:
```go
func restSearchKey(vendor, key string, d time.Duration, why string, now time.Time) {
	r := keyRest{Until: now.Add(d), Why: why}
	if was, ok := searchKeyRest(vendor, key, now); ok && was.Until.After(r.Until) {
		return
	}
	searchRests.Store(searchKeyID(vendor, key), r)
}
```
- trigger: two searches in flight on one key, both refused, with the two `searchKeyRest` loads landing before either `Store`. Reachable because a request's search calls run concurrently (search.go:1150-1183, one goroutine per tool call, `wg.Wait()` after), each running `askSearchAPIKeys` over the same key list (search_api.go:133-160), and because `restSearchKey` is also called from the quota read (search_quota.go:178); with a resting or rotating key list two searches do land on the same key. The 1 h vs 30 s pair needs one refusal to be 402 (`outOfCreditRest`, search_keys.go:55) and the other a 429 with `Retry-After: 30`.
- effect: the later store wins regardless of which rest is longer, so an hour's rest can be replaced by 30 s. The key is then asked again 30 s later instead of resting an hour: one extra retry cycle, and a repeated-402 key is the shape vendors escalate — but only for the requests that land inside the store window, not the "~120 more refused requests an hour" the input claims, since the next refusal restores the long rest.
- why: the documented invariant "a rest it has that ends later is kept" (search_keys.go:78-79) is enforced by a non-atomic load-compare-store; `sync.Map` gives per-operation atomicity only, and no mutex guards `searchRests`.
- fix: keep the later rest atomically — `LoadOrStore` + `CompareAndSwap` retry on the stored `keyRest`, or one `sync.Mutex` around `restSearchKey`/`wakeSearchKey`/`searchKeyRest`.
- evidence: `rg -n 'restSearchKey|searchKeyRest|searchRests' internal/gateway` -> declaration 33, load-compare 82, Store 85, Delete 90, Clear 133, callers search_api.go:153-158 and search_quota.go:156/178/180; read of search_api.go:133-160 (sequential keys within a search, one rest per refusal) and search.go:1154-1183 (concurrent searches, `sync.WaitGroup`). verified-by: read+grep
- confidence: high that the race exists; low that it fires often, which is why the impact is P3.

## Added findings

none

## VERDICT

VERDICT: CONFIRMED 7, DOWNGRADED 1, REJECTED 1, ADDED 0; reviewed 21 files
