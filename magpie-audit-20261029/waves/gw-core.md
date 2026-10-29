# gw-core: request lifecycle, routing, fallback and the retry loop

Release tree: `/tmp/magpie-audit-Pzn4fNp7/src` (read-only). Lane: `internal/gateway`
lifecycle/routing/fallback/retry. Protocol translation (gw-proto), subscription backends
(gw-subs) and provider account state (prov-core) are out of lane and not reported.

## Coverage

- internal/gateway/gateway.go: 1736-3870 (serve: group/model pick, candidate loop, retry
  classification, hold writer, affinity update, status/usage write), 3957-4145 (attempt),
  4183-4330 (forward, forwardOnce), 4766-4850 / 5055-5100 (passthrough relay and its retry
  loops), 6855-6898 (keepThroughSilence, relayEvents), 7113-7245 (sessionHeaders, conversationID).
  Functions traced: serve, attempt, forward, forwardOnce, passthrough (relay + retry head), conversationID.
- internal/gateway/routing.go: 1018-1315 (restAfter, restAfterMarked, rest clamping),
  1544-1592 (retryAfter, resetsNoted), 1593-1700 (route, weigh, weighRouted, routingOf);
  whole-file function index read. Functions traced: restAfterMarked, retryAfter, route, weigh.
- internal/gateway/fallback.go: 1-92 (candidate, capHold, cooldowns), 400-680 (plan, planGroup,
  planLevel, sinkOwn), 1401-1490 (retryable, lateRests, tooLongUnsaid), 1538-1620 (retry
  constants, passing, withRoom), 1782-1900 & 1930-2015 (holdWriter, Write, pass), 2326-2380 &
  2650-2765 (keepAlive, watch, keepQuiet, started); whole-file function index read.
- internal/gateway/affinity.go: 1-360 (stick, hourCache, sticks state, noteSticks, saveSticks,
  stickOf), 396-486 (strictKeep, strictOnly, stickKey), 487-666 (affine, leastHeld, after),
  667-790 (answered, opened, unanswered, seals). Functions traced: affine, leastHeld, stickOf,
  answered, unanswered, saveSticks.
- internal/gateway/rules.go: 1-200 (RuleHit, turnRule, turnRules, ruleKey, firstWords, ruleFor
  head), 240-360 (turn decision write, classifier ask, prune). Functions traced: ruleFor, firstWords.
- internal/gateway/decide.go: 300-460 (jevLevelled, askJev, systemOne, postDecide); whole-file
  function index read. Functions traced: askJev, jevLevelled, systemOne, postDecide.
- internal/gateway/decide_group.go: 1-317 full (decisionGroupOf, decideTries, decideFailed,
  serveDecisionGroup, serveDecideTries, askDecideTries).
- internal/gateway/groupfast.go: 1-83 full (withFast, fastPick, own/ultra context marks).
- internal/gateway/grouppattern.go: 1-31 full (emptyGroup, emptyGroupError).
- internal/gateway/drawing_route.go: 1-65 full (drawingModel, drawingTurnID, codexDrawingProvider).
- internal/gateway/session_route.go: 1-176 full (sessionRouteOf, sessionRoute, sessionLatest).
- internal/gateway/unrouted.go: 100-175 (heldWriter), 405-530 (watchUnrouted, show, giveUp);
  whole-file function index read. Functions traced: watchUnrouted, giveUp, heldWriter.
- internal/gateway/lan.go: 300-363 (lanGuard), 425-491 (rebound, ownHost, callerGuard),
  493-520 (identifyCaller), 634-666 (local, proxied, trustProxy); whole-file grep of auth paths.
- internal/gateway/desktop.go: 460-600 (desktopTurn, servedBeside, desktopDefault); whole-file
  index of model-pick and file/state paths.
- internal/gateway/mantle.go: 1-73 full (mantleRoutes, mantleReroute).
- internal/gateway/upstream_transport.go: 1-169 full (upstreamTransport, vendorTransport,
  uploadTransport, drawsApart, replayable) — read as the deadline context of the attempt loop.
- internal/gateway/trace.go: 555-600 (changed, beginFor, begin), plus grep of the routes ring cap
  and wake broadcast. Functions traced: changed, begin.

COVERAGE: read 17 files, 19601 non-test LOC in scope; skipped 0 in-lane files (large files read
by hot path: gateway.go 1736-3870 and selected regions of its 7813 lines; 4 neighbouring files —
classify.go, continuation.go, search.go, codex_backend.go — only grepped, their root causes belong
to other lanes).

## Findings

