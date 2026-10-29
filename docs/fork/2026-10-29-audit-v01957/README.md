# 2026-10-29 — audit v0.1.1957 close-out

Read-only audit of upstream release `v0.1.1957` (commit `a0ba8e1f2`), then
absorption-style delivery: each fix lives on a `fix/*` branch of
`skygunner/magpie`, based on the upstream `main` of its time, with a red/green
proof and one issue (or one comment on an existing issue) that carries an inline
diff and the branch/commit links. Upstream `yetone/magpie` is never pushed to.

## Files

| File | What it is |
| --- | --- |
| `FIX-MAP.md` | Every live finding of the reconciled audit, its cluster, delivery mode and batch |
| `delivered-issues.md` | Batch 1: issue, branch, commit, comment and language per delivery |
| `../../../magpie-audit-20261029/AUDIT-v0.1.1957.md` | The audit report this round delivers from |
| `../../../magpie-audit-20261029/upstream/evidence/` | Per-branch red/green, build and vet logs |

## Method

- The audit and its falsification wave were done on `v0.1.1957`; the report was
  reconciled before any issue was opened (rejected rows annotated, ADDED=4,
  counts made consistent).
- Delivery candidates are checked against `upstream/main` at `926ef769f` and
  screened against prior deliveries and upstream issues. See the ledger for
  the outcome of each candidate; an existing issue may contain a candidate fix.
- One finding = one branch = one issue, except where two findings share a file
  and mechanism and are delivered as one cluster.
- Red/green evidence is recorded per branch. For the Windows-only WSL change,
  native and `GOOS=windows` build and vet passed, but the Windows tests and
  actual WSL behavior have not been run or proved.
- B5 is a conservative mitigation: it stops on corrupt frames to preserve
  prior results, but does not recover later records without source repair.
  B6 stops for manual reconciliation when an initial blind write has an
  ambiguous competing update. See the issue bodies and evidence for details.
- B4 is a single commit (`9b83ba670`) covering stash failure propagation,
  cross-process locking, restore ordering, malformed-stash rejection and the
  StepCode write-ahead journal. Its complete diff is ~458 KB across 114 files,
  over GitHub's issue-body limit; the issue carries inline excerpts of the
  safety-critical hunks and a comment carries the complete diff of `stash.go`,
  `stepcode_recovery.go` and the restore-failure test. An external editor can
  still race the comparison/write window; process crash and power loss were
  not verified.

## Final state

- Six Batch 1 issues/comments carry inline diffs: #4118 (comment), #4137, #4136,
  #4144, #4141 and #4258 (body plus a full key-file diff comment).
- All six branches are pushed to `skygunner/magpie`; `upstream` received no push.
- Final `nogui` build and vet pass on all six branches; B3 also passes
  `GOOS=windows` build and vet. Windows tests and real WSL remain unrun.
- The six `/tmp/mg-fix-*` worktrees were removed after their heads and clean
  status were recorded in `upstream/evidence/cleanup-receipt.log`.
- The main worktree still carries its pre-existing `internal/redact`
  conflicts (`DU`/`UU`/`M`) and other untracked user files; this round added
  only `magpie-audit-20261029/` and `docs/fork/2026-10-29-audit-v01957/` and
  did not commit them.

## Verification limits

- The original audit was static except for the P1-1 synthetic reproduction.
  Subsequent branch tests exercise selected behavior; they do not establish all
  reported impacts end to end. WSL runtime behavior remains unverified.
- No branch claims a green full-repository suite; each evidence log names the
  packages that were run. B2, B3, B5 and B6 each contain multiple commits;
  cherry-pick their complete branch diff or all commits, not only the head.
- The main worktree of this checkout carries unresolved `internal/redact`
  conflicts. It was not switched, committed, stashed or edited by this round;
  all branch work happened in `/tmp/mg-fix-*` worktrees.
