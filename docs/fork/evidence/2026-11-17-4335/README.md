# 0.1.4335-p1 release evidence

Pinned upstream: v0.1.4335,
`cb4ad12bd1038c93cae08c877aac73bd1c4c1c2e`.
Source on which the image was built:
`f0a22204c4d202016802b73f0b46ac446d527a13`.
Previous accepted source:
`31af508decb1054c13cf94b25381d5c268fea28b` (0.1.3622-p1).

Candidate image:
`sha256:4f647e2c95fbb7d38cb653a08ab2a304dd1a654cbad16cef79a7d6f8b356b656`.
The image smoke check reports `magpie 0.1.4335-p1`.

## Source gate

- The candidate contains v0.1.4335 and has no conflict markers or
  whitespace errors against the pinned tag.
- Linux `nogui` build and vet pass.
- The deployment entry passes `bash -n` and all 80 mock transaction tests.
- Eight affected browser fixtures pass 288 cases in Chromium and WebKit.
- A pristine v0.1.4335 worktree reproduces the four `internal/agent`
  failures seen by the full Go run.
- After the budget repair, the two wheel fixtures were rerun over the
  candidate assets (`click-scroll`, `view-open-wheel`): 186 cases, 0 failures
  in Chromium and WebKit.
- `TestCoreScriptBudget` is extended by `forkPixelWheelBudget` = 164 bytes
  so the retained #4385 pixel-wheel scroll code in `app.js` has an explicit
  place in the core budget; the retained bytes are described by a comment in
  the test rather than hidden inside the upstream base constant.
- The combined full Go run reached the gateway package's built-in 10-minute
  timeout while other packages were running in parallel. A standalone gateway
  rerun with a 20-minute timeout completed every test, including the stream
  suite that was interrupted before; its remaining failures are the same
  Go-version and WSL environment groups listed above, with no gateway
  behavioral failure attributable to the merge.

The source gate is reviewed evidence, not a claim that every upstream test is
green on this workstation. The workstation runs Go `1.27.4-X:nodwarf5`, while
the production image builds with Go `1.26`; the full-run failures in unmodified
upstream CLI, provider and gateway code are listed rather than hidden by
retrying the suites:

- `internal/agent`: the four failures above, reproduced on a pristine tag.
- `internal/cli` (unmodified upstream code in this candidate):
  `TestEveryAgentFieldNameWritesNothing`,
  `TestAgentOpenCodeDefaultPutsBackTheModel`,
  `TestClaudeSignInSaysClaudeCodeIsTooOld`.
- `internal/provider` (unmodified upstream code in this candidate):
  `TestClaudeSignInSaysClaudeCodeIsTooOld`.
- `internal/gateway` standalone rerun: fastjson `marshalRaw` escape
  comparisons (`TestMarshalRawIsEncodingJSON`,
  `TestMarshalSizedIsJSONMarshal`, `FuzzMarshalRaw` seed #7) and the two
  `TestServeWSLListens*` WSL checks, the same failure group recorded for the
  0.1.3622 round.
- `internal/gui`: `TestCoreScriptBudget` is repaired in the candidate by
  extending the budget for the retained pixel-wheel scroll patch;
  `TestSessionReadsRoutes` failed once in the combined run and is recorded as
  load-sensitive, following the 0.1.3622 round's precedent.
- Post-repair Go re-checks: `TestCoreScriptBudget` and
  `TestSessionReadsRoutes` pass over three runs; Linux `nogui` build and
  vet still pass.

## Merge and patch scope

The merge contains 1,153 upstream commits and four textual conflicts. The
conflict resolutions and all eight retained patch decisions are recorded in
[the patch review](patch-review.md). No retained patch required a large or
semantics-changing rewrite for v0.1.4335.

## Production acceptance

Pending the docker1 transaction result.

## Scope boundaries

No complete GUI matrix, Windows/Darwin desktop validation or exhaustive live
vendor test was run. The isolated production-config replay was not repeated;
the reviewed deployment transaction supplies baseline/candidate snapshots,
real inference probes, observation and automatic rollback.
