# Library, sync, backup and update — audit of the 0.1.957 release source

Snapshot: `/tmp/magpie-audit-Pzn4fNp7/src` (read-only). Packages: `internal/library`,
`internal/davsync`, `internal/backup`, `internal/update`.

## Coverage

- `internal/davsync/davsync.go` 1-1260 (full) — Config/Server/state, `Load`, `Configure`, `Check`, `password`/`sameAccount`, `Off`, `View`/`Status`, `loadState`/`saveState`, `run`/`backoff`, `collect`, `hashes`, `take`, `changed`, `bring`, `keepUnseenGatewayKeys`, `syncOnce` (push/merge/blind/rebuild).
- `internal/davsync/dav.go` 1-517 (full) — `version`/`conditions`, `versionOf`, `limited`/`retryAfter`, `newDAV`, `send`/`do`, `forbidden`/`there`, `get`, `put`, `check`/`length`, `mkcol`, `nowhere`/`folders`.
- `internal/davsync/s3.go` 1-336 (full) — `newS3`/`urlOf`, `sendTo`/`unsigned`, `readError`/`explain`, `get`, `put` (conditional + unconditional), `head`, `noConditions`.
- `internal/davsync/github.go` 240-430 (partial) — `contentAt`, `bytes`, `get`, `githubParts` read, `commit` split, `errTooLarge`/`errBehind`, `gitCommit` head.
- `internal/davsync/restore.go` 1-271 (full) — `Restore`, `Upload`, `Undo`, `keepHere`, `SetAuto`, `isFile`.
- `internal/davsync/archive.go` 1-88 (full) — `S3Bucket`, `Bucket.at/Put/Get/Open`.
- `internal/davsync/lock.go` 1-69 (full) — `lock` (file lock + 200 ms retry), `locked`, `wait`.
- `internal/davsync/stall.go` 1-142 (full) — `stallTransport.RoundTrip`, `watch.moved/allSent`, `watched.Read/Close`, `cause`.
- `internal/davsync/notbackup.go` 55-190 (partial) — `described`, `whatIs`, `backupHead`, `fileWhere`, `cmdOf`.
- `internal/backup/backup.go` 1-568 (full) — `isBackupJSON`, Bundle, `Collect`, `Seal`, `Open`, `header`/`aead`/`derived`, `Restore`, `AddPlugins`, `profileKeys`.
- `internal/backup/data.go` 1-99 (full) — `saltFor`, `derived` key cache, `SealData`, `OpenData`.
- `internal/library/library.go` 1-282 (full) — `load`, `save`, `checkName`/`agentRe`, `change`, `changeLocked`, `lock.held`.
- `internal/library/backup.go` 1-124 (full) — `freshBackup`, `backups.keep`, `pruneBackups`, `holdsSkills`.
- `internal/library/carry.go` 20-149, 188-228, 300-439, 520-549 — `Bundle`/`CarriedSkill`, `maxCarried`/`maxCarriedAll`, `Collect`, `readSkills`, `check`, `Put`, `Changed`.
- `internal/library/skills.go` 180-319, 820-1019 (of 2468) — `link`, `markCopy`/`markedHash`, `copyIn`, `oldPath`/`dropOld`, `fetchSkill`, `untar`, `candidates`, `unhiddenFirst`.
- `internal/library/sync.go` 60-189 (of 947) — `sync`, `syncSteps`, `Sync` (per-step lock/memo), `syncMCP` head.
- `internal/library/skill_edits.go` 20-259 (full) — `takeEdits`, `copiedAt`, `changedAt`, `keepLibrarySkill`, `mirrorDir`, `skipInSkill`.
- `internal/update/update.go` 40-159, 240-369, 460-619, 760-1019 (of 1062) — `Feed`, `Release`/`Asset`, `LatestIn`, `Newer`/`comparePre`, `Stage`, `sameSigner`/`team`, `stageDir`, `Install`, `runAfterExit`, `ReplaceBinary`, `StageBinary`, `InstallBinaryAside`, `RemoveStaleNew`, `download`, `fetch`, `fileHash`, `counter`.
- `internal/update/mirror.go` 1-155 (full) — `mirrorOf`, `mirrored`, `CheckMirror`/`CheckMirrorURL`, `notProxy`, `MirrorError`.
- `internal/profile/profile.go` 180-215 (outside the four packages; read only to check `Save`/`Delete` name handling for `backup.Restore`).

