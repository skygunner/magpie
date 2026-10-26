The same natural failure occurred during my full suite on Linux/amd64, Go 1.26.8: `TestClaudeAccountServesModelsListedSinceItsFetch` found no Claude account immediately after fixture sign-in.

I adopted @ggbdpq’s helper fix above with co-author credit and retained a deterministic regression: [873c5dfd](https://github.com/skygunner/magpie/commit/873c5dfd59afc86d4fbace3c4fdf4af57a67dbac) (`fix/claude-signin-test-cache`). The regression first reads the empty fixture home to cache a miss, then calls `claudeSignIn` and requires its credential and Claude provider to be visible. It fails on the old helper; the fix invalidates after the write, covering the helper’s callers in 25 test files. No product credential-cache policy changed.

The retained regression plus the affected model/account/sign-out tests passed 20 race-enabled repetitions. This proves the cached-miss mechanism without leaving a background test goroutine running; it does not identify which earlier asynchronous reader produced the naturally observed miss.

Validation on combined head [0437ae4c](https://github.com/skygunner/magpie/commit/0437ae4cf3be61c64546dc1573a8211e3c6ad7fd) (`fix/issue-batch-20261023-c`): Go 1.26.8, isolated HOME and pinned caches, umask 027; native/Linux-nogui/Windows vet and builds, full `go test -tags nogui ./... -count=1 -timeout=45m`, gofmt and diff checks. The preceding run exposed #3174; this combined branch includes its credited fixture fix. No production deployment or real subscription account was changed.


---

Round-25 verification follow-up (2026-10-26):

Follow-up on the October 24 feedback: I rechecked fork head 873c5dfd59afc86d4fbace3c4fdf4af57a67dbac on fix/claude-signin-test-cache. The narrow `go test -count=10 ./internal/provider -run TestClaudeAccountServesModelsListedSinceItsFetch` returned 0 with Go 1.26.8. That is only a focused result and does not prove the loaded full-package scenario is fixed.

An expanded `go test -race -count=10 ./internal/provider -run "Claude|claude" -timeout=900s` returned 1: TestClaudeAccountServesModelsListedSinceItsFetch reported `race detected during execution of test`. The race detector shows a write in isolate (account_test.go:75), reached through claudeHome, conflicting with a background Allowances path started by TestClaudeAllSignedOutStaysListed via InUseLogin/loginRoom. This run did not fail with the original missing-Account assertion. I have preserved the complete race report and have not established whether this additional race predates the fork patch; it is not being called fixed or harmless.

The existing patch clears the cached miss after the fixture writes its login. @ggbdpq has independently checked that mechanism and supplied additional straggler-stack evidence in this thread. I am retaining the narrow patch without broadening production credential-cache policy; the fresh expanded race result prevents a claim that the whole package is clean. I did not reproduce the maintainer's load-300 environment or use a live Claude account.
