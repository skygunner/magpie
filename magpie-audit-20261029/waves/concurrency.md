# Concurrency audit — Magpie release source (snapshot /tmp/magpie-audit-Pzn4fNp7/src)

Scope: goroutines, locks, channels, timers, contexts in the request path and the
background loops of `internal/gateway`, `internal/provider`, `internal/plugin`,
`internal/usage`, `internal/sessions`, `internal/gui`. Read-only; no file in the
snapshot was modified.

## Coverage

- internal/gateway/concurrency.go 1-405: `lanes.take`, `free`, `releaser`, `grant`, `drop`, `Lanes`, `concurrency`.
- internal/gateway/body_room.go 1-230: `bodyRoomT.take`, `letIn`, `give`, `bodyRoomSize`.
- internal/gateway/search_keys.go 1-150: `restSearchKey`, `searchKeyRest`, `wakeSearchKey`, `searchKeyOrder`, `searchTurns`.
- internal/gateway/rpm.go 1-340: `rpms.reserve`, `prune`, `wait`, `giveBack`, `free`, `rpmTransport.RoundTrip`.
- internal/gateway/patience.go 1-215: `replan`, `waitsOf`, `pauseAlive`.
- internal/gateway/hub_mcp.go 150-270: `hubMCPRun`, `keepHubPruned`, `hubPruneEvery`.
- internal/gateway/gateway.go 650-790, 2930-3030, 6800-6980: serve/shutdown drain goroutine, lane+rpm wait around `attempt`, `keepThroughSilence`, `relayEvents`.
- internal/gateway/fallback.go 1742-1852, 1900-2010, 2570-2700: `holdWriter` fields and lock contract, `writeHeader`, `Write`, `pass`, `keepQueued`, `watch`.
- internal/gateway/active.go 1-200: `inflights.add`, `notePhase`, `noteRoute`, `endBroken`, `Active`.
- internal/gateway/busy.go 1-140: `counted`, `bridgeTurn`, `Busy`, `subscriptionBridge.parked`.
- internal/gateway/sse.go 1-130: `readSSEAlive`, `sseWriter.begin/comment/event/quiet`.
- internal/gateway/session_route.go 100-190: `sessionRoute`, `trace.sessionLatest`.
- internal/gateway/trace.go 850-940: `Trace` wait loop, `traceGather`.
- internal/gateway/claude_room.go 200-290: `claudeRoomT.take`, `nudge`, `leave`.
- internal/gateway/search.go 1120-1200, 1190-1310: search-round fanout (`sync.WaitGroup`), `askTranslated` and its `keepThroughSilence`.
- internal/gateway/continuation.go 500-610: stream hold, `keepThroughSilence` second call site.
- internal/gateway/claude_subscription.go 430-520, 3580-3700: `newSubscriptionBridge`, `bridges` sync.Map, `handedOver`, `continueWith` waiter delivery, `queue`.
- internal/gateway/draw_modelscope.go 20-110: `modelScopeQueues`, `modelScopeTask` polling.
- internal/gateway/wsl.go 235-335: `serveWSL` ticker and per-listener `srv.Serve`.
- internal/gateway/codex_follow.go 110-220: `codexLeads.sideOf` wake channel.
- internal/gateway/image_mcp.go 150-260: `imageMCPStore`, `imageMCPCall`.
- internal/gateway/cpu.go 60-150: `cpuProfile` recording guard and timer.
- internal/gateway/handover.go 230-310: `unreadConns.wait`, `stop`, `ServeCallbacks`, `forwardCallback` lock.
- internal/gateway/routing.go 610-700: `narrowWhenRead`, `allowanceWait`, `narrowing` WaitGroup.
- internal/gateway/codex_backend.go 1275-1345: `stopWatch`, hold writer over the backend stream.
- internal/plugin/host.go 150-450, 700-880, 1450-1580: `call.push/pop/credit/transitionLocked/watchCtx`, `host.retire/stop/busy`, `beginStream`, `admit`, `releaseSlot`.
- internal/plugin/gateway.go 100-275: `GatewayStates`, `HasGateway`, `GatewayBegin`, `GatewayRun.Finish/Close`.
- internal/provider/keepalive.go 1-120: `RenewLogins`, `KeepLoginsAlive`.
- internal/provider/account_relist.go 1-70: `KeepAccountsListed`, `relistAccounts`.
- internal/sessions/sessions.go 1050-1300: `saved`, `writeCache`, `refresh` worker pool, `publish`, `waitParsing`, checkpoint ticker.
- internal/sessions/quiet.go 1-80: `Warm`, `Ask`, `readSlot`, `readBegin`.
- internal/gui/backend.go 50-160: gateway start goroutine, `watchGateway`, `serveGateway`, `serveGatewayLocked`.
- internal/gui/api.go 2615-2705: `sentRows`, `rowsSize`, `writeJSONRows`.
- internal/usage/ledger.go: function index only (`grep -n 'func |Lock()|go func|WriteFile'`), no body read.