COVERAGE: read 20 files, 32239 non-test LOC in scope; skipped 76 (19 of the 95 non-test files were opened, fully or at their hot functions: the rest is library's mcp/hooks/targets/rtk/market/projects/dsh files, update's elevation/COM/CLI-shim/notes/blocked paths, davsync/usage.go, and github.go after line 430).

## Findings

### F1 [P1] A WebDAV/S3 backup over 64 MiB is silently truncated on read and reported as a damaged file

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/davsync/dav.go:291
- code:
```go
	data, err = io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return nil, version{}, err
	}
```
- trigger: the file on the server is a valid sealed backup larger than 64 MiB. Reachable without malice: `library.Collect` carries up to `maxCarriedAll = 32 << 20` of skill files (carry.go:43-51) and every provider icon up to `MaxIcon = 1 << 20` (provider/icon.go:31), each base64'd twice (bundle JSON, then the sealed envelope); with the skill budget spent, ~4 MiB of icons pushes the sealed file past 64 MiB. The writers never check: `syncOnce`'s `push` and `restore.Upload` seal and PUT whatever size they have.
- effect: on every computer except the writer, `backup.Open` sees a 64 MiB prefix, `isBackupJSON` matches, and it returns `ErrCorrupt`; the last-good rebuild then refuses (`cached(st.Sum)` is the full file, so `bytes.HasPrefix(good, data)` fails) and `syncOnce` returns `described(...)`, i.e. "is a magpie backup cut short or damaged (67108864 bytes): a write to it was interrupted. Sync on the computer that wrote it last to rebuild it, or upload this computer's setup over it". `whatIs` sets `Replace: true` (notbackup.go:136), so the only offered way out is Upload, which writes this computer's older setup over the other computer's intact backup. Sync fails again every 3 minutes, forever; the writer keeps working from its cached ETag, so it never notices.
- why: the read path enforces a limit the write path does not, and the truncation is classified as damage to the remote file rather than as a local read limit — so the recommended recovery destroys the file it misdiagnosed.
- fix: read `64<<20+1` and fail with an explicit "the backup is larger than 64 MiB" error (and refuse to push one), as `github.commit` already does.
- evidence: `rg -n '64<<20' internal/davsync/dav.go internal/davsync/s3.go` → `dav.go:291`, `s3.go:234` (both plain read caps); `rg -n 'githubMaxFile' internal/davsync/github.go` → `21:const githubMaxFile = 64 << 20`, `290`, `321`, `385` — GitHub caps the read *and* refuses the write, WebDAV/S3 do neither. verified-by: read
- confidence: high

### F2 [P2] `library.Put` moves the old skills aside before the new ones are in, and never puts them back on error

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/library/carry.go:369
- code:
```go
			return move(p, filepath.Join(aside, name))
		}
		for _, s := range l.Skills {
			if !keep[s.Name] {
				if err := out(s.Name); err != nil {
					return err
				}
			}
		}
		skills := []*Skill{}
```
- trigger: a restore or a sync that brings the library in, failing after the first skill has been moved aside: `move` on a later skill (a Windows file lock is called out in skills.go:250-253 as a real hazard), `os.MkdirAll(aside)` (carry.go:366), `dirLink`/`copyDir` (carry.go:390-392), or `os.Rename(stage/<skill>, skillDir)` (carry.go:397) — disk full, EPERM under a locked `~/.magpie/library/skills`, antivirus.
- effect: `Put` returns the error, and `changeLocked` (library.go:266-275) returns before `l.save()`, so `library.json` still lists every skill while `skillsDir()` no longer holds the ones already moved. The next `library.Collect` walks the missing directories and `readSkills` swallows the walk error (carry.go:129-132), so those skills are carried with `Files: {}`; that bundle is pushed, every other computer's `Put` renames an empty staged folder in place of the real skill, and their agents get it too. A skill lost on one machine becomes an empty skill on all of them, with nothing in the UI saying so (the real folder sits in `~/.magpie/backups/<stamp>/skills`).
- why: the incoming skills are staged, but the outgoing ones are moved away before anything is placed and there is no rollback, so a mid-restore failure turns an unchanged `library.json` into a library whose files are gone — and an unreadable skill folder is indistinguishable from an empty skill.
- fix: record the moves made and, on any error, move them back before returning (a `defer` that rolls back unless the function returned nil); at minimum make `readSkills` report an unreadable skill directory instead of an empty one.
- evidence: read carry.go:350-397 (aside path built at 356, `move` at 369, loop at 371-377, no rollback afterwards) and library.go:266-275 (no save when `f` fails); `sed -n '118,134p' internal/library/carry.go` shows `readSkills` returning an empty `CarriedSkill` for a directory it cannot walk. verified-by: read
- confidence: medium

