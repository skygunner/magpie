# v0.1.3190 upstream-first integration review

Accepted release `0.1.3190-p1`: source `171e12415ddce084a3c9417e80cee808a5d1589a`,
accepted 2026-11-06 21:05:07 UTC / 2026-11-07 07:05:07 AEST.
Production evidence: [deployment result](deployment-result.json). Pinned upstream:
`8cbaeecae976d2a8561293c1abc5daf6aa89d77f` (`v0.1.3190`).
Previous integration: `7308a6331`; previous accepted source: `16f33cdcf`.

## Patch decisions

| Issue | Decision | Current source/behavior comparison |
| --- | --- | --- |
| #3035 format | retain | Upstream `formatRefusal` still lacks guided_grammar/compile_grammar; preserve the existing schema-to-JSON cascade. |
| #3035 bare request extension | suspend | Upstream `11df525f0` restricts opaque-overflow inference to observed vendors and `8cbaeecae` adds pass-through errors. Remove the extra bare-refusal branch, pre-translation flag, decision-group hook and its old expectations together; opaque errors follow upstream. This is policy convergence, not a claim of upstream-equivalent failover. |
| #3635 | retain | Upstream optional-field parsing still misses NVIDIA's quoted multi-field list and the safe redundant reasoning alias; preserve selected member effort and cached compatibility guards. |
| #3636 | retain | Explicit unsupported-parameter lists still need the narrow shape classifier; preserve terminal invalid-value errors and no account rest. |
| #3624 | retain | Upstream openRoute selects the trace but does not reveal it on explicit Usage navigation; retain the event-scoped reveal after view loading. |
| #3948 | retain diagnostics | Upstream request metrics and bytes display changed, but bounded non-streaming envelope checks remain absent. Preserve committed HTTP status; no retry/rest from diagnostics. |
| #3794 | retain | Upstream extends Gemini anyOf shape refusals, not Kilo's explicit maximum tool count. Keep both classifications; never truncate tools. |
| #3623 | retain | New upstream bytes/image-savings detail fields do not provide the ordered recorded attempt chain. Merge both detail sections and translations. |
| #4385 | retain | Upstream delayed-wheel restoration still adds only erased progress. Keep the complete interrupted pixel step, per view, and the unchanged touch/non-pixel path. |
| #3487 remainder, #3788, #3791, cooldown settings | remain suspended | Do not restore older quota interpretations, non-streaming timers, cancellation holds, policy overrides or editor/API over current upstream. |
| #3417 | absorbed | Keep upstream configurable lead; no fixed-lead fork patch. |
| build -pN | retain | Base-version update/release-note comparison remains fork-specific. |
| plugin PR #366 | separate | Keep production Cloudflare Workers AI 0.0.5; no test-only plugin package publication. |

## Upstream and issue follow-up

Fetched the release tag and repository history. The 3036-to-3190 update changes
792 files. Four content conflicts were reviewed: shape-error matcher, opaque
length expectations, Usage translations and Usage detail fields. They are not
resolved by blindly taking the fork version.

Read current status for all 172 issues in the historical issue ledger, and all
comments for the production-patch threads plus the suspended quota/timeout/model
cooldown threads. The retained runtime issues are still open; their replies add
no new upstream implementation or reviewer request since the previous replies.
No duplicate issue/comment is needed for unchanged implementations. Historical
issue-campaign branches are not bulk-applied to production: their existence does
not establish runtime need or compatibility. Private API snapshots and full
readback are under `~/.local/state/magpie-rounds/0.1.3190-p1/`.

## Verification record

Initial Linux/Windows/Darwin nogui vet/build and default native vet/build:
8 checks pass. Existing deployment transaction tests: 80 pass. Retained gateway
regressions: 20 race repetitions pass; update tests pass.

The first full Go attempt found the core script 11 bytes above its unchanged
budget. Only two comments in the existing wheel fix were shortened, and the
budget check then passed; the failed attempt is retained privately. The final committed source was retested as recorded below; initial failed attempts remain preserved.

## Deployment probe correction

