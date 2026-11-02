# Agent process runtime — magpie v0.1.1957 (lane: agent runtime / WSL / CLI update / aside / proc / netproxy / autostart / daemon)

Read-only audit of the extracted release snapshot. No file under `/tmp/magpie-audit-Pzn4fNp7/src` was modified; no repo-wide build or test was run.

## Coverage

- internal/agent/wsl.go 1-270, 630-710, 990-1300, 1360-1637 — `place`/`distro` path+gateway mapping, `wslDistros`, `wslRunningLocked`, `wslUp`, `wslProbe`, `parseProbe`, `wslSaveLocked`, `wslReach`, `wslRun`, `decodeWSL`.
- internal/agent/cliupdate.go 1-1125 (full) — `howInstalled`, `fromNodeModules`, `installedVersion`, `memo.get`, `latestVersion`, `npmLatest`, `brewLatest`, `cliBin`, `CLI`, `CLIs`, `UpdateCLI`, `runsBroken`, `updateEnv`.
- internal/agent/aside.go 1-80 (full) — `asideIn`, field closures, `asideSet`.
- internal/agent/aside_stage.go 1-61 (full) — `stage`, `asideFieldPath` use.
- internal/agent/aside_account.go 1-150 (full) — `asideAccounts`, `asideChoice`, `asidePicked`, `pickAccount`.
- internal/agent/aside_native.go 1-766 (full) — `asideRead`, `settings`, `reconcileSettings`, `provider`, `connectLocked`, `state`, `apply`, `setSetting`, `planLocked`, `execute`, `executeOffline`.
- internal/agent/apprestart.go 100-385 — `findRestartable`, `AppBehind`, `RestartApp`, `waitGone`, `reopenBundle`.
- internal/agent/apprestart_windows.go 1-127 (full) — `listWinProcs`, `readWinProc`, `closeWinProcs`, `endWinTree`, `launchWin`.
- internal/agent/running_other.go 1-28, internal/agent/running_darwin.go 1-40 — `listRunning` (pgrep/ps, pgrep+sysctl).
- internal/agent/gone.go 1-204 (full) — `cliHeld`, `extensionHere`, `appHere`, `jetBrainsHere`, `zedAgentsHere`.
- internal/agent/rescan.go 1-34 (full) — `Rescan` (drops `wsl.probed`, `proc.Forget`).
- internal/agent/cliupdate_shim.go 1-202 (full) — `shimAt`, `miseShim`, `shimWhich`, `throughShim`, `installedAt`.
- internal/agent/agent.go 300-440 — `Running` (pgrep), `Detected`, `binAt`, `cliHeld` callers.
- internal/proc/proc.go 1-47 (full) — `Command`, `CommandContext`, `waitDelay`.
- internal/proc/tree.go 1-89 (full) — `StartTree`, `Wait`, `Kill`, `Stop`; group helpers `join`, `sweep`.
- internal/proc/tree_other.go 1-75 (full) — `ownGroup` (Setsid), `kill` (repeated group SIGKILL), `term`, `sweep`.
- internal/proc/tree_windows.go 1-64 (full) — job object `join/kill/term/sweep`.
- internal/proc/killtree_windows.go 1-108 (full) — `cancelTree`, `descendants` (pid-reuse check by start time), `endTree`, `killAll`.
- internal/proc/probe.go 1-83 (full) — `ProbeContext`, `probe.kill/end`, `EndProbes`.
- internal/proc/probe_other.go 1-27, internal/proc/probe_windows.go 1-23 — probe `Cancel` group kill, `killWait`.
- internal/proc/proc_other.go 1-15, internal/proc/proc_windows.go 1-32 — `cancelTree`, `hide` (CREATE_NO_WINDOW), `batch`.
- internal/proc/disclaim.go 1-30, internal/proc/disclaim_other.go 1-16 — `CombinedOutputOwn`.
- internal/proc/wrapper.go 1-250 (full) — `Unwrap`, `parseMiseWrapper`, `miseWhich`, `usableProgram`, `findMise`.
- internal/proc/version.go 1-144 (full) — `Version`, `runVersion` (10 s probe), `npmVersion`, `shimVersion`.
- internal/proc/path_other.go 30-150, 205-285, 290-450 — `UserPath` (2 s cap), `LoginPath`, `shellEnv`, `askShell`, `askShellAs` (5 s per flag set), `shellProbe`, `parseShellEnv`.
- internal/proc/look.go 1-147 (full) — `LookPath`, `Look`, `looked`, `lookStamp`.
- internal/proc/listener.go 1-108, internal/proc/listener_linux.go 1-81 — `ListeningOn`, `IsMagpie`, `IsForwarder`, `/proc/net/tcp` scan, `Executable`.
- internal/proc/family.go 1-108 (full) — `EndTrees`, `picks`, `under`.
- internal/proc/list_windows.go 1-91 (full) — Toolhelp `list`, `family`, `commandLine`.
- internal/netproxy/netproxy.go 1-564 (full) — `Install`, `Func`, `With`, `Dispatch`/`transport`, `For`, `Env`/`EnvWith`, `withProxy`, `System`/`refreshSystem`, `Redact`, `noProxy`, `bypassed`.
- internal/netproxy/bridge.go 1-330 (full) — `Bridge`, `bridgeConn`, `DialSOCKS`, `socksHandshake`, `EnvForBun`, `EnvForBunWith`.
- internal/netproxy/v4fallback.go 1-200 (full) — `v4Fallback`, `v6Conn.seen`, `v4Retry`.
- internal/netproxy/unchecked.go 1-100 (full) — `WithUncheckedTLS`, `uncheckedHost`, `uncheckTLS`.
- internal/netproxy/lan.go 1-110 (full) — `onLAN`, `lanIP`, lookup cache.
- internal/netproxy/scutil.go 1-70, internal/netproxy/system_darwin.go 1-5 — `readTool` (10 s), `parseScutil`.
- internal/autostart/autostart.go 1-87 (full) — `Enabled`, `Set`, `Refresh`, `self`.
- internal/autostart/autostart_other.go 1-63 (full) — XDG `.desktop` `enabled/enable/disable`.
- internal/autostart/autostart_windows.go 1-122 (full) — Run key, StartupApproved marker, `enable/disable`.
- internal/autostart/autostart_darwin.go 1-146 (full) — plist `enabled/enable/refresh`, `heldBack`.
- internal/autostart/autostart_sm_darwin.go 1-33 (full) — `SMAppService statusForLegacyURL`.
- internal/daemon/daemon.go 1-420 (full) — `Install`, `Uninstall`, `Start/Stop/Restart`, `Get`, `Web`, `TLS`, `pick`, `environ`, `run`, `private`.
- internal/daemon/systemd.go 1-195 (full) — `unitText`, `unitQuote`, `install/uninstall`, `status`, `reachable`, `lingerAdvice`.
- internal/daemon/launchd.go 1-174 (full) — `plist`, `install`, `unload`, `start/stop/restart`, `status`.
- internal/wslrun/wslrun.go 1-523 (full) — `Run`, `Find`, `In`, `running`, `Up`, `Known`, `Tool.Command/Probe/cancelable/Stop`, `argv`, `Env`, `Linux`.

