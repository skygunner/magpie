# Agent config wiring — release snapshot v0.1.957

Read-only audit of `/tmp/magpie-audit-Pzn4fNp7/src`: agent registry, Claude Code, Claude Desktop, Codex, DeepSeek Harness, omp, Aside, and magpie's applied/drift state (`internal/agent/applied.go`).

## Coverage

- `internal/agent/applied.go` — read 1-943 (whole file). Traced `appliedPath`, `appliedLoad`, `appliedSave`, `appliedOf`, `record`, `Apply`, `Drift`, `theInstalled`, `bypassed`, `magpieValue`, `groupNamed`, `sameGroup`, `setAsGroup`, `Reapply`, `Keep`, `Wired`, `Pick`, `Connect`/`connect`/`ConnectHow`, `Disconnect`, `reconnect`, `lastJSONLTime`, `wiringOff`.
- `internal/agent/agent.go` — read 1-751 (whole file). Traced `Field`, `Agent`, `Running`, `Detected`, `binAt`, `agentDir`, `Taken`, `hasGoMagic`, `Field`, `Values`, `Picks`, `Find`, `Spell`, `notOffered`, `split`, `atomic`.
- `internal/agent/agents.go` — read 1-250, 246-345, 500-610, 860-990, 1090-1270, 1520-1650. Traced `init`, `Clients`/`All`/`homes`/`builtins`/`local`/`onMachine`, `jsonGet`/`jsonSet`, `usesMagpie`, `prefixed`, `openCodeSpelled`, `pairGet`/`pairSet`, `magpieEffort`, `beforeMagpie.take`/`restore`, `emptyJSON`, `piLike`/`piLikeKeyed`, `openCodeRef*`, `ownGatewayProvider`, `sameGateway`, `homeDir`, `crushAt.setPick`.
- `internal/agent/claude.go` — read 100-360, 460-840, 2217-2317, 2703-2717. Traced `claudeSwitchOn`, `claudeCloud`, `claudeName`, `claudeEffortsFor`, `claudeClamp`, `claudeReadsTop`, `follow`/`followAt`/`claudeLight`, `tierEnv`, `claudeSameModel`, `tierAt`/`tierWith`/`tierEfforts`, `claudeIn` head (`rawEnv`, `env`, `routed`, `gwAt`, `mainModel`, `wroteMain`, `keyOurs`, `own`, `autoOurs`, `staleAbout`/`dropAbout`, `follows`, `get`, `writeBudgets`/`dropBudgets`), `claudeMain`, `claudeTakes`, `claudeModelWord`, `claudeDropModelEffort`, `claudeOwn`/`claudeDated`.
- `internal/agent/codex.go` — read 60-260, 2140-2400, plus 550-630 and 2192-2230. Traced `codexStandIn`, `multiAgentV2OnIn`, `codexMultiAgentV2On`, `codexSubAgentsV1Warning`, `codexEfforts`, `catalogParses`, `writeOwnList`, `codexIn` head, `keptBearerToken`, `codexOwnViaMagpie`/`codexOwnOf`, `codexKeptGateway`, `codexMoveGateway`, `isCodexGatewayOn`, `codexStaleCache`, `codexFailover`, `EndCodexFailover`, `CodexOwnOnly`, `CodexSideSet`, `codexChatGPTKnown`.
- `internal/agent/dsh.go` — read 60-190, 470-630, 860-1100. Traced `dshAt` (dir/`DSH_HOME`, profiles, `Unwire`, `Restore`, field Set), `dshParse`/`dshWrite`/`dshRead`, `dshNode`/`dshNodeLines`/yaml helpers, `dshConfig`, `dshRoutesIn`, `dshSet`, `dshKeepPick`, `dshEnv`, `dshCreds`, `dshCheck`/`dshCheckStripped`.
- `internal/agent/claudedesktop.go` — read 380-680, 780-890. Traced `desktopWired(Any)`, `desktopEntries`, `desktopObject`, `desktopAdoptStash`, `desktopOn`, `desktopSeedImport`, `desktopServers`, `desktopCarryMCP`, `desktopCarryLocale`, `desktopOff`.
- `internal/agent/omp.go` — read 80-210. Traced `ompSplit`, `ompYAMLList`, `ompProfileName`/`ompReserved`/`ompProfileOK`, `ompRoot`, `ompDir`, `omp`, `ompProfiles`, `ompIn`, `ompAt` head.
- `internal/agent/aside_native.go` — read 500-766. Traced `jsString`, `asideOwnOptions`, the Aside field setter, `setSetting`, `plan`/`planLocked`, `execute`, `executeOffline`.
- Supporting reads outside the lane, used only to check callers' contracts: `internal/edit/file.go` 90-210 and 330-395 (`WriteAtomic`, `Atomically`), `internal/edit/json.go` 1-125 (`GetJSON`, `SetJSON`, `setJSONBytes`), `internal/edit/env.go` 1-90, `internal/agent/gatewaykey.go` 1-100, `internal/agent/stash.go` 29-102, `internal/appdir/appdir.go` 226-234, `internal/appdir/env.go` 27-32 and 110-126, `docs/reference.md` 236-248.