COVERAGE: read 33 files, 225219 non-test LOC in scope; skipped 674 of the 707 non-test files in the six packages (budget of 25-40 tool calls: the rest of gateway.go/fallback.go/claude_subscription.go, all of provider/plugin/usage/gui beyond the hot paths above, and ledger.go bodies).

## Findings

### F1 [P2] `restSearchKey` keeps the wrong rest: read-then-store on a `sync.Map` can shorten an hour's rest to 30 seconds

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/search_keys.go:82
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
- trigger: one search API given several keys (`provider.SearchAPI`) and two searches in flight on the same key that are both refused within the same instant — e.g. key K is last in both orders, search A gets 402 (out of credit, `outOfCreditRest` = 1h `search_keys.go:57`) and search B gets 429 with `Retry-After: 30`. Both call `restSearchKey` concurrently; both `searchKeyRest` loads happen before either `Store`.
- effect: the loser of the store race overwrites the hour with 30s. The key is asked again 30 seconds later instead of resting an hour, so a key the vendor said has no credit takes up to ~120 more refused requests an hour; the user sees recurring 402/429 for that API instead of a fast "out of credit" answer, and a repeated-402 key is exactly the shape vendors escalate to a ban.
- why: the "a rest it has that ends later is kept" invariant (`search_keys.go:79-80`) is enforced by a non-atomic load-compare-store; `sync.Map` gives per-op atomicity only. `rg` shows nothing else guards `searchRests` (no mutex, no `CompareAndSwap`).
- fix: keep the later rest atomically — `LoadOrStore` + a `CompareAndSwap` retry loop on the stored `keyRest`, or put a `sync.Mutex` around `restSearchKey`/`wakeSearchKey`/`searchKeyRest`.
- evidence: `rg -n 'searchRests' --glob '!*_test.go' internal` → `search_keys.go:33` (declared `sync.Map`), `:85` Store, `:90` Delete, `:94` Load, `:133` Clear; no other guard in the tree. verified-by: grep+read
- confidence: medium

