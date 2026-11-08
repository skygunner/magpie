# Integration and production — upstream first

User-approved policy (2026-11-05): remove an absorbed patch. Also suspend a
fork patch whose logic overlaps or conflicts with current upstream rules;
do not force both semantics into a conflict resolution. Suspension does not
mean upstream implements every former fork guarantee.

- `main` mirrors upstream. `integration/nightly` is the persistent integration
  branch, merging a pinned release tag per round. Source `fix/*` and `feature/*`
  branches stay available; do not re-pick unchanged patches every round.
- The existing `prod` branch has historical failed candidates in its ancestry.
  It is not evidence of production success. `production/stable` points only
  to successfully observed production source. `candidate/<version>` records a
  tested candidate; `deployed/<version>` is created only after acceptance and
  observation pass. Old `prod/*` tags retain their historical meaning.
- Production is **0.1.3295-p1**, upstream v0.1.3295. Accepted 2026-11-08 05:08:40 UTC
  / 2026-11-08 15:08:40 AEST; tested source `2fb297ebcd179ea488f4363a86c922f307d22449`.
  Source integration alone is not production acceptance evidence.

## Candidate integration: v0.1.3336-p1

Pinned upstream c6bdc172a; production remains0.1.3295-p1 until acceptance and
observation pass. [Review](docs/fork/evidence/2026-11-09-3336/patch-review.md).
The eight retained issue behaviors are still needed against this tag; their
functional delta is unchanged. Prior suspensions remain. Upstream streaming
failure envelopes, gateway-key concurrency and model settings are used as merged.

## Test scope for rolling integration

User direction,2026-11-09: keep validation proportional to the change. Run the
retained-patch and affected-path regressions, production Linux checks, image
smoke, API snapshot acceptance, real inference and the existing observation.
Playwright covers selected affected pages in both engines; a complete GUI matrix
is reserved for a broad regression that needs it or an explicit request.
Record unrelated upstream/UI failures separately. Test/documentation-only
changes do not trigger another complete Go/build/replay cycle when the runtime
and relevant inputs are unchanged. State the actual tested scope in the evidence.

## Changes in v0.1.3295-p1

