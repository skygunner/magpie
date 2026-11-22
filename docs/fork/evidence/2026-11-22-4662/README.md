# 0.1.4662-p1 production integration

The release merges 48 upstream commits through v0.1.4662
(b168277b8f525a339cacc1ced1f14c4f9c89b593) onto the integration line
containing the previous production source ae4d31530. The merge completed
without conflicts. The docker1 build used a Git archive of the merged
commit cd25f5290a1289a31f1817fe406410578797e320.

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

Each behavior passed on the candidate and had a failing probe on the pristine
upstream tag with the fork assertions. The candidate passed all 104 browser
cases across Chromium and WebKit, including the changed group speed median
and restart-to-update behaviors. Linux nogui vet, the retained Go checks,
affected gateway/provider/library/sessions tests and full update and usage
packages passed. The reviewed deployment entry passed bash -n, ShellCheck
and 80 transaction tests. Its SHA-256 matched the installed root-owned copy.
The docker1 image smoke returned magpie 0.1.4662-p1.

## Deployment and publication

On November 22 the authenticated baseline and candidate snapshots matched
across 268 model IDs, 53 groups and 36 providers. The real inference probe
returned HTTP 200 with nonempty content at 04:01:59 UTC. The first sample
was healthy with zero restarts, logged 5xx and protocol errors. The short
acceptance procedure from the preceding release was used. The observer ended
at 04:02:17 UTC before its optional follow-up checks and final snapshot.

Production container f6cbb0b5bb4ba3a9842698847f8ee88397802248568205fa7ac30b1615aadd4b
continued serving. The manifest bytes read back from Docker Hub for
bigshort/magpie-patched:0.1.4662-p1 and :latest both hashed to the running
image digest sha256:5debbabceac053159965bdaee67fecebf4fd2afe27463406ab319d899fbb2a1c.
