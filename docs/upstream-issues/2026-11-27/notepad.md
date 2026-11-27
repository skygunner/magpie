# Magpie upstream follow-up notepad — incremental sweep 2026-11-27

Scope: the same 357 upstream issues tracked in the 2026-11-17 sweep.
The baseline watermark was `2026-11-23T19:01:16Z`. The fresh fetch in
`/tmp/magpie-issue-sweep.5HQploUl` produced all 1,071 JSON files, with zero
missing. Fetch started `2026-11-26T14:03:02Z`, finished
`2026-11-26T14:13:46Z`.

For the next sweep, use **`2026-11-26T14:03:02Z`** as the conservative
coverage/safety watermark, preserving overlap for events during fetch.
Note: other sessions hold live worktrees in the repo (raccoon, slow-start,
chat-carry, openai-temp, the fork PR #3 integration line); this sweep
touched none of them.

## Fresh activity since the baseline

- Four new comments: #4953 (wuyak), #5359 (neighbads), #3302 (22143966),
  #1139 (comicchang).
- 23 timeline events: closes on #3745 and #4855, and cross-references to
  new upstream issues #7602 (Go 1.27: #2207, #3680, #5348), #7741 (#2808),
  #7827 (#3027), #7843 (#3152), #7844 (#3444), #7654 (#3137), #7668
  (#3165), #7746 (#3674), #7617/#7618 (#4923), #7833 (#3280), #7814
  (#3788), and fork PR #3 citing #4385. Referenced upstream commits are
  the project's own fixes (eab4b006d, 82127a850 — #3013 fixed upstream,
  b9ed47c8f, a02cb9175, f7179d96a, d6ff051bc); none absorb our branches.
  `upstream/main` moved to `28d48864f`.

## Delivered

- **#5359 — rebased the branch.** neighbads verified the fix on a build
  carrying `aca455708` and noted the branch was behind main. Rebased onto
  `upstream/main` (`28d48864f`), resolving two app.js conflicts: the
  account-editor save keeps main's `maxImages`/`billing` and adds the
  plugin-header spread; the custom-editor save takes the branch's
  `headerValues()` form with main's current `decide` line.
  `fix/5359-plugin-account-header-fields` is now `634a41fd4` (force-pushed
  over `aca455708`). `TestPluginAccountHeadersSaveReload` and
  `TestPluginConfiguredHeaders5359` pass at the rebased head
  (evidence/5359/5359-green.log, 5359-gateway.log); `node --check app.js`
  clean. Reply: comment `7194562624`.
- **#4953 — answered wuyak's clarification.** The 62-token numbers came
  from a local test script plus a middleware that prepends the official
  first line, not a real Claude Code session; real-Claude-Code first line
  has no sample. The reply states the branch's boundary honestly: it
  dedupes only when the first line is the official one, and if a real
  Claude Code's first line differs, sending the official prompt once is
  the current correct behavior (#2906/#1378 policy untouched). Awaiting a
  real sample before extending the judgment. Reply: `7194563247`.
- **#3302 — confirmed the branch stands.** The reproduction matches
  `fix/3302-embedding-indices` (`02a461098`); upstream still hasn't taken
  it. Reply: `7194563828`.
- **#1139 — scoped the new request.** comicchang's RFC1918-over-proxy
  case is not covered by `fix/1139-issue-1139` (loopback only), and the
  requested opt-out/force-proxy toggle is a behavior ruling beyond the
  original fix; recorded for the maintainer, workaround unchanged. Reply
  (opener language zh): `7194564350`.

All four replies read back verbatim (author skygunner; SHA-256 per reply
in evidence/readbacks.json).

## Still waiting (unchanged this sweep)

- #4144 `fix/4144-scanframes-resync` (7a1054161), #2785
  `fix/2785-window-place-before-screens` (3e93c30ea) — no maintainer
  response yet; #4953 `fix/4953-translated-first-line` (58f2c396d) —
  awaiting real-Claude-Code sample / pickup; #5150 (19db050a9), #2541
  (cdd25ad1), #4603 (5d99f4477) — awaiting pickup; #2878, #3655/#3331
  community branches — awaiting maintainer choice. #3331, #4945 closed
  upstream (recorded 11-24). #4385 stays hands-off unless the maintainer
  asks; fork PR #3 is another session's integration work — not ours to
  touch.

## Queue for coming sweeps

- #2729: translate-path regression test promised in the 11-24 reply
  (flash+web_search landing on Anthropic when both protocols exist).
- #1682: extend `fix/1682-...` to the keyless-`resolve()` shape
  (takomitkm's report).
- #3352→#7466: port the reporter's key-balance patch.
- #3434→#7388/#7425: evaluate wizardpc-com's thread-affinity branch, then
  decide the Claude Code variant.
- #3927→#7577: extend `fix/3927-...` with restore/sync options apply.

## Second sweep of 2026-11-27 (watermark `2026-11-26T14:03:02Z`)

Fetch in `/tmp/magpie-issue-sweep.Rh9UZJa5`: all 1,071 JSON files, zero
missing; started `2026-11-27T12:36:48Z`, finished `2026-11-27T12:46:22Z`.
For the next sweep, use **`2026-11-27T12:36:48Z`** as the watermark.

- Three new comments: #1139 (comicchang, the community branch), #5696
  (aEgoist, the Agnes phrasing, twice with plugin cross-refs). Four of our
  11-27 replies read back as known. 15 timeline events: cross-references
  to new upstream issues #7909/#7971/#7891/#7927/#7993; #4855's class was
  fixed upstream by `2a47b5a47` (plugins#1563); #5359's referenced commit
  is our own rebase `634a41fd4`. `upstream/main` moved to `ad69092c0`.
- **#5696 — Agnes phrasing delivered.** Rebased the branch onto
  `ad69092c0` (shape word joins the lazyre list main grew; docs row beside
  main's LAN-reach row) and added Agnes AI's words as a learned cap.
  Branch `fix/5696-image-count-fallback` now `206c9bdb3` (force-pushed
  over `0b865efce`). `TestImageCapLearnedFromAgnesWords` red with the
  phrasing set aside, green with it; package suite green except the known
  `TestClaudeClientCacheAbsentUnchanged` environment red (#3280), shown
  equal on a tree without these changes. Reply: comment `7218953336`.
- **#1139 — comicchang's branch verified and endorsed.** Their
  `fix/proxy-lan-bypass` (`1849425db`, base `bcb136c9e`) absorbs our
  loopback fix and adds the NoProxy Auto/Off/Custom design (16 files,
  761 lines). Harm check clean (no new deps, no new addresses or
  credential flows); `go build`/`go vet` clean; targeted tests green
  (netproxy, settings, settingsapi full packages, plugin LAN/NoProxy).
  Ablation witness: breaking the `none` mode in `Func` turns
  `TestNoProxyModesAndCIDR` red (`nas.tail1234.ts.net:3425 … want proxy`),
  green restored. Reply (zh) endorses it as taking precedence over our
  loopback-only branch, notes the rebase-to-main need, leaves attribution
  to the maintainer: comment `7218953724` (edited once to fix a
  "comicchang" typo). Both replies read back verbatim; SHA-256 in
  evidence/readbacks-round2.json.
- Fork PR #3's referenced commit check: #4385 citation comes from the
  other session's integration PR — untouched, per the standing rule.
