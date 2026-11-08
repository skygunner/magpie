# 0.1.3336-p1 release evidence

Pinned upstream: v0.1.3336, c6bdc172a01e1eb8713c596e33e1ef703eb9b4ee.
Tested source: b7b5539de03005a071a57eceaa06fde3f7faecfa.
Production acceptance is established by deployment-result.json, separately
from the source gate. The accepted image is published without rebuilding.

## Deliberately limited test scope

The user requested proportional testing for this integration. The source was
frozen once, then the following checks ran on that source:

- One complete nogui Go run: 88 passing packages, 26,094 passing tests/subtests;
  20 packages have no tests.
- Production Linux nogui build and vet: both pass.
- One selected race run, including retained gateway behaviors and new gateway-key
  admission/concurrency paths: passes, without repeated stress loops.
- Deployment transaction: 80 tests pass.
- Eight browser fixtures: 208 tests pass in Chromium and WebKit, with no failures,
  cancellations or skips. The fixtures cover late wheel restoration, Usage
  diagnostics/attempts/routing/speed, gateway-key concurrency, model limits and
  session context navigation. See [actual scope](test-scope.json) and
  [browser results](gui-result.json).
- Isolated old/candidate production-configuration replay: 268 models,32 providers,
  49 groups; no unexpected differences. Temporary copies and networks are removed.

No complete GUI matrix, unrelated fixture repair campaign, Windows/Darwin desktop
release checks or exhaustive live-vendor checks ran. Browser rendering uses Mesa
software rendering only in the test environment. Physical Mac/WKWebView/trackpad
behavior is not tested. Prior broad GUI findings belong to their historical
records; this scoped run does not claim every old failure is fixed or persists.

## Patch and publication records

The eight retained runtime fixes remain necessary; their functional delta is
unchanged. Prior quota/timeout/cooldown suspensions stay in place. New upstream
model settings, model headers, gateway-key concurrency and streaming failure
envelopes are preserved. No new runtime fork patch is added.

- [Patch review](patch-review.md)
- [Issue follow-up](issue-follow-up.json)
- [Source gate](gate-result.json)
- [Production API comparison](acceptance-result.json)
- [Accepted production result](deployment-result.json)
- [Published manifest and config digests](published-digest.json)
- [Evidence hashes](evidence-hashes.json)

Raw logs, production configuration, API snapshots and rollback backup remain
private. The separate Cloudflare Workers AI plugin remains0.0.5 in the production
volume; it is not bundled in the published image.
