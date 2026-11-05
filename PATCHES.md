# Integration and production — upstream first

User-approved policy (2026-11-05): remove an absorbed patch. Also suspend a
fork patch whose logic overlaps or conflicts with current upstream rules;
do not force both semantics into a conflict resolution. Suspension does not
mean upstream implements every former fork guarantee.

- `main` mirrors upstream. `integration/nightly` is the persistent integration
  branch, merging a pinned release tag per round. Source `fix/*` and `feature/*`
  branches stay available; do not re-pick unchanged patches every round.
- The existing `prod` branch has historical failed candidates in its ancestry.
  It is not evidence of production success. `production/stable` will point only
  to successfully observed production source. `candidate/<version>` records a
  tested candidate; `deployed/<version>` is created only after acceptance and
  observation pass. Old `prod/*` tags retain their historical meaning.
- Production currently remains 0.1.2767-p1. This round pins upstream v0.1.3007.
  Source integration alone does not authorize publication.

## This round's patch decisions

| Patch | Decision | Behavior / reason |
| --- | --- | --- |
| #3035 | retain | SenseNova format downgrade and narrowly matched unexplained request refusals still extend upstream. |
| #3487 | suspend remainder | Upstream changed the amount model and zero-count rules; do not carry competing unknown/zero interpretations or its associated old fixtures. |
| #3635 | retain | NVIDIA explicit parameter-list repair and synchronized equivalent reasoning settings. |
| #3636 | retain | Unresolved explicit parameter lists fail over without resting accounts. |
| #3624 | retain | Usage navigation reveals the selected trace. |
| #3788 | suspend | Upstream now owns the changing group-budget/slow-start rules; use its current streaming-only FirstToken behavior. The 540s total budget is not an equivalent non-streaming deadline. |
| #3791 | suspend | Fork deliberately counts client cancellation and imposes hard model holds; upstream cancellation attribution and resting-candidate policy differ. |
| #3948 | retain diagnostics only | Speed display follows upstream; retain bounded non-streaming envelope checks without changing the committed HTTP status. Stream error-frame status handling is upstream's separate behavior. |
| #3794 | retain | Tools on demand does not cover arbitrary external client tools or guarantee vendor limits. Keep full-tool fallback. |
| #3623 | retain | Expanded Usage details show ordered recorded attempts, including 499. |
| Cooldown settings | suspend | Layered policy, changed defaults and release semantics compete with current upstream rest rules. Editor/API and runtime overrides are removed together. |
| #3417 | absorbed | Upstream configurable Codex reset lead remains; fixed-lead fork stays removed. |
| build -pN | retain | Update checks and release notes compare the base version. |
| plugins PR #366 | separate | Keep production Cloudflare Workers AI package 0.0.5. No package downgrade or test-only rebuild. |

The retained delta is reviewed against the pinned tag, including callers,
fixtures, GUI and subsystem references. Source commit ancestry and issue state
are evidence, not the sole retirement test. Tests must exercise current public
behavior; baseline failures do not automatically authorize deployment.

## Deployment checks

1. Preserve acceptance evidence before deciding: baseline.json, acceptance.json,
   acceptance-diff.json, final.json and final-diff.json are private on docker1.
2. An optional --expected-snapshot file contains reviewed baseline/candidate
   snapshots and the immutable target_image_id. A changed baseline or image
   refuses before stopping production. Unexpected candidate differences roll back.
3. `/api/cooldown` exists only in older fork builds. Its deliberate removal is
   represented as null in the snapshot, not silently ignored. Other HTTP failures
   remain errors. Authentication, providers, groups and caller-visible models stay
   checked, with explicitly reviewed model exposure changes only.
4. `/var/lib/magpie-deploy/BLOCKED` prevents a switch. Leave it present while an
   unexplained acceptance difference remains; clear it only after isolated
   reproduction and final-commit tests. Integration may continue while blocked.
5. Build one image from the tested commit. Run the existing detached transaction
   with a host lock, stopped-volume backup, immutable container/image assertions,
   acceptance, three real inference probes and 46-minute observation. Any failure
   rolls back and verifies the previous container; preserve candidate config.
6. Promote production/stable and deployed/<version>, and publish the exact image
   digest as a version and latest only after exit 0, this run's PASSED result,
   matching live IDs and fresh healthy checks. Never rebuild during promotion.

