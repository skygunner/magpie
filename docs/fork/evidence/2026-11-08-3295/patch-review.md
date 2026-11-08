# v0.1.3295 upstream-first integration

Pinned upstream: `0fc3455e2aa955466081b494795298e5ebbc8dc2` (`v0.1.3295`).
Previous accepted source: `171e12415ddce084a3c9417e80cee808a5d1589a` (0.1.3190-p1).
Integration base: `f816b7daa65a05b0e5272ba08c9e126566cda844`.
Accepted production source: 2fb297ebcd179ea488f4363a86c922f307d22449.
Observed and published after acceptance at 2026-11-08 05:08:40 UTC / 2026-11-08 15:08:40 AEST.

## Decisions against current source

| Patch | Decision and current behavior |
| --- | --- |
| #3035 | Retain only guided_grammar/compile_grammar recognition in the existing format downgrade. Upstream still lacks these words. The bare Invalid request extension stays suspended: current passErrors/opaque-overflow policy owns it. |
| #3635 | Retain quoted unsupported-parameter lists and proven duplicate reasoning controls. Upstream's new member EffortMaps calls withFixedEffort; its Chat path reaches withBodyEffort, synchronizing equivalent aliases. A real-handler regression verifies mapped high→xhigh in both fields. Sole/conflicting/richer controls stay intact. |
| #3636 | Retain narrow structured unsupported-list classification, invalid-value exclusions, and no account rest. Account selection refactoring does not replace request-shape handling. |
| #3624 | Retain explicit Usage navigation reveal; normal Routing selection keeps the reader's position. Upstream observer-loop and render-cadence fixes remain. |
| #3948 | Retain bounded nonstream protocol diagnostics and committed status; no retry/rest. Upstream now excludes client cancellation from failure totals. Preserve that rule and its canceled row indicator; protocol/client-report labels remain scoped. |
| #3794 | Retain explicit Kilo tools-count limit fallback with all tools. Upstream's Gemini schema classifier is preserved. |
| #3623 | Retain ordered actual attempts, including 499. New cancellation attribution is displayed alongside the attempt chain; it does not change retry behavior. |
| #4385 | Retain complete interrupted pixel-wheel motion per view. Upstream's restoration still restores only erased fragments. Touch/nonpixel and unheard-wheel policy unchanged. |
| build -pN | Retain base-version comparison and release-note query normalization. Upstream's download stall timer and one-time notes display remain. |
| #3487 remainder / #3788 / #3791 / cooldown settings | Remain suspended. Upstream changed default routing/rest inheritance again and now treats cancellation as no failure; do not reintroduce competing semantics. |
| #3417 | Already absorbed; no fixed-lead fork. |
| plugins#366 | Separate; merged test-only head remains 38ddeb132. Production Cloudflare Workers AI stays 0.0.5. |

Remove the comment-only opencode_client.go difference: no runtime behavior is lost.
The functional runtime delta touches 12 files. The full non-test internal diff
also contains one comment-only trace.go change: 13 files in total, +250/-44.
The trace comment repair is the thirteenth non-test path.
Two merge conflicts were resolved semantically: the Usage row combines scoped diagnostics
with upstream cancellation; the gateway reference keeps retired Kiro/plugin ownership and
adds upstream's user-selectable Claude cache TTL. No retired built-in translator is restored.

## Issue follow-up

Read current state of all 172 distinct issue/PR numbers in the historical ledger, plus
all comments on the retained/suspended patch issues. The eight retained issue behaviors
are still open; their threads contain no later upstream replacement patches.
Read all comments on the four discussions updated since the previous production acceptance:

- #3430 is closed and absorbed in upstream fc7fde82c (v0.1.3199): Messages thinking-token
  accounting is included through this merge, without another fork patch. Reporter confirmed
  live Copilot behavior on v0.1.3216; that is the reporter's check, not our own live-vendor test.
- #3165's original tool-turn alias issue is absorbed; two residual spelling faces remain
  proposed in fix/3165-reasoning-faces. Do not re-add the older broad reasoning rewrite,
  whose treatment of reasoning_details conflicts with upstream's typed-entry policy.
- #3174 still offers a test-only credential-cache fixture reset; review if its failure is
  observed in this round. Do not add unrelated runtime code.
- #3392's new third-party proposal combines empty required arrays with stream-index repair.
  It overlaps rapidly evolving upstream request/stream rules. Leave it out of this production
  delta pending its independent review, under the agreed overlap/suspension rule.

No duplicate issue comments or upstream PRs were posted.

## Verification record

