# Magpie upstream follow-up notepad — incremental sweep 2026-11-20

Scope: the same 357 upstream issues tracked in the 2026-11-17 sweep.
The baseline watermark was `2026-11-19T19:30:11Z`. The fresh fetch in
`/tmp/magpie-issue-sweep.FrisJF` produced all 1,071 JSON files (issue,
comments and timeline for each issue), with zero missing. GraphQL was
cross-checked at `2026-11-20T03:17:54Z`.

For the next sweep, use **`2026-11-20T03:00:34Z`** as the conservative
coverage/safety watermark, so events during the fetch are rechecked. The
fetch directory's `start.txt` records `2026-11-20T03:00:57Z`; the earlier
watermark deliberately preserves the overlap. The GraphQL check time is
not the next watermark.

## Fresh activity since the baseline

- Four new comments on two issues: three on [#2785](https://github.com/yetone/magpie/issues/2785) and one on [#2878](https://github.com/yetone/magpie/issues/2878).
- No new close or reopen events and no newly linked PRs in the tracked set.
- [#3793](https://github.com/yetone/magpie/issues/3793) received a new
  cross-reference from [#7265](https://github.com/yetone/magpie/issues/7265).

## Follow-up

- **#2785 — macOS window position.** The reporter says both a dual-display
  Mac and a single-display MacBook reopen the window centered on the main
  screen while retaining its size, and points to #6363 for the same
  regression. The existing `fix/2785-window-monitor` patch at `f4e454fa2`
  addresses screen restoration for maximized windows; it does not cover
  ordinary windows. An empty screen cache is a possible explanation for
  the ordinary-window behavior, not a confirmed root cause: it has not
  been verified on a real Mac. Investigate and reproduce before asserting
  the cause or claiming the patch resolves this report.
- **#2878 — credit pools.** The new comment reports branch
  `fix/2878-credit-pools` at `79f50f6e9`; the commit-message updates leave
  the file tree unchanged from `d174c5e41`. The focused validation and
  red/green witness are already recorded in `evidence/2878/`. The branch
  is awaiting upstream pickup; no further branch change is indicated by
  this sweep.