## Deployment history

- **2026-11-05, 3007-p2 preparation — both previous blockers resolved.**
  This remains pinned to upstream v0.1.3007; the seven retained issue behaviors
  and all four suspended entries stay as listed above.

  OpenCode's reference identifier.ts increments a 12-bit-position counter
  without a limit of 15. The generator and incoming-seed heuristic have both
  existed since upstream 8e6354714 (2026-10-13), including production 2767.
  `openCodeRealSession` is not the protocol validator for generated IDs.
  Tests now validate generated IDs against the actual shape/time/counter
  algorithm, including 32 IDs at fixed time; existing native-seed and legacy
  hash-seed assertions remain. Runtime logic is unchanged. The prior blocker
  classification was too strong; a vendor failure was never demonstrated.
  Reference: https://github.com/anomalyco/opencode/blob/dev/packages/schema/src/identifier.ts

  Claude's missing account is reproducible by caching a credential miss before
  claudeSignIn writes the test login. That fixture bypassed the normal writer
  and did not invalidate the miss. Reuse test-only #3174 fix de9593aca (with
  ggbdpq credit), clearing the credential cache after writing. The actual
  failing model-list test now seeds this miss so it cannot pass by timing.
  The same test fails before the helper repair and passes after it. No carried
  runtime patch caused either issue. Both regression families pass 20 times
  with race detection. Complete final-commit gates and deployment follow;
  this preparation entry is not a production success record.


- **2026-11-05, v0.1.3007 integration — BLOCKED, NOT DEPLOYED.**
  `integration/nightly` merges pinned upstream v0.1.3007. #3487 remainder,
  #3788/#3791 and cooldown-settings are suspended under the user's new
  upstream-first policy; retained runtime additions are listed above.
  Production remains 0.1.2767-p1; no candidate/deployed tag or Hub release
  was created for this integration.

  A network-isolated replay of the pre-3005 backup reproduced the incident's
  baseline exactly (245 models, 32 providers, 47 groups). The failed image
  yielded 251 models, 32 providers, 49 groups: four zcode models, two auto
  model aliases and two generated groups were added, with cooldown unchanged.
  The first new candidate snapshot also had 251/32/49 and deliberately no
  cooldown API. Original incident candidate API data is still unavailable:
  this is reproduction evidence, not recovered historical data. Later offline
  refresh failures changed the old replay's cached lists, demonstrating why
  snapshots must be saved at capture time. No external vendor calls were
  possible from the replay's internal Docker network.

  Test maintenance keeps the original behavior assertions: isolate the POSIX
  shell from a fish login, correct literal JSON-newline expectations, and gate
  the Darwin tray image test on its cgo implementation. Restore three upstream
  timing/order fixtures after cooldown removal. Reuse the existing #2207
  deterministic SOCKS refusal fixture (test-only, source a93c613a0). The Codex
  reset UI test clears previous notification state before each independent
  click and permits one-pixel engine rounding; all 20 browser/language/width
  cases pass, including the two previously failing WebKit German cases.
  All 80 deployment fault tests pass; Linux/Windows/Darwin nogui vet/build,
  native build/vet and retained gateway race regressions (20 repetitions) pass.

  **New deterministic blocker:** upstream openCodeNewID emits counter 16
  within one millisecond, but openCodeRealSession requires counter <16.
  The checked-in reproducer fails on clean v0.1.3007 at generated session 16.
  This explains a real consistency failure behind the previously called
  load-sensitive TestOpenCodeSessionOnEveryCall; a later isolated pass does
  not resolve it. Do not weaken the validator or alter the wire-ID algorithm
  without checking OpenCode's protocol requirements. No new protocol patch
  was added during this integration.

  Evidence: [deterministic reproducer](docs/fork/evidence/2026-11-05-3007/opencode-boundary-repro.go.txt),
  [failure](docs/fork/evidence/2026-11-05-3007/opencode-boundary-red.log),
  [isolated replay](docs/fork/evidence/2026-11-05-3007/incident-reproduction.json).
  A subsequent whole-suite run also failed
  `TestClaudeAccountServesModelsListedSinceItsFetch`; clean baseline isolation
  with `-race -count=20` passed, so this failure is not yet attributed and must
  not be labelled a flake or ignored. The complete release gate is not green.
  The final source includes later test-only fixture repairs; no claim is made
  that a final clean-commit release gate passed.

  `production/stable` is initialized to successful source 96cf7af969
  (prod/0.1.2767-p1), while `integration/nightly` carries this candidate.
  The existing six-hour ZCode automation was temporarily paused for this
  investigation, then restored with the new upstream-first policy and an
  explicit prohibition on clearing BLOCKED or deploying while it exists.
  Its schedule was preserved. All three isolated replay containers, their
  internal network and copied configuration directories were removed.

  Complete local gate logs remain private under
  `~/.local/state/magpie-rounds/0.1.3007-p1/`. Deployment stays blocked on
  docker1, and the rolling automation must honor this block. No production
  inference, acceptance switch or 46-minute production observation occurred.