Before switching production, the old 16-token probe was run against the healthy
0.1.3036-p2 instance. It returned HTTP 200 with finish_reason=length,
16 completion tokens all attributed to reasoning, and zero answer characters.
A 256-token probe returned finish_reason=stop with 89 reasoning + 13 answer
tokens and nonempty content. This is an insufficient probe allowance, not a
candidate protocol regression. Raise only the probe cap to 256; preserve the
nonempty-content requirement, one POST, timeout and rollback behavior. The
existing offline HTTP test now replays this reasoning-budget boundary: red at
16, green at 256. All 80 deployment tests pass after the correction.

Isolated replay of a private production copy is verified: old and candidate
snapshots both expose 269 models, 32 providers, 49 groups, no unexpected fields.
The count differs from the older release record because the live configuration
has evolved; the comparison used a fresh baseline. Replay containers, network,
config copies and seed archive were removed after verification.

A full Go attempt at committed runtime 74540cc71 exposed
TestKeepOffWaitsForTheSetting. The exact same failure (two reads in 50ms) was
reproduced five times in 500 repetitions on pristine v0.1.3190. This subsystem
has no fork delta. Preserve the initial failure and final-source rerun instead
of claiming it never occurred. Completed GUI attribution and the release boundary are recorded below.

## Usage presentation convergence found by the complete GUI run

The original candidate exposed a real #3948 presentation conflict with newer
upstream behavior: ledger-retried failed all 20 cases because a handed-on plain
400 became `400 · Failed · handed on` instead of upstream's `400 · handed on`.
The old global source/no-speed detail rewrites also displaced ordinary fields.
Remove those overlapping rewrites: ordinary vendor and session-log errors keep
upstream labels; missing-speed explanations use upstream tooltips. Protocol and
client-reported failures still identify their source and preserve HTTP status.
The client-processing boundary follows the standard fields.

The corrected details test restores upstream's ordinary field expectations and
adds only the boundary at the end. ledger-hscroll now distinguishes the attempt
history read on expansion from the fresh trace read on navigation, asserting
one request per action rather than treating all reads as the link's action.
All 58 focused cases pass in Chromium/WebKit, including all five languages for
HTTP-200 diagnostics and the upstream handoff count tests. The complete original
GUI run was kept frozen on its own worktree; the final 237 affected fixtures and
full Go/build/race checks were completed separately on the correction.

## Final-source gate and production acceptance

[Gate result](gate-result.json) and [GUI review](gui-review.json) distinguish the
original complete matrix (`c9ca5935f`, 1,251 files) from the corrected final
source (`171e12415`, 237 affected fixtures). Runtime assets outside the Usage
plugin page and translations are byte-identical between these candidates.
The original 114 and final 24 initial failing files, baseline attempts, one recorded
candidate retry and additional diagnostics remain available privately. The
expanded GUI suite is **not fully green**. Exact assertion matches, measured
layout/frame variations, controlled timing cases and first-only transients are
identified separately; an initial timeout after 27 reported passes is not treated
as a completed 30-case run. The official final and pristine masking-demo fixtures
both pass 30/30 and exit normally. Mobile screenshots with `BASE_REF=v0.1.3190`
also pass 30/30 on both; the default old `origin/main` mismatches are preserved.

Final Go passes 87 packages and 25,594 tests/subtests (20 no-test packages).
Eight build/vet checks and the retained race/update checks pass; deployment
transaction tests pass 80/80. The official final 230-case core run has 227 passes
and 3 failure counts (two WebKit horizontal-scrollbar cases plus their parent),
with the same scrollbar assertion on clean upstream. All wheel cases and the
narrow protocol/attempt/handoff checks pass in that environment.

Plugin [PR #366](https://github.com/magpie-community/plugins/pull/366) is merged;
its unchanged head `38ddeb1` modifies only `test/plugin.test.mjs` (+18/-3), with
no later review request. The actual production package remains 0.0.5.

A fresh isolated replay of the final image and a stopped-volume production backup
precede the switch. Baseline/candidate/final snapshots agree on 269 models,
32 providers and 49 groups, without an unexpected-field allowance. Deployment
run `20261106T201706Z-f5c6e83e` exits 0 with PASSED at 2026-11-06 21:05:07 UTC, after
2,760.902 seconds, 11 healthy samples and 3 successful real inference probes.
Restart/OOM/5xx/protocol-error counts are zero. The original container and volume
backup remain available. See [production result](deployment-result.json) and
[published digests](published-digest.json); version and latest are the exact
accepted image, without rebuilding. Native Windows/macOS execution was not done;
their checks are cross-build/vet only.
