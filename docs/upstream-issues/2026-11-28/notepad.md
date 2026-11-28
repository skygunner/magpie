## Third sweep of 2026-11-27 → 2026-11-28 (watermark `2026-11-27T12:36:48Z`)

Fetch in `/tmp/magpie-issue-sweep.dw8ADMIQ`: all 1,071 JSON files, zero
missing; started `2026-11-28T03:17:19Z`, finished `2026-11-28T03:26:03Z`.
For the next sweep, use **`2026-11-28T03:17:19Z`** as the watermark.

## Fresh activity since the baseline

- Three new comments: #2301 (NkAntony777, deep reproduction), #3213 (xiao-hans,
  real Qoder 503 queue evidence), #3392 (yetone, v0.1.4972 release notes).
- 12 timeline events: closes on #3392, cross-references to #7070/#2301,
  #7701/#3213, #8011/#3392, #8008/#3849, #7971/#3556, #7993/#3475/#4854;
  upstream commits `e2a35008c` (referenced), `c1cd20407`/`8ff20ada7`
  (absorbed #3392), `206c9bdb3` (our own #5696 rebase). `upstream/main` moved
  to `0d310af00`.
- Our three 11-27 replies read back as known.

## Delivered & conclusions

- **#3392 — absorbed upstream.** `v0.1.4972` (c1cd20407, 8ff20ada7) shipped
  with `Co-authored-by: skygunner <16a4053b>` and yangfei1024. Our
  `fix/3392-xai-required-empty` (`16a4053b`) is done; CI clean single-pass.
  Reply: comment `7233182879`.
- **#3213 — real evidence supports A.** xiao-hans reproduced on current
  release: 15 models 503 in one second, refusal text names single model's
  queue, whole account rested. Exactly A (rest the one model). Our
  `fix/queue-model-rest` (`50eac57f6`, `TestQueuedModelRestsAlone` red→green)
  already points that way; rebase needed after `restAfterMarked` moved.
  Reply (en): comment `7233183088`.
- **#2301 — deep reproduction, out of scope.** NkAntony777 shows Usage
  reads MiniMax Code (1.21B tokens, config path with space OK), but Sessions
  reader returns zero rows; synthetic minimal sample also reproduces the
  divergence between Usage reader and Sessions reader. Issue not in the 357
  tracked set; ask the maintainer if they want it added. Reply (zh): comment
  `7233183274`.

All three replies read back verbatim (author skygunner; SHA-256 in
evidence/readbacks.json).

## Still waiting (unchanged)

- #4144, #2785, #4953, #5150, #2541, #4603, #5696, #3213 branch — awaiting
  maintainer pickup/rebase. #4385 still hands-off.