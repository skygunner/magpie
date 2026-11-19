# 0.1.4537-p1 production fork

The image was built on docker1 from a Git archive of
928b930da32f24341838fcbe7fd2ba0e7624eb74. The merge brings 69 upstream
commits through v0.1.4537 (aeca7ae07473e072d813a557f207b411ad1ea925)
onto the integration line containing the accepted 0.1.4480-p2 source
443cccd518133f49210aeaf5e0c7505cc511a1bb.

## Production patch checks

| Retained issue | Behavior checked |
| --- | --- |
| #3035 | Retry guided-grammar output-format refusals |
| #3635 | Keep the requested reasoning effort when retrying an explicit NVIDIA parameter refusal |
| #3636 | Fail over on unsupported parameter lists without resting the account |
| #3794 | Retry a tool-count rejection on the next group member |
| #3948 | Record a malformed reply as a protocol error while preserving the HTTP status |
| #3623 | Show the ordered attempt chain when a request is canceled |
| #3624 | Reveal the selected routing story when navigating from Usage |
| #4385 | Preserve an interrupted pixel wheel step on WebKit |
| Build -pN | Compare the patched build to its base release for updates and release notes |

The fork behavior checks passed on the candidate. The corresponding control
probes failed on pristine v0.1.4537 with the fork's test code and upstream
runtime and GUI assets. The previous suspended patches were kept out.

Linux nogui vet, the full update package, targeted gateway/provider/update
tests from changed upstream fixtures and the retained-patch Go tests passed.
The deployment entry passed bash -n, ShellCheck and 80 transaction tests.
In Chromium and WebKit, 66 cases passed for click-scroll and Usage details,
routing and protocol diagnostics. Another 66 passed for the upstream changes
to remote update, small-first routing, sent model IDs and stable Usage cards.
The docker1 image smoke test returned magpie 0.1.4537-p1. Full multi-platform
and GUI suites were outside the tested scope.

## Deployment and publication

On November 19 at 11:23:35 UTC the new production container passed the
authenticated snapshot comparison and a real inference returned HTTP 200.
Both snapshots contained 277 models, 53 groups and 36 providers. The first
sample was healthy with zero restarts, logged 5xx and protocol errors.

This release used the short acceptance procedure from the previous release.
The observer was ended at 11:24:08 UTC, before its scheduled 46-minute final
snapshot. The container remained healthy and continued serving. The previous
container and the pre-switch volume archive remain on docker1.

Running container: 042c1d3f4e0c92cda662e4574e8402d9e068977118d32ed2cbf131a083debf7b.
The running image and the Docker Hub 0.1.4537-p1 and latest tags all have
digest sha256:43cea9a0fc1a6f303ac318fc54af629402bcd8cc11acea56b7c951903ba5f6aa.
Both registry manifests were read back and hashed after pushing the image.