COVERAGE: read 50 files, 10771 non-test LOC in scope; skipped 143 (109 remaining internal/agent non-test files are per-agent config wiring owned by the agent-wiring lane — claude.go, codex.go, dsh.go, agents.go, applied.go, installed/credential writers — and 34 platform-only proc/netproxy/autostart files, e.g. path_windows.go, listener_darwin.go, disclaim_darwin.go, nodever*.go, batch_windows.go, which hold no process-spawn path in this lane).

## Findings

### F1 [P1] A WSL probe runs while holding the package mutex, so one slow distro freezes every WSL reader
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/wsl.go:1133
- code:
```go
	for _, n := range wsl.names {
		installed[n] = true
		// a distro that stopped is probed again once it runs: WSL may have
		// been restarted meanwhile in another networking mode
		if !wsl.running[n] {
			delete(wsl.probed, n)
		}
		if wsl.running[n] && !wsl.probed[n] {
			if t, ok := wsl.failed[n]; !ok || time.Since(t) > wslRetryAge {
				if d := wslProbe(n); d == nil {
```
- trigger: Windows with WSL 2 and at least one running distro not yet probed in this process — first GUI load after start (or after `wsl.json` is unparsable), a distro restarted since (`wsl.probed` is deleted at 1128-1130), the Agents page refresh (`Rescan`, rescan.go:27, clears `wsl.probed` for all), or a retry every `wslRetryAge` = 10 min after a failed probe (985, 1132). `wslDistros` took `wsl.Lock()` at 1094 and holds it to 1095's deferred unlock, across the loop at 1124-1160.
- effect: every other reader of `wsl` — `wslAgents()` (890) on the GUI's provider state, `wslHomes()` (929) for sessions, and `wslUp()`/`WSLRunning()` (1236, 1245) used by `internal/sessions` — blocks for up to 30 s per running distro, serially: `wslProbe` is `wslRun(30*time.Second, …)` (1429), and a distro whose login shell hangs consumes its whole timeout. With three distros, a `/api/state` that touches WSL can stall ~90 s, and the GUI's own reply holds its WSL rows empty meanwhile. `wslSaveLocked()` (1258) also does file I/O under the same lock.
- why: an I/O-bound external probe is executed inside the critical section that every WSL reader needs, and the loop is serial. The file already protects the cheaper `-l --running` call with `wslRunningAge`/`wslRunningFailedAge` for exactly this reason (984-1000, comment "#2341"), but the 30 s probe is not protected: the invariant "no blocking call while the package lock is held" is broken.
- fix: probe outside the lock (compute the distro list under it, release, run `wslProbe`, then re-take it to store `wsl.seen[n]`/`wsl.probed[n]`), or mark a distro in-flight so concurrent readers skip it.
- evidence: `rg -n 'wslProbe\(|wslRun\(30|wsl.Lock\(\)' internal/agent/wsl.go` -> `1094: wsl.Lock()`, `1133: if d := wslProbe(n); d == nil`, `1429: wslRun(30*time.Second, "-d", name, "-e", "sh", "-lc", wslProbeScript)`; `rg -n 'wslDistros\(\)' internal/` -> only wsl.go:890 (agents) and 929 (homes), both reached from `All()`; verified-by: read|grep
- confidence: high

