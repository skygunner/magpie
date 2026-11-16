# v0.1.4335 integration review

Pinned upstream: cb4ad12bd1038c93cae08c877aac73bd1c4c1c2e (v0.1.4335).
Previous accepted source: 31af508decb1054c13cf94b25381d5c268fea28b (0.1.3622-p1).
Candidate source: f0a22204c4d202016802b73f0b46ac446d527a13.

The merge brings in 1,153 upstream commits. Eighteen paths changed on both
sides in this range. The merge had four textual conflicts; each was resolved
without dropping either current behavior:

- `internal/gateway/fallback.go` keeps upstream's `InferHub.001001005.400`
  recognition and the fork's explicit full-tool-count limit fallback.
- `docs/subsystems/gateway-routing.md` and `docs/subsystems/gui-shell.md`
  keep upstream's expanded documentation and the fork's patch notes.
- `internal/gui/assets/plugins/usage/i18n.js` keeps both sides' translation
  additions.

## Patch decisions

Retain #3035's explicit guided-grammar format downgrade; #3635's NVIDIA
parameter-list repair and synchronized reasoning settings; #3636's explicit
unsupported-list group fallback; #3624's Usage-to-Routing reveal; #3948's
bounded non-streaming protocol diagnostics with committed HTTP status; #3794's
full-tools count-limit fallback; #3623's ordered attempt history, including
499; and #4385's complete interrupted pixel-wheel step per view. The `-pN`
update/notes normalization remains necessary because upstream still compares
the raw release string.

The #3487 remainder, non-streaming #3788 timeout work, #3791 model cooldown and
the layered cooldown editor/API remain suspended. They still overlap or
directly conflict with upstream quota, timeout and resting-account policy.
#3417 remains absorbed by upstream's configurable Codex reset lead. #3430 is
closed upstream and its thinking-token handling is present in v0.1.4335.

## Convergence follow-up

The retained pull-request issues remain open. No commit between v0.1.3622 and
v0.1.4335 references the retained issue numbers, and searching the pinned
source for the fork-specific `guided_grammar`, `maximum number of items`,
`protocol_error`, `led-attempts` or `attempt history` behavior
produced no upstream implementation. Keeping the current fork patches remains
the lower-risk choice; deleting them would reopen known production failures.

| Issue | Upstream status at v0.1.4335 | Convergence decision |
| --- | --- | --- |
| #3035 | Open; no implementation added in this range | Retain |
| #3635 | Open; no implementation added in this range | Retain |
| #3636 | Open; no implementation added in this range | Retain |
| #3624 | Open; no implementation added in this range | Retain |
| #3948 | Open; no implementation added in this range | Retain diagnostics |
| #3794 | Open; no implementation added in this range | Retain |
| #3623 | Open; no implementation added in this range | Retain |
| #4385 | Open; no implementation added in this range | Retain |
| #3487 | Open; competing upstream quota semantics remain | Keep suspended |
| #3788 | Open; upstream group budget is not an equivalent non-stream timeout | Keep suspended |
| #3791 | Open; upstream cancellation/rest policy differs | Keep suspended |
| #3417 | Issue open, feature already present as `CodexResetLead` | Do not repick |
| #3430 | Closed completed; `thinking_tokens` handling is present | Absorbed |

This is a monitoring result, not a request to post another upstream duplicate.
Revisit a retained patch when upstream lands equivalent public behavior, or
when a focused upstream proposal can replace the fork implementation without
the registration, cancellation or retry semantics changing.

Live re-check after deployment (GitHub API, 2026-11-17): every retained and
suspended issue is still open, and upstream main had no new commits after
the pinned v0.1.4335 at re-check time; the #3623 attempt-chain branch is
this fork's own feature branch, still waiting on an upstream decision.

## Candidate delta after review

The only runtime code change added during this round's gate is
`internal/gui/core_budget_test.go`: a named `forkPixelWheelBudget` of 164
bytes extends the core script budget for #4385's retained wheel code and
comments why. No other patch decision required a code change after the merge;
this accounts for the retained bytes and adds no runtime behavior.

## Verification

The source gate uses the ordinary proportional scope for a rolling upstream
merge: one `nogui` Go run, a standalone gateway rerun where the combined run
reached its default timeout, Linux `nogui` build/vet, the 80 deployment
transaction tests and the eight affected browser fixtures. After the budget
repair, `TestCoreScriptBudget` and `TestSessionReadsRoutes` pass over three
Go runs and Linux `nogui` build/vet still pass.

On this workstation a pristine v0.1.4335 worktree reproduced the four
`internal/agent` failures from the full run. The candidate changes no
`internal/cli` code and only one line in an `internal/agent` test fixture.
The remaining full-run failures, and the standalone 20-minute gateway rerun's
fastjson/WSL group, are listed in the release README rather than retried
away; the gateway rerun also completed the stream suite the combined run had
cut off. No complete GUI matrix, cross-platform desktop check or exhaustive
live-vendor test was run.
