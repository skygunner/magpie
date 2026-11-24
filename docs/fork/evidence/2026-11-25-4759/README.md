# 0.1.4759-p1 production integration

The source image was built on docker1 from the Git archive of
`5595cf78152ff44d9bbd845ffb7e1bbf4e843e31`. The merge added 60 upstream
commits through `v0.1.4759` (`95e6a4180866afde186e0f1a25369388c7fd7e7f`)
to the tested 4717-p3 source, without textual conflicts.

## Retained behavior

Candidate tests covered the eight retained issue behaviors (#3035, #3635,
#3636, #3623, #3624, #3794, #3948, #4385) and the patched-build version
comparison. The pristine upstream tag failed the corresponding fork behavior
checks. Existing suspensions remain recorded in the patch ledger.

Two more cases failed on the pristine tag and passed on the candidate.
[Upstream #7668](https://github.com/yetone/magpie/issues/7668) reproduces
duplicate assistant reasoning on Raccoon SenseNova Flash. The scoped adapter
keeps one copy of identical text and preserves distinct fields and tool
history. The slow-start test lets every rest expire and verifies waits of
1, 2, 4, 8, 16, 32, 60 and 60 minutes. The existing provider restrictions,
successful-response reset and account/model isolation are covered separately.

## Validation

- Go 1.26.8 passed the full gateway, update and usage packages, targeted race
  checks, the core script size budget and Linux headless vet. The unchanged
  deployment entry passed 83 transaction tests. The isolated image reported
  the expected version before production switched.
- Chromium and WebKit ran 228 checks. Three narrow Chinese Usage-page cases
  failed, plus their two parent tests (223 passing). The page can scroll back
  when its data first lands during a wheel event it never receives. The same
  Chromium case reproduced on pristine 4759 and the prior production assets
  (51 to 21 pixels). The WebKit case passed an isolated repeat. The separate
  #4385 interrupted-wheel regression passed in both browsers on the candidate
  and failed on unpatched WebKit. The Usage attempt-chain and selected-story
  checks also passed on the candidate.

## Production verification

The acceptance and final snapshots matched the pre-switch baseline of 266
caller-visible models, 54 groups and 36 providers. Three real inference probes
returned HTTP 200. A separate pinned Hermes SDK streaming request with both
reasoning aliases received nonempty text and a normal stop in one POST before
and after the switch. Production also recorded a first slow start with a
60-second rest on the new image. The same backoff code on its 4717-p3 rollout
had already recorded 60 and 120 seconds on successive timeouts of one model
on the same account.

The 46-minute observation completed at 16:00:28 UTC on November 24 (02:00:28
Brisbane time, November 25), with 72 recorded HTTP 200s, zero container
restarts, zero logged 5xx and zero protocol errors. The deployment entry exited
0. The production image and Docker Hub manifests for bigshort/magpie-patched
tags `0.1.4759-p1` and `latest` matched
`sha256:ee5fff371b758cc72708c5c3531ae051983369fd7837fd963b4672db6b7cf52e`.
