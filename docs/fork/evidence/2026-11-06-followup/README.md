# 2026-11-06 issue-campaign follow-up sweep

Full re-check of all 172 issues/PRs the ledger carries comments on
(1072–3828): state, every comment posted after our last one, and
timeline cross-references. 81 open with nothing new, 29 landed upstream,
9 closed, 8 needed action. Comments posted in each opener's language and
read back via the API (author/issue/created_at/UTF-8 body SHA256):
`comment-readback.json`.

Reworked branches (all red re-proved before the fix, green at the head;
full logs in the per-issue directories):

- `3619/` — branch rewritten as one commit `bbd53987c` on `2df5d40ee`
  after ggbdpq's verification found #153 regressing:
  `red-153-at-old-head-e9307596c.log` (the old head's red, reproduced),
  `green-targeted-bbd53987c.log`, `full-gateway-bbd53987c.log` (only the
  five baseline reds clean main shows here),
  `../baseline-main-five-reds-set.txt`.
- `3566/` — `28919f45b` on `2df5d40ee` with the `$ref` skip guard:
  `red-inner-union-on-clean-main.log`, `green-kimi-28919f45b.log`,
  `full-gateway-28919f45b.log`.
- `3183/` — sealed-task guard `c20d3e202` on
  `fix/issue-batch-20261023-g`: `red-sealed-task-at-parent.log`,
  `green-native-relay-c20d3e202.log`.
- `3329/` — console-marker regression test `fc222ef12` on
  `fix/dsh-comment-between-entries` (rebased onto `2df5d40ee`):
  `red-console-markers-on-clean-main.log`,
  `green-dsh-comments-fc222ef12.log`.

Comment-only follow-ups: #3572 (branch stands down — upstream
`2260390a0` covers both faces), #3259, #3285, #3188.

Ledger: units 3183 3329 3566 3572 3619 3259 3285 3188 updated;
`generate.py --write/--selftest` pass (169 units, 173 issues, 362 remote
heads).
