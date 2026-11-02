# Verification of sessions-store and library-sync findings against 0.1.957

Source: `/tmp/magpie-audit-Pzn4fNp7/src`. Every quoted block was compared with the cited line; every trigger was traced through its callers. No cited line was off by more than 3 lines, so no file:line was replaced (the function or secondary lines are named in the evidence).

## Verdicts

- sessions-store/F1 CONFIRMED - rollback of a failed move is unchecked and `os.RemoveAll(dir)` then destroys whatever the rollback could not put back; the move-back really can fail (`move` refuses an existing path, manage.go:554) and the trash holds the only copy.
- sessions-store/F2 CONFIRMED - `parseWith` publishes the partially applied clone with the old offset, so the next re-parse of the file (its next append, not literally the next poll) applies those lines to a clone of the already-counted state; `readCalls` has the same split at call_cache.go:345-370.
- sessions-store/F3 CONFIRMED - `Delete` holds `storeMu.Lock` and `mu.Lock` across the whole move loop, and a cross-volume `move` copies the tree inside it; all eight `storeMu.RLock` readers block.
- sessions-store/F4 CONFIRMED - a whole frame whose `DecodeAll` fails ends `scanFrames` with `err == nil`, `end` at that frame and `next` there (or one frame before when a line was pending), so the tail pins and later frames are never read; `dsh.go:297` is not the only caller (`calls_dsh.go:106` too).
- sessions-store/F5 REJECTED - `defer in.Close()` is inside the `filepath.WalkDir` callback, so it runs when each callback returns: one file open at a time, no fd pile-up and no EMFILE.
- library-sync/F1 CONFIRMED - `dav.get`/`s3.get` stop at 64 MiB with no error while `push`/`Upload` will write a larger sealed file; other computers see `ErrCorrupt`, the rebuild refuses their older cached copy, and the offered Upload overwrites the intact file (keeping only the truncated prefix aside).
- library-sync/F2 CONFIRMED - `Put` moves every replaced skill aside before it renames the staged ones in, with no rollback, and `changeLocked` does not save on error, so `library.json` lists skills whose folders are gone; `readSkills` carries them as empty and the next sync spreads the empty folder.
- library-sync/F3 CONFIRMED - a WebDAV PUT with no ETag goes over unconditionally and comes back `Blind: false`, so the loser's next round treats its pushed parts as unchanged here and silently brings the winner's older parts in; the trigger needs only a GET without ETag (a usable Last-Modified does not help, `put` only ever uses `etag`).
- library-sync/F4 CONFIRMED - Windows and Linux binaries are installed from the SHA-256 in the same feed response that names the URL; the only independent check is macOS `codesign` (update.go:291-307), and Windows builds are not even Authenticode-signed (`blocked.go`: "magpie's unsigned builds").
- library-sync/F5 CONFIRMED - `saveState` returns nothing and `edit.WriteAtomic`'s error is dropped, so `syncNow` reports success with the new baseline and `Blind` guard unwritten.

## Corrected findings

### library-sync/F1 [P1] A WebDAV/S3 backup over 64 MiB is silently truncated on read and reported as a damaged file

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/davsync/dav.go:291 (also s3.go:234)
- code:

```go
	data, err = io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return nil, version{}, err
	}
```

