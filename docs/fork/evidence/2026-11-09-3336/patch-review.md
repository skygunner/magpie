# v0.1.3336 integration review

Pinned upstream:c6bdc172a01e1eb8713c596e33e1ef703eb9b4ee. Integration base:1eeeeda55040bffede5cbc322b7efaa1c021410c.
Previous accepted source:2fb297ebcd179ea488f4363a86c922f307d22449.
51 upstream commits,237 changed paths. Candidate status is separate from production.

## Patch decisions

Retain #3035's explicit grammar-format downgrade; #3635's quoted parameter-list
repair and proven-equivalent reasoning aliases; #3636's narrow unsupported-list
group fallback; #3624's explicit Usage-to-Routing reveal; #3948's bounded
nonstream protocol diagnostics with committed HTTP status; #3794's full-tools
count-limit fallback; #3623's actual ordered attempts, including499; and #4385's
complete interrupted pixel-wheel step per view. Build-pN normalization remains.

The functional delta is unchanged:12 files; a separate trace.go comment repair
makes13 non-test internal paths,+250/-44. Upstream postDecideSent model headers
reach the classifier while existing alias synchronization remains. Gateway-key
slots wrap admission/queueing; the scoped fallback and protocol diagnostics keep
their existing meanings. The new whole Responses response.failed envelope is a
streaming path and does not replace nonstream malformed-envelope detection.
Routing's caller-key names/groups-back behavior and session context lines remain
upstream; Usage's explicit trusted-click reveal and attempt details remain scoped.
Upstream's image base64 normalization and native compaction routing are preserved.

#3487 remainder,#3788,#3791 and the layered cooldown editor/API stay suspended:
new expiring typed balances,queueing and default routing do not justify restoring
competing quota/cooldown semantics. #3417 remains absorbed. #3430 remains absorbed
through upstream. The Cloudflare Workers AI plugin remains0.0.5 separately.

## Issue follow-up

All eight retained issues are still open,with unchanged discussions since3295.
Read the new comments on #3174 and #3392 to their ends. #3174's author is tracing
an earlier goroutine caching credentials after a test; no ready new fix is posted.
Only revisit if its failure is actually encountered. #3392 now recommends the
independently verified third-party2c9cf0f proposal over our old narrow branch.
Our old branch stays archived; that broader xAI request/stream proposal remains
for independent review rather than expanding this deployment. #3165's residual
spelling faces stay separate. No duplicate issue reply or upstream PR is posted.

Production already has the required ZCode0.1.57,WorkBuddy0.2.26,Grok0.2.6 and
Kiro0.1.33 packages. No plugin package update is needed for these mover floors.

## Proportional validation

Freeze one candidate source,then run Go once,Linux nogui build/vet,one selected
race run,80 deployment tests and eight browser fixtures: click-scroll,
view-open-wheel,usage-ledger-detail,usage-routing-visible,usage-speed-every-row,
gateway-key-concurrency,model-limits and sessions-context-line. Browser fixtures
run both engines with their existing language/width cases; Mesa software rendering
is test-only. No complete GUI matrix,unrelated fixture repairs,or cross-platform
desktop release testing is planned. Only a new relevant failure expands this scope.
Actual results and image-bound production replay are recorded separately.