### F3 [P2] A WebDAV write the server did not check is reported as a checked one, so a lost concurrent write is never reconciled

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/davsync/dav.go:70
- code:
```go
	lm := h.Get("Last-Modified")
	t, err := http.ParseTime(lm)
	if err != nil {
		return version{}
	}
	if d, err := http.ParseTime(h.Get("Date")); err != nil || d.Sub(t) < time.Second {
		return version{}
	}
	return version{Modified: lm}
```
- trigger: a WebDAV server (or proxy/gateway in front of it) that answers GET/PUT without an `ETag` and either without a `Date` header or with a `Last-Modified` inside the same second as `Date` — the gate above turns all of these into `version{}`. `dav.put` then sends no `If-Match` at all (dav.go:319-323), and two computers syncing within the same window both get a 2xx.
- effect: the later PUT silently replaces the earlier one; both machines record their own bundle as the server's state (`st.Sum`, `st.Remote` set from the write in `push`, davsync.go:1097-1100). On the loser's next round the part reads as "unchanged here, changed there", so it is brought in with no `Notice`, and the changes it pushed are reverted locally — the exact loss the `Blind` guard exists to detect.
- why: `version.Blind` is set only by `s3.put`'s HEAD-then-PUT path (s3.go:302); the WebDAV path returns `version{ETag: ...}` with `Blind` false even when the PUT carried no condition at all, so the state machine believes a write-up that was never checked.
- fix: in `dav.put`, when `etag == ""` (no `If-Match` sent), return `version{ETag: ..., Blind: true}` as s3 does.
- evidence: `rg -n 'Blind' internal/davsync/*.go` (non-test) → only `s3.go:302` sets it true, plus the state plumbing at davsync.go:212-214/1098-1100; `sed -n '319,323p' internal/davsync/dav.go` shows the `if etag != "" { h["If-Match"] = etag }` unconditional branch. verified-by: grep
- confidence: medium

### F4 [P2] On Windows and Linux an update binary is trusted on a checksum delivered by the same response that names it

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/update/update.go:993
- code:
```go
	// the whole file, what earlier tries kept included, against the feed's
	if !strings.EqualFold(fileHash(path), a.SHA256) {
		err = fmt.Errorf("%s does not match its checksum", name)
		if u != a.URL {
			err = fmt.Errorf("%s from the mirror %s does not match its checksum from %s; not installed", name, mirror, Site)
		}
		return fail(err)
	}
```
- trigger: any update on a Windows or Linux build (`ReplaceBinary` → `StageBinary` → `download` → `InstallBinary`). The only integrity input is `Asset.SHA256` from `https://usemagpie.ai/api/latest` (`Feed`, update.go:46), the same response that carries the asset URL that is fetched.
- effect: whoever can answer that feed — a compromised site/CDN account, or anyone who can complete TLS as `usemagpie.ai` on the machine (an installed root, a corporate MITM proxy) — installs arbitrary code: the hash is not signed, so it binds the download to nothing but that response. macOS is protected independently: `Stage` calls `sameSigner` (update.go:284, 293-302), which requires `codesign --verify --deep --strict` and the installed bundle's `TeamIdentifier`.
- why: an in-band checksum protects against a broken transfer, not against a forged release; on the two platforms without a code-signing check, a forged release has no second gate.
- fix: publish a detached signature over each asset's SHA-256 and verify it against a key compiled into magpie, before `InstallBinary`, on every platform.
- evidence: `rg -n -i 'authenticode|signtool' internal/update` → no output; `rg -n 'codesign' internal/update` → `update.go:294` and `update.go:305` only, both inside `sameSigner`/`team`, which only `Stage` (the macOS bundle path) calls. verified-by: grep
- confidence: medium