### F1 [P1] No per-attempt or total deadline: a reply that stalls mid-stream hangs the turn, its lane slot and the client forever
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/gateway.go:2918
- code:
```go
		} else {
			// a stream that fails before anything is said is let go at
			// once (hw.stop), not read on until the vendor hangs up
			ctx, stop := context.WithCancel(withWireAt(r.Context(), at))
			if isGroup {
				ctx = groupTry(ctx)
			}
```
- trigger: an upstream that answers 200 and one event, then keeps the connection open and sends
  nothing more (or sends only SSE comments), while `MaxConcurrency`/`QueueWait` are set on that
  key or account. Same for a non-streaming reply whose body stalls after the headers.
- effect: `attempt` sits in the body read, so the deferred `release()` (gateway.go:2979) never
  runs. The lane slot stays taken; with `MaxConcurrency=1` every other request to that key queues
  and is then turned away 429 with a `Retry-After` (gateway.go:2987-3013) for as long as the
  stalled stream lives. The agent sees a reply that never ends until its own client timeout.
  Keepalives stop at 5 minutes and nothing replaces them: `keepQuiet` returns on
  `time.Since(h.heard) >= keepaliveLongest` with the comment "left for a timeout to end"
  (fallback.go:2729-2731), and no such timeout exists — the only bound in the path is
  `ResponseHeaderTimeout` (10 min), which covers headers alone (upstream_transport.go:65).
- why: the attempt loop derives every try from the client's context with `WithCancel` only, so the
  invariant "a try ends in bounded time" rests on the vendor and on HTTP/2 connection pings, neither
  of which is a deadline for the *reply*. `hw.stop` (the slow-start cancel) is the only deadline the
  loop owns, and it fires before the first content only.
- fix: give each try its own `context.WithTimeout(r.Context(), tryLongest)` (or a body read
  deadline) instead of `WithCancel`, so a stalled reply ends as the next candidate's turn.
