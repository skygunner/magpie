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
- Production is 0.1.3036-p2 (accepted 2026-11-05 10:30:47 UTC / 20:30:47 AEST),
  tested source 16f33cdcf00f4c445be7be35b285f3ef1c241df7, upstream v0.1.3036.
  Source integration alone does not authorize publication.

## This round's patch decisions (v0.1.3036-p2 production)

| Patch | Decision | Behavior / reason |
| --- | --- | --- |
| #3035 | retain | SenseNova format downgrade and narrowly matched unexplained request refusals still extend upstream. |
| #3487 | suspend remainder | Upstream changed the amount model and zero-count rules; do not carry competing unknown/zero interpretations or its associated old fixtures. |
| #3635 | retain | NVIDIA explicit parameter-list repair and synchronized equivalent reasoning settings. |
| #3636 | retain | Unresolved explicit parameter lists fail over without resting accounts. |
| #3624 | retain | Usage navigation reveals the selected trace. |
| #3788 | suspend | Upstream now owns the changing group-budget/slow-start rules; use its current streaming-only FirstToken behavior. The 540s total budget is not an equivalent non-streaming deadline. |
| #3791 | suspend | Fork deliberately counts client cancellation and imposes hard model holds; upstream cancellation attribution and resting-candidate policy differ. |
| #3948 | retain diagnostics only | Speed display follows upstream; retain bounded non-streaming envelope checks without changing the committed HTTP status. Stream error-frame status handling is upstream's separate behavior. |
| #3794 | retain | Tools on demand does not cover arbitrary external client tools or guarantee vendor limits. Keep full-tool fallback. |
| #3623 | retain | Expanded Usage details show ordered recorded attempts, including 499. |
| Cooldown settings | suspend | Layered policy, changed defaults and release semantics compete with current upstream rest rules. Editor/API and runtime overrides are removed together. |
| #3417 | absorbed | Upstream configurable Codex reset lead remains; fixed-lead fork stays removed. |
| #4385 | retain new fix | Restore the full interrupted pixel-wheel step from its origin, grouped per view; preserve native progress and keep touch/non-pixel restoration unchanged. Source `fix/webkit-delayed-wheel`. |
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

- Current accepted release: **0.1.3036-p2**, upstream v0.1.3036, source
  [`16f33cdcf`](https://github.com/skygunner/magpie/tree/deployed%2F0.1.3036-p2).
  Accepted 2026-11-05 after 46 minutes and three successful inference probes.
- [Full deployment history and earlier patch notes](docs/fork/history.md#deployment-history).
- [Current acceptance and scoped test evidence](docs/fork/evidence/2026-11-05-3036/p2/gate-result.json),
  [production result](docs/fork/evidence/2026-11-05-3036/p2/deployment-result.json),
  [published digest](docs/fork/evidence/2026-11-05-3036/p2/published-digest.json).

## Known boundaries

- Expanded GUI checks retain documented upstream #4132 (unheard-wheel driver
  cadence) and #4689 (nested-scroller) failures. They are separately assessed
  against clean upstream, not described as a fully green expanded suite.
- The fork's layered cooldown editor/API and non-streaming timeout/cancellation
  model policies are suspended. Current behavior follows upstream; the old
  feature's design is [archived](docs/fork/archive/cooldown-settings.md).
- Cloudflare Workers AI is a separate plugin, kept at 0.0.5 for this production
  release. Plugin PR #366 is not a Magpie runtime patch.

Workspace layout, source branches and maintenance:
[the fork guide](docs/fork/README.md). This file contains current decisions;
old test details and deployment records belong in the history/evidence pages.