### F5 [P3] `saveState` discards the write error, so a sync reports success with no baseline recorded

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/davsync/davsync.go:555
- code:
```go
func saveState(st state) {
	if lastgood.KeepUnparsed(path("sync-state.json"), lastgood.Into[state]) != nil {
		return
	}
	b, _ := json.MarshalIndent(st, "", "  ")
	edit.WriteAtomic(path("sync-state.json"), b)
}
```
- trigger: `sync-state.json` cannot be written while a sync succeeds — disk full, the settings directory read-only, or the file held by another process — and `syncNow` reaches `saveState(st); return err` with `err == nil` (davsync.go:683-685).
- effect: the round is reported as a success while `st.Local`, `st.Sum`, `st.Server`, `st.Last` and the `Blind` guard from a blind write are all still the previous ones. Every later round then sees every part as changed here and re-reads or re-pushes the file, and the merge runs from a pre-write baseline, so another computer's part can be dropped without a `Notice` — with nothing telling the user the state could not be recorded.
- why: a persistence step on the success path returns no error, so the merge's own baseline (the thing that makes a later merge correct) can be missing while the caller is told the sync is done.
- fix: have `saveState` return the `edit.WriteAtomic` error and return it from `syncNow`, `Upload`, `Restore` and `Undo` before they report success.
- evidence: read davsync.go:555-561 (no error returned) and davsync.go:683-685 (`saveState(st)` then `return err`, with `err` already cleared). verified-by: read
- confidence: high

## Checked, no defect

- `load`/`save`/`changeLocked` (library.go:136-161, 266-280): a `library.json` that doesn't parse returns the unmarshal error and no change is applied, so a failed parse can never become empty state; saves go through `edit.WriteAtomic`, and `Collect` refuses a library file it cannot stat or read (carry.go:55-64, the #2269 guard).
- `Bundle.check` (carry.go:188-228): every restored/synced skill name goes through `checkName` (`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`, library.go:215-222), every carried file path through `localFile`, and set/agent ids likewise, before `Put` joins a name onto `skillsDir()`/`stage` — so a hostile bundle cannot traverse out of them. `profile.Save` (profile.go:187-199) stores into a JSON map keyed by name, never a per-name path.
- `untar` (skills.go:876-920): the tarball's one top folder is dropped, the cleaned path is required to stay under the destination, symlink and hardlink entries are ignored (only `TypeDir`/`TypeReg` are handled), and the total is capped by `maxFetch` — a repository tarball cannot write outside the temp dir or fill the disk.
- `dav.put`'s short-write retry (dav.go:324-352): bounded at three writes, verified by HEAD and then PROPFIND when the HEAD's length is 0, and the retry matches the half file's ETag — a cut upload is detected by size rather than read back, and `errChanged` is returned for a 412 instead of overwriting.
- `backup.Seal`/`Open` (backup.go:266-330): AES-256-GCM with the envelope fields in the AAD, PBKDF2-SHA256 at 600k iterations, and three distinct errors (`ErrPassphrase`, `ErrCorrupt`, `ErrNotBackup`) so a wrong passphrase cannot trigger the damaged-file rebuild; `Restore` only applies a bundle that opened.
- `backups.keep` (library/backup.go:43-95): each file is copied aside on the first write of a change, the folder is created exclusively (`freshBackup`, millisecond stamp with `-2`, `-3`… suffixes), and `pruneBackups` keeps the newest 30 plus every folder that holds skills — the "backup before overwrite" invariant holds for agent files.
- Sync concurrency: `lock()` (davsync/lock.go:25-50) takes the cross-process file lock before `mu`, with a 30 s wait, so Sync/Configure/Off/Dismiss/Restore cannot interleave; `library.Sync` lets go of `mu` between per-agent steps and only reuses the memo when `mu.held == heldAt+1` (sync.go:136-160), so a change that slipped in between is not overwritten by stale hashes.
- `stallTransport` (stall.go:41-72): each request gets its own watchdog, reset per read/write, with `allSent` giving the tail of a large PUT `max(since start, 4×stallAfter)`; a slow but moving transfer is not killed and a dead server is, so no sync can hang on a silent socket.

## Open questions

- Does a WebDAV URL with embedded credentials (`https://user:pass@host/dav`) reach an error path that echoes it? `newDAV` quotes `c.URL` as typed (dav.go:144-148) and `nowhere` builds `u.String()` from `d.base` including `User` (dav.go:455-470); the resulting `st.Error` is persisted in `sync-state.json` and logged by `run`. Confirming needs a live server that answers 404/405 at that address.
- Whether a real install's sealed backup actually crosses 64 MiB (F1's trigger) needs a runtime measurement of a large skill library plus provider icons; the snapshot can only show the arithmetic (32 MiB of skill files ≈ 57 MiB sealed per carry.go:43-48, plus 1.33 MiB of envelope per 1 MiB icon).
