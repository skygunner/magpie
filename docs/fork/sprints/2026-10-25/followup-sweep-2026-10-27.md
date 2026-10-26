# Full follow-up sweep — 2026-10-27

All 139 issues/PRs the fork has delivered code or comments for were re-checked
against upstream yetone/magpie (main `d9a8148d7c1e56af756bceee74aadd4faa4aee60`)
and every thread's comments read to its latest reply. Fetched 2026-10-26T17:34:35Z.

Buckets: 19 acted / reworked / answered, 2 closed with no further action,
26 independent-verification-only threads, 92 quiet (no action, evidence reviewed).

Actions this sweep:
- #3632 branch follow-up: merged upstream/main, wake() made lazy too, 47eb5102 pushed, comment 6408812631 edited (zh).
- #3365 probe reply to wh5938316's verifier sample (zh, comment 6423389640).
- #3622 apply-check on current main recorded; no comment (JoeyWu-tech's patch duplicates semantics).
- Ledger: adoptions 3043/3567/3650 (+3276 earlier), independent_alternative 3339/3444/3165,
  reporter-verified notes 3392/3152, unit 3194 recorded, r25 coverage texts corrected,
  generate.py chain check generalized to topological order (merged deliveries), selftest 19/19 green.

| # | state | disposition |
| --- | --- | --- |
| 1072 | open | no change since our last touch; no action, evidence reviewed |
| 1073 | open | no change since our last touch; no action, evidence reviewed |
| 1105 | open | no change since our last touch; no action, evidence reviewed |
| 1258 | open | no change since our last touch; no action, evidence reviewed |
| 2008 | open | no change since our last touch; no action, evidence reviewed |
| 2207 | open | HANDLED by round-25 rework: our comment edited 2026-10-26 after erickguan's 10-25 sighting (focused race count3 passed; no broad claim); no new comment needed. |
| 2259 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 2287 | open | no change since our last touch; no action, evidence reviewed |
| 2323 | open | no change since our last touch; no action, evidence reviewed |
| 2389 | open | no change since our last touch; no action, evidence reviewed |
| 2460 | open | no change since our last touch; no action, evidence reviewed |
| 2473 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 2530 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 2674 | open | no change since our last touch; no action, evidence reviewed |
| 2741 | open | independent verification/rebase by Edward4226 (2026-10-25); no action — evidence reviewed |
| 2759 | open | no change since our last touch; no action, evidence reviewed |
| 2777 | open | no change since our last touch; no action, evidence reviewed |
| 2866 | open | no change since our last touch; no action, evidence reviewed |
| 2867 | open | no change since our last touch; no action, evidence reviewed |
| 2923 | open | no change since our last touch; no action, evidence reviewed |
| 2949 | open | no change since our last touch; no action, evidence reviewed |
| 2953 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 2998 | open | no change since our last touch; no action, evidence reviewed |
| 3020 | open | REVIEWED: WardLu ping (2026-10-23) is for the maintainer; ggbdpq verified our commit; no action. |
| 3027 | open | no change since our last touch; no action, evidence reviewed |
| 3035 | open | no change since our last touch; no action, evidence reviewed |
| 3043 | open | ADOPTED upstream 27ca0bef5 (v0.1.1585), skygunner attribution kept (yetone 2026-10-26); ledger updated; no comment (announcement is the maintainer's). |
| 3062 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 3076 | open | independent verification/rebase by ggbdpq (2026-10-25); no action — evidence reviewed |
| 3095 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 3097 | open | independent verification/rebase by ggbdpq (2026-10-24); no action — evidence reviewed |
| 3100 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 3117 | open | no change since our last touch; no action, evidence reviewed |
| 3121 | open | independent verification/rebase by ggbdpq (2026-10-22); no action — evidence reviewed |
| 3131 | open | no change since our last touch; no action, evidence reviewed |
| 3139 | open | no change since our last touch; no action, evidence reviewed |
| 3141 | open | independent verification/rebase by JoeyWu-tech (2026-10-25); no action — evidence reviewed |
| 3143 | open | no change since our last touch; no action, evidence reviewed |
| 3144 | open | no change since our last touch; no action, evidence reviewed |
| 3147 | open | no change since our last touch; no action, evidence reviewed |
| 3148 | open | no change since our last touch; no action, evidence reviewed |
| 3150 | open | no change since our last touch; no action, evidence reviewed |
| 3151 | closed | closed 2026-10-22; no fork action |
| 3152 | open | REPORTER VERIFIED end-to-end on Windows 11 (1024MrL, 2026-10-23); ggbdpq closed the field half; ledger note added. |
| 3153 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 3156 | open | no change since our last touch; no action, evidence reviewed |
| 3165 | open | UPSTREAM LANDED REPORTED FACE independently (1bb8143b, no trailer from this thread); two defensive faces remain uncovered upstream, covered by fork replay 57a50716; ggbdpq verdict: nothing further requested; ledger updated, no comment. |
| 3169 | open | no change since our last touch; no action, evidence reviewed |
| 3173 | closed | closed 2026-10-22; no fork action |
| 3174 | open | HANDLED by round-25 rework: our comment edited 2026-10-26 (race attribution unresolved, narrow patch retained); ggbdpq re-offered the two-line drop 2026-10-24; no new comment needed. |
| 3176 | open | independent verification/rebase by ggbdpq (2026-10-22); no action — evidence reviewed |
| 3183 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 3193 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 3197 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 3205 | open | independent verification/rebase by ggbdpq (2026-10-22); no action — evidence reviewed |
| 3213 | open | no change since our last touch; no action, evidence reviewed |
| 3224 | open | no change since our last touch; no action, evidence reviewed |
| 3229 | open | no change since our last touch; no action, evidence reviewed |
| 3233 | open | no change since our last touch; no action, evidence reviewed |
| 3234 | open | no change since our last touch; no action, evidence reviewed |
| 3246 | open | no change since our last touch; no action, evidence reviewed |
| 3248 | open | no change since our last touch; no action, evidence reviewed |
| 3251 | open | PARTIAL ADOPTION recorded earlier (8278fd9b0, layer 1); sweep note: 3276 closed with reporter confirmation; layer-2 candidate by Edward4226 (2026-10-25) awaits maintainer decision — not ours to take. |
| 3255 | open | no change since our last touch; no action, evidence reviewed |
| 3258 | open | no change since our last touch; no action, evidence reviewed |
| 3259 | open | no change since our last touch; no action, evidence reviewed |
| 3260 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 3266 | open | no change since our last touch; no action, evidence reviewed |
| 3276 | closed | ADOPTED upstream 8278fd9b0 (v0.1.1325), credited; issue closed 2026-10-23, reporter confirmed; ledger updated. |
| 3277 | open | no change since our last touch; no action, evidence reviewed |
| 3280 | open | no change since our last touch; no action, evidence reviewed |
| 3282 | open | no change since our last touch; no action, evidence reviewed |
| 3284 | open | independent verification/rebase by Edward4226 (2026-10-25); no action — evidence reviewed |
| 3285 | open | no change since our last touch; no action, evidence reviewed |
| 3289 | open | independent verification/rebase by ggbdpq (2026-10-25); no action — evidence reviewed |
| 3302 | open | no change since our last touch; no action, evidence reviewed |
| 3303 | open | no change since our last touch; no action, evidence reviewed |
| 3306 | open | no change since our last touch; no action, evidence reviewed |
| 3311 | open | no change since our last touch; no action, evidence reviewed |
| 3329 | open | no change since our last touch; no action, evidence reviewed |
| 3339 | closed | UPSTREAM INDEPENDENT a7ed2f61d (antigravity disabled-ids filter, Edward4226-verified 2026-10-24); issue closed with reporter's thanks; no fork action; ledger updated. |
| 3345 | open | no change since our last touch; no action, evidence reviewed |
| 3349 | open | independent verification/rebase by Edward4226 (2026-10-24); no action — evidence reviewed |
| 3352 | open | independent verification/rebase by ggbdpq (2026-10-23); no action — evidence reviewed |
| 3358 | open | no change since our last touch; no action, evidence reviewed |
| 3365 | open | ANSWERED (zh): probe on upstream main 19253f67 shows a bare unlearned 'verifier' never rule-matches; only known-propagation after a credential-value occurrence replaces it (comment 6423389640); decisive earlier-occurrence check requested from wh5938316. |
| 3370 | open | REVIEWED: Kurisu0306's PR-attempt note (2026-10-24) is about their own permissions, not our delivery; no action. |
| 3374 | open | no change since our last touch; no action, evidence reviewed |
| 3392 | open | REPORTER VERIFIED (kongkongklng 2026-10-25), asked maintainer to land; merge question is the maintainer's; no comment from us (would be noise). |
| 3401 | open | REVIEWED: FuchaZ 2026-10-26 summary restates the Library-layer gap for issue scanners; Reasonix lane state unchanged (awaiting reporter's Windows run); no action. |
| 3430 | open | no change since our last touch; no action, evidence reviewed |
| 3434 | open | no change since our last touch; no action, evidence reviewed |
| 3440 | open | independent verification/rebase by ggbdpq (2026-10-24); no action — evidence reviewed |
| 3444 | open | SUPERSEDED by upstream 3e9efe1d (ggbdpq 2026-10-25); qiuyu21 zstd measurement + machenbin315-design pre-send split documented in thread; ledger updated to independent_alternative. |
| 3447 | closed | CLOSED: upstream 1d6b72ad (release since v0.1.1369), reporter-verified; recorded previously, no action. |
| 3455 | open | no change since our last touch; no action, evidence reviewed |
| 3460 | open | no change since our last touch; no action, evidence reviewed |
| 3470 | open | no change since our last touch; no action, evidence reviewed |
| 3475 | open | no change since our last touch; no action, evidence reviewed |
| 3487 | open | no change since our last touch; no action, evidence reviewed |
| 3513 | open | independent verification/rebase by ggbdpq (2026-10-25); no action — evidence reviewed |
| 3514 | open | no change since our last touch; no action, evidence reviewed |
| 3519 | open | independent verification/rebase by ggbdpq (2026-10-25); no action — evidence reviewed |
| 3528 | open | no change since our last touch; no action, evidence reviewed |
| 3529 | open | no change since our last touch; no action, evidence reviewed |
| 3530 | open | no change since our last touch; no action, evidence reviewed |
| 3531 | open | no change since our last touch; no action, evidence reviewed |
| 3541 | open | no change since our last touch; no action, evidence reviewed |
| 3553 | open | no change since our last touch; no action, evidence reviewed |
| 3561 | open | no change since our last touch; no action, evidence reviewed |
| 3566 | open | no change since our last touch; no action, evidence reviewed |
| 3567 | closed | ADOPTED upstream cec54c1d0, credited as co-author; issue closed 2026-10-25; ledger updated. |
| 3572 | open | no change since our last touch; no action, evidence reviewed |
| 3585 | open | no change since our last touch; no action, evidence reviewed |
| 3595 | open | no change since our last touch; no action, evidence reviewed |
| 3597 | open | no change since our last touch; no action, evidence reviewed |
| 3598 | open | no change since our last touch; no action, evidence reviewed |
| 3601 | open | no change since our last touch; no action, evidence reviewed |
| 3614 | open | no change since our last touch; no action, evidence reviewed |
| 3619 | open | no change since our last touch; no action, evidence reviewed |
| 3620 | open | no change since our last touch; no action, evidence reviewed |
| 3621 | open | no change since our last touch; no action, evidence reviewed |
| 3622 | open | VERIFIED: patch applies --3way clean onto upstream/main d9a8148d (2026-10-27); JoeyWu-tech's 63f4c0d5+f5bc278f carries the same pruneDist semantics plus lock repair — no further comment posted to avoid duplication. |
| 3624 | open | no change since our last touch; no action, evidence reviewed |
| 3628 | open | no change since our last touch; no action, evidence reviewed |
| 3629 | open | no change since our last touch; no action, evidence reviewed |
| 3630 | open | no change since our last touch; no action, evidence reviewed |
| 3632 | open | REWORKED: branch merged upstream/main d9a8148d (de49433f wake) with lazy close in wake too, follow-up commit 47eb5102 pushed (normal), comment 6408812631 edited with red/green + provider-flake boundary; JoeyWu-tech's 0576b589 credited as independent two-site cover. |
| 3645 | open | no change since our last touch; no action, evidence reviewed |
| 3647 | open | no change since our last touch; no action, evidence reviewed |
| 3648 | open | no change since our last touch; no action, evidence reviewed |
| 3650 | open | ADOPTED upstream 9733ed1b4 (v0.1.1608), ported from fix/usage-absent-used with credit; ledger updated. |
| 3655 | open | no change since our last touch; no action, evidence reviewed |
| 3656 | open | no change since our last touch; no action, evidence reviewed |
| 3657 | open | no change since our last touch; no action, evidence reviewed |
| 3661 | open | no change since our last touch; no action, evidence reviewed |
| 3668 | open | no change since our last touch; no action, evidence reviewed |
| 3674 | open | no change since our last touch; no action, evidence reviewed |
| 3684 | open | no change since our last touch; no action, evidence reviewed |

## Not delivered (round-25 lanes, honestly open)

- fix/r25-proc-listening-bound, fix/r25-antigravity-live-catalog, fix/r25-opencode-session-count:
  local branches at base only, clean worktrees, nothing pushed or commented. Round-25 shortfall stands at 1
  (14 of 15 production units). Not part of this sweep's mandate (no delivered fix to follow up).
- #2622 (sensenova tools normalization / "inference request is invalid"): an uncommitted
  fallback.go sketch was found stray in the dev tree; parked as
  evidence/2026-10-27/parked-2622-invalid-request-words.patch and reverted from the tree.
  #2622 is not in the fork's delivered set; not claimed here.
