# 0.1.4628-p1 production integration

Image source ae4d31530c187de92acc531cf9388da2a4fc63b5 merges upstream
v0.1.4628 (9cacf1fbb4102812e8192884f0f0505a2ecd637a) into the fork that
served 0.1.4537-p1 (928b930da32f24341838fcbe7fd2ba0e7624eb74).
The upstream range contains 113 commits. A two-paragraph conflict in
gui-shell.md was resolved by keeping the upstream scroll-button description
and the fork's description of interrupted pixel-wheel input.

## Retained behavior

| Issue | Probe |
| --- | --- |
| #3035 | Retry guided-grammar format refusals with a compatible output format |
| #3635 | Keep requested reasoning effort when handling explicit NVIDIA parameter refusals |
| #3636 | Fail over on named unsupported parameter lists without resting the account |
| #3794 | Pass a tool-count rejection to the next group member |
| #3948 | Preserve the HTTP status and mark confirmed malformed replies as protocol errors |
| #3623 | Show the actual attempt chain of a canceled request |
| #3624 | Reveal the selected routing story when opening it from Usage |
| #4385 | Restore the interrupted pixel-wheel step in WebKit |
| Patched versions | Compare a -pN release as its base for updates and release notes |

Each behavior above passed on the candidate. The control used the same
behavioral assertions with pristine v0.1.4628 runtime code and GUI assets;
each retained behavior had a failing upstream probe.

## Verification

Linux nogui vet and the deployment script's 80 tests passed. Go checks
covered retained gateway/GUI/update behavior, tests in gateway and provider
fixtures affected by upstream, and the full update, usage, access, pii,
settings and settingsapi packages. All 66 cases in four retained-patch
browser fixtures and 42 cases in three changed routing fixtures passed
in Chromium and WebKit. Other platforms and the full GUI matrix were outside
this release's test scope.

The immutable candidate was built on docker1 from a Git archive of the
merge commit. Isolated image checks verified the version, Debian 13,
glibc 2.41, Node v24.22.3, uv 0.13.7 and uid 65532. The deployment entry's
SHA-256 matched the previously reviewed root-owned copy.

## Production and registry

The initial authenticated API snapshots matched across 270 models, 52 groups
and 36 providers. Real inference returned HTTP 200 at 05:52:33 UTC on
November 21. The first sample was healthy with zero restarts, logged 5xx
and protocol errors. The short acceptance procedure from the previous release
was used; the observer ended at 05:52:59 UTC before its 46-minute final
snapshot. The production container continued serving.

Running container: 49718f175109f8af0ef7bef4a631be3bd4bba261a437983e1c44001b4566b382.
The running image and the retrieved manifests of bigshort/magpie-patched
tags 0.1.4628-p1 and latest shared this digest:
sha256:72fe5e84a2eee05d3f25d91305ba8a685d07b5b40f4355f6544fa8a6f00c1add.
