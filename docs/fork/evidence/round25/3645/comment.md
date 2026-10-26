Fixed on the fork in [fix/r25-routing-log-icons-wait](https://github.com/skygunner/magpie/tree/fix/r25-routing-log-icons-wait), commit [833c7cd61728607fdba75637e3271099847c2e70](https://github.com/skygunner/magpie/commit/833c7cd61728607fdba75637e3271099847c2e70), based on upstream `19253f67bca2310c32739a6eaac88546b3574dbc`.

The race is in `internal/gui/tests/routing-log-icons.test.cjs` → `serveLive`: it advanced the fake trace after 300ms regardless of whether the test had inspected the previous story. The Routing client's `poll()` → `renderAll()` → `requestAnimationFrame(drawAll)` path can therefore draw a later trace before the test observes `.rt-steps li.wait`. Increasing the locator timeout would not retain a state that has already gone.

The fixture now holds each trace response behind an explicit promise gate. The test checks the initial waiting story and records its logo nodes, releases the failed-attempt update, observes `li.bad`, then releases the completed update and checks the existing logo/node identity assertions. The terminal long poll stays pending. This fixes the test fixture; no product rendering change was needed.

RED → GREEN: added `live logo fixture holds trace updates until the story is checked` in `routing-log-icons-wait.test.cjs`. It runs the actual fixture with old timer callbacks immediately eligible, modeling a reader that falls behind. On a clean copy of the base with only that new test added, it fails with `AssertionError [ERR_ASSERTION]: trace advanced before the story was checked` (`actual: [1, 2]`, `expected: [1]`). At this commit the same test passes, including checking that the final long poll remains pending.

The failure also reproduces in the browser on the base when the reported mechanism is injected the way the standards prescribe (a late frame, not reruns): with every `requestAnimationFrame` callback delayed 1000ms while the fixture's fake still advances on its 300ms timers, the base's own test dies with the reported `locator.waitFor: Timeout 8000ms exceeded - waiting for locator('.rt-steps li.wait')` on WebKit, in both English and Chinese. At this commit the same injected run passes: the fixture delivers each step only after the one before has been seen, so no fixed 8s budget is racing a fake's cadence any more.

Validation:
- Both affected Node test files: **15 passed, 0 failed**, including Chromium and WebKit and all five live-story languages.
- `GOTOOLCHAIN=go1.26.8 go test ./internal/gui -count=1`: **PASS**, 44.363s.
- `gofmt -l internal/gui`: no output; `GOTOOLCHAIN=go1.26.8 go vet ./...` and `GOTOOLCHAIN=go1.26.8 go build ./...`: PASS.
- `GOTOOLCHAIN=go1.26.8 GOOS=linux GOARCH=amd64 go build -tags nogui ./...` and `GOTOOLCHAIN=go1.26.8 GOOS=windows GOARCH=amd64 go build ./...`: PASS.
- Both changed JavaScript files passed `node --check`; the pushed branch SHA matches the commit above and contains exactly one commit over the base.

What was not tried: the complete 112-file Routing GUI run under the reporter's exact load (three rounds at `--test-concurrency=6` under a synthetic 8-way load passed without reproducing it; the injected late frames above stand in for that), live provider requests, and Windows/macOS runtime testing. The JavaScript language server is unavailable locally. No Docker container was needed.