Pinned upstream `0fc3455e2`; [semantic review](docs/fork/evidence/2026-11-08-3295/patch-review.md).
Preserve upstream cancellation attribution, routing defaults and member effort maps.
The mapped-effort handler regression protects synchronized NVIDIA reasoning aliases.
Messages thinking-token accounting (#3430) is absorbed upstream; no extra patch.
The eight retained issue behaviors remain necessary. Prior suspensions remain.

Remove the OpenCode comment-only difference. Functional runtime differences touch
12 files; including the separate trace.go comment repair, the full non-test
internal diff is 13 files (+250/-44).

Test repairs are recorded on `fix/gui-fixtures-3295`: posted #3476/#3479,
view-entrance contrast measurement, #4715 platform pin and #3501 atomic tab read.
The #5425 navigation guard also corrects five upstream setup omissions and one
in our own Usage routing fixture. Original assertions and short action deadlines stay.
The posted four-comment i18n ownership repair is credited to Gaoshou101.

Final Go passes 88 packages and 25,914 tests/subtests; 8 build/vet checks, gateway
race regressions repeated 20 times, update checks and 80 deployment tests pass.
The original 1,275-file GUI matrix has 108 initially failing files; the final 12
changed/guard fixtures pass 250 tests. See [scope and review](docs/fork/evidence/2026-11-08-3295/gui-review.json).
Expanded GUI checks are **not fully green**. The separate Cloudflare Workers AI
plugin remains 0.0.5. [Final source gate](docs/fork/evidence/2026-11-08-3295/gate-result.json).

## This round's patch decisions (v0.1.3295-p1 production)

| Patch | Decision | Behavior / reason |
| --- | --- | --- |
| #3035 | retain | Keep the explicit guided-grammar format downgrade. The bare Invalid request extension is suspended; opaque errors follow upstream. |
| #3487 | suspend remainder | Upstream changed the amount model and zero-count rules; do not carry competing unknown/zero interpretations or its associated old fixtures. |
| #3635 | retain | NVIDIA explicit parameter-list repair and synchronized equivalent reasoning settings. |
| #3636 | retain | Unresolved explicit parameter lists fail over without resting accounts. |
| #3624 | retain | Usage navigation reveals the selected trace. |
| #3788 | suspend | Upstream now owns the changing group-budget/slow-start rules; use its current streaming-only FirstToken behavior. The 540s total budget is not an equivalent non-streaming deadline. |
| #3791 | suspend | Fork deliberately counts client cancellation and imposes hard model holds; upstream cancellation attribution and resting-candidate policy differ. |
| #3948 | retain diagnostics only | Retain bounded non-streaming envelope checks and the committed HTTP status. Ordinary vendor/session labels and speed tooltips follow upstream; protocol/client-reported failures identify their source. |
| #3794 | retain | Tools on demand does not cover arbitrary external client tools or guarantee vendor limits. Keep full-tool fallback. |
| #3623 | retain | Expanded Usage details show ordered recorded attempts, including 499. |
| Cooldown settings | suspend | Layered policy, changed defaults and release semantics compete with current upstream rest rules. Editor/API and runtime overrides are removed together. |
| #3417 | absorbed | Upstream configurable Codex reset lead remains; fixed-lead fork stays removed. |
| #4385 | retain | Restore the full interrupted pixel-wheel step from its origin, grouped per view; preserve native progress and keep touch/non-pixel restoration unchanged. Source `fix/webkit-delayed-wheel`. |
| build -pN | retain | Update checks and release notes compare the base version. |
| plugins PR #366 | separate | Keep production Cloudflare Workers AI package 0.0.5. No package downgrade or test-only rebuild. |

The retained delta is reviewed against the pinned tag, including callers,
fixtures, GUI and subsystem references. Source commit ancestry and issue state
are evidence, not the sole retirement test. Tests must exercise current public
behavior; baseline failures do not automatically authorize deployment.

## Deployment checks

1. Preserve acceptance evidence before deciding: baseline.json, acceptance.json,
   acceptance-diff.json, final.json and final-diff.json are private on docker1.
2. An optional --expected-snapshot file contains reviewed baseline/candidate
   snapshots and the immutable target_image_id. A changed baseline or image
   refuses before stopping production. Unexpected candidate differences roll back.
3. `/api/cooldown` exists only in older fork builds. Its deliberate removal is
   represented as null in the snapshot, not silently ignored. Other HTTP failures
   remain errors. Authentication, providers, groups and caller-visible models stay
   checked, with explicitly reviewed model exposure changes only.
4. `/var/lib/magpie-deploy/BLOCKED` prevents a switch. Leave it present while an
   unexplained acceptance difference remains; clear it only after isolated
   reproduction and final-commit tests. Integration may continue while blocked.
5. Build one image from the tested commit. Run the existing detached transaction
   with a host lock, stopped-volume backup, immutable container/image assertions,
   acceptance, three real inference probes and 46-minute observation. Any failure
   rolls back and verifies the previous container; preserve candidate config.
6. Promote production/stable and deployed/<version>, and publish the exact image
   digest as a version and latest only after exit 0, this run's PASSED result,
   matching live IDs and fresh healthy checks. Never rebuild during promotion.

## Deployment history

- Current accepted release: **0.1.3295-p1**, upstream v0.1.3295, source
  [`2fb297ebc`](https://github.com/skygunner/magpie/tree/deployed%2F0.1.3295-p1).
  Accepted 2026-11-08 05:08:40 UTC after 2761.013 seconds, 11 healthy samples
  and three successful real inference probes. Acceptance and final snapshots
  match 268 models, 32 providers, 49 groups; zero unexpected fields.
  Zero restarts, OOM, 5xx and protocol errors. The exact accepted manifest is
  published as 0.1.3295-p1 and latest; the old container and backup are retained.
- [Full deployment history and earlier patch notes](docs/fork/history.md#deployment-history).
- [Final source gate](docs/fork/evidence/2026-11-08-3295/gate-result.json),
  [GUI review](docs/fork/evidence/2026-11-08-3295/gui-review.json),
  [production result](docs/fork/evidence/2026-11-08-3295/deployment-result.json),
  [published manifest/config digests](docs/fork/evidence/2026-11-08-3295/published-digest.json).

## Known boundaries

- Expanded GUI checks are **not fully green**. The original 1,275-file matrix
  ran on 575d7bda3 with 108 initially failing files. All failures have pristine 3295
  comparisons and one candidate retry. Eleven changed fixtures plus the static
  navigation guard pass 250 tests on the accepted source. Runtime, binary and the
  other 1,264 fixtures are byte-identical to the original matrix. Case-level and
  assertion-template matches, first-only failures and diagnostic limits are
  separated in the [scoped review](docs/fork/evidence/2026-11-08-3295/gui-review.json).
  The inherited duplicate panel Usage read (#4717) and other upstream layout,
  scrolling and fixture lifecycle limits remain individually recorded.

- The fork's layered cooldown editor/API and non-streaming timeout/cancellation
  model policies are suspended. Current behavior follows upstream; the old
  feature's design is [archived](docs/fork/archive/cooldown-settings.md).
- Cloudflare Workers AI is a separate plugin, kept at 0.0.5 for this production
  release. Plugin PR #366 is not a Magpie runtime patch.

Workspace layout, source branches and maintenance:
[the fork guide](docs/fork/README.md). This file contains current decisions;
old test details and deployment records belong in the history/evidence pages.
