# internal/sessions - persistence, decoding, indexes, stats

## Coverage

- internal/sessions/cache_decode.go: 1-156 - decodeCache (gjson head, per-file json.Unmarshal, unique-string sharing), encodeCache (streamed 256 KiB key/value writes), share.
- internal/sessions/sessions.go: 1-120 (package doc, Tokens, ccTokens); 200-430 (state, day, file/fileMore, tool, dateOf, day, use, unuse, saw, clone); 460-570 (stat, statFiles, statOne, parallel); 615-735 (allFiles, distinctPaths, codexFiles, warmListing); 880-980 (CachePath, cacheVersion, zone, loadCache); 980-1200 (saveCache, Saved/saved, writeCache, refresh head); 1200-1375 (publish, waitParsing, checkpoint ticker, indexing, offOf, pricer, Reset); 1395-1525 (List, listSessions, assemble, assembleListed head); 1600-1712 (parserFor, parseWith); 1724-1861 (scan, scanHead, scanAt, scanThin); 1855-2000 (takeLong, giveLong, blocks, lineLen, scanBuffer).
- internal/sessions/call_cache.go: 46-260 (versions/caps, resetCalls, pruneCalls, keepCalls, reader tables); 250-410 (callLock, readCalls); 410-610 (callsAlong, ride, finish, callPatch/callFrame, callDiskState/restore, frameBuffer, loadCalls head); 600-739 (loadCalls tail, writeCalls).
- internal/sessions/call_frame.go: 1-118 - decodeFrame, shardDecoder/frameInput, gobTypes, gobUint.
- internal/sessions/frames.go: 1-270 - readFrame, frameHeaderLen, blockLen, frameSize, scanFrames.
- internal/sessions/call_continuation.go: 1-72 - keepCallContinuation, restoreCallContinuation.
- internal/sessions/call_revisions.go: 1-74 - callRevisions, writeCallRevisions, withCallRevisions.
- internal/sessions/revision_cache.go: 1-208 - workingRevision, recentRevision, weights, retainedRevisions, trimSummaryRevisions, trimCallRevisions.
- internal/sessions/listmemo.go: 1-62 - round, get, done, reset.
- internal/sessions/rollup.go: 1-147 - Duration, Tokens.Spent, Sum, ranked.
- internal/sessions/stats.go: 100-465 (Day/Usage/Active, StatsFor, StatsAndNative, statsIn, statsRead); 540-615 (Overview); 600-771 (shapes, toolUse, skillUse, ToolCategory).
- internal/sessions/tail.go: 1-71 - tail, markTail, holds, dropUse.
- internal/sessions/quiet.go: 1-81 - Warm, Ask, quiet, readSlot, readBegin.
- internal/sessions/calls.go: 186-296 (sharedCalls flight, callsFor, callFiles/callFilesWith); 370-560 (prepareCalls, clone, cloneState, callHead, cxMarks); 1100-1225 (codexCompact, hashHead, sameHead, prefixHash, CallSources, ReadCallSource).
- internal/sessions/manage.go: 150-265 (Manage, nativeIDs, listAgent, listed); 300-450 (Delete, sessionPaths); 495-625 (inside, recent, sizeOf, writeManifest, move, copyAll, Trash head).
- internal/sessions/gateway_delete.go: 1-244 - readHidden, writeHidden, keepUnreadHidden, GatewayHidden, GatewaySizes, DeleteGateway, restoreGateway, clearGatewayTrash.
- internal/sessions/dsh.go: 253-330 - dshCarry, parseDsh, dshCount head.
- internal/sessions/agy.go: 276-372 - parseAgy, agyCount.
- internal/sessions/codex_provider.go: 120-190, 288-305, 675-695 - codexProviders and its saveCache call sites under storeMu/mu.
COVERAGE: read 19 files, 27043 non-test LOC in scope; skipped 67 (other agent parsers such as codex.go/opencode.go/kimi.go/claude.go/grok.go/droid.go/wsl.go, transcript and trace readers, search, cc_copies, thin, platform-specific manage_*/cursor_clone_* files, and carry_convert - outside this lane's write/read/index/stats core).

## Findings

### F1 [P2] Failed session-delete rollback is unchecked, then the trash directory holding the unreturned file is deleted
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
- trigger: `Delete` of a session with at least two paths where a later `move` fails (ENOSPC, EACCES, EIO, or a WSL 9p copy) and a move-back of an earlier path also fails - `move` refuses when the original path exists again (`"%s is already there"`), which a still-running agent reproduces by recreating its file, since `recent()` only samples mtimes before the first move.
- effect: the already-moved path stays in `trash/.../files/` and `os.RemoveAll(dir)` deletes it there, so the user's transcript, tool results and history files are erased permanently; `Delete` returns only the second move's error and the returned state contradicts the comment ("the session stays as it was"). Magnitude: one whole session, unrecoverable, and the manifest it needed for `Restore` is removed too.
- why: the compensating action is treated as infallible; a failed compensation must leave the trash (and its manifest) in place, not be swept away.
- fix: capture each rollback `move` error and, when any fails, keep `dir` and return an error naming what stayed in the trash instead of calling `os.RemoveAll(dir)`.
- evidence: `grep -n 'what was moved goes back' -A 7 -B 3 internal/sessions/manage.go` -> rollback loop at 385-387 discards `move`'s error, `os.RemoveAll(dir)` at 388; `rg -n 'back\.From|moved goes back' internal/sessions --glob '!*_test.go'` -> only these rollback sites, none checks the error. verified-by: read|grep
- confidence: high

### F2 [P2] A read error mid-file publishes the half-read parse against the old offset, so the next poll counts those lines twice
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
- trigger: a non-EOF read error (EIO, ESTALE on an sshfs/NFS home, EACCES after a mid-poll chmod, a removed FUSE mount) after at least one complete line of a recently read session whose parse is kept: the per-line callback already added tokens and active time to `s` (a clone of the cached parse), and `s.Off` stays at the pre-scan value while `s.Size`/`s.Mod` were set from the current file before the scan.
- effect: `refresh` publishes that state unconditionally (`publish` has no error path), the next refresh passes the append guard in `prepareCalls`/`parseWith`, and every line from the old offset is applied a second time: input/output/cache-write tokens and active seconds for that session double, doubling its cost on Usage and Sessions and per-day totals, until the file is rewritten or the cache version changes. The Requests path has the identical split at call_cache.go:350 - a partial scan keeps `st.Off` while `st.Calls` were appended - so request rows duplicate as well.
- why: the byte offset is the commit point of an incremental parse, but the state is mutated line by line before that commit and is published even when the commit did not happen.
- fix: on `err != nil` return the previous state unchanged (`if err != nil { return old }`) instead of the partially applied clone; same for `readCalls` (return `old` after a failed scan).
- evidence: read of sessions.go:1703-1706 (only the offset update is guarded) and call_cache.go:345-373 (offset/`st.Calls` guarded separately, `keepCalls` at 370 runs on both paths); `grep -n 'func parseWith' internal/sessions/sessions.go` -> `func parseWith(...) *state` (no error return, so the error is dropped). verified-by: read|grep
- confidence: high

### F3 [P2] Delete holds the exclusive session-store lock across a cross-volume (WSL/9p) copy
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
- trigger: `DELETE` of a session whose files are not all on one volume - the WSL case the function documents ("moved out of it over \\wsl.localhost"), or Claude Code tool-result folders on a second mount: `move` falls through to `copyAll` and copies the whole tree while both locks are held (the move loop is at manage.go:383).
- effect: every session read takes `storeMu.RLock()` (sessions.go:1408 List, stats.go:180 Stats, manage.go:163 Manage, codex.go:355) and blocks until the copy ends: the Sessions and Usage pages and the gateway's session listing stall for the duration (a 1 GB tool-result folder over 9p at ~30 MB/s is ~35 s per poll), so polling clients time out and retry while the copy runs.
- why: a global read-write lock is held across blocking filesystem I/O that needs no access to the in-memory index.
- fix: collect `paths`/`fs` under the locks, release both before the move loop, and re-take them only to drop `cache` entries and call `saveCache`.
- evidence: `rg -n 'storeMu.Lock\(\)' internal/sessions --glob '!*_test.go'` -> manage.go:319 plus four codex_provider/relocate sites; `rg -n 'storeMu.RLock\(\)' internal/sessions --glob '!*_test.go'` -> 8 read paths including sessions.go:1408 and stats.go:180. verified-by: read|grep
- confidence: high

### F4 [P3] One undecodable zstd frame pins a packed dsh session at that frame forever, silently
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/sessions/frames.go:228
- code:
```go
		if out, err = dec.DecodeAll(frame, out[:0]); err != nil {
			err = nil
			end = start
			return
		}
```
- trigger: a dsh `session.v*.jsonl.zst` with one frame that fails `DecodeAll` (bit rot, a frame cut and later extended with different bytes whose block headers still parse) - a fully present frame, not a partially written one.
- effect: `scanFrames` returns err nil with `end` = the bad frame's start and `next`/`nextSkip` unchanged, so `parseDsh` (dsh.go:297-309) marks the tail at that frame; every later read restarts there, fails again and never reads the frames after it. The session's later messages, prompts and tokens never appear in Sessions, Usage or Requests, with no error anywhere; each poll re-decodes the prefix up to the frame.
- why: "not written whole yet" (transient) and "cannot be decoded" (permanent) are given the same treatment, so a corrupt frame can never be passed.
- fix: when `readFrame` returned a whole frame (`n > 0`) and `DecodeAll` fails, advance `next, nextSkip = at, 0` and continue past it instead of returning at its start.
- evidence: read of frames.go:210-232; `rg -n 'scanFrames|err = nil' internal/sessions/frames.go internal/sessions/dsh.go` -> frames.go:216 and 229 are the only places these errors are dropped, dsh.go:297 the only caller. verified-by: read|grep
- confidence: high

### F5 [P3] copyAll defers every source file's Close inside the walk callback, so fds pile up for the whole copy
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/sessions/manage.go:596
- code:
```go
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fi.Mode().Perm())
		if err != nil {
			return err
		}
```
- trigger: cross-volume `move` of a session directory with more regular files than the process's descriptor budget (RLIMIT_NOFILE, commonly 1024): a Claude Code session's `<id>/` folder with hundreds of subagent/tool-result files; the `defer` belongs to `copyAll`, not to the walk iteration.
- effect: partway through, `os.Open` fails with EMFILE, `move` removes the partial destination and returns the error, `Delete` rolls back - and with F1's unchecked rollback that same path can then destroy the paths it had already moved. While the copy runs the process is also short of descriptors for files and sockets used by other requests.
- why: per-file resources are released at function scope rather than per iteration, so the peak is proportional to the tree, not to the file being copied.
- fix: close `in` explicitly after `io.Copy` in each iteration (`in.Close()` before returning) instead of deferring it to `copyAll`.
- evidence: read of manage.go:575-600 - `defer in.Close()` at 596 sits inside the `filepath.WalkDir` callback whose header is at ~570; the same walk copies an unbounded number of files. verified-by: read
- confidence: high

## Checked, no defect

- `decodeCache` (cache_decode.go:23-81) rejects the whole kept index on a non-JSON body, a foreign version/zone or any file whose parse fails to unmarshal, so a torn `sessions.json` yields a full rebuild, never wrong numbers; `writeCache` (sessions.go:1083-1096) always writes a temp file and renames, removing the temp on every error path, and no fsync is needed because a lost index only costs the next start a re-read.
- `loadCalls` (call_cache.go:561-660) checks each frame's length (0 < size <= 256 MiB), CRC32, gob decode, path/agent identity and non-decreasing `State.Off`, and returns nil when the shard ends in a `More` frame, so a torn append is discarded rather than merged into calls.
- Resume-from-offset is guarded three ways: `prepareCalls` (calls.go:388) and `parseWith` (sessions.go:1677) require a non-shrinking file, `Off <= size`, a head hash and `prefixHash` of the indexed prefix; `tail.holds` (tail.go:44-49) adds the same check for dsh/Grok/Droid/Kimi carries. The documented limit - 16 sampled 4 KiB windows for files over 64 KiB - is a deliberate shortcut, not an oversight.
- Published parses are treated as immutable by design, and the copies back that up: `state.clone` (sessions.go:348-380) deep-copies `Days` including each `*day`'s `Hours`, `Tools` and `Skills`, plus `Models`, `Codex`/`Claude`; the agent carries that own maps are copied before mutation (`parseAgy` rebuilds `Steps` at agy.go:294-298), so `encodeCache` marshalling without the mutex does not race a parser.
- `keepCalls`/`pruneCalls` (call_cache.go:96-208) keep `callOrder` and `callCache` in step (delete-then-reinsert, order rebuilt from survivors), so the `n -= len(callCache[p].Calls)` accounting cannot miss an entry; the budgets (131,072 calls, 64 files) bound resident rows.
- `scanThin` (sessions.go:1773-1861) returns the offset of the last whole line when the file ends mid-line, so a growing file resumes without losing or repeating a line, and `lineLen`/`takeLong`/`blocks` bound long-line buffering to the line's own size; the 32 MiB `maxLine` skip is explicit.
- `statsRead`'s window (stats.go:169-186, 395-430) builds `since` with `AddDate(0,0,1-days)` and day indices with `math.Round(Sub/24h)`, and `Overview`/`perDay` reuse the same `first`, so DST transitions do not shift a day; dates outside `[from, To]` are dropped before aggregation.
- `sharedCalls`'s flight (calls.go:190-232) publishes `fl.calls` before closing `done`, so waiters synchronise through the channel; the flight is removed under the mutex on both the deferred and the fast path, and the shared case returns a clone.

## Open questions

- Does the pooled zstd `Decoder` (frames.go:27-33, returned to `frameDecoders` after a failed `DecodeAll`) stay usable? If klauspost/compress leaves a failed decoder poisoned, every later dsh read would fail at its first frame until restart - a runtime experiment with a corrupted frame would settle it.
- Can a real agent rewrite a >64 KiB session file in place leaving all 16 `prefixHash` windows unchanged (calls.go:1138-1175 says such rewrites can go undetected)? A live Codex/Claude compaction run plus a re-read would show whether the sampled fingerprint is sufficient in practice.
