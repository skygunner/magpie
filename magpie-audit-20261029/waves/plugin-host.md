# Plugin host, plugin runtime, gateway middleware hooks

## Coverage

- internal/plugin/host.go:1-560 — call/callState/callEvent, `push`, `rejectFrameLocked`, `pop`, `credit`, `watchCtx`, `transitionLocked`/`transition`, `host`, `lineLimit`, `headCost`, `ctrl`/`writeReq`/`Loaded`/`Clashes`.
- internal/plugin/host.go:560-1120 — `OnChange`/`OnChangeNow`/`OnSignIn`/`OnListed`, `telling`/`Told`, `Restart`, `retire`, `busy`, `Running`, `Warm`, `stop`, `hostScript`, `get`/`getLocked`/`start`/`startOn`, `drainStderr`, `read`, `chunkFrame`, `readLine`, `dispatch`.
- internal/plugin/host.go:1120-1972 — `enqueue*`, `fetchLine`, `writerLoop`, `fail`, `credit`/`abort`/`wakeCtrl`/`ctrlLoop`/`pumpCtrl`/`dropCtrl`, `begin`/`beginStream`/`admit`/`releaseSlot`, `call`, `callWithTimeout`, `Fetch`, `headRead`, `body.Read`/`Close`/`giveUp`.
- internal/plugin/gateway.go:1-260 — `Gateway`, `GatewayOnly`, `GatewayContext`, `GatewayStates`, `HasGateway`, `GatewayBegin`, `GatewayRun.Finish`, `Close`.
- internal/plugin/middleware.go:1-75 — `Middleware`, `ListStamp`, `OptionsExample`.
- internal/plugin/store.go:1-1016 — `list`/`Load`/clone, `writeWhole`/`save`, `listStamp`/`checkList`, `IsPath`/`IsGit`/`Name`/`Target`, `Add`/`add`/`Update`/`Remove`/`SetOff`/`Prefer`/`SetOptions`, `lockAuth`, `install`/`bunAdd`/`installAt`/`reinstall`, `mendPackageJSON`, `ensurePi`.
- internal/plugin/providers.go:200-430, 490-630, 660-790, 855-945 — `forgetProviders`, `Providers`, `commitProviders`, `sameList`, `refreshProviders*`, `UseCached`, `Listed`, `Refreshed`, `Answered`, `Settle`, `Cached`, `accountsOf`, `readAuth`/`authRaw`/`parseAuth`, `Finish`, `APIKey`, `SignOut`.
- internal/plugin/bun.go:1-402 — `BunVersion`/`bunSums`, `bunTarget`, `Bun`, `HasBun`, `downloadBun`, `getURLFrom`, `bunCommand`, `wontLoad`.
- internal/plugin/page.go:1-289 — `Window`, `htmlInside`, `inside`, `FileIn`, `SetPageOptions`.
- internal/plugin/tools.go:1-156 — `Tools`, `RunTool`, `noteTools`, `ToolsOn`, `ToolNames`.
- internal/plugin/dist.go:100-210 (+ grep of all archive/index paths) — `readDistIndex`, `parseDistIndex`, `mergeDist`, `distFile`.
- internal/plugin/market.go:430-570 — `npmInfo`, `npmAsk`, `Search`, `searchNPM`.
- internal/plugin/host.js:1320-1480, 2600-2680, 2800-2870 — `gatewayRuns`/`GATEWAY_RUN_LIMIT`/`gatewayBudget`, `closeGateway`, `gatewayBegin`, `gatewayFinish`, `gatewayRequest`, `gatewayStates`, `gatewayRequest`/`fetch` line dispatch.
- internal/pluginrt/pluginrt.go:1-416 — `Register`, `Registered`, `New`, `Reconcile`, `needsGoing`, `missing`/`satisfied`, `apply`, `dispose`, `undo`, `Stop`, `Statuses`, `Active`, `Compatible`.
- internal/pluginrt/config.go:1-410 — `Load`, `Compose`, `merge`, `Settle`, `Set`, `SetConfig`, `SetPreset`, `update`, `write`, `Refresh`, `KeepFresh`, `Running`.
- internal/pluginrt/schema.go:1-192 — `SchemaOf`, `Property.parse`/`check`/`value`, `Validate`.
- internal/gateway/middleware.go:1-158 — `serveAgent`, `triedOn`, `ownAsk`, `middlewareWanted`, `gatewayHooks`, `gatewayQuery`, `gatewayOrigin`.
- internal/gateway/gateway_hooks.go:1-94 — `gatewayResponse`, `gatewayReply`.
- plugins/all/all.go:1-20 — registration list.
- plugins/budget/budget.go:1-47 — `init`/`InUse`, `gwbudget.Limit`.
- plugins/gateway/gateway.go:1-34 — `Settings`, `init`.
- plugins/groups/groups.go:1-30 — `init`/`ServeGroups`.
- plugins/library/library.go:1-20 — `init`/`lib.InUse`.
- plugins/middleware/middleware.go:1-34 — `init`, `gwmw.Use`/`usage.Observe`.
- plugins/otel/otel.go:1-45 — `init`/`InUse`, `otlp.New`.
- plugins/redact/redact.go:1-46 — `init`/`InUse`, `gwmw.Mask`.
- plugins/rotation/rotation.go:1-48 — `init`/`inUse`, `routes.Register`.
- plugins/routing/routing.go:1-22 — `init`.
- plugins/search/search.go:1-49 — `init`/`InUse`, `gwtool.Serve`.
- plugins/sessions/sessions.go:1-30 — `init`/`InUse`.
- plugins/tools/tools.go:1-29 — `init`, `plugin.ToolsOn`.
- plugins/usage/usage.go:1-16 — `init`.
- plugins/warmups/warmups.go:1-41 — `init`/`count`.
- Cross-read for guard checks (not my lane, cited only): internal/gateway/capture.go, internal/gateway/conversations.go, internal/gateway/turndone.go, internal/middleware/run.go, internal/beat/beat.go, internal/filememo/watch.go.

