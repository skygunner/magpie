# Magpie upstream follow-up notepad — incremental sweep 2026-11-19

Scope: the 357 issues of the 2026-11-17 sweep, re-fetched fresh
(2026-11-19, all 357 + comments, 0 failures), plus the three pending
items the 11-17 notepad left open. Base of record: the same lineage as
docs/upstream-issues/2026-11-17.

## Fresh activity (comments after 2026-11-17T13:30Z, 6 issues)

- #3331 [zh] westlinkin "0.1.4376 仍未解决" + VectorLu's alternative branch
  (d7b551199, v0.1.4335 base, co-author credit). Verified: 7 usage pricing
  tests green, gofmt/vet clean. Endorsed as superseding our 0bd3651d (zh).
- #3655 [zh] doskoi's independent branch (1f92921f). Verified: 121
  tool-loop tests green, gofmt/vet clean; wider tool-name normalization.
  Compared with ours (29218ae3b), both left for the maintainer (zh).
- #4385 [en] yetone: third "Not taking it" — the answer to our 11-17
  rationale question. Acknowledged once (en), standing down.
- #4854 [en] ggbdpq forwards our fix/4956-4854 branch — no action needed.
- #5150 [en] yetone diagnoses the mote red (sshd forward holds
  127.0.0.1:3425). Our branch (rebased onto today's main as 19db050a9,
  force-pushed over 3ade7c26) isolates owner discovery via MAGPIE_ADDR —
  covers exactly that (en).
- #5425 [en] yetone fixed upstream (549813abb) — closed, no action.

## Pending items from 11-17

- #2785 [zh] Root cause found: a maximised window's place is never kept
  and no screen was kept either, so maximised-on-second-display reopens on
  the main display. Branch fix/2785-window-monitor rewritten on today's
  main (f4e454fa2, force-pushed over fc5cb114a): WindowScreen kept even
  maximised, window put on its screen before the zoom; settingsapi.Apply
  and KeepOwn carry the new field. Asked pluo to verify.
- #4945 [zh] Promised note-placement work delivered: fix/4945-note-apart
  (39c5413b3, on today's main) skips a tool_use-carrying assistant message
  when landing the placeholder note; red on clean main, green at head.
- #4144 [zh] Rework (read-past-corrupt-frame resync + real damaged-file
  test) NOT delivered this round — still pending as of 11-17, unchanged.

## Delivered

| # | lang | reply | branch/head |
|---|---|---|---|
| 2785 | zh | 7009874663 | fix/2785-window-monitor f4e454fa2 (force-push) |
| 4945 | zh | 7009874983 | fix/4945-note-apart 39c5413b3 (new) |
| 5150 | en | 7009875350 | fix/5150-lane-test-owner-isolation 19db050a9 (force-push) |
| 3655 | zh | 7009875705 | (comparison reply; both branches stand) |
| 3331 | zh | 7009876066 | (endorsement of VectorLu's d7b551199) |
| 4385 | en | 7009876387 | (standing down) |

All posted in opener language and read back (posted-replies.json).

## Verification notes

- Full internal/gui run: TestSessionReadsRoutes red under the full run on
  clean upstream/main AND on the branch (passes alone) — environmental
  baseline, not a regression (evidence/gui-*.log).
- #2785 window family + settings guard: green (evidence/2785-*.log).
- #4945: red on clean main, green at head (evidence/4945-*.log).

## Still pending

- #4144 rework (committed 11-17): scanFrames read-past-corrupt-frame
  resync + a test from a real damaged session file. Heaviest item; needs a
  dedicated session.
- #3331/#3655 wait on maintainer pickup; #2785 awaits pluo's dual-display
  verification.
