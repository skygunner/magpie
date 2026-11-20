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
  Follow-up delivered after the sweep: [skygunner's reply](https://github.com/yetone/magpie/issues/2785#issuecomment-7029353823)
  (`2026-11-20T03:22:44Z`) corrects the earlier root-cause claim and asks
  whether `windowAt` exists before restart and whether ordinary exit followed
  by restart also centers the window. Awaiting the reporter's answers and
  real-Mac verification; neither the cause nor a fix for ordinary-window
  restoration is confirmed. The API readback is in
  [evidence/2785/2785-readback.json](evidence/2785/2785-readback.json);
  its `body` UTF-8 bytes (without an added newline) have SHA-256
  `a4026f1d964ea12a10bad084f2751dfc307778d6c8f178b19620f1f6939a002d`.
- **#2878 — credit pools.** The new comment reports branch
  `fix/2878-credit-pools` at `79f50f6e9`; the commit-message updates leave
  the file tree unchanged from `d174c5e41`. The focused validation and
  red/green witness are already recorded in `evidence/2878/`. The branch
  is awaiting upstream pickup; no further branch change is indicated by
  this sweep.
