Reproduced on Linux with the real Bun 1.4.3 and fixed the test in my fork: [d16f32c](https://github.com/skygunner/magpie/commit/d16f32c358755633bca5c6835ffd4c7e8fc24533), branch `fix/socks-proxy-test`.

A failed HTTPS CONNECT is correctly surfaced as a transport error. The gateway already recognizes both `proxyconnect` and `socks connect` as proxy failures, falls back, and does not rest the account, so no host-side remapping is needed.

The test now accepts only the specific HTTPS `proxyconnect tcp: 502 Bad Gateway: socks connect ...` error as well as the existing HTTP 502 form. Other errors and bad responses still fail, and the working SOCKS route is still checked. Original test: FAIL; fixed plugin test plus plugin/built-in gateway proxy-fallback tests: PASS with `-race -count=3`, sandboxed Go 1.26.8.

Bun 1.4.2 and macOS/Windows runtime were not rerun. This resolves the Bun 1.4.3 assertion mismatch; it does not claim to explain the separately mentioned historical 60-second timeouts.

Combined validation: these changes are included in [065f0c8f](https://github.com/skygunner/magpie/commit/065f0c8ffc9e30225f3d468ef41ec414aab75e02) on `fix/issue-batch-20261023`. At that exact commit, `go test -tags nogui ./... -count=1 -timeout=45m`, native/Linux/Windows vet and native/Linux/Windows builds all passed with Go 1.26.8, a temporary HOME and pinned Go caches (Linux host, umask 027). Cross-platform builds/vet are not Windows runtime tests. These full-suite results apply to the combined commit; the focused results above apply to the individual fix.


---

Round-25 verification follow-up (2026-10-26):

Follow-up on the October 25 report: I rechecked fork head d16f32c358755633bca5c6835ffd4c7e8fc24533 on fix/socks-proxy-test. `GOTOOLCHAIN=go1.26.8 go test -race -count=3 ./internal/plugin -run SOCKS -timeout=600s` returned 0. This covers TestPluginSOCKSProxy and TestInstallThroughSOCKSProxy, including the specific HTTPS CONNECT error shape in your comment. It is a focused rerun; I did not reproduce your Go 1.27.3 WSL2 full-suite environment, so I cannot claim that scenario is resolved. The existing fork patch accepts the specific proxyconnect/SOCKS failure while retaining the working-proxy checks. I am retaining that patch unchanged rather than treating a report from upstream main as proof of behavior with the fork patch applied. No new implementation or upstream PR was created.
