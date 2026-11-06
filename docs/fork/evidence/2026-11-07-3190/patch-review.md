# v0.1.3190 upstream-first integration review

Candidate in progress, not a production acceptance record. Pinned upstream:
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
budget check then passed; the failed attempt is retained privately. Final source
must be retested before deployment. Full Go, full UI, image smoke, isolated
snapshot replay and production observation results will be recorded separately.

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
of claiming it never occurred. Browser-suite failures remain under review.

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
HTTP-200 diagnostics and the upstream handoff count tests. The original complete
GUI run remains frozen on its own worktree; affected Usage/navigation cases and
full Go/build checks are rerun on the correction separately.