- evidence: `rg -n 'context\.WithTimeout|context\.WithDeadline' internal/gateway -g '!*_test.go'`
  → 26 hits across 20 files, none inside `serve`, `attempt`, `passthrough` or `forwardTranslated`
  (gateway.go's hits are lines 703, 900, 980 — drain, quotas, quota cards). verified-by: grep
- confidence: medium

### F2 [P2] The affinity map grows without bound and every write past 4096 walks all of it under the global lock
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/affinity.go:707
- code:
```go
	sticks.m[key] = stick{rest: c.rest, who: c.who(), model: c.model, effort: c.effort, turn: turn, at: now, cacheRead: cacheRead, cacheWrite: cacheWrite, hour: hour}
	delete(sticks.opening, key)
	if len(sticks.m) > 4096 {
		for k, st := range sticks.m {
			if now.Sub(st.at) > stickKeep {
				delete(sticks.m, k)
			}
		}
	}
```
- trigger: a client that varies any of the eight session headers per request
  (`x-opencode-session`, `x-session-affinity`, `x-session-id`, `session_id`, `session-id`,
  `x-claude-code-session-id`, `x-conversation-id`, `x-deepseek-harness-session-id` — gateway.go:7116),
  so `conversationID` (gateway.go:7194) yields a fresh key for every request and `answered` inserts
  a fresh entry.
- effect: `stickKeep` is 24h (affinity.go:71), so the prune only ever deletes entries older than a
  day: distinct session ids seen in 24h all stay resident (~200B each; 10^5 ids ≈ tens of MB,
  10^6 ≈ hundreds). Once over 4096, *every* answered request walks the whole map while holding
  `sticks.Lock`, so N answered requests cost O(N²) comparisons and each answer stalls all other
  requests' affinity lookups (`affine` locks the same mutex, affinity.go:505). `rules.go:349-355`
  repeats the same bounded-by-age prune for `turnRules.m`, keyed by the same client-supplied id.
- why: the invariant "per-conversation state is bounded" is enforced by age, not by count, and the
  eviction work is charged to the next writer on the request path instead of to a background sweep.
- fix: evict by count (drop the oldest past a hard cap, e.g. `sticksKept`/4096) instead of scanning
  for entries older than 24h, so the map is bounded and each write is O(1).
- evidence: `rg -n 'sticks\.m|turnRules\.m|4096' internal/gateway` → the only caps are
  affinity.go:707 (age-based) and rules.go:349 (age-based) plus classify.go:167; nothing bounds
  `len(sticks.m)` by count. verified-by: grep
- confidence: high

### F3 [P2] `stickOf` stats, reads and JSON-decodes affinity.json while the global affinity mutex is held, on the request path
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/affinity.go:256
- code:
```go
func stickOf(key string) (stick, bool) {
	if st, ok := sticks.m[key]; ok {
		return st, true
	}
	path := sticksPath()
	fi, err := os.Stat(path)
	if err != nil || path == sticks.from && fi.ModTime().Equal(sticks.mod) {
		return stick{}, false
```
- trigger: any request whose conversation is not yet in memory — a new session, or the first request
  after a restart/handover — while other requests are in flight. `affine` calls it as
  `sticks.Lock(); st, had := stickOf(key); sticks.Unlock()` (affinity.go:505-507), and
  `unanswered` (affinity.go:733-739) and `strictKeep` (affinity.go:400-402) do the same. The
  `os.Stat` runs on every miss; once the file's mtime moves — `saveSticks` rewrites it within
  `sticksFlushAfter` (2s) of each answered conversation — the miss also pays
  `os.ReadFile` + `decodeJSON` of up to 512 sticks (affinity.go:261-278) under the lock.
- effect: every concurrent request's affinity decision serialises behind disk I/O of another
  request's. On a busy or slow disk this is the failure the gateway itself documents for the write
  side ("sticks and the provider's last answer, saved to disk, took seconds on a busy disk",
  gateway.go:2590-2593): here it is on the read side, on the hot path, under the one global mutex.
- why: the invariant "no blocking I/O under a lock that every request needs" is broken — the file is
  read for one conversation's key while holding the mutex that guards all of them.
- fix: read and decode the file outside `sticks.Lock` (stat/read first, then merge the decoded
  snapshot under the lock), leaving only the map access inside.
- evidence: read affinity.go:249-281 with its caller at 505-507; `rg -n 'sticks\.Lock\(\)' internal/gateway -g '!*_test.go'`
  → 10 sites in 2 files (affinity.go 299, 333, 400, 505, 628, 699, 727, 733; codex_backend.go 397,
  500); the block opened at 505 encloses the `stickOf` call, which does `os.Stat` (256),
  `os.ReadFile` (261) and `decodeJSON` (266). verified-by: read
- confidence: high

## Checked, no defect

- request bodies are never consumed by a retry: the body is a `[]byte` all the way down
  (`serve` → `attempt` → `passthrough`/`forwardTranslated` → `forwardOnce`, which builds a fresh
  `http.NewRequestWithContext` and installs the bytes with `provider.SetBody`, gateway.go:4315-4321),
  so a fallback after a refused attempt re-sends the same bytes, not a drained reader.
- every upstream response body is closed before a retry replaces `res`: forward's 400/403 loop
  (4194, 4219), the format, history-tools and optional-field loops (4783, 4810, 4828, 5001, 5016,
  5042) and the single deferred close of the surviving response (5072); mantleReroute closes the
  first refusal before asking the other prefix (mantle.go:46-48, 59-61).
- re-ask loops in `serve` are each one-shot-guarded, so no unbounded attempt loop exists:
  `reloops < loopReasks` (3136), `repicked < 2` (3140), `resealed < 2` (3150), `updated` /
  `floored` / `roomFor` one-shot flags (3181, 3193, 3203), `creditsAsked[c.rest]` (3543),
  `autoReset` (3680), `small = nil` (3721).
- retry backoff arithmetic is bounded and cannot go negative: `retryPause=1s`, `longestPause=8s`,
  `lastRetries=2`, `rateRetries=3` (fallback.go:1538-1554); `passing` returns ok only while
  `again < lastRetries|rateRetries`, so `retryPause << again` stays ≤ 8s and `again` cannot grow
  past 3 (gateway.go:3453-3678).
- rest durations are clamped on every path: `restAfterMarked` mins `d` to `longestQuota` /
  `longestMonthly` (routing.go:1137) and `retryAfter` to `longestWait` (routing.go:1580), so a
  vendor's bogus `x-ratelimit-reset` (unclamped at routing.go:1544-1550) cannot bench an account
  for years — it is applied only through the clamped path.
- a failing decision model still rests even though `askDecideTries` discards the returned `Rest`
  (decide_group.go:308): `restAfterMarked` writes `restingUntil.m`/`.note` itself
  (routing.go:1282-1284), so no unrested decision model results.
- lanGuard's order holds: allow list → loopback DNS-rebinding check → `remoteKeyed` → key
  identification (lan.go:328-360), with `local()` folding in the page port and forwarding headers
  (lan.go:634-641); the only keyless bypass is the per-run 24-byte callback token compared in
  constant time (lan.go:398-424).

## Open questions

- Whether F1's hang is ever cut short in practice by netproxy's HTTP/2 health ping for vendors on
  HTTP/2: needs a runtime experiment with a stalling upstream and `MaxConcurrency=1` to see whether
  the lane slot is released. Not run (read-only snapshot, no network).
- How large `sticks.m` actually grows in a real deployment (F2's magnitude): needs a soak with
  many distinct session ids; the count-per-day is deployment specific.