COVERAGE: read 39 files, 10616 non-test LOC in scope; skipped 11 (internal/plugin/{updates,github,gitrev,bunupdate,installcache,agent,resolve}.go and the four systemca*.go files, ~2300 LOC: install/update/registry/CA paths, no request or hook path).

## Findings

### F1 [P2] Request hooks are silently skipped once 128 runs are open, and refusal looks like success to Go
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/plugin/gateway.go:206
- code:
```go
	err = h.call(start, "gatewayRequest", map[string]any{"operation": "begin", "body": json.RawMessage(body), "context": info}, &out)
	if err != nil || out.ID == "" || out.Token == "" || len(out.Body) == 0 || out.Body[0] != '{' || !json.Valid(out.Body) {
		return nil
	}
	// the run ends when its owner finishes or closes it, never when the
	// client goes: Codex hangs up on response.completed, and the reply it
	// got is still the exchange to observe
	return &GatewayRun{Body: out.Body, h: h, id: out.ID, token: out.Token}
```
- trigger: 129 concurrent agent requests while any installed plugin exposes `magpie.hooks`. host.js:1360 (`if (gatewayRuns.size >= GATEWAY_RUN_LIMIT) return null`, limit 128 at host.js:1345) answers the 129th `begin` with `result: null`; Go maps that to `out.ID == ""` and returns nil in the same branch as "the host is down" and "this plugin has no hook".
- effect: the overflow request is served with the original body and no `onResponse` call, with no log line and no counter (`grep -n 'log\.' internal/plugin/gateway.go` is empty). A hook that rewrites the prompt, pins a model or scrubs secrets does not run for that request, and the owning plugin is told nothing. At 200 concurrent requests, ~36% of them bypass their hooks; the only signal is `gatewayStates` counters, which do not count refusals.
- why: "no hook" and "too many runs" are collapsed into one nil return, so a resource limit at the plugin boundary turns into a silent policy gap instead of a per-request error or a fallback.
- fix: return a distinguishable refusal (`errTooManyCalls`-style) from host.js (`{error: …}`) and log/count it in `GatewayBegin` instead of returning nil.
- evidence: `grep -n 'GATEWAY_RUN_LIMIT\|gatewayRuns.size' internal/plugin/host.js` → `1345:const GATEWAY_RUN_LIMIT = 128`, `1360:  if (gatewayRuns.size >= GATEWAY_RUN_LIMIT) return null`; `grep -n 'log\.' internal/plugin/gateway.go` → no output. verified-by: grep
- confidence: high

### F2 [P2] `lockAuth` spins forever when the lock file can be neither created nor removed
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/plugin/store.go:139
- code:
```go
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			return func() { os.Remove(lock) }
		}
		if !os.IsExist(err) && !os.IsPermission(err) {
			return func() {} // no folder to lock in: no file to change either
		}
		if fi, err := os.Stat(lock); err == nil && time.Since(fi.ModTime()) > authLockStale || time.Since(start) > 2*authLockStale {
			os.Remove(lock)
		}
		time.Sleep(5 * time.Millisecond)
	}
```
- trigger: `SignOut` (providers.go:901, called with no host running) on a machine where `plugin-auth.json.lock` cannot be created or removed — a plugins dir left root-owned by an earlier `sudo magpie`, a read-only volume, or EACCES on the lock itself. `os.IsPermission(err)` is true, so the loop neither returns nor breaks; the `os.Remove` at line 148 is unchecked and keeps failing.
- effect: the sign-out never returns (no error, no timeout); the caller (GUI request handler or `magpie` CLI) blocks forever while burning ~200 `open`/`stat`/`remove` syscalls a second, and the pending sign-in/out work behind it never happens.
- why: the retry loop treats a permanent permission failure as "someone else holds the lock" and has no exit path after `2*authLockStale`, and its final escape hatch (`os.Remove`) has its error discarded.
- fix: after `time.Since(start) > 2*authLockStale`, return an error (make `lockAuth` return `(func(), error)`) instead of removing-and-retrying.
- evidence: `grep -n 'func lockAuth' -A 24 internal/plugin/store.go` → loop at 138-151 has exactly two exits (line 142 lock taken, line 145 non-EACCES/non-EEXIST error); `grep -rn 'lockAuth()' internal/plugin` → `SignOut`, `Import`, `Take`, `Restore` all call it directly. verified-by: read
- confidence: high