- `prod/0.1.3005-p1` — 2026-11-05. **Candidate failed acceptance;
  rolled back, not published.** Source commit `b46d2c12f` and the annotated
  tag were pushed to the fork before the image was built and transferred.
  The final gate was run at that commit: all eight Go failures were
  reproduced on the clean v0.1.3005 baseline (seven in the full suite and
  `TestOpenCodeSessionOnEveryCall` in a repeated provider package run); the
  two WebKit German failures and Darwin nogui vet error also reproduced on
  the clean tag. The other builds, Linux/Windows vet, patch regressions,
  deployment fault tests and smoke checks passed.

  The detached switch began at 02:03 UTC. Before the switch the gateway
  listed 245 models, with 32 providers and 47 groups in the console. The
  candidate's acceptance snapshot differed (`acceptance_snapshot_changed`),
  so the script did not send the real inference probe or start the 46-minute
  observation. Its log does not identify whether models, providers, groups
  or the effective cooldown differed; the cause is not established. The
  previous `magpie:0.1.2767-p1` container was restored and verified healthy
  at 02:05:32 UTC with zero restarts. No retry was attempted, and neither
  `0.1.3005-p1` nor `latest` was pushed to Docker Hub; the public candidate
  tag returned 404 and `latest` remained `sha256:bbc221d0…`.

  Private pre-switch volume and container backups are under
  `/var/lib/magpie-deploy/0.1.3005-p1/`; a candidate configuration snapshot
  was also preserved before the old container resumed. The two copies have
  the same configured provider IDs and raw cooldown setting, which does not
  establish which acceptance field differed. The candidate API snapshot was
  not saved. A read-only investigation found one plausible cause: upstream
  `7c115f6f5` changed the model exposure rule for an empty picks list with
  delisted models and no `DelistedAt` timestamp, and one configured provider
  meets those conditions. It had no direct model IDs in the baseline. The
  candidate's model IDs and the cause of the mismatch remain unconfirmed.
  Identify the mismatch before a future attempt; preserve the required
  acceptance checks and do not publish this failed candidate.

- `prod/0.1.2877-p1` — 2026-11-04. **Candidate failed acceptance;
  rolled back, not published.** The v0.1.2877 release was merged into
  `prod` and the candidate tag was pushed. After deployment, the first
  real `group/fast-cheap` inference timed out after 90 seconds; a retry
  returned 200, but the failure required rollback. Rollback was delayed:
  the previous session retried instead of rolling back; after the session
  resumed, `magpie:0.1.2767-p1` was restored and healthy at 07:06 UTC.
  The restored console returned 401 without a key and 200 with a cookie;
  `/v1/models` returned 200 with 249 models, and a real inference returned
  200. The prior candidate's acceptance did not collect the actual
  `/v1/models` count before switching, so equality for that endpoint was
  not established. The candidate was not observed for 46 minutes and was
  not published to Docker Hub. Its pre-switch volume backup is
  `/home/hermes/backups/magpie/pre-0.1.2877-p1-20261104-013704`.

  Do not redeploy the candidate until #3487's carried behavior is fixed
  and tested across the real plugin-to-tray and GUI paths: `Amount` is a
  float in both `plugin.UsageWindow` and `provider.QuotaWindow`, so the
  plugin's null and a stated zero are indistinguishable after decoding,
  and `omitempty` omits either from the API response. This candidate's
  `QuotaWindow.Count` treats that unknown as zero and can show a full
  allowance in the tray; the fake GUI fixture alone did not test the real
  serialization path. The final tagged commit changed the corresponding
  test expectation after the last full-suite run; it was not followed by
  a full-suite rerun at that commit. Resolve that regression and complete
  the required gate before another deployment attempt. The inference
  timeout's cause remains unverified and is not attributed to #3487.