COVERAGE: read 9 files, 13514 non-test LOC in scope; skipped ~8170 (the per-agent catalog/notice/CLI bodies of claude.go, codex.go, dsh.go, agents.go, claudedesktop.go, omp.go and aside_native.go, outside the config read/write and drift paths; those regions were covered by rg sweeps for JSON/TOML/YAML writes, backups, env handling, deletion sites and locks).

## Findings

### F1 [P2] Disconnect deletes its only restore point before the write that needs it
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/applied.go:774
- code: ```go
	// a field its own Set left empty goes back to what it was on before
	// Connect (Goose's own provider and model)
	var was map[string]string
	json.Unmarshal([]byte(unstash(a.ID+".connect.was")), &was)
	after := a.Values()
	for _, f := range a.Fields {
		if w := was[f.Key]; w != "" && after[f.Key] == "" && w != magpieID && !magpieValue(a, f, w, was) {
			if err := f.Set(w); err != nil {
```
- trigger: an agent connected while it was on its own model (the comment names Goose's own provider/model, which has no `.before-magpie` copy), then Disconnect runs and any `f.Set(w)` at line 778 fails — config file mode 0444 or a read-only mount, a full disk, or the atomic rename failing. `reconnect` has the same shape at applied.go:831: `unstash(a.ID+".reconnect")` before `a.Apply(k, v)`.
- effect: the user's pre-magpie provider/model is permanently gone. `.connect.was` was already deleted from stash.json, and only `piLike` (agents.go:1265) and `mothx.go:111` ever take a file copy (`before.take()`), so a retried Disconnect restores nothing and the agent stays on its own default. Disconnect also already stashed `<id>.disconnected` before Unwire, so it reads as "user disconnected" with the config half restored.
- why: `unstash` is read-and-delete, so the restore point is consumed before the loop that consumes it succeeds, and the restore loop is not inside `edit.Atomically` either — the one place where the transaction boundary and the data's lifetime disagree.
- fix: read with `stashLoad()[a.ID+".connect.was"]` and `forget` it only after every `f.Set(w)` returns nil (wrap the loop in `edit.Atomically` over the agent's files).
- evidence: `rtk bypass rg -n 'func unstash' -A 10 internal/agent/stash.go` → `delete(m, key)` at stash.go:98 (read-and-delete); `rtk bypass sed -n '755,800p' internal/agent/applied.go` → unstash at 774, writes at 778; `rtk bypass rg -n 'before\.take\(\)|beforeMagpie\{' internal/agent/*.go` → only agents.go:1265 and mothx.go:111. verified-by: read
- confidence: high

### F2 [P2] dsh `.env`: magpie overwrites, then deletes, a gateway key it did not write
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/dsh.go:1030
- code: ```go
	for _, k := range []string{dshKeyRef, dshOldKeyRef} {
		// the old name only as magpie left it
		if v, ok := edit.GetEnvFile(env, k); ok && !(k == dshKeyRef && on) && (k == dshKeyRef || ourKey(v)) {
			if err := edit.DelEnvFile(env, k); err != nil {
				return err
			}
		}
```
- trigger: `~/.dsh/.env` holds `MAGPIE_GATEWAY_KEY` with a key magpie did not write — docs/reference.md:238-242 tells users to put "the sharing key from another one" there. Then magpie applies to dsh here (`on` true: the line is overwritten in place at dsh.go:1022-1025 with no stash), or dsh is disconnected / one of dsh's own models is picked (`on` false: the loop above deletes the line).
- effect: the user's key disappears from `.env` with no copy anywhere, so any dsh session they point at that other magpie fails with "no credential for provider route magpie" until they retype a key they can no longer read. The sibling store path does the opposite: `dshCreds` stashes the user's value (dsh.go:1072) and puts it back on the way out (dsh.go:1091).
- why: for the current key name the delete guard is vacuous — `(k == dshKeyRef || ourKey(v))` is always true when `k == dshKeyRef` — while the legacy name keeps `ourKey(v)`; and the `.env` write path has no stash/restore for a non-magpie value, unlike `.credentials.yaml`.
- fix: guard with `ourKey(v)` for `dshKeyRef` too, and stash a non-magpie value before overwriting it, as `dshCreds` does.
- evidence: `rtk bypass rg -n 'dshKeyRef|MAGPIE_GATEWAY_KEY' --glob '!*_test.go' .` → delete at dsh.go:1030 with no `ourKey` for the new name, versus store delete at dsh.go:1085 behind `ourKey(v)`; `rtk bypass sed -n '1058,1100p' internal/agent/dsh.go` → stash at 1072, restore at 1091. verified-by: grep
- confidence: high

### F3 [P2] applied.json is rewritten with a bare os.WriteFile and both of its errors are dropped
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/applied.go:60
- code: ```go
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
- trigger: any Apply/Connect/Pick/Reapply — `record` calls `appliedSave` once per changed field (up to ~10 for Claude Code, whose Set moves the base URL, token, tiers, context, output, capabilities, auto-mode and picker keys), so the file is rewritten repeatedly on the user path; the process is killed (OOM, power loss) during one of those writes, or the config dir is full/read-only.
- effect: a truncated file makes `appliedLoad` (applied.go:41-47) silently return an empty map, so `Drift` reports no `replaced` drift for *every* agent — the whole ledger, not one entry — until each is re-applied; on a full or read-only dir `record` returns silently and Apply still answers success, so the row stops warning that its config was changed outside magpie.
- why: the ledger is the one state file magpie owns that is not written atomically, while every config file it edits goes through `edit.WriteAtomic`; discarding `MkdirAll`/`WriteFile` errors makes a failed ledger write indistinguishable from a successful Apply.
- fix: `return edit.WriteAtomic(appliedPath(), b)` and log its error, as `stashWrite` does.
- evidence: `rtk bypass rg -n 'appliedPath\(' --glob '!*_test.go' .` → the only writer is applied.go:60; `rtk bypass rg -n 'os\.WriteFile' --glob '!*_test.go' internal/agent/` → the other hits seed a file magpie is the first to create (dsh.go:1078, agents.go:1550, claudedesktop.go:560) while config edits use `edit.WriteAtomic` throughout the package. verified-by: grep
- confidence: high

### F4 [P3] applied.json's read-modify-write is serialized only inside one process, though two magpies may share it
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/agent/applied.go:71
- code: ```go
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
- trigger: a second magpie on another port over the same config dir (`MAGPIE_ADDR`, the case `elsewhere`/`theInstalled` at applied.go:250-273 is written for) applies to one agent — its start-up Sync, a Connect — while the first magpie's user picks a model for a different agent, or `Keep` runs from a Disconnect.
- effect: both processes loaded the same map and the later writer stores its own snapshot, dropping the other's entry. That agent then has no record, so `Drift` skips every field through `if !ok || … { continue }` (applied.go:206) and the row stays green while its config points somewhere else — the exact false "connected" state Drift exists to prevent.
- why: `appliedMu` (applied.go:30) is package-scoped and the file has no lock, so the whole-map load-modify-store is not serialized across the two live processes the rest of the file explicitly supports.
- fix: re-read and merge only this agent's entry under a lock file (`O_EXCL`/flock), or store one file per agent.
- evidence: `rtk bypass rg -n 'flock|Flock|LockFile|O_EXCL' internal/agent/*.go` → no lock primitive, only stash.go:59's O_EXCL seed; `rtk bypass rg -n 'appliedPath\(' --glob '!*_test.go' .` → a single shared path for every process. verified-by: grep
- confidence: medium

## Checked, no defect

- `edit.SetJSON` refuses a corrupt config instead of replacing it: `setJSONBytes` returns `invalid JSON/JSONC` when `json.Valid` fails (internal/edit/json.go:110-111), so a malformed `settings.json`/`opencode.json` fails the Apply rather than being overwritten with `{}`.
- dsh's patch-list parser (`dshParse`, dsh.go:448-495) returns an error for any shape it does not model ("is not a list of entries magpie can edit") and `dshWrite` only writes `[]` over a file that already exists, so an unparsable `cordis.patch.yml` is never replaced by an empty list.
- `writeOwnList` (codex.go:231-236) runs `lastgood.KeepUnparsed(path, catalogParses)` before writing Codex's model list, so keys the user added to a list magpie can't parse are copied aside first.
- Claude Code's env deletions are ownership-checked: `staleAbout`, `windowOurs`, `defaultOurs`, `budgetsOurs` and `autoOurs` (claude.go:670-800) each compare the current value with the stash magpie wrote, stash a user's own value under `claude.env.<key>`, and leave anything that doesn't match alone.
- `codexKeptGateway` (codex.go:2187-2212) keeps a non-default base URL only when it ends at magpie's path on another host or on loopback-with-a-path, and rejects URLs carrying userinfo, query or fragment, so a user's own OpenAI endpoint is not mistaken for magpie's.
- Relative environment values cannot retarget writes: `homeDir`/`ompDir` require `filepath.IsAbs` (agents.go:1100, omp.go:136) and `appdir.LookupEnv` already passes over relative paths (appdir/env.go:27-32, 110-126), which is why `homes()` can use `appdir.Getenv("XDG_CONFIG_HOME")` without its own check.
- Aside's disconnect re-validates before writing: `execute`/`executeOffline` rebuild the plan under `asideMu`, compare `Revision`, and re-read the providers file with `bytes.Equal(current, file.Before)` before replacing it (aside_native.go:686-735), so a concurrent Aside edit aborts the restore instead of being clobbered.
- Multi-file writes roll back: `atomic()` wraps each field `Set`/`Sync`/`Unwire` in `edit.Atomically` (agent.go:733-751), which restores every touched file's bytes and mode when a later step fails (internal/edit/file.go:345-395), so a half-written Codex provider table plus model is not left behind.

## Open questions

- Claude Desktop: `desktopOn` seeds `coworkEgressAllowedHosts: ["*"]` and `disableDeploymentModeChooser: true` once, when the key is absent (claudedesktop.go:571-575). Whether that widens a narrower Desktop default for the Cowork sandbox's egress allowlist needs the app running (it would matter for LAN reachability of a code-running sandbox). No offline check possible.
- F4 needs two live magpies over one config dir to be observed end to end (second instance on another port, simultaneous Apply on two agents); not run here because the snapshot must not be modified and no service was started.
