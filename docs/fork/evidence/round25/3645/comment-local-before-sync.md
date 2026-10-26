Fixed on the fork in [fix/r25-routing-log-icons-wait](https://github.com/skygunner/magpie/tree/fix/r25-routing-log-icons-wait), commit [833c7cd61728607fdba75637e3271099847c2e70](https://github.com/skygunner/magpie/commit/833c7cd61728607fdba75637e3271099847c2e70), based on upstream `19253f67bca2310c32739a6eaac88546b3574dbc`.

The race is in `internal/gui/tests/routing-log-icons.test.cjs` → `serveLive`: it advanced the fake trace after 300ms regardless of whether the test had inspected the previous story. The Routing client's `poll()` → `renderAll()` → `requestAnimationFrame(drawAll)` path can therefore draw a later trace before the test observes `.rt-steps li.wait`. Increasing the locator timeout would not retain a state that has already gone.

The fixture now holds each trace response behind an explicit promise gate. The test checks the initial waiting story and records its logo nodes, releases the failed-attempt update, observes `li.bad`, then releases the completed update and checks the existing logo/node identity assertions. The terminal long poll stays pending. This fixes the test fixture; no product rendering change was needed.

RED → GREEN: added `live logo fixture holds trace updates until the story is checked` in `routing-log-icons-wait.test.cjs`. It runs the actual fixture with old timer callbacks immediately eligible, modeling a reader that falls behind. On a clean copy of the base with only that new test added, it fails with `AssertionError [ERR_ASSERTION]: trace advanced before the story was checked` (`actual: [1, 2]`, `expected: [1]`). At this commit the same test passes, including checking that the final long poll remains pending.

Validation:
- Both affected Node test files: **15 passed, 0 failed**, including Chromium and WebKit and all five live-story languages.
- `GOTOOLCHAIN=go1.26.8 go test ./internal/gui -count=1`: **PASS**, 44.363s.
- `gofmt -l internal/gui`: no output; `GOTOOLCHAIN=go1.26.8 go vet ./...` and `GOTOOLCHAIN=go1.26.8 go build ./...`: PASS.
- `GOTOOLCHAIN=go1.26.8 GOOS=linux GOARCH=amd64 go build -tags nogui ./...` and `GOTOOLCHAIN=go1.26.8 GOOS=windows GOARCH=amd64 go build ./...`: PASS.
- Both changed JavaScript files passed `node --check`; the pushed branch SHA matches the commit above and contains exactly one commit over the base.

What was not tried: the complete 112-file Routing GUI run under the reporter's exact load, live provider requests, and Windows/macOS runtime testing. The isolated base browser test passed, as reported in the issue; the deterministic fixture regression supplies the failing baseline evidence. The JavaScript language server is unavailable locally. No Docker container was needed.