- `prod/0.1.2767-p1` — 2026-11-03. Rolling merge of v0.1.2767 (32
  commits since v0.1.2753); two conflicts, both sides kept: shapeRefused
  now also checks upstream's new `reasoningBackRefused` (Beam's missing
  required parameter refusal) beside our existing `refusedParameterList`,
  `zhipuImageRefusal`, `upstreamFailedWords`; the notes site client
  (`update/notes.go`) rides upstream's `NotesBetweenTotal` refactor
  while keeping our `base()` stripping on both `after` and `upto`
  parameters. Budgets re-measured 1,326,801 / 1,548,546 (upstream moved
  some words/translation code out of the core). Freshness sweep all
  negative; none absorbed. Gate: ours 9 reds all inside the clean
  baseline's 10 (the extra there is the known load-sensitive
  `TestOpenCodeSessionOnEveryCall`); GUI 134/136 with the same two
  webkit-de `codex-auto-reset` reds reproducing on the baseline. Deployed
  by the batched script (third round on it, healthy in 30s); accepted
  with providers 32 / groups 49 / 1201 model ids equal before and after
  and three real probes 200 across 46 minutes, 10 samples healthy with
  zero panic/fatal/5xx/protocol_error. Published to Docker Hub as
  `bigshort/magpie-patched:0.1.2767-p1` and `:latest`, digest
  `sha256:bbc221d0…`. Hub README updated via one-shot fetch→edit→PATCH→
  verify (JWT expired once, re-login re-PATCH and verified from live
  API). Backup
  `/home/hermes/backups/magpie/pre-0.1.2767-p1-20261103-070823`; old
  container kept stopped as `magpie-pre-0.1.2767-p1-20261103-070823`.

- `prod/0.1.2753-p1` — 2026-11-03. Rolling merge of v0.1.2753 (127
  commits; seven conflict files, both sides kept: fallback.go's refusal
  predicates now also carry upstream's new `zhipuImageRefusal` (GLM's webp
  refusal, #4732) beside our `refusedParameterList` and
  `upstreamFailedWords`; gateway.go's shape-refused block gains upstream's
  webp-re-as-JPEG re-ask while keeping our `bareRequest` disjunct; the
  docs' textcalls line went to upstream's superset (relayed Responses
  replies are read too, `textcalls_relay.go`) while the format line kept
  our #3035 SenseNova wording; both i18n files unioned; the Usage ledger
  test's labels kept our attempt-chain columns with upstream's renamed
  word 接口地址). Budgets re-measured 1,326,629 / 1,548,303. Freshness
  sweep all negative; none absorbed. Gate: ours 10 reds all inside the
  clean baseline's 11 (the two extras there were the known load-sensitive
  `TestOpenCodeSessionOnEveryCall` and allocation-counting
  `TestCCListingKeepsItsNames`); GUI 134/136 with the same two webkit-de
  `codex-auto-reset` reds reproducing on the baseline. Deployed by the
  batched script (second round on it, no session loss, healthy in 30s);
  accepted with providers 32 / groups 49 / 1202 model ids equal before
  and after and six real probes 200 across 46 minutes, 10 samples healthy
  with zero panic/fatal/5xx/protocol_error. Published to Docker Hub as
  `bigshort/magpie-patched:0.1.2753-p1` and `:latest`, digest
  `sha256:4b144f66…`. Hub README updated fetch→edit→PATCH→verify in one
  step and written back to /tmp/hub-readme.md (the JWT expired mid-PATCH;
  after re-login the first write again used a stale re-fetched template —
  the round's lesson is now do fetch, edit, PATCH and verify inside one
  script so the local template cannot be overwritten by a stale fetch
  between steps). Backup
  `/home/hermes/backups/magpie/pre-0.1.2753-p1-20261102-192605`; old
  container kept stopped as `magpie-pre-0.1.2753-p1-20261102-192605`.

