# 0.1.3295-p1 release evidence

Pinned upstream: v0.1.3295, 0fc3455e2aa955466081b494795298e5ebbc8dc2.
Tested source: 2fb297ebcd179ea488f4363a86c922f307d22449.
Production acceptance is recorded separately in deployment-result.json;
the source gate alone does not establish a successful deployment.

## Test scope

| Check | Result |
| --- | --- |
| Full Go suite on the tested source | 88 packages; 25,914 passing tests/subtests; 20 packages without tests |
| Build and vet | Linux, Windows and Darwin nogui; native default build/vet: 8 checks pass |
| Retained gateway regressions | Race detector, 20 repetitions; update tests pass |
| Deployment transaction | 80 unit tests pass |
| Original complete GUI matrix | 1,275 files on 575d7bda3: 1,167 initially pass, 108 initially fail |
| Failed-file comparisons | Every initial failure has a pristine v0.1.3295 run and one candidate retry |
| Final affected GUI and guard | 11 changed fixtures plus the unchanged navigation guard: 12 files, 250 tests pass |
| Official Ubuntu browser image | Core regression 86/86; late-heard wheel regression 82/82 |
| Isolated production configuration replay | 268 models, 32 providers, 49 groups; no unexpected differences |

The expanded GUI matrix is not fully green. First failures are preserved.
Case-level matches, assertion-template matches, passing baseline attempts and
passing retries are distinguished in [gui-review.json](gui-review.json).
File completion does not imply every nested case ran after a parent failed.

After the original matrix, only documentation and 11 GUI test fixtures changed.
All runtime files, the other 1,264 fixtures and the built program are identical.
The official browser regressions ran against the original matrix's runtime;
they are reused through that verified identity, rather than described as fresh
runs on the final source label. See [runtime binding](final-runtime-binding.json)
and [binary binding](binary-equivalence.json).

Fixture repairs preserve the behavior being checked: #3476 compares actual
scroll positions across a redraw; #3479 classifies one exact known observer
notification; contrast waits for the existing view entrance; #5425 separates
navigation room from short action deadlines; #4715 pins the Mac-only transition
cases to the Mac platform; #3501 reads a rebuilt tab in one page turn. Posted
fix authors are preserved on the source branch fix/gui-fixtures-3295.

The panel duplicate Usage read in #4717 remains an inherited runtime issue,
already observed in the previous release. Its newly posted runtime proposal
is left for independent review; it is not described as merely load or as lacking
a proposed fix. Other layout, scrolling, fixture lifecycle and engine boundaries
are documented individually in the GUI review.

Physical Mac/WKWebView/trackpad behavior is not tested. Real inference probes
cover the deployed probe route, not every configured vendor and account.

## Evidence

- [Source gate](gate-result.json)
- [Patch decisions](patch-review.md)
- [Final issue follow-up](issue-follow-up.json)
- [Production API comparison](acceptance-result.json)
- [GUI failure review](gui-review.json)
- [Accepted production result](deployment-result.json)
- [Published manifest and config digests](published-digest.json)
- [Evidence hashes](evidence-hashes.json); raw logs and production snapshots are retained privately
