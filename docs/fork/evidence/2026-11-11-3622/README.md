# 0.1.3622-p1 release evidence

Pinned upstream: v0.1.3622, 025123431c140a4988efc3d5c44546e8d73d16d3.
Source on which the image was built: 31af508decb1054c13cf94b25381d5c268fea28b.
Previous accepted source: b7b5539de03005a071a57eceaa06fde3f7faecfa (0.1.3336-p1).
Image id (accepted and published, never rebuilt):
sha256:cd3f4290fd404ae7652711c780f73be7b9ad2f24c64f1264242ea22ff7e7b74a

## Acceptance (shortened observation by owner direction, 2026-11-17)

The owner directed that sample 1's healthy result be treated as successful and
that publishing proceed without waiting for the full 46-minute observation,
because the next nightly integration/deployment round had already started and
a 46-minute wait would overlap. The detached transaction was therefore NOT
stopped — killing it would trigger its rollback and replace the new container
with the old one. It continues to completion independently and will write its
own result and release the host lock; the container, image and published tags
are not touched by it.

Observed before publication:
- Container `magpie` running, health=healthy, restarts=0, oom=false.
- Unauthenticated `/api/providers` = 401.
- Acceptance real inference (`group/fast-cheap`) = HTTP 200 at 18:04:37 UTC.
- ACCEPTED event emitted 18:04:37 UTC.
- SAMPLE 0 (3.3s), SAMPLE 1 (276.7s), SAMPLE 2 (552.4s): all healthy, 0 restarts.
- Observation real-inference probe 1 = HTTP 200 at 18:09:21 UTC.
- Second real request returned 200; no 5xx, protocol_error, panic, fatal, OOM
  or restart observed in the acceptance + observation window to that point.

## Source gate (completed before the switch)

- One complete nogui Go run: 91 passing packages, 4 failing packages whose
  failures all reproduce on a pristine v0.1.3622 checkout under local Go 1.27.3
  (TestAlmaProvidersSeenDecodeOnce, TestDshACPSettingsGetTheRoute,
  TestServeWSLListensBesideLoopback, the fastjson marshal tests,
  TestClientPathCoversHandler, TestReadOnlyKeyEveryRoute); one GUI test
  (TestSessionReadsRoutes) is load-sensitive and passes alone. No regression.
- Linux nogui build and vet: pass.
- One selected race run over retained gateway/key behaviors: 225 pass, count=3.
- Deployment transaction: 80 mock tests pass (no side-effect rehearsal).
- Eight browser fixtures: 288 pass in Chromium and WebKit after adapting two
  fixtures to upstream's request page and the phone tab bar.
- Isolated production-config replay was NOT performed in this round: a prior
  attempt caused the 2026-11-11 docker1 ownership incident, and new hard rules
  forbid improvised parallel scripts. The deploy transaction's own
  baseline/acceptance snapshot comparison and automatic rollback were used.

## Scope boundaries

- Full 46-minute observation was cut short by explicit owner direction.
- No full GUI matrix, Windows/Darwin desktop checks, or live vendor coverage.
- No isolated config replay; see incident and rules above.
- Raw snapshots, private round artifacts and rollback backup remain on docker1
  under /var/lib/magpie-deploy/0.1.3622-p1/ (private, mode 0700).

## Patch and publication records

- [Patch review](patch-review.md)
- [Published manifest digests](published-digest.json)