- `prod/0.1.2657-p1` — 2026-11-02. Rolling merge of v0.1.2657 (344
  commits; six conflict files, both sides kept: fallback.go's imports and
  the refusal predicates (`refusedParameterList` beside upstream's new
  `upstreamFailedWords`), gateway.go's cooldown hard hold beside
  upstream's `handed.release` restructure of laneMate, provider.go's
  cooldown field assignments beside `Patience` and upstream's new
  `accountSettings()` literal (our four fields ride in
  `accountSettings`, not the literal), and the docs bullets. **Upstream
  refactored the settings keep-what-not-sent list into
  `settingsapi.Apply`/`Merge` (#4745): the cooldown editor's ownership
  line moved with it into `settingsapi.Merge`** — caught by the audit
  (api.go left the carried set; a first scripted insert had silently not
  fired because the old anchor line no longer exists). Freshness sweep
  all negative (every branch tip equals its fix of record; no issue
  comments after 10-29; plugin PR #366 head still `38ddeb1`, test-only);
  none absorbed. Gate: full suite 9 reds identical to the clean
  v0.1.2657 baseline (fastjson escapes, Claude cache, Codex pool version,
  DSH umask, WSL, SOCKS, route-command naming); GUI 134/136 with the two
  webkit-de `codex-auto-reset` reds reproducing on the baseline too.
  Deployed with `scripts/deploy-docker1.sh` run detached on docker1 — the
  first round on the batched switch: stop, stopped-volume tar, rename,
  run, healthy after 30s, no session loss. Accepted: version
  0.1.2657-p1, no-key 401 / cookie 200, providers 32 / groups 49 / 1202
  model ids equal before and after, `/api/cooldown` 200, five real
  `group/fast-cheap` probes 200 across 46 minutes, 10 observation samples
  healthy with zero panic/fatal/5xx/protocol_error. Published to Docker
  Hub as `bigshort/magpie-patched:0.1.2657-p1` and `:latest`, digest
  `sha256:eec9eb8b…`, README updated and written back to
  /tmp/hub-readme.md (after one mis-PATCH: re-fetching the template over
  the local edit published the stale text once; re-patched and verified
  from the live API). Backup
  `/home/hermes/backups/magpie/pre-0.1.2657-p1-20261102-172853` (old
  container kept stopped as `magpie-pre-0.1.2657-p1-20261102-172853`).

- `prod/0.1.2450-p1` — 2026-11-01. Rolling merge of v0.1.2450 (150
  commits; three conflicts, both sides kept: app.js `openModal`'s new
  `quiet` beside the cooldown close guard, i18n's four language blocks
  unioned, core budgets re-measured to 1,326,038 / 1,547,127 — upstream
  moved Usage fragments out of the core). Freshness sweep all negative:
  every carried branch tip equals its fix of record; nvidia's extras are
  the three fixes already carried plus a docs-only commit; the 1025a
  branch's six newer commits are other issues' fixes touching none of
  #3487's files; no issue comments after 10-31; plugin PR #366 head
  still `38ddeb1`, test-only. Gate: 9 reds all reproduced on the clean
  v0.1.2450 baseline (fastjson escapes, WSL/SOCKS network, umask 0640,
  upstream's `SearchAll` keep-test gap); `TestNoCommandBypassesProc` now
  green (upstream fixed it). **Deployed by hand**: the scheduled round
  stopped the old container, which cut the agent's own gateway route and
  ended the session; the operator rolled the container onto the pushed
  image himself. Verified on the running container: healthy, 0 restarts,
  version 0.1.2450-p1, no-key 401 / cookie 200, providers 31 / groups 49
  / 1200 model ids equal before and after, `/api/cooldown` 200, real
  `group/fast-cheap` 200 ("OK"), 40 minutes of logs with zero
  panic/fatal/5xx/protocol_error. Published to Docker Hub as
  `bigshort/magpie-patched:0.1.2450-p1` and `:latest`, digest
  `sha256:b8e6db5d…`, README updated. Backup
  `/home/hermes/backups/magpie/pre-0.1.2450-<ts>` (volume.tgz taken with
  the container stopped).

- `prod/0.1.1556-p1` — 2026-10-26. First patched build: the six bug
  patches above at v0.1.1556.
- `prod/0.1.1612-p1` — 2026-10-27. Rolling merge of v0.1.1612.
- `magpie:0.1.1612-p2-hindsight` — 2026-10-27 (no tag; pre-scheme).
  `dd670e6` deployed for the Hindsight 499 problem; running before this
  round. Superseded by `prod/0.1.1670-p1`.
- `magpie:0.1.1612-p3-tools` — candidate only, never deployed (its binary
  swap is replaced by this round's proper #3794 integration).
- `prod/0.1.1670-p1` — 2026-10-27. Rolling merge of v0.1.1670 plus
  #3788/#3791 (`dd670e6`) and #3794 (`82d916e3`); plugin tarball rebuilt
  from PR #366 `bdc302a`. Published to Docker Hub as
  `bigshort/magpie-patched:0.1.1670-p1` and `:latest` — from this round
  on, every deployment is published there too.
- `prod/0.1.1691-p1` — 2026-10-27. Rolling merge of v0.1.1691; no
  conflicts, none absorbed, same nine patches.
- `prod/0.1.1734-p1` — 2026-10-27. Rolling merge of v0.1.1734; #3035's
  evolved fix (`223f9b0a`, bare-request failover) joined `8f931a28`.
- `prod/0.1.1767-p1` — 2026-10-28. Rolling merge of v0.1.1767; clean
  merge, no sweep movement, none absorbed.
- `prod/0.1.1767-p2` — 2026-10-28. Same upstream, plus the non-streaming
  SSE follow-up (`dfd128ca`) and the #3487 span-fixture repair
  (`3ea2e279`).
- `prod/0.1.1767-p3` — 2026-10-28. Same upstream, plus #3948
  (`56eb1ba1`).
- `prod/0.1.1790-p1` — 2026-10-28. Rolling merge of v0.1.1790; #3950's
  wallet `Balance` pointer kept beside #3487's `AmountUnknown`.
- `prod/0.1.1840-p1` — 2026-10-28. Rolling merge of v0.1.1840; two
  mechanical conflicts (the `Provider` literal, the usage plugin's
  `i18n.js`), none absorbed, same ten patches.
- `prod/0.1.1900-p1` — 2026-10-28. Rolling merge of v0.1.1900 (93
  commits); clean merge, none absorbed, same ten patches. Upstream
  refactored gateway internals (format/gateway/routing split); the
  carried fixes' regressions pass at their new homes.
- `prod/0.1.1948-p1` — 2026-10-29. Rolling merge of v0.1.1948 (69
  commits); three conflicts. Upstream took #3487's semantics over
  (bare `Amount`, plugins#468); this line now carries only its card
  display and millisecond-span guard, re-expressed at the merge. Nine
  cherry-picked patches unchanged.
- `prod/0.1.2051-p1` — 2026-10-29. Rolling merge of v0.1.2051 (126
  commits); four conflicts, both sides kept (the Provider literal, the
  usage plugin's i18n.js, gateway-routing.md, the quota CLI line). The
  Codex auto-reset strings, i18n.js and their test were restored to
  upstream's held-up wording, which an earlier round's conflict
  resolution had trimmed here. No patch movement.
- `prod/0.1.2051-p2` — 2026-10-29. Same upstream, plus the build-line
  fix: a `-pN` build compares as its base release in What's new and the
  update check (`update.parse` strips the suffix; the site is asked in
  whole release numbers). History had run to p1 only — this and the
  next two lines were back-filled at 0.1.2134-p1.
- `prod/0.1.2051-p3` — 2026-10-29. Same upstream, plus #3623
  (`33c0ade7a`, attempt chain in Usage details).
- `prod/0.1.2051-p4` — 2026-10-29. Same upstream, plus the cooldown
  settings surface (`1da2e1391`, `feature/cooldown-settings`),
  fork-only trial, no upstream feature request.
- `prod/0.1.2134-p1` — 2026-10-30. Rolling merge of v0.1.2134 (99
  commits); three conflicts, both sides kept (the Provider literal's
  cooldown fields beside upstream's `BestInFamily`, the account row's
  cooldown button beside the acc-line restructure, this line's core
  budgets raised to measured values for upstream's own growth). No
  sweep movement, none absorbed.
- `prod/0.1.2170-p1` — 2026-10-30. Rolling merge of v0.1.2170 (38
  commits); three conflicts, both sides kept (Usage detail's
  client-reported/protocol-error wording beside upstream's mcp-source
  condition, gateway-routing.md's fork policy bullets beside upstream's
  newest prose, core budgets raised to measured values for upstream's
  own growth). The v0.1.2134 budget raise, left unstaged when
  0.1.2134-p1 was cut, is committed here (the deployed image had it;
  the tag had not). No sweep movement, none absorbed.
- `prod/0.1.2214-p1` — 2026-10-30. Rolling merge of v0.1.2214 (71
  commits); four conflicts, both sides kept: (this entry's
  conflicts as before). None absorbed.
- `prod/0.1.2264-p1` — 2026-10-30. Rolling merge of v0.1.2264 (66
  commits); seven conflicted files, both sides kept — except #3417,
  which is upstream's now: settings' `CodexResetLead` makes the spend
  deadline's lead configurable and a reset held up past its spend point
  is spent at once; this line's fixed-lead variant and its test were
  superseded and removed at the merge. No sweep movement otherwise.
- `prod/0.1.2289-p1` — 2026-10-31. Rolling merge of v0.1.2289 (30
  commits); three conflicts, both sides kept or upstream's (the account
  literal's cooldown fields re-inventoried, providerInfoWith taking
  upstream's `CountFree`/`LimitRestOff` — the GUI's cooldown policies
  come from `/api/cooldown`, not this literal — core budgets standing).
  No sweep movement, none absorbed. Upstream's new awake/turn-taking
  tests proved timing-flaky under the full suite on both this line and
  the clean base (`TestColdTurnsTaken`, `TestKeepOffWaitsForTheSetting`,
  `TestOpenCodeSessionOnEveryCall`).
- `prod/0.1.2344-p1` — 2026-10-31. Rolling merge of v0.1.2344 (61
  commits); three conflicts, both sides kept (Usage detail's
  client-reported/protocol-error wording beside upstream's new
  search-source condition, the account literal's cooldown fields
  re-inventoried, core budgets standing). No sweep movement, none
  absorbed. Two of upstream's own reds joined the known set, both
  verified on the clean base: `TestSettingsSaveKeepsWhatItDoesNotSend`
  (b09349d17 added the `SearchAll` setting without teaching the
  keep-test) and `TestNoCommandBypassesProc` (new
  `internal/claudetui/child_unix.go` bypasses `proc.Command`).
 the account pick's
  cooldown-policy assignments beside upstream's reworded billing
  comment (the split account literals collapsed onto upstream's, this
  line's fields added), Unrest keeping the release-an-active-rate-rest
  behavior beside upstream's saveRests bookkeeping, gateway-routing.md
  taking upstream's newest candidate-loop prose, and this line's core
  budgets standing (upstream's growth in this range fit under them).
  No sweep movement. A new upstream awake test
  (`TestKeepOffWaitsForTheSetting`) proved timing-flaky under the full
  suite: it passes standalone on both this line and the clean base.

## Non-streaming SSE regression follow-up (2026-10-28)

The timeout extension in `dd670e6` also enabled `holdWriter.watch` for
non-streaming requests. After 15 seconds without response headers, its
`keepQuiet` path sent an SSE keepalive and committed HTTP 200. The eventual
JSON answer (or HTTP error) was then emitted as SSE. Hindsight's non-streaming
OpenAI SDK call received a string instead of a completion object and reported
missing choices, even when the SSE contained an answer.

`keepQuiet` now requires the caller to request streaming. First-response
cancellation/fallback and the opt-in cooldown policy are unchanged. The
existing streaming fixture explicitly sets `streams`; no new configuration,
dependency or retry mechanism is introduced.

Regression checks:

- `TestNonstreamFirstResponseFallsBack`: silent and header-only candidates,
  crossing the keepalive threshold, must fall back to valid JSON.
- `TestNonstreamWaitKeepsStatusAndJSON`: Chat, Responses and Messages wait
  20 virtual seconds and preserve JSON and both 200/503 statuses. Replacing
  `fallback.go` with its pre-fix contents makes these checks fail with early
  SSE, including 503 becoming 200.
- The timeout/cooldown checks pass 20 repetitions under `-race` on both this
  branch and an isolated integration checkout at `c70e0610` (v0.1.1767 plus
  carried patches). The expanded streaming checks also pass.
- Native/Linux/Windows vet and builds pass on the integration checkout.
  Its compiled Linux binary, using a throwaway HOME and fake upstream over
  real HTTP, returns JSON after a 20.011-second wait and after an
  18.012-second first-candidate timeout/fallback. Neither sends early SSE.

Full integration suite: `go test -p 2 -tags nogui ./... -count=1
-timeout=10m` ran with Go 1.26.8 in an isolated Linux container. Gateway
(410.021s), GUI, plugin and PII passed. The suite was not green:

- Four agent write-refusal cases failed as root; all four passed when rerun
  as an unprivileged user.
- `TestKeyNote` expects host wording but detects the container (also reported
  in upstream #3188); the unchanged code fails the same check.
- Existing upstream failures: WSL/Hermes file reads (#3924), provider shared
  model-list requests (#3854), and the TUI Codex picker (#3919).
- `TestQuotaSpanCarriesEveryWindowField` needs `AmountUnknown: true` in its
  all-fields fixture after the #3487 patch. Runtime `quotaReport` already
  copies it. That one-line test repair is on the separate
  `fix/3487-quota-span-fixture-1767` branch, for the new release's test.

Deployment handoff: this follow-up is not deployed by this work. Cherry-pick
only this follow-up onto the v0.1.1767 integration line that already carries
`dd670e6`; do not merge the older release base into the deployment branch.

## Request protocol and client-error visibility (2026-10-28)

Branch `fix/request-protocol-visibility`, based on deployed source `1207467e`
(v0.1.1767-p2). Cherry-pick this follow-up onto the deployment integration;
it does not replace the earlier non-streaming SSE fix and is not deployed here.

A completed, non-streaming Chat/Responses/Anthropic response is checked using
its existing bounded capture. SSE instead of JSON, malformed JSON, missing
response arrays and empty Chat choices become `protocol_error` while retaining
the committed HTTP status, including 200. This is diagnostic recording, not a
new retry or cooldown policy. Truncated/encoded captures and streaming response
bodies are outside this check; valid tool calls, refusals and empty text pass.

Usage shows `200 · Protocol error`, identifies gateway checks versus existing
client reports/logs, and explains that HTTP status cannot establish client
processing success. The original change labeled missing speed in the cell;
since v0.1.2877, upstream's speed display instead shows a dash or lower bound,
with the reason in its tooltip and details. No Hindsight log collector, client
changes, feedback endpoint, historical backfill or report-to-request merging
is added.

Validation:

- New HTTP integration checks fail on the old code because missing/empty choices
  remain HTTP 200 without a recorded failure. New UI checks fail on the old
  untranslated protocol status and blank speed. Both pass with this change.
- Protocol/capture, Routing totals and OTLP fallback checks pass 20 repetitions
  under `-race`. Two existing success fixtures now use valid choices rather
  than empty arrays; their original success assertions are retained.
- Chromium and WebKit: speed, request details and protocol tests pass, including
  all five languages for the new labels and existing narrow-window speed tests.
- Native/Linux/Windows vet and builds pass.
- The complete `go test -tags nogui ./...` run is not green. Aside from those two
  corrected fixtures, its failures reproduce on unchanged `1207467e`:
  `TestAgyShell`, `TestWSLColdPlaceOpensNothing`,
  `TestClaudeClientCacheAbsentUnchanged`, `TestCodexFittedImagesPastTheBudget`,
  `TestMarshalRawIsEncodingJSON`, `FuzzMarshalRaw`,
  `TestServeWSLListensBesideLoopback`, `TestPluginSOCKSProxy`, `TestEnvOfChild`,
  `TestCodexAppServersOnHome`, `TestTUIFetchesSubscriptionModels` and intermittent
  `TestHandedOverGatewayFinishesItsStream` (EOF, also reproduced in 20 baseline
  repetitions). These are not repaired in this observability patch.
- The final gateway package run passes in 318.139s with only these five
  reproduced baseline failures excluded via `-skip`:
  `TestClaudeClientCacheAbsentUnchanged`, `TestCodexFittedImagesPastTheBudget`,
  `TestMarshalRawIsEncodingJSON`, `TestServeWSLListensBesideLoopback`, and
  `FuzzMarshalRaw`. The intermittent handover test was included and passed.

## Fork-only feature: cooldown settings

`feature/cooldown-settings` adds global/provider/account/key policy editors with
resolved-value/source display, modelCooldown controls, detailed examples and
active-rest release. Rate-limit defaults become 2 minutes initially and 64 minutes
maximum; quota exhausted without a reset defaults to a fixed 30-minute rest.
Explicit old provider values remain overrides. This feature is kept on the user's
fork for trial; no upstream feature request is submitted. It is not deployed by
this branch push. See the fork policy section of docs/subsystems/gateway-routing.md.