### F2 [P2] A timed-out WSL probe leaves the distro-side shell running: only wsl.exe is killed
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/wsl.go:1053
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
- trigger: a `wslProbe` (`sh -lc <probe script>`, 1429) or `wslReach` (`sh -c <reach script>`, 1053) whose distro-side shell outlives the timeout — a login profile that blocks (`exec tmux`/zellij auto-start, a prompt, a hung network mount). `proc.CommandContext` cancels only the Windows-side `wsl.exe` and its Windows descendants; the codebase states the platform rule itself at internal/wslrun/wslrun.go:349: "run inside the distro too: killing wsl.exe alone leaves it there".
- effect: the `sh -lc` and whatever the profile started stay in the distro with no owner. `wslDistros` retries the failed distro every `wslRetryAge` (10 min, 985/1132) for as long as magpie runs, so a hung profile accumulates one stray shell tree per retry per distro (the wrapper's own nested `timeout 10` bounds only its `--version` child, not the profile shell). The same leak hits `wslReach`, whose curl is bounded but whose shell is not.
- why: cancellation is bound to the Windows process, while the work being cancelled lives in another kernel/VM; nothing in `wslRun` marks or stops distro-side processes, unlike `wslrun.Tool.cancelable` (wslrun.go:344-360) which runs a `MAGPIE_WSL_RUN` stop script for exactly this.
- fix: give the probe the same marker/stop treatment as `wslrun.Tool` (pass a `MAGPIE_WSL_RUN`-style env var and kill marked processes on cancel), or run it through `wslrun`.
- evidence: `rg -n 'killing wsl.exe alone leaves it there|func \(t Tool\) Stop' internal/wslrun/wslrun.go` -> `349: … killing wsl.exe alone leaves it there …`, `382: func (t Tool) Stop(mark string)`; `rg -n 'wslRetryAge' internal/agent/wsl.go` -> `985: wslRetryAge = 10 * time.Minute`, `1132: … time.Since(t) > wslRetryAge`; verified-by: read|grep
- confidence: medium

### F3 [P2] LoginPath holds the login mutex across up to 10 s of shell probes and never caches a failed answer
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/proc/path_other.go:89
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
- trigger: a machine whose login shell never prints the probe's mark — `askShell` tries `-ilc` then `-lc` at 5 s each (353-372, `shellAsk = 5 * time.Second` at 363), and the code's own comment (347) names the case: a profile that `exec tmux`/zellij, or a shell that fails. `Agent.CLI()` calls `loginPath()` for every agent with a Bin (cliupdate.go:747, 823), and `CLIs` runs one goroutine per agent (cliupdate.go:864-885) on every GUI state build (internal/gui/api.go:998, `agent.CLIs(3*time.Second)`).
- effect: because `at` is never set when `p == ""` (95-98), every call re-runs both shell probes; the mutex makes the ~10 s work serial across agents instead of shared, so the 24 CLI rows take ~4 minutes per poll while the GUI's 3 s budget expires every time — the versions/`OffPath` of every agent stay `pending` indefinitely, and each 3 s poll adds another set of goroutines queued on `login.Lock()`, each of which then re-runs the same 10 s ask: the queue and the shell-process rate grow without bound.
- why: a failed, expensive read is cached by no field (`login.at`/`login.dirs` only on success) and the read is performed with the lock held, so failures cost more than successes — the opposite of `UserPath` (40-63), which caps its wait at `shellWait = 2 s` and never blocks a caller.
- fix: cache the failure (`login.at = time.Now()` on the empty path, with a short retry), and compute `shellEnv()` outside the lock so concurrent callers wait on one result.
- evidence: `rg -n 'shellAsk|func shellEnv|func askShell\(' internal/proc/path_other.go` -> `211`, `353`, `363: var shellAsk = 5 * time.Second`; `rg -n 'loginPath|OffPath =' internal/agent/cliupdate.go` -> `747: var loginPath = proc.LoginPath`, `823: c.OffPath = a.Bin != "" && offPath(bin, loginPath())`; verified-by: read|grep
- confidence: high

### F4 [P2] daemon status/install/uninstall run systemctl/launchctl/loginctl with no deadline
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/daemon/daemon.go:392
- code:
```go
func run(name string, args ...string) (string, error) {
	out, err := proc.Command(name, args...).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return text, err
```
- trigger: any service-manager call that does not answer — `magpie daemon status` / `start` / `stop` / `install` (daemon_cli.go:254, 275, 293), the web CLI's startup probe (`web_cli.go:94`, `daemon.Get()`), the TUI's install offer (`tui/keep.go:46`), or `Get()`'s `lingerAdvice` running `loginctl` (systemd.go:161) — on a Linux whose `systemd --user`/dbus is wedged, a `su`/`sudo` session with a stale bus socket, or a macOS `launchctl print` blocked during login. `proc.Command` is `exec.Command` with no context, no `WaitDelay` (proc.go:26-31).
- effect: `Get()`/`Install()`/`Uninstall()` never return; the CLI hangs at its first status line and the GUI/TUI spinner never ends, holding the request goroutine and the child `systemctl`/`launchctl`/`loginctl` process for the life of magpie (no timeout, no cancellation, no way to retry short of Ctrl-C).
- why: every other external tool on a read path in this tree is bounded — `netproxy.readTool` 10 s (scutil.go:17), `proc.Version` 10 s (version.go:56), `wslRun` 10/30 s (wsl.go:1050) — while the package that talks to the service manager, the one tool whose failure mode is "bus wedged", has no deadline at all.
- fix: run it through `proc.CommandContext` with a short timeout (e.g. 10 s) and report the timeout as a status error.
- evidence: `rg -n 'context\.|CommandContext|Timeout' internal/daemon/*.go` -> no matches (exit 1), i.e. the package never bounds a call; `rg -n 'proc.Command' internal/daemon/daemon.go` -> `393: out, err := proc.Command(name, args...).CombinedOutput()`; verified-by: read|grep
- confidence: high (code path); medium (how often a user bus wedges)

### F5 [P2] The bridge's plain-HTTP path drops its 30 s deadline, so a stalled peer pins the goroutine and both connections forever
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/netproxy/bridge.go:135
- code:
```go
	t := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			return DialSOCKS(ctx, socks, addr)
		},
	}
	defer t.CloseIdleConnections()
	req.RequestURI = ""
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Proxy-Authorization")
	res, err := t.RoundTrip(req.WithContext(context.Background()))
```
- trigger: a SOCKS5 proxy in settings or the system's (`Bridge`/`EnvForBun`, bridge.go:40-80, 300-330) and a plain `http://` request through the bridge whose SOCKS connection is accepted but whose origin never sends response headers (a black-holed origin, a proxy that parks the request). The 30 s context created at 106 is used only by the CONNECT branch (`DialSOCKS(ctx, …)`); the plain branch replaces the request context with `context.Background()`, and the transport sets neither `ResponseHeaderTimeout` nor a client timeout.
- effect: `bridgeConn` hangs in `RoundTrip` forever: `defer c.Close()` never runs, the client connection, the SOCKS connection and the goroutine stay allocated, and the client sees a request with no deadline. One such request per stall, unbounded over time; a proxy that parks many requests turns into unbounded goroutine/socket growth in magpie's process.
- why: the function's own deadline is not propagated to the round trip it performs, so the "no request outlives 30 s" invariant holds only for CONNECT. (`SetReadDeadline` at 100 is the request head only and is cleared at 105.)
- fix: `t.RoundTrip(req.WithContext(ctx))` and set `ResponseHeaderTimeout` (e.g. 30 s) on `t`.
- evidence: `rg -n 'SetDeadline|ResponseHeaderTimeout|WithTimeout' internal/netproxy/bridge.go` -> `100/105` (request head), `106: context.WithTimeout(…30*time.Second)`, `181/183/189` (SOCKS handshake deadline); no response-header timeout anywhere in the package (`rg -n 'ResponseHeaderTimeout' internal/netproxy/` -> only the doc comment at netproxy.go:52); verified-by: read|grep
- confidence: high (code path); medium (needs an http:// request through a socks5 proxy)

## Checked, no defect

- `proc.Tree`/`group.kill` (tree.go 24-89, tree_other.go 33-75): a SIGKILL to `-pgid` is repeated every 5 ms for up to `killFor` (10 s) so a child forked as the signal was sent is not orphaned; `Wait` reaps then sweeps, and `Stop` waits for `waited` or the grace before `Kill`. The PID-reuse window after `Wait` is acknowledged in the struct comment; `kill(-pgid)` is only reached while a leader pid can still own its group.
- `proc.ProbeContext`/`EndProbes` (probe.go 24-83, probe_other.go 12-27): the context cancel joins the process group, `p.end()` is closed when the group is empty, the map entry is removed by `context.AfterFunc`, and `EndProbes`'s two bounded waits (`endWait`, `killWait`) keep shutdown from hanging on a probe that never started.
- `aside_native.apply`/`execute`/`executeOffline` (aside_native.go 467-766): the restore point is saved before the runtime write, each restore verifies `Aside changed %s since the preview`, the offline path re-runs `planLocked` and compares `Revision` before writing, and `edit.Atomically` supplies rollback over settings/models/record; `asideMu` makes each read-modify-write atomic and is not taken by `state()` (so a read never waits on a write).
- `netproxy.Dispatch`/`transport` (netproxy.go 162-250): clones are capped at 32 with `CloseIdleConnections` before eviction, per-provider choices never share connections, `With` replaces (not nests) a choice, and `uncheckTLS` refuses an `https://` proxy whose certificate would be skipped with the provider's.
- `daemon.unitText`/`unitQuote` and `launchd.plist` (systemd.go 30-55, launchd.go 32-60): `\`, `"`, `%` and newlines are escaped, plist values go through `html.EscapeString`, and the file holding the environment (proxy passwords, `MAGPIE_WEB_KEY`) is created 0600 before any byte is written (`private`, daemon.go 410-420).
- `cliupdate.UpdateCLI` (cliupdate.go 968-1060): one update per agent via `updating.LoadOrStore`, a 10 min context bounds the command, success is re-verified by `runsBroken`, memos are forgotten before the version is read again, and a declined Windows elevation prompt is reported as `AdminError` without claiming the command ran.
- `autostart` (all platforms): state is read from the system's own store, `enable` writes a complete record before the system can act on it, and on Windows `Set(true)` clears the Task Manager StartupApproved marker so "on" is not reported while Windows starts nothing.

## Open questions

- Windows/WSL runtime behavior of killing `wsl.exe` for `agent/wsl.go`'s probes: F2 rests on the release's own statement (wslrun.go:349) plus WSL's documented behavior; a Windows host is needed to count surviving `sh` processes after a 30 s probe timeout.
- Whether Go's `exec.Cmd.Wait` waits for a slow `Cancel` callback: `wslrun.Tool.cancelable` runs a 10 s `wsl.exe` stop script inside `Cancel`, so a cancelled WSL run may return up to 10 s after its context ends. Needs a run to confirm the delay on `Output()`/`Wait`.
- `proc.Tree.Kill` on Windows when `AssignProcessToJobObject` failed (tree_windows.go 26-40) falls back to killing the leader alone; whether a real app's helpers survive that needs a Windows run.