### F2 [P2] The client write happens under the lock the first-token watchdog needs: a stalled agent freezes failover

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/fallback.go:1998
- code:
```go
	if h.passing {
		h.sent(b)
		return h.w.Write(b)
	}
	n, err := h.held.Write(b)
```
- trigger: a streaming turn through `holdWriter` (`fallback.go:1950-1951` takes `h.mu` and holds it to the end of `Write`) whose agent stops draining its socket — a paused/killed-but-open client, or a receive window that fills while a large reply streams — so `h.w.Write(b)` blocks on TCP backpressure. Any try with `Group.FirstToken` set, or one whose agent must be kept alive (`keepQueued`, `watch`), reaches this state.
- effect: `watch()`'s goroutine takes the same mutex every `watchEvery` (`fallback.go:2677`, `:2697`) and `keepQueued`'s takes it every second, so both block behind the client write. The `firstWait` deadline at `fallback.go:2692` cannot fire: a try that never produced its first token is not let go, the next member of the routing group is never asked, and no SSE keepalive reaches the agent. The agent sees its own idle timeout instead of the failover, and the goroutine stays parked until the client resumes or the connection is torn down.
- why: one mutex is both the state lock and the write lock, so a blocking syscall on an untrusted peer serializes the watchdog that exists to bound exactly that wait; "a deadline that never fires" because the only goroutine able to enforce it is queued on the lock the deadline is meant to interrupt.
- fix: copy what is to be written (and the passing flag) under `h.mu`, unlock, then write to `h.w` outside the lock; or set a write deadline on the response so `h.w.Write` cannot block indefinitely.
- evidence: read of `fallback.go:1950-1952` (`h.mu.Lock()` + `defer h.mu.Unlock()`), `:1998` (`return h.w.Write(b)`), and `:2677`/`:2692`/`:2697` (`watch`'s tick and deadline cases both take `h.mu`). verified-by: read
- confidence: medium

### F3 [P3] The GUI's gateway watchdog holds the process-wide gateway mutex across a dial and an HTTP probe

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/backend.go:96
- code:
```go
		gatewayMu.Lock()
		if ctx.Err() == nil && served.Load() == nil && !gateway.Running() {
			serveGatewayLocked()
		}
		gatewayMu.Unlock()
```
- trigger: another program holds the gateway port and does not answer as magpie (or magpie is starting on a small box), so `serveGatewayLocked` reaches `gateway.ServedBy()` (an HTTP GET) and `gateway.Taken()` (`gateway.go:351-358`, a 500 ms `net.DialTimeout`), both while `gatewayMu` is held; the 15 s watchdog ticks again, and at the same time the user restarts or takes over the gateway (`gui/gateway_takeover.go:99`, `gui/port.go:123`, `backend.go:43` all funnel through `serveGateway`/`serveGatewayLocked`).
- effect: the restart/takeover call blocks behind network I/O it does not need, up to the probe timeouts (0.5 s dial plus the `ServedBy` HTTP timeout) per tick; with the port squatted the GUI's "restart gateway" and port-change paths stall for that long every 15 s window instead of failing or succeeding immediately.
- why: a mutex that protects in-process state (`served`, `servedRun`) is held across outbound network I/O, so a slow or hostile peer on the loopback port serializes unrelated control paths.
- fix: read `served`/`gateway.Running()` under `gatewayMu`, release it, probe outside, then re-take the lock only to install the server if it is still needed.
- evidence: `awk 'NR>=95&&NR<=102' internal/gui/backend.go` shows `gatewayMu.Lock()` … `serveGatewayLocked()` … `Unlock()`; `rg 'func Taken' -A8 internal/gateway/gateway.go` shows the 500 ms `net.DialTimeout` inside it; `rg 'serveGateway\(\)|serveGatewayLocked\(\)' internal/gui` lists the restart/takeover callers. verified-by: read+grep
- confidence: high

## Checked, no defect

- `lanes.take` (`concurrency.go:78-160`): the ctx/timeout give-up path re-locks, searches its own channel in the queue, and correctly distinguishes "still queued" (remove, `drop`) from "granted as it gave up" (`busy--`, `grant`) — no double grant, no lost slot; `grant` closes each waiter's channel exactly once under `l.mu`.
- `rpms.reserve/wait/giveBack` (`rpm.go:110-200`): no `rpms.mu` is held across the `rpmTimer` wait; a cancelled wait calls `stop()` before `giveBack` so the place cannot be both spent and returned, and `prune` keeps the slice ordered because times only grow.
- `bodyRoomT.take/letIn/give` (`body_room.go:110-200`): room is returned by `context.AfterFunc`, a waiter that times out re-checks `w.in` under the mutex before leaving the queue, and the `by` map keys on the request context so a body read twice is counted once.
- `plugin.host` call lifecycle (`host.go:194-420`, `1466-1580`): `transitionLocked` owns cleanup under `h.mu` and always takes `c.mu` after it; `push` bounds the queue by the credit window and refuses an overrun; `beginStream` waits for a window outside `h.mu`; `call` cannot be left waiting because the `h.dead` case returns `errHostGone`.
- `inflights`/`counted` (`active.go:56-86`, `busy.go:82-120`): the counter decrement and `active` deletion run in one deferred function that also re-panics after `endBroken`, so a panicking handler cannot leave a request counted forever; the route list is copied under `f.mu` before use.
- `sessions.refresh` (`sessions.go:1090-1270`): a path is claimed in `parsing` under `mu` before a job is queued and `publish` closes exactly that claim once, so two listings of one path cannot double-close the channel (the #2932 shape); workers only touch their own `jobs[i]`, and the checkpoint ticker is stopped.
- `search.go`'s per-call fanout (`search.go:1152-1180`): the loop goroutines write to distinct indices of `results`/`searches` and `go.mod` declares `go 1.26.8`, so `i` and `c` are per-iteration variables — not the pre-1.22 capture race.
- `keepThroughSilence` (`gateway.go:6855-6910`): both call sites (`search.go:1249`, `continuation.go:547`) call the returned `stop` with the passed lock released, so the helper's `close(done); <-over` cannot deadlock against its own `lock.Lock()`.

## Open questions

- `notePhase` (`active.go:96-105`) saves the previous phase and restores it unconditionally. Phases set by nested defers in one goroutine restore in LIFO order, so I could not construct a trigger where a stale value wins; whether a cross-goroutine phase (a keepalive/watch path calling `notePhase`) exists needs a runtime check of `/v1/magpie/concurrency` output under a request that waits for a slot and then for its minute.
- `hw.keepQueued()` at `gateway.go:2953` is stopped by a plain call at `gateway.go:2969`, not by `defer`. A panic between the two (recovered in `counted`) leaves a 1 s ticker goroutine and its `holdWriter` alive for the process. I could not name a reachable panic in that window, so this is unconfirmed rather than reported as a finding.
- Whether a client that stops reading can actually block `holdWriter.Write` past the agent's idle timeout on this platform (F2's magnitude) needs a socket experiment: a 10 MiB streamed reply to a client with a zero receive window, then observe `magpie goroutines` for the parked `watch` goroutine.
