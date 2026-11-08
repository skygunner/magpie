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
- Production is **0.1.3190-p1**, upstream v0.1.3190. Accepted 2026-11-06 21:05:07 UTC
  / 2026-11-07 07:05:07 AEST; tested source `171e12415ddce084a3c9417e80cee808a5d1589a`.
  Source integration alone is not production acceptance evidence.

## Candidate integration: v0.1.3295-p1

Pinned upstream `0fc3455e2`; [semantic review](docs/fork/evidence/2026-11-08-3295/patch-review.md).
Production remains 0.1.3190-p1 until final tests, isolated API comparison and 46-minute
observation pass. Test-only fixes from #3476/#3479 and a contrast measurement repair
are tracked on `fix/gui-fixtures-3295`; their original failures remain in the record.
The #5425 navigation guard also repairs five upstream fixture omissions and
one in our own Usage routing test, with separate navigation waiting only.
Retained functional runtime differences touch 12 files after removing a comment-only
delta. Preserve upstream cancellation attribution and new effort maps. Prior suspensions
remain; Messages thinking-token accounting (#3430) arrives through upstream.

## Changes in v0.1.3190-p1

- Suspend the #3035-associated bare Invalid request failover extension. Upstream
  `11df525f0` scopes opaque overflow handling to observed vendors; `8cbaeecae`
  adds pass-through errors. Keep the independent guided-grammar format downgrade.
- Remove #3948's overlapping generic error-label/source and duplicate no-speed
  detail rewrites. Ordinary vendor/session labels and speed tooltips follow
  upstream; bounded protocol/client-report diagnostics remain. The client-processing
  boundary follows the standard fields.
- Raise only the deployment probe allowance from 16 to 256 tokens: the old cap
  could be exhausted by reasoning before any answer. Keep the nonempty-answer
  check, one POST, deadlines and rollback behavior.
- The suspended quota/cooldown/timeout policies stay suspended. The separate
  Cloudflare Workers AI plugin remains 0.0.5.
- [Integration and semantic review](docs/fork/evidence/2026-11-07-3190/patch-review.md),
  [final test gate](docs/fork/evidence/2026-11-07-3190/gate-result.json),
  [GUI scope and failure review](docs/fork/evidence/2026-11-07-3190/gui-review.json).

## This round's patch decisions (v0.1.3190-p1 production)

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

- Current accepted release: **0.1.3190-p1**, upstream v0.1.3190, source
  [`171e12415`](https://github.com/skygunner/magpie/tree/deployed%2F0.1.3190-p1).
  Accepted 2026-11-06 21:05:07 UTC after 2,760.902 seconds, 11 healthy samples
  and three successful real inference probes. Snapshots match the fresh baseline:
  269 models, 32 providers, 49 groups; no unexpected fields.
- [Full deployment history and earlier patch notes](docs/fork/history.md#deployment-history).
- [Final source gate](docs/fork/evidence/2026-11-07-3190/gate-result.json),
  [GUI review](docs/fork/evidence/2026-11-07-3190/gui-review.json),
  [production result](docs/fork/evidence/2026-11-07-3190/deployment-result.json),
  [published manifest/config digests](docs/fork/evidence/2026-11-07-3190/published-digest.json).

## Known boundaries

- Expanded GUI checks are **not fully green**. The complete original 1,251-file
  matrix and the final 237 affected fixtures retain reviewed upstream layout,
  scrolling and asynchronous test failures, including #4132/#4689. See the
  [scoped failure review](docs/fork/evidence/2026-11-07-3190/gui-review.json); first failures and retries
  are preserved. The old fork's ordinary-400/detail presentation conflicts were
  corrected before deployment.

- The fork's layered cooldown editor/API and non-streaming timeout/cancellation
  model policies are suspended. Current behavior follows upstream; the old
  feature's design is [archived](docs/fork/archive/cooldown-settings.md).
- Cloudflare Workers AI is a separate plugin, kept at 0.0.5 for this production
  release. Plugin PR #366 is not a Magpie runtime patch.

Workspace layout, source branches and maintenance:
[the fork guide](docs/fork/README.md). This file contains current decisions;
old test details and deployment records belong in the history/evidence pages.
