# #3392 follow-up: xaiBody (null content, system turns)
Date: 2026-10-25 (UTC 2026-10-24 evening). Account: skygunner. This file records a zcode session's direct delivery, not a review round.

## Trigger
kongkongklng merged 8ecab590 into their fork and reported both halves' follow-up on
#3392 (comment 6385388355, with both original request bodies and error bodies):
- codex: first call succeeds, every call after it 400s, `xAI: Could not decode the
  compaction blob. Ensure it is unmodified from the compact response.` — the replayed
  reasoning item carries `"content": null` beside `encrypted_content` (verified present
  in the posted request body).
- claude: every call 400s `xAI: Invalid request content: Invalid message role.` — Claude
  Code 2.1.292 sends a mid-conversation `system`-role message (messages[1]); Anthropic
  made them GA mid-2026, xAI's Anthropic-compatible endpoint has not.

## Root cause
Both quirks are known to the Grok subscription's plugin path (`grokShaped` in
`internal/provider/grok.go` deletes the null content), but the xai preset (official key)
shapes nothing: `Provider.Prepare` is identity for key presets, and the custom-tool fix
had only now let a Codex second turn reach api.x.ai. The Anthropic relay is verbatim, so
the system turn reached xAI's Messages as-is. Translation paths are safe already:
parseResponses lifts in-place system/developer turns (#502), buildAnthropic coerces
non-assistant roles to user.

## Product delivery
- Branch: `fix/issue-batch-1024pm` in https://github.com/skygunner/magpie
- Worktree: `/home/vincenty/Projects/magpie-fix-1024`; base unchanged `dd1e8822`; new
  head `d8ad9ab2ae999a9920b0945d3beba42b27eb27fc`, pushed and read back from
  `origin/fix/issue-batch-1024pm` (this file's run), no force-push.
- Commit `d8ad9ab2`: `internal/gateway/xai_body.go` (`xaiBody`, gated `p.Host() ==
  "api.x.ai"` as `foldsRoots` is, applied in `passthrough` and on the built body in
  `forwardTranslated`), `xai_body_test.go`, a `gateway-routing.md` bullet.

## Verification (d8ad9ab2, worktree)
- Red/green: with `xaiBody` gated off, `TestXAITakesCodexsSealedReasoningBack`,
  `TestXAILearnsCustomToolsThenTakesTheReplay` and `TestXAIFoldsClaudesSystemTurn` all
  FAIL (the failures print the shaped-off bodies: `"content":null` reaching the fake
  xAI, the system turn left in messages); green at head. Wordings are the reporter's.
- `gofmt -l` on the changed files: clean. `go vet ./...`, `GOOS=linux go vet -tags
  nogui ./...`, `GOOS=windows go vet ./...`: clean. `go build`, `GOOS=linux go build
  -tags nogui`, `GOOS=windows go build`: clean.
- Full `go test -tags nogui ./...` in a sandbox HOME with pinned GOPATH/GOMODCACHE/
  GOCACHE: 66 packages ok; 4 failing packages, 7 failing tests; exit 1. Log:
  `full-suite-d8ad9ab2.log`, sha256
  `1d20f74c10fc4e8fa0c3ed6880fc56081c47f43e74ef31c69cbcb486a7be1ed8`.
- Baseline: the same 7 tests fail identically on clean `14c1551e` (stashed changes,
  same session, same sandbox method): `TestWriteAtomicReadOnlyHardLink`,
  `TestProgram` (edit/testenv: this machine's filesystem/Go, not the round-13 documented
  pair), `TestClaudeClientCacheAbsentUnchanged`, `TestCodexFittedImagesPastTheBudget`,
  `TestMarshalRawIsEncodingJSON`, `FuzzMarshalRaw` (gateway; Go-version-sensitive
  marshalRaw assertions), `TestPluginSOCKSProxy` (plugin #2207). Zero new failures
  from d8ad9ab2.
- Not run against the real xAI API: no key available. The issue reply asks the reporter
  to run a round (codex→grok and claude→grok).

## GitHub publication
- https://github.com/yetone/magpie/issues/3392#issuecomment-6386768624 , author
  skygunner, created 2026-10-24T16:37:27Z, body sha256
  `b60f9d1399f3c21de51810a3c849c5b6fcc8e0136cc650c333f3d8b318e0052a` (read back from
  the API after posting). Chinese, delivery_followup, names both root causes, the fix,
  the branch and commit, the verification, and the ask.

## Scope
- No other units touched; upstream main checked `dd1e8822..upstream/main` (94 commits)
  for an overlapping fix: none (`xai_body.go` absent upstream).
- The unit `u-gateway-an-upstream-naming-the-type-an-unknown-3392-8ecab590` is extended
  (title, coverage, commits, comment, verify head) rather than duplicated; one unit per
  issue is the ledger's standing shape.

## #3139 follow-up: balance and plan halves of the nil-cache drop
Trigger: Lemonawa's isolated reproduction on upstream 0.1.1406 (comment 6385388355 thread, posted
2026-10-24T14:26:39Z): a key's balance card, first single-card refresh, keeps no in-memory cache and
`CachedCards` back-fills the just-written disk reading with its `asOf`; the plan path shares it.
- Code reading: `KeyBalances` and `PlanQuotas` both commit through the same `mergeCards` whose `kept == nil`
  early return a11d605e removed, so the fix already covers both; no behavior change needed.
- Delivery: `47f44679c94bb22b22647e63d6b48a33c51e392d` on `fix/initial-single-quota-refresh`, fast-forwarded
  onto `fix/issue-batch-20261023-k`; both pushed and read back (this file's run), no force-push. Worktree
  `/home/vincenty/Projects/magpie-quota-3139`.
- `TestFirstKeyCardRefreshStaysFresh` reproduces the report's five steps on balance and plan kinds, the fake
  balance endpoint returning the posted `{"data":{"balance":12.34}}` via `$data.balance`; the plan provider
  speaks OpenCode Go's host (plan reads are host-gated) with the rewriting transport, as the file's older
  tests do. A plan card sets no `ReadAt`; its freshness is `AsOf` nil alone.
- Red/green: the old `mergeCards` put back, balance, plan and `TestFirstClaudeCardRefreshStaysFresh` all FAIL;
  restored, green. Targeted `Quota|Refresh|Cards|Balance|Plan`: 175 tests ok. The four first/keeps-newer tests
  `-race -count=3`: 57 ok. gofmt and package vet clean.
- Reply: https://github.com/yetone/magpie/issues/3139#issuecomment-6386993902 (created 2026-10-24T16:58:22Z,
  sha256 0807581009dd012674d5bfaf653caf2f39736bdb4f4bedea4b6fa7cd6163c3c8). Notes the wording split already
  exists (`asOf` vs `error`) and the fix only changes what enters the cache; reporter asked to rerun the steps.
- Full nogui suite not run for this commit: provider-package test-only change; the round-14 log above covers
  the session's other delivery.

## #3401: Windows Studio 2.50.0 verification recorded
Trigger: FuchaZ's verification (comment 6385801640, 2026-10-24T15:07:23Z) of cf87dc37 on a real Studio 2.50.0
/ Windows 11 install: PreToolUse/PostToolUse probes fired around a real bash call; a user-owned Stop hook and
six user `[[plugins]]` survived add and remove byte-identical both ways; `~/.agents/skills` accounted through
`readsShared`; their probe run passed `go test -tags nogui ./internal/library -run Reasonix` 3/3.
- This closes the round-13 open item ("operator asked to verify real Hook firing and shared skill recognition
  before claiming cross-platform behavior"). Remaining unverified: the Electron GUI path itself (the CLI shares
  the loader and settings.json); recorded as such.
- Two non-blocking observations judged, both left as coded:
  - match double-anchoring (`^(?:^X$)$` ≡ `^X$`): the anchored shape is magpie's canonical recorded entry
    (claims and removals match on it) and exactly the shape the real-app verification fired; unanchoring would
    need its own real-app pass for no behavioral gain.
  - the `✓ written into <agent>` line: `Result.Changed` names agents whose files were written, true even when
    one section changed; per-section reporting would restructure the Result for a cosmetic read.
- Reply: https://github.com/yetone/magpie/issues/3401#issuecomment-6387030804 (created 2026-10-24T17:01:52Z,
  sha256 a24c21d109cf61ae8e98a4f2e53bb24de34f1a9290402787690193f46878fb8e).
- No product code changed for #3401 this session.
