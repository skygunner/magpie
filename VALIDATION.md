# NVIDIA parameter compatibility and Usage routing navigation

Code head: `4f7e63a88592439c79bb43d1309432ad5fce44f3`.
Base: upstream `ef3b9f85a46769713d55d21c6c3259fe70b69acd`.

## Scope

- #3635: recognize explicit unsupported-parameter lists; remove only a redundant effort-only reasoning alias; keep proven duplicates aligned through member/classifier effort changes and model fitting. Recheck the equivalence condition on cached refusals.
- #3636: after safe compatibility handling, continue a group on explicit parameter-list refusals without cooldown. Preserve invalid-value errors and nonredundant controls.
- #3624: Usage navigation reveals the selected Routing story and records its scroll position; selection inside Routing stays stationary.
- #3623 is intentionally unchanged. No live gateway deployment or client configuration changes are part of this branch.

## Regression evidence

- `TestNvidiaRedundantReasoningCompatibility` and `TestNvidiaParameterListBoundaries`: fail against unmodified gateway code (400 after one call; no fields recognized), pass with the fix. Covers subsequent cached requests, sole/conflicting/richer reasoning, invalid-value input echoes and nested field names.
- `TestGroupFailsOverUnsupportedParameterList`: fails without fallback changes for both 400 and 422; passes with them. Two successive requests each reach the second member and the first trace has `failShape` and no rest.
- `TestUnsupportedParameterListEscaped`: checks JSON-escaped field delimiters.
- `TestNvidiaReasoningAfterMemberEffort`: fails before alias alignment (400 after two calls), passes afterwards with a group member fixed at xhigh and the client's original high/high fields.
- Final targeted gateway run passed: `go test ./internal/gateway -run 'TestNvidia|TestOptionalField|TestGroupFailsOver|TestUnsupportedParameterList|Test.*Effort' -count=1`.
- `usage-routing-visible.test.cjs`: fails on original assets because the selected story remains above the viewport. Passes in Chromium and WebKit for en, zh, zh-TW, ja and de (10 cases).
- Existing `routing-failed-open.test.cjs` passes in Chromium and WebKit (8 cases), preserving the no-scroll-on-selection behavior.
- Existing `usage-ledger.test.cjs` was run in both engines. Its two newest-live-route assertions and asynchronous route-callback cleanup error also reproduce on the unchanged baseline in both engines. Other assertions passed. These are not claimed fixed.

Browser checks used Playwright 1.64.1. WebKit 2370 needed ICU 74 and Flite shared libraries already available in the machine's older Playwright browser cache; these were supplied to the test browser, with no application dependency added.

## Broad acceptance

At the code head above:

- PASS: `go vet ./...`, `GOOS=linux go vet -tags nogui ./...`, `GOOS=windows go vet ./...`.
- PASS: native `go build`, Linux nogui build, Windows build (outputs written outside the checkout).
- PASS: `gofmt -l` for changed Go files, `git diff --check`, JavaScript syntax checks for both changed assets.
- NOT GREEN: `go test -tags nogui ./...`. Failures are the baseline-reproduced cases listed below, plus `TestHandedOverGatewayFinishesItsStream`. The latter passes in isolated runs on both baseline and this code head, but fails during the full run; its suite-order dependency is unresolved and is not claimed fixed.

The final targeted gateway tests and browser regressions passed; these results do not turn the full suite green.

The failed-case baseline rerun reproduced `TestWriteAtomicReadOnlyHardLink`, `TestClaudeClientCacheAbsentUnchanged`, `TestCodexFittedImagesPastTheBudget`, `TestMarshalRawIsEncodingJSON`, `TestServeWSLListensBesideLoopback`, `FuzzMarshalRaw/seed#7`, `TestCoreScriptBudget`, `TestPluginSOCKSProxy`, `TestProgram/first`, and the provider package's synctest channel panic in `TestUnreadLoginsWriteErrorsKeepCredentials` on unmodified upstream code. An initial full run also failed `TestHandedOverGatewayFinishesItsStream`; that failure did not reproduce in the isolated baseline rerun, so it is not labelled an established baseline failure.

Go acceptance runs use a temporary HOME with GOPATH, GOMODCACHE and GOCACHE pinned to their pre-sandbox values. No real provider account was used to validate this branch. Synthetic upstream servers reproduce the supplied error and public request fields; private conversation exports are not committed.
