# prod line — upstream releases + skygunner patches

How the fork is organized (since 2026-10-27):

- `main` mirrors `upstream/main` (fast-forward only). A fix forks off it as
  `fix/<topic>`, one fix per branch, is verified there, and is replied to
  the upstream issue.
- `prod` is the rolling integration line: each round merges the newest
  upstream **release tag** (never `upstream/main` — unreleased commits
  don't ship) and cherry-picks the fixes chosen for deployment. This file
  is the manifest of what the line carries.
- Each deployment is tagged `prod/<image-tag>` (e.g. `prod/0.1.1670-p1`).
  The tag is the deployment record; `magpie:<tag>` images build from it.
- A patch leaves the manifest only when upstream absorbs it: the fix of
  record is an ancestor of the merged tag, or an upstream commit says it
  fixes the same issue, or the issue closes as fixed.

Each round:

1. `git fetch upstream --tags`; merge the newest `vX.Y.Z` into `prod`.
   Resolve conflicts by keeping both sides' semantics; adapt new upstream
   tests that meet a carried type change (noted below when it happens).
2. **Track each carried patch's latest state — the table's SHAs are what
   was last integrated, never a freeze.** For every row's source ref:
   - `git fetch origin`, then compare the row's `fix/*` branch tip against
     the carried state (`git rev-list --count <carried>..origin/<branch>`).
     New commits mean the fix evolved: cherry-pick them (branch order),
     re-run that patch's regressions, and move the row's fix of record to
     the new head. Also skim the row's issue for new comments — a
     reporter's follow-up or a maintainer's note can make the carried fix
     need rework even with no new commits.
   - The plugin row is a PR: fetch its head (`git fetch
     <plugins-remote> pull/366/head` in the plugins checkout) and compare
     the runtime files (`src/`, `index.mjs`, `package.json`) against the
     tarball in the volume. Runtime change → rebuild the tarball with
     `npm pack`, swap it into `/config/magpie/plugins/dist/`, replace the
     package under `node_modules/`, restart `magpie`, re-verify. Test-only
     commits → note them, nothing to ship.
3. Add genuinely new fixes: `git cherry-pick -x` from their `fix/*`
   branches, in branch order — only when the user names them (an evolved
   carried patch is not a new patch).
4. Absorption: a patch leaves the table only when upstream absorbed it —
   the fix of record is an ancestor of the merged tag, or an upstream
   commit says it fixes the same issue, or the issue closes as fixed.
5. Gate: each patch's own regression tests, the GUI tests below in both
   engines (`node --test` with playwright), `go vet`,
   `TestCoreScriptBudget` (if app.js goes over, shorten the patch's own
   comments before considering a budget raise), and
   `go test -tags nogui ./...` — attribute every red against a clean
   checkout of the same tag before blaming the line.
6. Tag `prod/<tag>-pN` (annotated); build
   `docker build --build-arg VERSION=<tag>-pN -t magpie:<tag>-pN .`;
   smoke-test on a throwaway config (`magpie version`, 401 without a key).
   Publish to Docker Hub: `docker tag` the image as
   `bigshort/magpie-patched:<tag>-pN` and `:latest` (latest always points
   at the newest round), push both, and refresh the repo's README — the
   version header, the patch table, and the source tag link come from
   this file's current round.
7. On docker1: back up the volume and container config under
   `/home/hermes/backups/magpie/pre-<tag>-<ts>/` (volume tar + container
   inspect), stop and rename the running container to
   `magpie-pre-<tag>-<ts>` (kept stopped), run the new image with the same
   flags (`-p 3425:3425 -p 3430:3430 -v magpie-config:/config:z --user
   nonroot --restart unless-stopped`, the two MAGPIE_* env vars,
   `web --addr 0.0.0.0:3430 --no-open`).
8. Accept: healthy, 401/200 auth pair, model count equal before/after, a
   real `group/fast-cheap` request 200. Push `prod` and the tag.

## Carried patches (as of v0.1.2051-p1, 2026-10-29)

Checked against v0.1.2051 (2026-10-29): nine patches carried as cherry-
picks plus #3487's remainder, re-expressed at the v0.1.1948 merge
(#3948 joined at 0.1.1767-p3). The v0.1.2051 merge kept both sides of
four collisions: the Provider literal (ModelCooldown beside upstream's
HiddenDrawers), the usage plugin's i18n.js (#3948's four language
blocks beside upstream's hide-from-Overview strings), gateway-routing.md
(our FirstToken/modelCooldown bullets beside upstream's new
fallback-header bullet) and the quota CLI's wording, which takes
upstream's newest. This round also restored the held-up wording of the
Codex auto-reset strings, i18n.js and their test, which an earlier
round's conflict resolution had trimmed on this line. Upstream `1390aa458` (community plugins#468) took #3487's
point over its own way — a plugin window's `Amount` is a bare count
again, where 0 is what a count left out reads as and a limit alone says
nothing — so the carried pointer form (`d4944ace`'s `amountUnknown`
flag) could not sit beside the type change and was taken over: this
merge carries only what upstream lacks, the quota card and tray
refusing to draw a full allowance for a limit with no count (`quotaCount`
and `QuotaWindow.Count` treat an amount of 0 as not said) and the
millisecond-span guard (`quotaSpanLimit`), plus the fake plugin's
`Daily` (`amount: null`) window as the pipeline regression; told
upstream in the issue. The other three conflicts kept both sides:
`fallback.go`'s `shapeRefused`/`retryable` (our `refusedParameterList`
beside upstream's `creditWords` guard) and the usage plugin's `i18n.js`
(our #3948 strings beside upstream's clear-usage strings). The #3788
issue's comments are our own integration announcements, no rework.

Equivalence notes for the sweep: `fix/hindsight-nonstream-cooldown`'s
tip `b48a91bb` and `dfd128ca` are the same SSE fix on different bases —
never pick both; #3788/#3791 track `fix/hindsight-nonstream-sse-1767`.
#3035's evolved fix `223f9b0a` (fix/bare-request-failover) is carried
beside `8f931a28`. The #3487 source branch's newer commits are other
issues' fixes touching none of its files; the nvidia branch's extra
commit is docs-only.

| upstream | fix of record (skygunner/magpie) | source to track | what it does |
| --- | --- | --- | --- |
| yetone/magpie#3035 | `8f931a28` + `223f9b0a` | `fix/sensenova-grammar-refusal`, `fix/bare-request-failover` | recognize SenseNova's guided_grammar compile_grammar refusal and retry the schema as json_object; an otherwise unexplained 400/422 `Invalid request` also fails over to the next candidate, resting nothing |
| yetone/magpie#3417 | `e75480ec` | `fix/codex-expiry-window` | Codex expiry-only resets stay inside the expiry window |
| yetone/magpie#3487 | merge-carried (was `d4944ace`) | `fix/issue-batch-1025a` | quota card: a limit with no count is not a full allowance — on upstream's bare-amount model (plugins#468) the card and tray treat an amount of 0 as not said; a millisecond span is not charted as seconds (`quotaSpanLimit`, upstream lacks it) |
| yetone/magpie#3635 | `ad3badd3` + `4f7e63a8` | `fix/nvidia-parameters-routing` | NVIDIA parameter-list repair keeps reasoning effort; duplicate effort controls stay aligned through routing |
| yetone/magpie#3636 | `0d3dca3d` | `fix/nvidia-parameters-routing` | an explicit unsupported-parameter 400 fails over to the group's remaining members |
| yetone/magpie#3624 | `c931166a` | `fix/nvidia-parameters-routing` | Usage → View routing reveals the selected trace, not off-screen |
| yetone/magpie#3788 | `dd670e6` + `dfd128ca` | `fix/hindsight-nonstream-sse-1767` | the group's first-response timeout also bounds non-streaming requests — headers alone don't count — so a slow member fails over before the caller gives up; the last candidate still waits. `dfd128ca` keeps that wait out of SSE: a non-streaming wait no longer sends a keepalive and HTTP 200 (see the SSE follow-up section below) |
| yetone/magpie#3791 | `dd670e6` + `dfd128ca` | `fix/hindsight-nonstream-sse-1767` | provider JSON `modelCooldown` opts exact model ids into model/account/key cooldowns on 499s and slow starts: 1, 2, 4, 8, 16, 32, then the cap minutes; success resets the ladder; a cooling model is not probed even as last candidate (503 + Retry-After); `RestCapOff` overrides; production opts `raccoon`'s `sn-sensenova-6-8-flash` at a 60-minute cap |
| yetone/magpie#3948 | `56eb1ba1` | `fix/request-protocol-visibility` | a completed non-streaming response is checked by its existing bounded capture: SSE-instead-of-JSON, malformed JSON, missing arrays and empty Chat choices record `protocol_error` while keeping the committed HTTP status (Usage shows `200 · Protocol error`); missing speed says non-streaming/untimed or insufficient sample — diagnostic recording only, no new retry/cooldown policy |
| yetone/magpie#3794 | `82d916e3` | `fix/kilo-tools-count-fallback` | an explicit upstream tool-count limit (Kilo refusing 129 tools, max 128) fails the group over to the next member with the full tool list; the refusing member stays eligible for smaller requests |
| yetone/magpie#3623 | `33c0ade7a` | `feature/request-attempt-chain` | Usage's expanded request loads its recorded routing trace and lists every actual attempt in order — provider/model, status, duration, error, a 499 included; an expired or unavailable trace says so instead of inferring from the current group config. No routing or retry behavior changes |
| build line (ours) | this branch | `prod` | What's new and the update check read a `-pN` build as its base release: `update.parse` strips the suffix, and the notes site is asked in whole release numbers (its own `newer()` reads `-pN` as a pre-release, which hid the range's base releases at both ends — 0.1.2051's notes stayed out of "since 0.1.1948-p1" and 0.1.1948's crept in). `prod_build_test.go` pins it |

Plugin (not this branch): `@magpie-community/opencode-cloudflare-workers-ai-auth`
0.0.4 tarball, source to track = plugins PR #366 (`pull/366/head`). The
tarball in the volume was packed at `bdc302a`; `38ddeb1` after it is
test-only, so the runtime content matches the PR head. Rebuilt and
reinstalled whenever the PR's runtime files change.

Carried after registering here: #3948 was registered on 2026-10-28
while no fix existed; `fix/request-protocol-visibility` (`56eb1ba1`)
appeared the same day and is carried in the table above. #3623 was
registered on 2026-10-29 and its fix (`33c0ade7a`,
`feature/request-attempt-chain`) appeared within the hour, carried in
the table above.

## Deployment history

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
processing success. Missing speed says non-streaming/untimed or insufficient
sample, with its specific reason. No Hindsight log collector, client changes,
feedback endpoint, historical backfill or report-to-request merging is added.

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