### F3 [P2] Every request with any gateway hook spools, parses and re-renders the whole reply, even when no hook has `onResponse`
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/gateway_hooks.go:15
- code:
```go
	capture := &conversationWriter{captureResponseWriter: &captureResponseWriter{
		ResponseWriter: w, full: &spool{limit: conversationBodyMax, pattern: "magpie-hook-*"},
	}}
	return capture, func() {
		defer capture.full.discard()
		if streamOf(body) {
			_ = http.NewResponseController(w).Flush()
		}
		output, err := capture.full.read()
```
- trigger: any agent request while a plugin that only implements `onRequest` is installed (a context/recall plugin), or while a hook's `onResponse` never runs because `gatewayFinish` decided to skip it: Go has no idea whether a response hook exists (grep A below).
- effect: per request the whole reply is written twice to a temp file in $TMPDIR (up to `conversationBodyMax` = 8 MiB, conversations.go:19) and read back whole into memory; `gatewayReply` then re-parses the entire SSE stream (`readSSE` + per-event `decode` + `finish` + `render`) and `Finish` clones it again — just to hand bytes to a child that discards them. On the request goroutine, before the handler returns: ~8 MiB disk write + ~16 MiB of transient heap + full SSE re-render per request, for zero effect.
- why: the capture path is gated on "the plugin list has any `magpie.hooks` file" instead of "some loaded hook has a response hook", so the expensive half of the exchange runs unconditionally.
- fix: have `gatewayRequest{operation:"begin"}` report whether any participating hook has `onResponse`, and skip `gatewayResponse`/`gatewayReply`/`Finish` when it does not.
- evidence: `grep -rn 'onResponse' --include=*.go .` → no match under internal/plugin or internal/gateway (only unrelated `FunctionResponse` in internal/sessions); `grep -n 'conversationBodyMax' internal/gateway/*.go` → `conversations.go:19:const conversationBodyMax = 8 << 20`. verified-by: grep
- confidence: high

### F4 [P2] A reply over 8 MiB is reported to the hooks as a failed exchange with no body
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/gateway_hooks.go:27
- code:
```go
		output, err := capture.full.read()
		// a client that hangs up once it has the whole reply (Codex, on
		// response.completed) still got it: gatewayReply judges the bytes
		if err != nil || capture.full.cut() || capture.status >= 400 {
			run.Finish(r.Context(), nil, false, body)
			return
		}
		reply, err := gatewayReply(proto, output)
```
- trigger: a successful streamed reply larger than `conversationBodyMax` (8 MiB) — a long tool-heavy turn, an image-carrying answer, a big code dump — with a gateway `onResponse` hook installed. `spool.cut()` is `s.kept < s.size` (capture.go:114), so any reply past the cap takes this branch.
- effect: `onResponse` receives `reply: null` and `success: false` for a request the client received in full and without error. Anything the hook does with success (logging, ledger, retry hints, usage accounting, "did the agent's turn work") sees a failure that never happened; the reply body is dropped silently.
- why: "the body was truncated for storage" is conflated with "the exchange failed", so a storage cap decides the plugin's correctness verdict.
- fix: keep `success` independent of `cut()` (compute it from status and the bytes kept), and pass the truncation as its own flag.
- evidence: `grep -n 'func (s \*spool) cut' -A 2 internal/gateway/capture.go` → `114:func (s *spool) cut() bool { return s.kept < s.size }`; `grep -n 'conversationBodyMax =' internal/gateway/conversations.go` → `19:const conversationBodyMax = 8 << 20`. verified-by: read
- confidence: high

