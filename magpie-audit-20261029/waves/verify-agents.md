## Verdicts

agent-wiring/F1 CONFIRMED - unstash is read-and-delete (stash.go:92-101, delete at 98) and applied.go:774 eats the only copy of Goose's own provider/model before the restore loop's Set at 779; reconnect has the same shape at 831/868.
agent-wiring/F2 DOWNGRADED P3 - the vacuous (k == dshKeyRef || ourKey(v)) guard at dsh.go:1030 and the unstashed overwrite at 1023 are real, but docs/reference.md:240-242 reserves MAGPIE_GATEWAY_KEY and says magpie takes it out on Disconnect, so the loss needs a user key under magpie's own documented name.
agent-wiring/F3 CONFIRMED - appliedSave (applied.go:53-60; the cited 60 is its os.WriteFile line) writes with a bare os.WriteFile and drops both errors, and record calls it once per field, so a torn or failed write reads back as no record for every agent (applied.go:41-47, 206).
agent-wiring/F4 CONFIRMED - appliedMu (applied.go:30) is process-local and applied.json has no filelock (git grep filelock a0ba8e1f2 -- internal/agent is empty), so two magpies over one appdir.Config() lose entries and Drift then skips replaced fields at 206.
agent-runtime/F1 CONFIRMED - wsl.Lock at wsl.go:1094 is held across wslProbe at 1133 (wslRun 30 s, 1429); All() reaches it through wslAgents (agents.go:143), Detected() (agent.go:588) is what /api/state builds from (gui/api.go:2479,2497), and Rescan clears wsl.failed so a retry is immediate (rescan.go:23-27).
agent-runtime/F2 DOWNGRADED P3 - wslRun (wsl.go:1050-1057) is plain proc.CommandContext with no distro-side stop, unlike wslrun.Tool.cancelable/Stop (wslrun.go:364,382, comment at 349), but the stated "profile execs tmux" trigger cannot hang a non-tty sh -lc; only a genuinely stuck command or mount (NFS home, wslinfo) outlives the 30 s, so the leak is real but narrower and rarer.
agent-runtime/F3 DOWNGRADED P3 - LoginPath (path_other.go:89-101) does re-run both 5 s asks on every call inside login.Lock when the shell never answers (at is only set on success), but the GUI caps its retries at 3 (app.js:2409, 4 s apart), so the queue is bounded to ~4 rounds; the effect is rows stuck pending for minutes, not unbounded growth.
agent-runtime/F4 CONFIRMED - daemon.run (daemon.go:392-393) is proc.Command with no context and the package sets no timeout anywhere; it is reached from daemon_cli.go:254/275/293, web_cli.go:94, tui/keep.go:46 and the loginctl in systemd.go:178.
agent-runtime/F5 CONFIRMED - bridge.go:135 round-trips with context.Background() while the 30 s ctx at 106 is used only by CONNECT/DialSOCKS, and the transport sets no ResponseHeaderTimeout, so a parked plain-http reply pins the goroutine plus both sockets.
peripheral/F1 CONFIRMED - catalog.go:776-781 writes models.json in place; loadGen skips a file that does not parse (563-570) and Stale only stats mtime (793-800), so a truncated cache still reads as fresh for 24 h and Source() keeps SyncIfMissing away.
peripheral/F2 CONFIRMED - imagemcp.go:676-680 numbers names until Stat says ErrNotExist; an EACCES/EPERM/EIO answer never does, so the job goroutine (158-163) spins for the life of the server and the paid image is never written.
peripheral/F3 CONFIRMED - settle holds store.mu across MkdirAll/OpenFile/Write/prune (maskrec.go:200-233) and List holds it across every day-file read (440-452); Hold takes the same mutex at 172.
peripheral/F4 CONFIRMED - settings.go:1754 marshals the struct alone and nothing carries unknown keys, so a key this build does not model leaves the live file; the previous bytes are in settings.json.bak (lastgood.Keep at 1748), but nothing merges them back.
peripheral/F5 CONFIRMED - Current holds mu (omarchy.go:380-381) across fontName and two hyprInt calls (390-394); each is bounded by 2 s, not the 1 s the finding says (418-424, 433-440), so the worst stall is ~6 s, not ~4 s.

