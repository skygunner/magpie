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