### F5 [P3] A plugin that fails `Apply` while wanted is never applied again, even after its config changes
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/pluginrt/pluginrt.go:252
- code:
```go
	for {
		applied := false
		for _, id := range ids {
			p := rt.plugins[id]
			if !p.wanted || p.state == Active || p.state == Failed {
				continue
			}
			if !Compatible(p.m.API) {
				p.state, p.why = Failed, fmt.Sprintf("written for host API %s, this is %s", p.m.API, API)
				continue
```
- trigger: a built-in plugin whose `Apply` returns an error or panics once (e.g. `gwmw.Use`/`usage.Observe` panicking during a partially written `plugins.yml`), then the user corrects the setting with `magpie plugin set-config <id> …` or edits plugins.yml; `KeepFresh`/`Refresh` runs `Reconcile` again with the same `wanted=true`.
- effect: the plugin stays `Failed` (and every point it serves stays unserved, so a fail-closed point such as the key limiter refuses requests) for the life of the process; the window shows the original error while the fixed config sits unused. Only turning the plugin off and on (`Set` twice) or restarting magpie recovers it.
- why: `Failed` is used both as "not applicable" (an API mismatch, permanent) and "failed to start" (retryable on a config change), so `Reconcile` cannot tell the two apart.
- fix: skip only the permanent case (`!Compatible`) and let a `Failed` plugin be re-applied when its composed config changed.
- evidence: `grep -n 'p.state == Failed' internal/pluginrt/pluginrt.go` → `252:` the only occurrence, inside the apply loop that `Reconcile` re-enters on every `Refresh`; `grep -n 'func (rt \*Runtime) apply' -A 20` shows `Failed` set from `Apply`'s error/panic (line 317). verified-by: read
- confidence: medium

## Checked, no defect

- `call.push` / `rejectFrameLocked` (host.go:194-270): a call's queue is bounded by its credit window and every malformed, oversized, empty or over-window frame gives the call up (`errProtocolOverrun`) instead of growing memory; the free `head` frame is capped by `maxHeadEntries`/`maxHeadBytes`, so a hostile child cannot queue unbounded lines.
- `readLine` + `chunkFrame` + `dispatch` (host.go:1115-1200): partial lines are reassembled across `bufio.ErrBufferFull`, a line past `maxWireLine` kills the host instead of allocating, and a final line arriving with `io.EOF` is still dispatched.
- `h.fail` / `writerLoop` / `enqueueLine` (host.go:1270-1420): no caller can be parked forever on a dead child — every wait selects on `h.dead`, `ctx.Done()` or a bounded queue; `fail` drains the writer queue and answers pending write results.
- `call.transitionLocked` (host.go:325-395): one lifecycle under `h.mu`, slot released exactly once (only on entering `callReleased`), and an aborting call keeps its slot and stays registered until its abort write is acknowledged, so a child that stops reading stdin cannot grow the control backlog.
- `body.Read` / `giveUp` (host.go:1560-1660): the consumer does the waiting, nothing in magpie parks on a reader that stopped; credit is returned per byte read plus per frame finished, and context cancellation or `Close` ends the call once via `sync.Once`.
- `pluginrt.apply` / `undo` / `dispose` (pluginrt.go:308-370): a panicking plugin is isolated (recovered into `Failed`), what it registered before the panic is undone in reverse, and a panic during disposal is logged and the rest still disposed.
- `save`/`writeWhole`/`lockAuth` write path (store.go:172-235) and `pluginrt.write` (config.go:352-382): both publish by temp-file + rename (with `lastgood.Keep` and fsync in `steady.WriteFile`), so a reader never sees a half-written plugins.json or plugin-auth.json.
- `page.go inside` (page.go:250-275): plugin-declared files are joined under the plugin folder with `..`/absolute/backslash rejection and `EvalSymlinks` + `Rel` verification, so a page or panel cannot serve a file outside the plugin.
- `downloadBun` (bun.go:290-345): the zip is SHA-256 checked against the release checksums and only the entry whose base name is `bun`/`bun.exe` is extracted to a fixed path, so a hostile archive cannot write elsewhere (no zip-slip).
- `npmAsk` (market.go:440-480): the two goroutines write disjoint variables (`info`, `weekly`) and are joined by a `WaitGroup` before any read, so the fan-out is race-free under `-race`.
- `Compatible`/`SchemaOf`/`Validate` (pluginrt): API ranges are parsed and compared numerically, and settings are type-checked (NaN/Inf, min/max, enum, whole numbers) before `plugins.yml` is written, so `SetConfig` cannot store a value the form cannot draw.

## Open questions

- Whether `GATEWAY_RUN_LIMIT` = 128 is deliberately a fail-open budget or an oversight: no doc or test names it, and `internal/plugin/gateway_test.go` never reaches the cap. Settling it needs a run with 128 in-flight hooked requests against a real host.
- Whether an 11 s `Finish` goroutine per request (gateway.go:238-242) is reachable in a memory-visible way: it copies the whole reply (up to 8 MiB) and the request body, so peak RSS is bounded by reply size × requests finished in the last 11 s. Confirming needs a load run with a slow `onResponse` and a large reply.
