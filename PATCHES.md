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
   Carried cherry-picks keep their SHAs, so the table below stays true.
   Resolve conflicts by keeping both sides' semantics; adapt new upstream
   tests that meet a carried type change (noted below when it happens).
2. Add new fixes: `git cherry-pick -x` from their `fix/*` branches, in
   branch order.
3. Gate: each patch's own regression tests, the GUI tests below in both
   engines (`node --test` with playwright), `go vet`,
   `TestCoreScriptBudget` (if app.js goes over, shorten the patch's own
   comments before considering a budget raise), and
   `go test -tags nogui ./...` — attribute every red against a clean
   checkout of the same tag before blaming the line.
4. Tag `prod/<tag>-pN` (annotated); build
   `docker build --build-arg VERSION=<tag>-pN -t magpie:<tag>-pN .`;
   smoke-test on a throwaway config (`magpie version`, 401 without a key).
   Publish to Docker Hub: `docker tag` the image as
   `bigshort/magpie-patched:<tag>-pN` and `:latest` (latest always points
   at the newest round), push both, and refresh the repo's README — the
   version header, the patch table, and the source tag link come from
   this file's current round.
5. On docker1: back up the volume and container config under
   `/home/hermes/backups/magpie/pre-<tag>-<ts>/` (volume tar + container
   inspect), stop and rename the running container to
   `magpie-pre-<tag>-<ts>` (kept stopped), run the new image with the same
   flags (`-p 3425:3425 -p 3430:3430 -v magpie-config:/config:z --user
   nonroot --restart unless-stopped`, the two MAGPIE_* env vars,
   `web --addr 0.0.0.0:3430 --no-open`).
6. Accept: healthy, 401/200 auth pair, model count equal before/after, a
   real `group/fast-cheap` request 200. Push `prod` and the tag.

## Carried patches (as of v0.1.1691-p1, 2026-10-27)

Checked against v0.1.1691 (2026-10-27): none absorbed yet; all nine
issues still open. The v0.1.1691 merge needed no conflict resolution. Integration notes: at
v0.1.1612 upstream made a plugin window's `Used` a pointer (#3650), kept
beside #3487's pointer `Amount`; at v0.1.1670 upstream added runs-out
spans (#3762) whose new test wrote a bare `Amount` — adapted to the
pointer on this line.

| upstream | fix of record (skygunner/magpie) | what it does |
| --- | --- | --- |
| yetone/magpie#3035 | `8f931a28` | recognize SenseNova's guided_grammar compile_grammar refusal and retry the schema as json_object |
| yetone/magpie#3417 | `e75480ec` | Codex expiry-only resets stay inside the expiry window |
| yetone/magpie#3487 | `d4944ace` | quota card: a null amount is not a full allowance; a millisecond span is not charted as seconds |
| yetone/magpie#3635 | `ad3badd3` + `4f7e63a8` | NVIDIA parameter-list repair keeps reasoning effort; duplicate effort controls stay aligned through routing |
| yetone/magpie#3636 | `0d3dca3d` | an explicit unsupported-parameter 400 fails over to the group's remaining members |
| yetone/magpie#3624 | `c931166a` | Usage → View routing reveals the selected trace, not off-screen |
| yetone/magpie#3788 | `dd670e6` (fix/hindsight-nonstream-cooldown) | the group's first-response timeout also bounds non-streaming requests — headers alone don't count — so a slow member fails over before the caller gives up; the last candidate still waits |
| yetone/magpie#3791 | `dd670e6` (fix/hindsight-nonstream-cooldown) | provider JSON `modelCooldown` opts exact model ids into model/account/key cooldowns on 499s and slow starts: 1, 2, 4, 8, 16, 32, then the cap minutes; success resets the ladder; a cooling model is not probed even as last candidate (503 + Retry-After); `RestCapOff` overrides; production opts `raccoon`'s `sn-sensenova-6-8-flash` at a 60-minute cap |
| yetone/magpie#3794 | `82d916e3` (fix/kilo-tools-count-fallback) | an explicit upstream tool-count limit (Kilo refusing 129 tools, max 128) fails the group over to the next member with the full tool list; the refusing member stays eligible for smaller requests |

Plugin (not this branch): `@magpie-community/opencode-cloudflare-workers-ai-auth`
0.0.4 tarball, rebuilt from plugins PR #366 head `bdc302a` (span in
seconds per the magpie contract, used percent for the router, 429 rests
the account instead of the card's percent, Analytics reads), installed in
the `magpie-config` volume at `/config/magpie/plugins/`. Re-verify after
each image upgrade.

Not carried: #3623 (no fix exists on the fork yet; nothing to apply).
Checked again at v0.1.1670: still zero comments, still nothing upstream.

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
