# 0.1.4343-p1 release evidence

Pinned upstream: v0.1.4343, `2071195ff4c7076dc79820502e0f1f80378b9660`.
Source on which the image was built: `095fc5f96bb15f9f9fd56e3de237e9700fa81daa`.
Previous accepted source: `82e303f0ccf5cc4584f033af9ec80c8601b7b09a` (0.1.4335-p1).

Candidate image: `sha256:5c7223d19dd60f8be4c06e9e19809fddfa0132438e44a01d369d8805154adeb2`.
The image smoke check reports `magpie 0.1.4343-p1`.

## Source gate

- The merge (12 upstream commits, 77 files) applied with zero textual conflicts; no conflict markers remain in the tree.
- All eight retained patches verified present in the merged tree by their behavior markers (`guided_grammar`, `protocol_error`, tool-count fallback, led-attempts, pixel-wheel).
- `internal/gui/core_budget_test.go` extended: `forkPixelWheelBudget` is now 430 bytes (164 wheel + 266 other retained patches), because v0.1.4343 used the last 266 upstream bytes of app.js room.
- Linux `nogui` build and vet pass.
- Four affected browser fixtures (group-free-rule, instance-name, click-scroll, view-open-wheel): 206 cases, 0 failures.
- `TestSessionReadsRoutes` (gui): 3/3 pass on rerun; its combined-run failure is load-sensitive, the same as last round.
- `TestClaudeSignInSaysClaudeCodeIsTooOld` (provider) fails both before and after the merge with the 4335-based tree: a pre-existing environment issue already recorded for 0.1.4335, not a regression.

## Production acceptance

Docker1 transaction PASSED (`clean_observation`) at 2026-11-17T07:01Z
after 2761 s: 42 real inferences (32 x 200, 10 client-closed 499), 0
restarts, 0 OOM, 0 5xx, 0 protocol errors. Accepted container
`ec792ef5b392` (image `5c7223d19dd6`); published on docker1 as
`bigshort/magpie-patched:0.1.4343-p1` and `:latest` from the same image
ID, no rebuild. Git refs `production/stable`, `candidate/0.1.4343-p1`
and `deployed/0.1.4343-p1` point at `e1a2b3c4d` (the acceptance docs
commit on `integration/nightly`).

## Scope boundaries

No complete GUI matrix, Windows/Darwin desktop validation or exhaustive live-vendor test was run. The deployment transaction supplies baseline/candidate snapshots, real inference probes, observation and automatic rollback.