## Corrected findings

### agent-runtime/F1 [P1] A WSL probe runs while holding the package mutex, so one slow distro freezes every WSL reader
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/wsl.go:1133 (lock at 1094, probe timeout at 1429)
- code:
```go
		if wsl.running[n] && !wsl.probed[n] {
			if t, ok := wsl.failed[n]; !ok || time.Since(t) > wslRetryAge {
				if d := wslProbe(n); d == nil {
```
- trigger: Windows with WSL 2 and a running distro whose probe has not succeeded in this process: the first GUI load after a start (wsl.seen is persisted, wsl.probed is not), a distro restarted since (probed deleted at 1128-1130), the Agents page's Rescan (rescan.go:23-27 clears probed and failed), or the 10 min retry after a failure (wslRetryAge at 985, checked at 1132). wslDistros takes wsl.Lock at 1094 and holds it to the deferred unlock; wslProbe at 1133 runs inside that section and is wslRun(30*time.Second, ...) at 1429.
- effect: every other reader of wsl blocks for up to 30 s per running distro, serially: wslAgents (886) reached from All()/Detected(), wslHomes (924) from copilotWSLHomes (949), wslUp/WSLRunning (1235/1245) from sessions. Detected() is what /api/state builds from (agent.go:588, gui/api.go:2497), so the GUI's state reply can stall ~90 s with three slow distros, and wslSaveLocked (1258) adds file I/O under the same lock.
- why: the probe is an external, I/O-bound call inside the critical section every WSL reader needs, and the loop is serial. The cheaper -l --running call is explicitly cached for this reason (wslRunningAge/wslRunningFailedAge at 992-996, comment #2341), but the 30 s probe has no such protection beyond the 10 min failure retry.
- fix: probe outside the lock (snapshot the distro list under it, release, probe, re-take to store wsl.seen/wsl.probed), or mark a distro in flight so concurrent readers skip it.
- evidence: wsl.go:1093-1167 and 1428-1429 read; rg -n 'wslProbe\(|wslRun\(30|wsl.Lock\(\)' internal/agent/wsl.go -> 1094, 1133, 1429; rg -n 'wslAgents\(\)|wslHomes\(\)' internal/agent -> agents.go:143, wsl.go:886, 924; Detected at agent.go:588; state() at gui/api.go:2479-2497. verified-by: read, grep
- confidence: high

### agent-wiring/F1 [P2] Disconnect deletes its only restore point before the write that needs it
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/applied.go:774 (restore loop 776-783; reconnect 831/868)
- code:
```go
	// a field its own Set left empty goes back to what it was on before
	// Connect (Goose's own provider and model)
	var was map[string]string
	json.Unmarshal([]byte(unstash(a.ID+".connect.was")), &was)
	after := a.Values()
	for _, f := range a.Fields {
		if w := was[f.Key]; w != "" && after[f.Key] == "" && w != magpieID && !magpieValue(a, f, w, was) {
			if err := f.Set(w); err != nil {
```
- trigger: an agent connected while it was on its own model (connect stashes .connect.was at 571; Goose's provider/model has no .before-magpie copy: beforeMagpie is only agents.go:1214 and mothx.go:64), then Disconnect runs and one of its f.Set(w) at 779 fails after the first loop's clears succeeded — the disk filling up (deletes and clears succeed, the restore's create does not), an I/O error on the create, or an atomic rename failing after the old file was removed. A read-only mount is not enough: it fails the first loop too, before the unstash. The reconnect record has the same shape: unstash at 831 is consumed before the Apply loop at 862-868.
- effect: the user's pre-magpie provider/model is gone from the config and from stash.json at once, so a retried Disconnect restores nothing and the agent stays on its default; the .disconnected marker was already stashed at 750, so the row reads as "user disconnected" over a half-restored config.
- why: unstash is read-and-delete (stash.go:92-101), the restore is not inside one edit.Atomically (only each field's Set is), so the restore point's lifetime ends before the work that consumes it succeeds.
- fix: read with stashLoad()[a.ID+".connect.was"], run the loop, and forget the key only after every Set returns nil (wrapping the loop in edit.Atomically over the agent's files).
- evidence: applied.go:769-786, 824-869, 571, 750 read; stash.go:92-101 (delete at 98); rg -n 'before\.take\(\)|beforeMagpie\{' internal/agent -> agents.go:1214/1265, mothx.go:64/111 only. verified-by: read, grep
- confidence: high

### agent-wiring/F3 [P2] applied.json is rewritten with a bare os.WriteFile and both of its errors are dropped
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/applied.go:53 (os.WriteFile at 60)
- code:
```go
func appliedSave(m map[string]applied) {
	if err := lastgood.KeepUnparsed(appliedPath(), lastgood.Into[map[string]applied]); err != nil {
		log.Printf("applied.json wasn't written over: %v", err)
		return
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	os.MkdirAll(filepath.Dir(appliedPath()), 0o755)
	os.WriteFile(appliedPath(), b, 0o600)
}
```
- trigger: any Apply/Connect/Pick/Reapply: record (71-86) calls appliedSave once per recorded field (Apply records the changed fields at 116 and the picked one at 119; reconnect at 868), so the file is rewritten several times per user action. The process dies during one write (OOM, power loss — the class #1505 names), or the config dir is full or read-only.
- effect: os.WriteFile truncates before writing, so a crash or ENOSPC can leave the ledger empty or partial; appliedLoad (41-47) unmarshals it into an empty map without an error, and Drift's rec.Fields is then empty, so !ok at 206 skips the replaced check for every agent. The otherwise-green rows stop warning that their config was changed outside magpie, and a failed MkdirAll/WriteFile is indistinguishable from a successful Apply.
- why: the ledger is the one state file magpie owns that is not written atomically (stashWrite uses edit.WriteAtomic at stash.go:64; every config edit goes through edit.WriteAtomic), and both error returns are discarded.
- fix: build the bytes, then return edit.WriteAtomic(appliedPath(), b) and log or propagate the error from record/Apply, as stashWrite does.
- evidence: applied.go:41-47, 53-60, 71-86, 116-119, 200-207 read; rg -n 'appliedPath\(' --glob '!*_test.go' -> only applied.go; stash.go:41-65 for the contrast. verified-by: read, grep
- confidence: high

### agent-runtime/F3 [P2] LoginPath holds the login mutex across both 5 s shell asks and never caches a failed answer
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/proc/path_other.go:89 (shellAsk 5 s at 363; two flag sets at 350-355; OffPath call at cliupdate.go:823)
- code:
```go
func LoginPath() []string {
	login.Lock()
	defer login.Unlock()
	if login.dirs != nil && time.Since(login.at) < time.Minute {
		return login.dirs
	}
	p, vars := shellEnv()
	if p == "" {
		return nil
	}
```
- trigger: a machine whose login shell never prints shellMark — a profile that blocks on a hung network mount, a shell that fails, or an interactive hand-over that also breaks -lc. askShell tries -ilc then -lc, each bounded by shellAsk = 5 s (363), so shellEnv can take 10 s per shell; login.at is only set after a successful answer (95-98), so every call re-probes. Agent.CLI() calls loginPath() for every agent with a Bin (cliupdate.go:823); CLIs runs one goroutine per detected agent (856-885) from /api/agents/cli, which returns after 3 s with pending (gui/api.go:998).
- effect: the N agents' calls serialize behind login.Lock, each paying its own two 5 s asks; a 24-agent machine spends ~4 min of shell probes per round, the handler returns pending every time, and every CLI version/OffPath stays pending for that whole period. Downgraded from the finding's unbounded-queue claim: the GUI retries only 3 times, 4 s apart (app.js:2409), so about four rounds overlap at most; the shell churn and the pending rows are bounded, the stall is not.
- why: an expensive, I/O-bound read is performed with the mutex held and a failed answer is not cached, so failures cost more than successes; UserPath (40-63) shows the pattern that does not block (2 s cap, background probe).
- fix: cache the failure (set login.at on the empty answer, with a short retry age) and compute shellEnv outside the lock so concurrent callers wait on one result.
- evidence: path_other.go:86-101, 197-235, 340-375 read; rg -n 'shellAsk|loginPath|OffPath =' internal/proc/path_other.go internal/agent/cliupdate.go -> 363, 747, 823; gui/api.go:997-1000; app.js:2409 (again < 3, 4000 ms). verified-by: read, grep
- confidence: high

### agent-runtime/F4 [P2] daemon status/install/uninstall run systemctl/launchctl/loginctl with no deadline
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/daemon/daemon.go:392 (the only runner, 392-393)
- code:
```go
func run(name string, args ...string) (string, error) {
	out, err := proc.Command(name, args...).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
```
- trigger: any service-manager call that does not answer — magpie daemon status/start/stop/install (daemon_cli.go:254, 275, 293), the magpie web startup probe (web_cli.go:94, daemon.Get()), the TUI's hand-over at quit (tui/keep.go:46, daemon.Install), or Get()'s lingerAdvice spawning loginctl on every status (systemd.go:178) — on a Linux with a wedged user bus, a stale XDG_RUNTIME_DIR socket, or a macOS launchctl blocked during login.
- effect: the package has no in-process bound: Get()/Install()/Uninstall() wait until the service manager answers or the process is killed, so the CLI sits at its first status line, magpie web never starts serving, and the terminal's exit hand-over blocks. The common Linux wedge is usually ended by the service manager's own D-Bus timeouts, so "forever" is the worst case, not the typical one; nothing in magpie bounds it.
- why: every other external tool on a read path here is bounded (netproxy.readTool 10 s, proc.Version 10 s, wslRun 10/30 s), while the package whose failure mode is "bus wedged" has no deadline at all.
- fix: run through proc.CommandContext with a short timeout (10 s) and report the timeout as a status error.
- evidence: daemon.go:269-282, 392-405 read; git grep -n 'proc.Command' a0ba8e1f2 -- internal/daemon -> daemon.go:393 only; rg -n 'context\.|CommandContext|Timeout' internal/daemon/*.go -> no matches; callers at daemon_cli.go:254/275/293, web_cli.go:94, tui/keep.go:46, systemd.go:178, launchd.go:174. verified-by: read, grep
- confidence: high on the missing bound, medium on how often a user bus wedges

### agent-runtime/F5 [P2] The bridge's plain-HTTP path drops its 30 s deadline, so a stalled peer pins the goroutine and both connections
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/netproxy/bridge.go:135 (deadline created at 106; transport at 125-131)
- code:
```go
	req.RequestURI = ""
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Proxy-Authorization")
	res, err := t.RoundTrip(req.WithContext(context.Background()))
```
- trigger: a SOCKS5 proxy in settings or the system's (Bridge/EnvForBun) and a plain http:// request whose SOCKS connection is accepted but whose origin never sends response headers (a black-holed origin, a proxy that parks requests). The 30 s context from 106 reaches only the CONNECT branch and DialSOCKS; the transport sets neither ResponseHeaderTimeout nor a client timeout, and DialSOCKS clears its deadline after the handshake (158-161).
- effect: bridgeConn blocks in RoundTrip forever: defer c.Close() never runs, the client connection, the SOCKS connection and the goroutine stay allocated, and the client sees a request with no deadline. One leaked set per stalled request, unbounded over time.
- why: the function's own deadline is not propagated to the round trip it performs, so "no request outlives 30 s" holds only for CONNECT; the read deadline at 100 is the request head and is cleared at 105.
- fix: t.RoundTrip(req.WithContext(ctx)) plus ResponseHeaderTimeout (30 s) on the transport.
- evidence: bridge.go:93-163 read; rg -n 'SetDeadline|ResponseHeaderTimeout|WithTimeout' internal/netproxy/bridge.go -> 100/105 (head), 106 (ctx), 181/183/189 (handshake); no ResponseHeaderTimeout anywhere in internal/netproxy. verified-by: read, grep
- confidence: high on the code path; medium on how often an http:// request crosses a SOCKS bridge

### peripheral/F1 [P2] models.dev cache is rewritten in place, so an interrupted sync leaves magpie with no models for up to a day
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/catalog/catalog.go:780 (write path 776-781)
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
- trigger: a catalog sync (hourly KeepFresh, magpie sync, Missing() after an unpriced model) interrupted after os.WriteFile has truncated models.json (the download is read up to 64 MiB) — process killed, power loss (the #1505 zero-filled-file class), or the disk filling mid-write.
- effect: models.json is empty or truncated; loadGen (563-570) skips a file that does not unmarshal, so mdev stays nil: provider model lists empty, PriceOf/PricedBy price nothing (usage shows $0), Provider(id) lists nothing. Stale (793-800) compares mtime only, so the just-truncated file is not stale, and SyncIfMissing/Source (740-747, 541-547) see a file and return without fetching. The empty catalog stands until the 24 h check or a manual magpie sync.
- why: the cache is the only copy of the model list and is written non-atomically, while its sibling labs.json is temp+rename (labs.go), and nothing validates what is read back.
- fix: write the cache with edit.WriteFile-style temp+rename (labs.go), and treat a cache that does not unmarshal into a non-empty map as stale in Stale().
- evidence: catalog.go:536-570, 740-800 read; catalog/fresh.go:17-61; the only payload validation is on the downloaded body (catalog.go:773); rg -n 'lastgood|KeepBad' internal/catalog/*.go -> no matches. verified-by: read, grep
- confidence: high

### peripheral/F2 [P2] image MCP save path numbers names forever when Stat fails for any reason but "not exist"
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/imagemcp/imagemcp.go:677 (loop 676-680)
- code:
```go
	for k := 2; ; k++ {
		if _, err := os.Stat(out); errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		out = filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, k, e))
	}
```
- trigger: generate_image/generate_video with a path inside a directory this process cannot stat — assets/hero.png in a root-owned 0700 folder, an ACL-denied folder, or a disconnected network share: os.Stat returns EACCES/EPERM/EIO, never ErrNotExist.
- effect: the job goroutine (start at 158-163, no context or deadline on run) spins for the life of the MCP server building hero-2.png, hero-3.png, ... while every tools/call answers "still being made" (await's 20 s limit) and the image that was already paid for is never written.
- why: "never written over" is implemented as "keep numbering until Stat says ErrNotExist", with no bound and no branch for any other error, so a failure that will never clear is retried forever.
- fix: return out only when Stat says ErrNotExist, and return the error otherwise: if _, err := os.Stat(out); errors.Is(err, os.ErrNotExist) { return out, nil } else if err != nil { return "", err }.
- evidence: imagemcp.go:158-163, 632-680, 875-878 read; rg -n 'os.Stat\(out\)|errors.Is\(err, os.ErrNotExist\)' internal/imagemcp/imagemcp.go -> 677 only, loop head 676 has no bound. verified-by: read, grep
- confidence: high

### agent-wiring/F2 [P3] dsh .env: magpie overwrites, then deletes, a gateway key it did not write
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/dsh.go:1030 (overwrite at 1023)
- code:
```go
	env := filepath.Join(dir, ".env")
	if on {
		if v, ok := edit.GetEnvFile(env, dshKeyRef); !ok || v != keyAt(gw) {
			if err := edit.SetEnvFile(env, edit.KV{Path: dshKeyRef, Value: keyAt(gw)}); err != nil {
				return err
			}
		}
	}
	for _, k := range []string{dshKeyRef, dshOldKeyRef} {
		// the old name only as magpie left it
		if v, ok := edit.GetEnvFile(env, k); ok && !(k == dshKeyRef && on) && (k == dshKeyRef || ourKey(v)) {
```
- trigger: ~/.dsh/.env holds MAGPIE_GATEWAY_KEY with a value magpie did not write, and dsh is connected (on true: 1023 overwrites it in place, no stash) or disconnected / given one of its own models (on false: 1030 deletes it).
- effect: the value disappears from .env with no copy; a dsh session that pointed at that key fails until it is retyped. The sibling store path does the opposite (dshCreds stashes at 1072, restores at 1091).
- why: for dshKeyRef the delete guard (k == dshKeyRef || ourKey(v)) is always true, while the legacy name keeps ourKey(v); and .env has no stash for a non-magpie value. Downgraded: docs/reference.md:240-242 reserves this exact name — "Keep it under a name of your own (MAGPIE_DS_KEY) ... magpie takes MAGPIE_GATEWAY_KEY out on Disconnect" — so the documented ownership makes this narrow rather than a general overwrite of the user's key; a user's own sharing key belongs under their own name, and only that case keeps the stash asymmetry worth fixing.
- fix: guard dshKeyRef with ourKey(v) as dshOldKeyRef does, and stash a non-magpie value before overwriting it, as dshCreds does.
- evidence: dsh.go:1016-1038, 1054-1099 read; docs/reference.md:236-244; the store delete at 1086 keeps ourKey(v). verified-by: read, grep
- confidence: high on the code path, lower on the trigger being a real user action

### agent-wiring/F4 [P3] applied.json's read-modify-write is serialized only inside one process, though two magpies may share it
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/applied.go:71 (appliedMu at 30; path at 38)
- code:
```go
func record(id, key, v string) {
	appliedMu.Lock()
	defer appliedMu.Unlock()
	m := appliedLoad()
	a := m[id]
	if a.Fields == nil {
		a.Fields = map[string]string{}
	}
	if v == "" {
```
- trigger: a second magpie on another port over the same appdir.Config() (MAGPIE_ADDR, the case elsewhere/theInstalled is written for) applies to one agent — start-up Sync, a Connect, a CLI magpie <agent> <field> — while the first magpie's user picks a model for another agent, or Keep runs from a Disconnect.
- effect: both loaded the same map; the later writer stores its own snapshot, dropping the other's entry. That agent then has no record, so Drift skips every replaced field through !ok at 206 and its row stays green while its config points elsewhere.
- why: appliedMu is package-scoped and the file has no cross-process lock, although the codebase has one for exactly this (internal/filelock.Mutex, used by access, pluginrt, provider/logins). appliedPath is a single path for every process.
- fix: hold a filelock.Mutex on applied.json (or one file per agent) around the load-modify-write.
- evidence: applied.go:30-38, 71-86, 400-410 read; git grep -n filelock a0ba8e1f2 -- internal/agent -> none; filelock/filelock.go:1-80 documents the two-magpie case; git grep -n 'filelock.Mutex' -- internal -> access/access.go:158, pluginrt/config.go:78, provider/logins.go:134. verified-by: read, grep
- confidence: medium (the window needs two processes writing at once)

### agent-runtime/F2 [P3] A timed-out WSL probe leaves the distro-side shell running: only wsl.exe is killed
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/wsl.go:1053 (wslRun 1050-1057; probe call 1429; reach call 1037)
- code:
```go
var wslRun = func(timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := proc.CommandContext(ctx, "wsl.exe", args...)
	// wsl.exe speaks UTF-16 unless told otherwise; either is read
	cmd.Env = append(os.Environ(), "WSL_UTF8=1")
	return cmd.Output()
}
```
- trigger: a probe (sh -lc, 1429) or reach check (sh -c, 1037) whose distro-side command outlives the 30 s / 10 s timeout — a hung NFS home, a wedged wslinfo, a stuck command -v over a dead mount. The wrapper's own nested timeout 10 bounds only the nvm and --version children (1399, 1412), not the login shell.
- effect: proc.CommandContext cancels only the Windows-side wsl.exe and its Windows descendants; the distro-side sh and what it started stay, with no owner. wslDistros retries a failed distro every wslRetryAge = 10 min (985, 1132) for as long as magpie runs, so one stray tree per retry per distro.
- why: cancellation is bound to the Windows process while the work lives in another kernel, unlike wslrun.Tool.cancelable/Stop (wslrun.go:364-390), which runs a MAGPIE_WSL_RUN stop script for exactly this; the release states the rule at wslrun.go:349. Downgraded: the finding's "profile execs tmux/zellij" trigger cannot hang a non-tty sh -lc (tmux fails fast without a tty, and a prompt reads EOF), so only a genuinely stuck command reaches this path.
- fix: give the probe the same marker/stop treatment as wslrun.Tool (MAGPIE_WSL_RUN env and a stop script on cancel), or run it through wslrun.
- evidence: wsl.go:1025-1057, 1129-1160, 1374-1430 read; wslrun.go:340-390; rg -n 'wslRun\(' internal/agent/wsl.go -> 848, 1037, 1367, 1429; retry at 985/1132. verified-by: read, grep
- confidence: medium (the leak needs a Windows host to count surviving sh processes)

### peripheral/F3 [P3] masking-record store keeps its one mutex across file writes and whole-file reads, so the request path waits on the Privacy page
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/maskrec/maskrec.go:218 (lock 200-201; List lock 440-441)
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
- trigger: a request whose masking record settles (settle, from Hold's done) while the Privacy page's List runs: List takes the same store.mu (440) and reads every day file as it goes (read(fs[i].path) at 452), up to 30 files; settle holds the mutex across MkdirAll, OpenFile, Write, Close and possibly prune.
- effect: the two serialize; while a large record set is listed every concurrent request's Hold (172) blocks, and a slow or network-mounted config dir stretches each settle by the file open+append. The GUI's own list/status calls queue behind request settles the same way.
- why: one mutex protects both the in-memory slices and the on-disk work, so I/O latency becomes request latency; SentOf already shows the safe pattern (copy under the lock, decode outside, 395-406).
- fix: marshal and append before taking store.mu (or copy the needed state out and write after unlocking), and have List read day files outside the lock.
- evidence: maskrec.go:158-235, 377-470 read; rg -n 'defer store.mu.Unlock' internal/maskrec/maskrec.go -> 173, 201, 247, 378, 441, 486, 504; callers gateway/masked.go:83 and gui/masking.go:121. verified-by: read, grep
- confidence: medium (the magnitude depends on the record set and the config dir)

### peripheral/F4 [P3] saving settings.json drops every key this build does not model
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/settings/settings.go:1754 (lastgood.Keep at 1748)
- code:
```go
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
```
- trigger: a settings.json holding a key this release does not know — written by a newer magpie or a fork sharing the config dir, or added by hand — followed by any save here (a GUI toggle, SetRedactModel, UseRedactModel, EditUsageAliases).
- effect: the unknown key leaves the live file; going back to the build that wrote it, that setting is at its default. The previous bytes (with the key) are kept in settings.json.bak by lastgood.Keep at 1748, so the value is recoverable by hand, but nothing merges it back and the newer build never reads the .bak.
- why: the file is rewritten as a marshalling of the Settings struct instead of the on-disk JSON with changed keys merged in, so forward-unknown keys cannot survive a round trip; there is no Extra field or custom Marshal/Unmarshal.
- fix: keep unknown top-level keys (an Extra map with custom Marshal/Unmarshal, or read the file into a map, overlay the struct's marshal, re-marshal).
- evidence: settings.go:1542-1780, 1325-1360 read; rg -n 'MarshalJSON|UnmarshalJSON|Extra |unknown key' internal/settings/*.go -> no matches; lastgood.go:112-140 for Keep. verified-by: read, grep
- confidence: high on the mechanism; the trigger needs a newer build's key

### peripheral/F5 [P3] omarchy.Current refreshes the theme inside its mutex, so a font/Hyprland change stalls every look for seconds
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/omarchy/omarchy.go:390 (lock 380-381; fontName 418-424; hyprInt 433-440)
- code:
```go
	if d != desk || cached.Stamp == "" {
		font = fontName()
		rounding, border = hyprInt("decoration:rounding", 0), hyprInt("general:border_size", 2)
		desk = d
	}
	colors, _ := os.ReadFile(filepath.Join(dir, "colors.toml"))
```
- trigger: a change to the files deskFiles() stamps — fontconfig/fonts.conf after omarchy-font-set, or anything in ~/.config/hypr/* (a Hyprland config edit; a theme switch that sets a font changes the former) — while Current is being read (a GUI theme request, a page load, or a second caller at the same moment). The refresh branch runs with mu held (380-381).
- effect: the refresh runs omarchy-font-current (2 s) and two hyprctl getoption calls (2 s each), so the first caller's theme arrives up to ~6 s late and every other caller blocks on the mutex for that whole time even though the cached theme would have answered; the code's own note at 331-336 says hyprctl asked during a reload can stall for seconds. The finding's 1 s/4 s figures are wrong in the small direction: each timeout is 2 s, so the worst stall is ~6 s.
- why: one mutex guards both the cache and the subprocess work, so a refresh that blocks on external tools blocks all readers instead of only the one refreshing.
- fix: read the stamps and do the subprocess work outside the lock, then re-take it and store the result only if the stamp is unchanged.
- evidence: omarchy.go:330-400, 418-451 read; sed/grep -> 380 mu.Lock, 390 branch, 418/422 fontName, 433/437 hyprInt. verified-by: read
- confidence: medium (magnitude not measured on a live Hyprland)

## Added findings

### ADDED 1 [P2] stash()/unstash() drop stashWrite's error, so the only copy of a user value can be lost silently
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/stash.go:78 (stash), 90 (forget), 99 (unstash); stashWrite returns the error at 41-65
- code:
```go
func stash(kv map[string]string) {
	stashMu.Lock()
	defer stashMu.Unlock()
	m := stashLoad()
	for k, v := range kv {
		if v == "" {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	stashWrite(m)
}
```
- trigger: Connect, Apply, a pick, or dshCreds stashes a value magpie is about to overwrite (.connect.was at applied.go:571, a Claude env key, dshStashKey(creds, dshKeyRef) at dsh.go:1072) while the stash cannot be written: config dir full or read-only, the O_EXCL 0600 seed or edit.WriteAtomic failing.
- effect: the restore point is not stored, yet Connect/Apply reports success; Disconnect then restores nothing and leaves the agent on its default (or the user's key replaced by magpie's with no copy). The failure is invisible in the GUI and in the log.
- why: stashWrite returns an error and all three callers call it as a bare statement (78, 90, 99); a failed write is indistinguishable from a successful one, unlike the KeepUnparsed failure inside stashWrite, which is logged and stops the write.
- fix: return the error from stash/unstash/forget and fail the caller that depends on the restore point (Connect, dshCreds), as settings.save already does when lastgood.Keep fails.
- evidence: stash.go:41-65, 67-79, 82-90, 92-101 read; callers applied.go:571, dsh.go:1072, claude.go's env stash; no caller checks a return value. verified-by: read, grep
- confidence: high on the mechanism; medium on the trigger's frequency

### ADDED 2 [P2] stash.json's read-modify-write has no cross-process lock; the CLI and the app can drop each other's saved values
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/stash.go:67 (stashMu at 27, path at 29, load 31-37, write 41-65)
- code:
```go
func unstash(key string) string {
	stashMu.Lock()
	defer stashMu.Unlock()
	m := stashLoad()
	v := m[key]
	if v != "" {
		delete(m, key)
		stashWrite(m)
	}
	return v
}
```
- trigger: the app (a service, or a magpie beside it) writes the stash — Connect at applied.go:571, Apply, dshCreds at dsh.go:1072 — while another process writes it: a terminal's magpie claude default (cli/main.go:662 -> Disconnect -> stash at applied.go:750 and unstash at 774), or a second magpie on another port.
- effect: both read the whole map and the later writer stores its own snapshot, dropping the other's entry. For entries like .connect.was, a stashed user key, or a Claude env value, that entry is the only copy of what magpie overwrote, so Disconnect restores nothing and the value is gone from both the live config and the stash.
- why: stashMu is process-local (27) and the file is read-modify-written with no filelock, although the package has one for exactly this case (internal/filelock, "across the magpies on a computer"), used for providers' logins and plugin config; stashPath is one path for every process sharing appdir.Config().
- fix: hold a filelock.Mutex on stash.json around load-modify-write (or store one file per agent).
- evidence: stash.go:27-101 read; git grep -n filelock a0ba8e1f2 -- internal/agent -> none; filelock/filelock.go:1-80; git grep -n 'filelock.Mutex' -- internal -> access/access.go:158, pluginrt/config.go:78, provider/logins.go:134; cli/main.go:662. verified-by: read, grep
- confidence: high on the mechanism; medium on the window (two processes writing at once)

## VERDICT

(The /tmp/magpie-audit-Pzn4fNp7/src snapshot was moved out from under this run mid-verification; all 14 quotes and line numbers above were re-checked byte-for-byte against the same release content at commit a0ba8e1f2 in the local git object store, which the wave names as its baseline.)

VERDICT: CONFIRMED 11, DOWNGRADED 3, REJECTED 0, ADDED 2; reviewed 37 files
