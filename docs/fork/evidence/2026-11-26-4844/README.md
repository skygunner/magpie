# 0.1.4844-p1 integration

The candidate merges 111 upstream commits through v0.1.4844
(`dc87829ebe38450f1c4ab3879728656a84679fb8`) into the integration line
containing the 0.1.4759-p1 production source. The image was built on docker1
from commit `cbf17aaef17bcbd197d04b8e06894385d98da028`.
The gateway merge keeps the Raccoon Chat alias repair before upstream's
updated reasoning replay predicate. The routing documentation carries both
upstream key-model notices and the existing fork's slow-start ladder.

| Retained production behavior | Candidate check |
| --- | --- |
| #3035 | Explicit guided-grammar refusals retry with a compatible output format |
| #3635 | NVIDIA optional-field repair preserves the requested reasoning effort |
| #3636 | An unsupported parameter list falls through without resting the account |
| #3794 | Tool-count refusals pass the entire tool list to the next member |
| #3948 | Usage preserves committed HTTP status and identifies invalid protocol replies |
| #3623 | Canceled requests show their actual attempt chain |
| #3624 | Usage navigation reveals the selected Routing story |
| #4385 | WebKit recovers the full interrupted pixel-wheel step |
| Patched version checks | Update and release-note comparisons treat -pN builds as their base |
| #7668 | Scoped Raccoon alias deduplication preserves distinct reasoning and tool history |
| Slow-start cooldown | Per-model and per-account ladder begins at the first timeout and caps at 60 minutes |

The candidate passed the targeted Go checks, selected gateway race tests, GUI
core size budget, 110 Chromium and WebKit checks, and 83 deploy transaction
tests. The deployment entry passed bash -n and ShellCheck. Its SHA-256
matches the root-owned HTTPS-aware script used for 0.1.4759-p1. A clean
container on docker1 reported the expected version and passed native HTTPS
gateway and console checks with CA verification and the container healthcheck.

## Baseline comparisons

Pristine 4844 failed the counterpart regressions for the eight issue
behaviors, the -pN comparison, Raccoon reasoning replay and the first-timeout
cooldown. These controls ran the fork assertions against upstream code or
assets. The candidate passed those same behaviors.

The broader Go run on the workstation's Go 1.27.4-X reported agent and CLI
failures. Both trees failed the same five isolated agent cases
(`TestAlmaProvidersSeenDecodeOnce`, `TestHermesProfilesAreAgents`,
`TestWSLProbeFindsBunOmp`, `TestAgyPlugin` and
`TestWSLProbeLooksPastWindowsCommand`) and two CLI cases
(`TestEveryAgentFieldNameWritesNothing` and
`TestAgentOpenCodeDefaultPutsBackTheModel`). The DSH settings permissions
test passed on both trees with umask 022.

The existing production gateway at https://docker1.home.arpa:3425/v1
returned 263 models and a nonempty real inference with HTTP 200 using the
workstation client's existing credentials and CA trust before the switch.
The console also passed certificate validation on port 3430.

The broad Go run passed 103 packages. Its two provider cases
(`TestClaudeSignInRunsTheNewestClaudeCode` and
`TestClaudeSignInSaysClaudeCodeIsTooOld`) reproduced on pristine 4844.
The broad gateway package hit Go's ten-minute timeout. The timed-out
streaming test, the two JSON serialization checks, and the GUI
`TestSessionReadsRoutes` check all passed on both trees on Go 1.26.8
when rerun directly. Targeted gateway race tests also passed on Go
1.26.8, as did the Linux headless vet check.

## Production acceptance and Docker Hub

The owner authorized the switch using the SHA-256-verified, root-owned
deployment entry. The transaction kept the old container and a backup of
the production volume. The new container passed acceptance at 11:24:54 UTC
on November 26. Authenticated before and after snapshots matched across
263 models, 53 groups and 36 providers. The first sample reported healthy
with zero restarts, logged 5xx and protocol errors.

The workstation verified the existing HTTPS gateway on
https://docker1.home.arpa:3425/v1 using its existing CA trust and credentials.
The route returned 263 models and real inference returned HTTP 200 with
nonempty content. The gateway CA fingerprint matched its pre-switch value.
The HTTPS console on port 3430 passed TLS verification and returned the
expected unauthenticated HTTP 401. The observer was stopped after short
acceptance, before the 46-minute observation and final snapshot.

Production container cfba568a770fd42f62e9c12db3184ca4a34ba37b48a7500eaa24997b4f7939bd
ran image `sha256:aae80e87207af90d0d1996b8c9acfcb801e3d0158cbb52ecf4eadab022b0ef32`.
Both Docker Hub tags, bigshort/magpie-patched:0.1.4844-p1 and :latest,
returned manifest bytes matching that digest after publication.
