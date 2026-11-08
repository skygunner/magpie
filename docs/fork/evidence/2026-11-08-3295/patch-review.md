# v0.1.3295 upstream-first integration

Pinned upstream: `0fc3455e2aa955466081b494795298e5ebbc8dc2` (`v0.1.3295`).
Previous accepted source: `171e12415ddce084a3c9417e80cee808a5d1589a` (0.1.3190-p1).
Integration base: `f816b7daa65a05b0e5272ba08c9e126566cda844`.
This source is a candidate; production promotion requires the final gate and observation.

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
The retained non-test internal runtime delta now touches 12 files, down from 13.
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

Before freezing the candidate: targeted gateway NVIDIA/member-map/cancellation tests pass;
Usage attempt-chain and diagnostics fixtures pass in Chromium and WebKit. Final complete
Go/GUI/build/race results and isolated production replay will be recorded separately, bound
to the immutable tested source. A baseline failure alone is not permission to ship it.
