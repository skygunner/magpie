I reproduced this at pinned v0.1.3036 and have a focused fix in [skygunner/magpie, fix/webkit-delayed-wheel](https://github.com/skygunner/magpie/tree/fix/webkit-delayed-wheel). The branch starts at v0.1.3036 and contains the repair, regression coverage, and subsystem notes; the following comment-only commit keeps it within the existing core-script budget.

The failure is reproducible on Linux in the official Ubuntu Playwright 1.64.1 image, and with 1.62.1. A trace of the original code showed a 300px pixel-wheel input leaving only 18px: the guard writes scrollTop after the first animated fragment, which cancels WebKit's remaining native wheel animation. When the delayed listener finally runs, givesBack only restores the recorded first fragment.

The repair records the restored position and groups erased motion per view. For a pixel wheel it completes the input from that origin, accounting for native motion already kept, instead of adding the same step twice. Non-pixel wheels and touch input still restore only recorded erased distance. It only acts when there is matching erased motion; ordinary wheels with no erased motion are unchanged.

The test keeps the original minimum-movement assertion and adds an upper bound against duplicate movement. The day-click load is allowed to settle before measuring the wheel's own distance, so the assertion does not count asynchronous list expansion as wheel movement.

Verification so far:
- The regression against unmodified v0.1.3036 fails with the same late-heard-wheel assertion.
- The repaired complete click-scroll file passes all 24 tests in Chromium and WebKit, without skips, including code-scroll rejection and the later upward wheel.
- Official Ubuntu final-source check of click-scroll plus account-rest: 34/34 pass, no skips.
- Full Go suite on the integrated fork: 87 packages pass, 20 have no tests. The core-script budget is unchanged and passes.

I have not tested this on a physical Mac/WKWebView or with a physical trackpad, so I am not claiming this resolves every symptom reported in #3086. Those platforms still need confirmation. This addresses the specific delayed-wheel regression here.

Follow-up on the broader load matrix: [fix/gui-wheel-test-contexts](https://github.com/skygunner/magpie/tree/fix/gui-wheel-test-contexts) separately fixes view-open-wheel attaching context cleanup to the enclosing engine test instead of its subtest. A new post-case assertion fails with 1 remaining context under the previous hook and passes with per-case cleanup. Runtime assets are unchanged by that supporting branch. The complete late-wheel mode now passes 82/82 checks in the official Ubuntu image, across both engines, all five languages, both widths and both pages, with no skips. The existing unheard-wheel cadence problem in #4132 and the independently reproduced nested-scroller cases in #4689 remain separate; this patch does not change or claim to fix those policies.
