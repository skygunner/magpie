# 0.1.4717-p1 production integration

The release merges 65 upstream commits through v0.1.4717
(e18f4841f68fc722dbb4e9e0e1f426289def39b9) onto the integration line
containing the previous production source cd25f5290. The merge completed
without conflicts. The docker1 build used a Git archive of merged commit
ca8d2f52f7f25da0d09fdb49d6b43e506d9c1252.

## Patch checks

| Issue | Behavior exercised |
| --- | --- |
| #3035 | Retry a guided-grammar rejection with a compatible output format |
| #3635 | Preserve requested reasoning effort when retrying an explicit NVIDIA parameter rejection |
| #3636 | Fail over on unsupported parameter lists without resting the account |
| #3794 | Hand a tool-count rejection to the next group member with the full tool list |
| #3948 | Keep the wire status and report a malformed response separately |
| #3623 | Show the recorded attempt chain for a canceled request |
| #3624 | Reveal the selected routing story when navigating from Usage |
| #4385 | Restore interrupted pixel-wheel input on WebKit |
| Patched builds | Compare -pN versions as their base release in update and release-note checks |

Each retained behavior passed on the candidate and had a failing probe on
pristine upstream with the fork assertions. The candidate passed all 110
Chromium and WebKit cases, the focused Go gateway, provider and library
checks, the full update and usage packages, and Linux vet. The deployment
entry passed bash -n, ShellCheck and 80 transaction tests. The SHA-256 of
the installed root-owned entry matched the reviewed script. The docker1
image smoke returned magpie 0.1.4717-p1.

## Known upstream test failure

TestCoreScriptBudget failed for total core scripts on both trees. Pristine
upstream measured 1,278,221 bytes versus its 1,277,787 budget. The candidate
measured 1,278,651 versus 1,278,217. Both exceeded their budget by exactly
434 bytes; the fork added exactly its existing 430-byte allowance. The
app.js budget passed on both trees. This size-only upstream failure was
accepted for this release with no change to the code or budget.

## Deployment and publication

On November 23 the authenticated pre-switch and candidate snapshots matched
across 269 models, 52 groups and 36 providers. Real inference returned HTTP
200 with nonempty content at 12:06:06 UTC. The 12:06:09 UTC sample reported
healthy, zero restarts, zero logged 5xx and zero protocol errors. The observer
was stopped after short acceptance, before the extended observation and
final snapshot.

Production container 716830935e8b092161511a213b5c63e674ae5db274a8081380cf700eb537a313
ran image sha256:2702de37785ac18f63711dece0aaaab6e15486d23a73efab494ef583a25e684c.
Both Docker Hub tags bigshort/magpie-patched:0.1.4717-p1 and :latest
returned manifest bytes matching that SHA-256 after publication.