Final tested source: 2fb297ebcd179ea488f4363a86c922f307d22449.
The complete Go run passes 88 packages and 25,914 tests/subtests; 20 packages
have no tests. Eight build/vet checks, gateway race regressions repeated 20 times,
update tests and 80 deployment transaction tests pass. The original 1,275-file
GUI matrix on 575d7bda3 has 108 initially failing files; all have pristine 3295
comparisons and one candidate retry. It is not fully green. The final 12 affected
fixtures/guard pass 250 tests. Runtime, binary and the other 1,264 fixtures remain
identical to the full run. Official Ubuntu core86 and late-wheel82 regressions
pass on that identical runtime. The final isolated production replay matches
268 models/32 providers/49 groups, with the separate Cloudflare package0.0.5.
See gate-result.json, gui-review.json and the runtime/binary bindings.

## Test fixture repairs prepared on the pinned tag

Source branch `fix/gui-fixtures-3295` starts at pristine v0.1.3295 and contains
five independent commits. The key-scroll/header fixes and the atomic Usage-tab
read adapt posted fixes by ggbdpq; the platform pin preserves Rick Chan as author.
The contrast fix follows a captured measurement of the page entrance. Tests only.

- #3476: compare scroll positions across the key editor redraw. The pointer-proximity
  `.holds` marker can change while page/body positions stay identical. Take the
  reader-only fix; no timeout increase. The complete file passes 40/40 in both engines,
  all five languages and both widths. Original paired failures are preserved.
- #3479: classify only the exact benign ResizeObserver undelivered-notification
  message in the header width-sweep test, following the existing purpose-filter/update
  precedent. Every layout assertion and every other page error remains checked.
  Original candidate and pristine 3295 reproduce the page-error string; repaired14/14.
- Group contrast: the first read captured ancestor MAIN.view opacity0 during entrance,
  producing1:1 for every text. Wait for that view to appear before the original
  computation and thresholds. The repaired full file passes8/8. A color-parser-only
  diagnostic still failed; no color-parser or runtime style change is carried.

The complete1,275-file GUI matrix stays frozen on 575d7bda3. Final-source Go/build/race
checks and all changed fixtures are run separately. Gate evidence must verify
that runtime assets and every unchanged fixture are byte-identical between that full
matrix and the final source. This preserves the original failures instead of hiding
or overwriting them. Remaining known GUI failures are reviewed separately.

## Navigation-room guard: fork omission repaired

The full matrix caught #5425's five upstream setup omissions and a sixth in
our own usage-routing-visible fixture. The extra fork entry is an integration
gap and is repaired, rather than waived as the same baseline failure. Add only
setDefaultNavigationTimeout(60000) beside the short action timeout in each of
the six files; existing action deadlines and every behavioral assertion stay.
The unchanged static guard goes from six entries to green. All six full browser
fixtures pass in Chromium/WebKit after the correction.

The final affected scope is eleven changed GUI fixtures plus the unchanged static
navigation guard. The original full1,275-file run remains frozen on 575d7bda3;
all runtime code/assets and the other 1,264 fixtures must be byte-identical to the
final source. Final-source Go/build/race and all twelve affected/guard checks are
bound to the final source, with fresh image/replay binding before deployment.

## Additional posted fixture fixes adopted

- #4715: side-toggle's two Mac-transition cases pin navigator.platform to MacIntel.
  A Linux host correctly skips the product transition, so leaving the platform
  implicit made those assertions invalid. The explicit Linux no-transition case
  remains. Posted0c0aedc91, Rick Chan; final whole file26/26 passes.
- #3501: Usage focus reads the selected tab and its parent in the same in-page
  turn. A detached locator handle could otherwise lose its parent during the
  existing boot redraw. Preserve the old5s patience and all assertions.
  Postedc2bd0b67d, ggbdpq; final whole file4/4 passes.
- #5426/#5332/#5482: adopt Gaoshou101's four Go-comment i18n ownership repairs
  from5659e3ac3. Plugin-owned phrases in comments must not be assigned to the
  core translation table. No runtime behavior changes; the original guard red
  and repaired4/4 result are preserved privately.

## Remaining GUI scope

Some first failures were not reproduced by the single pristine attempt; their
retry, historical reports, source ownership and diagnostic limits are retained
separately. A passing retry is not substituted for the first result. The UI
matrix is not described as all green or as all rerun on the final source label.

#4717's real duplicate panel Usage read was already recorded in3190. The newly
posted0fd066d runtime fix and3e2cb06 fixture changes remain for independent review,
rather than expanding this convergence release to the historical issue backlog.
#3500's superseded row-selector patch and#3922's already-absorbed newest-control
patch are not applied again. #2686's flash-parking patch targets a later assertion,
not this round's earlier#5215 clipped-card visibility failure.

## Accepted production

2761.013 seconds, 11 healthy samples and three successful inference probes.
Acceptance and final API diffs are empty:268 models/32 providers/49 groups.
No restarts, OOM, 5xx or protocol errors. The exact tested image was published
as 0.1.3295-p1 and latest, without rebuilding. See deployment-result.json,
acceptance-result.json and published-digest.json.