- trigger: the file on the server is a valid sealed backup larger than 64 MiB. Reachable without malice: `library.Collect` carries up to `maxCarriedAll = 32 << 20` of skill files (carry.go:43-52; the file's own comment says 32 MiB of files is about 57 MiB sealed) and each provider icon may be `MaxIcon = 1 << 20` (provider/icon.go:31); the skill bytes are base64 in the bundle JSON and again in the sealed envelope (backup.Seal, backup.go:266-278), and the icons likewise (~1.78 MiB sealed per 1 MiB raw), so ~4 MiB of icons past a spent skill budget crosses 64 MiB. Neither writer checks: `syncOnce`'s `push` and `restore.Upload` seal and PUT whatever size they have (no length test anywhere in davsync).
- effect: on every computer except the writer, `dav.get` returns a 64 MiB prefix; `backup.Open` fails to unmarshal, `isBackupJSON` matches the head and returns `ErrCorrupt` (backup.go:289-300). In `syncOnce` the damaged-rebuild branch refuses because `cached(st.Sum)` is that computer's older, smaller file, so `bytes.HasPrefix(good, data)` is false (davsync.go:1146-1163) - the audit's parenthetical inverted this: a cached copy that really were the full file would let the rebuild succeed. `described` then yields "is a magpie backup cut short or damaged (67108864 bytes)" with `whatIs`'s `Replace: true` (notbackup.go:60, 136), so the only action offered is Upload: `restore.Upload` keeps the truncated prefix aside (`keepAside`, restore.go:154-160) and PUTs this computer's older setup over the intact file. Sync fails again on every round on those computers. The writer never re-reads (its stored ETag answers 304 and `maps.Equal(L, st.Local)` returns early, davsync.go:1071-1074), so it never notices.
- why: the read path enforces a limit the write path does not, and the truncation is classified as damage to the remote file rather than as a local read limit, so the recommended recovery destroys the file it misdiagnosed.
- fix: read `64<<20+1` and fail with an explicit "the backup is larger than 64 MiB" error, and refuse to push a sealed bundle over the cap (github.go:385 already refuses), as `github.commit` does.
- evidence: `dav.go:291`, `s3.go:234` (plain caps, no Content-Length check); `github.go:21` `githubMaxFile`, `:321` read with +1, `:385` write refusal; `carry.go:43-52` budgets; `icon.go:31`; `backup.go:289-300` ErrCorrupt; `davsync.go:1146-1163` rebuild gate; `restore.go:149-160` Replace then keepAside; `notbackup.go:55,60,136`.
- confidence: high

### sessions-store/F1 [P2] Failed session-delete rollback is unchecked, then the trash directory holding the unreturned file is deleted

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/sessions/manage.go:383
- code:

```go
		if err := move(p, filepath.Join(dir, "files", m.Name)); err != nil {
			// what was moved goes back, and the session stays as it was
			for _, back := range t.Items {
				move(filepath.Join(dir, "files", back.Name), back.From)
			}
			os.RemoveAll(dir)
			return Trashed{}, err
		}
```

- trigger: `Delete` of a session with at least two paths where a later `move` fails (ENOSPC, EACCES, EIO, or a WSL 9p copy) and a rollback `move` of an earlier path also fails. The rollback target is the original path, and `move` refuses one that exists again (`"%s is already there"`, manage.go:554), which a still-running agent that opens by path reproduces by recreating its file; `recent()` sampled every mtime before the first move (manage.go:368-372, `ActiveWindow = time.Minute`), so the check cannot catch it. A cross-device rollback copy error does the same.
- effect: the already-moved path stays in `trash/<agent>/<stamp>/files/` and `os.RemoveAll(dir)` (manage.go:388) deletes it there together with the manifest, so the transcript, tool results and history are erased permanently and `Restore` can no longer name them; `Delete` returns only the second move's error and the returned state contradicts the comment ("the session stays as it was").
- why: the compensating action is treated as infallible; a failed compensation must leave the trash (and its manifest) in place, not be swept away.
- fix: capture each rollback `move` error and, when any fails, keep `dir` and return an error naming what stayed in the trash instead of calling `os.RemoveAll(dir)`.
- evidence: manage.go:383-389 (rollback errors discarded at 385-387, `os.RemoveAll(dir)` at 388); the sibling rollback in `Restore` (manage.go:708-712) ignores errors too but never removes the trash, so it cannot destroy the file; `rg -n 'back\.From|moved goes back' internal/sessions` finds the same unchecked-rollback-then-RemoveAll pattern twice more, at gateway_delete.go:145-150 and 167-173.
- confidence: high

### sessions-store/F2 [P2] A read error mid-file publishes the half-read parse against the old offset, so the next re-parse counts those lines twice

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/sessions/sessions.go:1703
- code:

```go
	off, err := scanThin(f.path, s.Off, head, thin, line)
	if err == nil {
		s.Off = off
	}
	if calls != nil {
		calls.finish(off, err)
	}
```

- trigger: a non-EOF read error (EIO, ESTALE on an sshfs/NFS home, a removed FUSE mount) after at least one complete line of a recently read, non-packed session file whose parse is kept. The per-line callback has already added tokens and active time to `s` (a clone of the cached parse), while `s.Off` stays at the pre-scan value and `s.Size`/`s.Mod` were set from the current file before the scan (sessions.go:1682).
- effect: `refresh` publishes that state unconditionally (`publish` has no error path, sessions.go:1209-1216), and the published `Size`/`Mod` are the current file's, so it is not retried while the file is unchanged; on the next re-parse (the next append, or any size/mod change) the guard at sessions.go:1677 still passes and every line from the old offset is applied a second time to a clone of the already-counted state. Those input/output/cache-write tokens and active seconds double in Usage, Sessions and per-day totals until the file is rewritten or the cache version changes; while the read error persists, each re-parse adds them again. The Requests path has the identical split: `readCalls` appends to `st.Calls` line by line, keeps the old `st.Off`/`ContentHash` on error and still calls `keepCalls` (call_cache.go:345-370), so the next read clones the extended call list and appends the same rows again.
- why: the byte offset is the commit point of an incremental parse, but the state is mutated line by line before that commit and is published even when the commit did not happen.
- fix: on `err != nil` return the previous state unchanged (`if err != nil { return old }`) instead of the partially applied clone; do the same in `readCalls` (return `old` without `keepCalls`).
- evidence: sessions.go:1703-1708 (only the offset update is guarded), 1677 (the clone guard), 1209-1216 (publish); call_cache.go:345-370 (`if err == nil` covers only the offset/State update, `keepCalls` at 370 runs on both paths); `grep -n 'func parseWith' internal/sessions/sessions.go` shows no error return, so the error is dropped.
- confidence: high

### sessions-store/F3 [P2] Delete holds the exclusive session-store lock across a cross-volume (WSL/9p) copy

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/sessions/manage.go:319
- code:

```go
	storeMu.Lock()
	defer storeMu.Unlock()
	mu.Lock()
	defer mu.Unlock()
	loadCache()
	defer useDBs()()
	all := allFiles()
```

- trigger: `DELETE` of a session whose files are not all on one volume - the WSL case the function documents ("One in a WSL distro is moved out of it over \\wsl.localhost", manage.go:308-311), or a Claude Code tool-result folder on a second mount: `move` (manage.go:549-566) falls through to `copyAll` (568-604) and copies the whole tree while both locks are held by the move loop at manage.go:380-390.
- effect: every session read takes `storeMu.RLock()` (codex.go:355, manage.go:84/122/137/163, sessions.go:1408, stats.go:180, transcript_opencode.go:11) and blocks until the copy ends: Sessions, Usage and the gateway's session listing stall for the duration (a 1 GB tool-result folder over 9p at ~30 MB/s is ~35 s), so polling clients time out and retry while the copy runs.
- why: a global read-write lock is held across blocking filesystem I/O that needs no access to the in-memory index.
- fix: collect `paths`/`fs` under the locks, release both before the move loop, and re-take them only to drop `cache` entries and call `saveCache`.
- evidence: manage.go:319-322 and the move loop at 380-390; `move`'s copyAll branch at 561-565; copyAll at 568-604; the eight `storeMu.RLock()` readers above.
- confidence: high

### library-sync/F2 [P2] `library.Put` moves the old skills aside before the new ones are in, and never puts them back on error

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

- trigger: a restore or a sync that brings the library in, failing after the first skill has been moved aside: `move` on a later skill (a Windows file lock is called out as a real hazard at skills.go:250-253), `os.MkdirAll(aside)` (carry.go:366), `dirLink`/`copyDir` (carry.go:386-394), or `os.Rename(stage/<skill>, skillDir)` (carry.go:397) - disk full, EPERM under a locked `~/.magpie/library/skills`, antivirus.
- effect: `Put` returns the error, and `changeLocked` returns before `l.save()` (library.go:266-279), so `library.json` still lists every skill while `skillsDir()` no longer holds the ones already moved. The next `library.Collect` walks the missing directories and `readSkills` swallows the walk error (carry.go:129-132), so those skills are carried with `Files: {}`; that bundle is pushed, every other computer's `Put` moves its real folder aside and renames the empty staged folder in (carry.go:371-377 then 397), and their agents get it too. A skill lost on one machine becomes an empty skill on all of them, with nothing in the UI saying so; the real files sit in `~/.magpie/backups/<stamp>/skills`.
- why: the incoming skills are staged, but the outgoing ones are moved away before anything is placed and there is no rollback, so a mid-restore failure turns an unchanged `library.json` into a library whose files are gone - and an unreadable skill folder is indistinguishable from an empty skill.
- fix: record the moves made and, on any error, move them back before returning (a `defer` that rolls back unless the function returned nil); at minimum make `readSkills` report an unreadable skill directory instead of an empty one.
- evidence: carry.go:350-397 (aside path at 356, `move` at 369, out loop at 371-377, no rollback afterwards); library.go:266-279 (no save when `f` fails); carry.go:125-133 (`readSkills` returns an empty `CarriedSkill` for a directory it cannot walk).
- confidence: medium

### library-sync/F3 [P2] A WebDAV write the server did not check is reported as a checked one, so a lost concurrent write is never reconciled

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

- trigger: a WebDAV server (or proxy/gateway in front of it) that answers GET/PUT without an ETag. `put` only ever uses an ETag (dav.go:319-323) and `syncOnce` passes only `ver.ETag` (davsync.go:1083), so any GET without one - including a server that does return a usable Last-Modified, not only the `version{}` cases above - sends no `If-Match` at all. If the server also has no Date or a Last-Modified inside the same second as Date, `versionOf` returns `version{}` on top of that. Two computers syncing over such a server both get a 2xx.
- effect: the later PUT silently replaces the earlier one; both machines record their own bundle as the server's state (`st.Sum`, `st.Server`, `st.Remote` from the write in `push`, davsync.go:1097-1100). On the loser's next round the part reads as "unchanged here, changed there" (`lc` false because `st.Local` was set to the pushed hashes, `rc` true), so it is brought in with no `Notice` (davsync.go:1186-1213), and the changes it pushed are reverted locally - the exact loss the `Blind` guard exists to detect (davsync.go:1060-1065).
- why: `version.Blind` is set only by `s3.put`'s HEAD-then-PUT path (s3.go:302); the WebDAV path returns `version{ETag: ...}` with `Blind` false even when the PUT carried no condition at all, so the state machine believes a write-up that was never checked.
- fix: in `dav.put`, when `etag == ""` (no `If-Match` sent), return `version{ETag: ..., Blind: true}` as s3 does.
- evidence: dav.go:61-74 (versionOf), 319-323 (`if etag != "" { h["If-Match"] = etag }`), 353 (the returned version); davsync.go:1083 (`etag := ver.ETag`), 1060-1065, 1097-1100, 1186-1213; `rg -n 'Blind' internal/davsync/*.go` (non-test) -> only s3.go:302 sets it true.
- confidence: medium

### library-sync/F4 [P2] On Windows and Linux an update binary is trusted on a checksum delivered by the same response that names it

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

- trigger: any update on a Windows or Linux build (`ReplaceBinary` -> `StageBinary` -> `download` -> `fetch`, update.go:493-538, 866-1001). The only integrity input is `Asset.SHA256` from `https://usemagpie.ai/api/latest` (`Feed`, update.go:42-51), the same response that carries the asset URL that is fetched; `download` merely requires it to be non-empty (update.go:867-869).
- effect: whoever can answer that feed - a compromised site/CDN account, or anyone who can complete TLS as `usemagpie.ai` on the machine (an installed root, a corporate MITM proxy) - installs arbitrary code: the hash is not signed, so it binds the download to nothing but that response. macOS is protected independently: `Stage` calls `sameSigner` (update.go:284, 291-307), which requires `codesign --verify --deep --strict` and the installed bundle's `TeamIdentifier`. Windows has no second gate at all: `canStart` only tries to launch the download (`update.go:557`), and the builds are unsigned (`blocked.go:35`, "magpie's unsigned builds").
- why: an in-band checksum protects against a broken transfer, not against a forged release; on the two platforms without a code-signing check, a forged release has no second gate.
- fix: publish a detached signature over each asset's SHA-256 and verify it against a key compiled into magpie, before `InstallBinary`, on every platform.
- evidence: `rg -n -i 'authenticode|signtool|winverifytrust|wintrust' internal/update` -> no output; `rg -n 'codesign' internal/update` -> update.go:294 and 305 only, both inside `sameSigner`/`team`, which only `Stage` (the macOS bundle path) calls; `download` 866-882 and `fetch` 919-1001 (the hash from `a`); `LatestIn` 86-119 (the feed's JSON); `canStart` at update.go:557.
- confidence: medium

### sessions-store/F4 [P3] One undecodable zstd frame pins a packed dsh session at that frame forever, silently

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/sessions/frames.go:228
- code:

```go
		if out, err = dec.DecodeAll(frame, out[:0]); err != nil {
			err = nil
			end = start
			return
		}
```

- trigger: a dsh `session.v*.jsonl.zst` with one frame that fails `DecodeAll` (bit rot, a frame cut and later extended with different bytes whose block headers still parse) - a fully present frame (`readFrame` returned it whole, so this is not `io.ErrUnexpectedEOF`), not a partially written one.
- effect: `scanFrames` returns `err == nil` with `end` at the bad frame's start and `next`/`nextSkip` unchanged, so `parseDsh` marks the tail there (dsh.go:306-309) and the Requests reader does the same (calls_dsh.go:114-116). Every later read resumes at that frame (or the frame before it when a line was left pending), fails again and never reads the frames after it; the session's later messages, prompts and tokens never appear in Sessions, Usage or Requests, with no error anywhere. Each read re-reads and re-decodes only from that frame, not the file's prefix - the audit's "re-decodes the prefix" is wrong, the pinning is not.
- why: "not written whole yet" (transient) and "cannot be decoded" (permanent) are given the same treatment, so a corrupt frame can never be passed.
- fix: when `readFrame` returned a whole frame (`n > 0`) and `DecodeAll` fails, advance `next, nextSkip = at, 0` and continue past it instead of returning at its start.
- evidence: frames.go:228-231; `readFrame`'s whole-frame return at frames.go:108; `scanFrames` callers `dsh.go:297` and `calls_dsh.go:106` (the audit's "dsh.go:297 the only caller" is wrong); the tail marking at dsh.go:306-309 and calls_dsh.go:114-116; `err = nil` at frames.go:216 and 229 are the only places these errors are dropped.
- confidence: high

### library-sync/F5 [P3] `saveState` discards the write error, so a sync reports success with no baseline recorded

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

- trigger: `sync-state.json` cannot be written while a sync succeeds - disk full, the settings directory read-only, or the file held by another process - and `syncNow` reaches `saveState(st); return err` with `err == nil` (davsync.go:684-685). `Off` (davsync.go:431), `Dismiss` (davsync.go:538) and `Restore`/`Upload`/`Undo` (restore.go:112/185/226) discard it the same way.
- effect: the round is reported as a success while `st.Local`, `st.Sum`, `st.Server`, `st.Last` and the `Blind` guard from a blind write are all still the previous ones. Every later round then sees every part as changed here and re-reads or re-pushes the file, and the merge runs from a pre-write baseline, so another computer's part can be dropped without a `Notice` - with nothing telling the user the state could not be recorded.
- why: a persistence step on the success path returns no error, so the merge's own baseline (the thing that makes a later merge correct) can be missing while the caller is told the sync is done.
- fix: have `saveState` return the `edit.WriteAtomic` error and return it from `syncNow`, `Off`, `Dismiss`, `Upload`, `Restore` and `Undo` before they report success.
- evidence: davsync.go:555-561 (no error returned), 684-685 (`saveState(st)` then `return err`, with `err` already cleared), 431 (`Off`), 538 (`Dismiss`), restore.go:112/185/226.
- confidence: high

## Added findings

### A1 [P3] A frame whose header no longer parses is treated as "not written whole" too, pinning the same tail forever

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/sessions/frames.go:212
- code:

```go
		frame, n, err = readFrame(r, frame, frameSize(f, at))
		if err != nil {
			// the end, a frame still being written, or not one at all:
			// what was read stands, the rest waits for the next read
			err = nil
			return
		}
```

- trigger: bytes corrupted in the magic or a block header of a fully present frame (bit rot, a partial overwrite): `frameSize` returns 0 and `readFrame` returns `errFrame` (frames.go:77, 94), not `io.ErrUnexpectedEOF`. The file has stopped growing, but the code has no way to tell that from a write in progress.
- effect: exactly F4's - `scanFrames` returns `err == nil` at the last good frame, `markTail` pins the session at that offset (`dsh.go:306-309`, `calls_dsh.go:114-116`), and every later read fails at the same bytes silently, so all later messages and tokens are missing from Sessions, Usage and Requests. F4's fix (advance on a `DecodeAll` failure) does not cover this path: `readFrame` never returned a whole frame.
- why: "not written whole yet" and "written but unreadable" are conflated on both frame-error paths, not only the decode one.
- fix: when no progress has been made at the same tail across several reads (or when the bytes are present but not a frame), record the offset as corrupt and surface an error instead of re-trying it forever; skip to the next frame magic when one can be found.
- evidence: frames.go:212-218 (the swallowed `readFrame` error), frames.go:77 and 94 (`errFrame`), frames.go:154-184 (`frameSize` returns 0 for a bad header), dsh.go:306-309, calls_dsh.go:114-116.
- confidence: medium

### A2 [P3] `put` drops a usable Last-Modified version, so the write stays unconditional and the next round re-downloads the whole backup

- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/davsync/dav.go:319
- code:

```go
func (d *dav) put(ctx context.Context, data []byte, etag string) (version, error) {
	h := map[string]string{"Content-Type": "application/octet-stream"}
	if etag != "" {
		h["If-Match"] = etag
	}
```

and the return at dav.go:353:

```go
			return version{ETag: res.Header.Get("ETag")}, nil
```

- trigger: a WebDAV server that supplies a usable Last-Modified but no ETag (or omits the ETag on the PUT response). `versionOf` gives `version{Modified: lm}` (dav.go:73), `syncOnce` passes only `ver.ETag` (davsync.go:1083), and `put` neither sends `If-Unmodified-Since` nor returns the `Modified` it saw.
- effect: (a) the PUT goes over unconditionally even though the server did give a version, so the lost-update window of library-sync/F3 exists on these servers too; (b) `push` stores the empty version as `st.Server` (davsync.go:1097), so the next sync's GET is unconditional and downloads the whole backup (up to the 64 MiB cap) once more instead of getting a 304 - a cost that matters on the metered servers the code calls out (dav.go:76-80, Nutstore counts requests and downloads). After that one read `st.Server` is set again and the conditional GET resumes.
- why: the version model carries an ETag and a Last-Modified, but the write path accepts and returns only the ETag, so a server that versions by time is treated as unversioned for writes and for the state it records.
- fix: give `put` the whole `version` and return the ETag and Last-Modified it saw; send `If-Unmodified-Since` when only `Modified` is known.
- evidence: dav.go:61-74 (versionOf), 319-323 (only `If-Match`), 353 (only the ETag returned); davsync.go:1083 (`etag := ver.ETag`), 1097 (`st.Server = v`), 1071-1074 (the read that follows the push), 76-80 (the Nutstore note); `rg -n 'If-Unmodified-Since' internal/davsync` -> no output.
- confidence: high

## VERDICT

VERDICT: CONFIRMED 9, DOWNGRADED 0, REJECTED 1, ADDED 2; reviewed 21 files
