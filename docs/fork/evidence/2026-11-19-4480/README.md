# Production fork 0.1.4480

Image source 443cccd518133f49210aeaf5e0c7505cc511a1bb merges upstream v0.1.4480
(35e5989f17705939547faaa598e25a1c36f523dc) with integration commit 393141a2a,
a descendant of the accepted 0.1.4343-p1 source 095fc5f96. The upstream
range since the running release contains 186 commits. No runtime conflict
resolution or new runtime patch was needed.

## Patch decisions

The same behavior probes ran against the candidate and a separate checkout of
the pinned upstream tag. The control used only test code from the fork, with
upstream production sources and browser assets. Each row below failed on
pristine upstream and passed on the candidate.

| Retain | Behavior exercised | Regression probe |
| --- | --- | --- |
| #3035 | Retry a guided-grammar compiler refusal with a compatible output format | TestPassthroughSchemaRefusedAsGrammarErrorAskedAsObject |
| #3635 | Retry explicit NVIDIA field refusals while keeping the requested reasoning effort | TestNvidiaRedundantReasoningCompatibility |
| #3636 | Fail over on a named parameter list in a 400/422 without resting the account | TestGroupFailsOverUnsupportedParameterList |
| #3794 | Fail over when a vendor rejects the tool count, preserving the tools for the next member | TestGroupFailsOverAToolCountLimit |
| #3624 | Bring the selected routing story into view from Usage | usage-routing-visible, Chromium |
| #3623 | Show the actual ordered attempt chain for a canceled request | usage-ledger-detail, Chromium |
| #3948 | Keep HTTP 200 on the wire and record a confirmed malformed response as a protocol error | TestProtocolFailureKeepsHTTPStatus and usage-speed-every-row |
| #4385 | Restore the interrupted pixel wheel step in WebKit, including after a click | click-scroll, late-heard wheel |
| Build -pN | Compare patched builds to their base for update checks and release notes | TestNewerTreatsAProdBuildAsItsBase and TestBetweenAProdBuildRange |

Previous suspensions (#3487 remainder, #3788, #3791 and cooldown settings)
stay suspended. The absorbed #3417 remains upstream-owned. The separate
Cloudflare Workers AI plugin was outside this image integration.

## Candidate verification

Linux nogui build and vet passed. Both the host binary and the docker1 image
reported magpie 0.1.4480-p1. The shell entry passed bash -n, ShellCheck and
80 deployment transaction tests. Five browser fixtures passed all 228 cases
across Chromium and WebKit. The retained gateway, update and GUI Go checks
passed, including the core script budget.

The wider local Go runs used Go 1.27.4. Gateway hit its 10-minute package
timeout; the remaining tests were rerun starting at TestThinkingStreamsWithNoOneElse.
The two JSON marshal comparisons, FuzzMarshalRaw seed, two WSL listener tests,
Claude sign-in fixture and TestSessionReadsRoutes failed on the original
upstream as well. TestSessionReadsRoutes passed three standalone candidate
runs. The full GUI and cross-platform matrices were outside this release gate.

## First attempt, 0.1.4480-p1

The initial acceptance snapshot matched and all three real inference probes
returned HTTP 200. At 2760.3 seconds the final API snapshot had gained six
model IDs and two automatic groups for GPT-6.2 Astra and Sol. Nothing was
removed. All eleven samples were healthy, with 28 HTTP 200, two client-closed
499, zero restarts, zero logged 5xx and zero protocol errors.

The transaction removed the candidate and restarted the original container.
Its snapshot still contained the additions, so the transaction recorded
FAILED_ROLLBACK (final_snapshot_changed). A fresh authenticated check of the
original container matched the candidate's final snapshot in all fields
(276 models, 53 groups, 36 providers); a fresh real inference returned 200.
The trigger for the additions was not established. The stopped-volume archive
and failed result remain on docker1. This build was not published.

The second attempt uses a fresh per-version transaction and a build from the
same source with the -p2 version string. Its acceptance and final snapshots
must both match its own live production baseline.

## Accepted and published, 0.1.4480-p2

Initial API parity was exact (276 models, 53 groups, 36 providers) and the
real inference returned HTTP 200 at 16:31:48 UTC on November 18. The owner
accepted this result and directed immediate publication. The observer ended
at 16:35:58 UTC, about four minutes after acceptance. This was an explicit
owner acceptance; the automated 46-minute final snapshot did not run for p2.
The production container continued serving, healthy with zero restarts and
no OOM. Its ID is 8be0766a5128fe1c54ed791705c59a7ba1b851e58c8f3be4bc220e7a3c25f2dd.

Both Docker Hub manifests were retrieved and hashed after the pushes.
bigshort/magpie-patched:0.1.4480-p2 and bigshort/magpie-patched:latest
match the running image, sha256:4ee5602b8d5dd0f4cb178136cf15beae564c76b819eb8ab1e6d7965f7867f1eb.
The accepted image was promoted directly from docker1.
