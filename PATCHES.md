# prod/ rolling line — upstream release + skygunner patches

This branch is the source of the production image on docker1 (10.0.0.31,
container `magpie`, image `magpie:<tag>-pN`, ports 3425/3430, volume
`magpie-config`). One branch per deployed release, named after the image
tag; old `prod/*` branches stay as deployment records.

Each round rolls the line forward:

1. `git fetch upstream --tags`; take the newest `vX.Y.Z` release tag (not
   `upstream/main`).
2. Branch `prod/vX.Y.Z-p1` off the previous `prod/*` branch (not off the
   tag), then `git merge vX.Y.Z` — the carried patch commits keep their
   SHAs, so the table below stays true; resolve conflicts by keeping both
   sides' semantics.
3. For each patch in the table below, check whether upstream absorbed it:
   the fix of record is an ancestor of the tag, or an upstream commit says
   it fixes the same issue, or the issue is closed as fixed. Absorbed →
   skip. Otherwise `git cherry-pick -x <sha>`, resolve conflicts by keeping
   both sides' semantics.
4. Verify: each patch's own regression tests, the GUI tests it ships
   (`node --test` with playwright, both engines), `go vet`, and
   `go test -tags nogui ./...` — attribute every red against a clean
   checkout of the same tag before blaming the patches. `TestCoreScriptBudget`
   is part of the gate: if app.js goes over, shorten the patch's own
   comments before considering a budget raise.
5. `docker build --build-arg VERSION=<tag>-p1 -t magpie:<tag>-p1 .`,
   smoke-test on a throwaway config (`magpie version`, 401 without a key),
   then on docker1: back up the volume and container config under
   `/home/hermes/backups/magpie/pre-<tag>-<ts>/`, stop and rename the old
   container to `magpie-pre-<tag>-<ts>` (kept stopped), run the new one
   with the same flags (`-p 3425:3425 -p 3430:3430 -v magpie-config:/config:z
   --user nonroot --restart unless-stopped` and the two MAGPIE_* env vars,
   `web --addr 0.0.0.0:3430 --no-open`).
6. Accept: healthy, 401/200 auth pair, model count equal before/after,
   real `group/fast-cheap` request 200. Push the branch; the branch is the
   record.

## Carried patches (as of v0.1.1612-p1, 2026-10-27)

Checked against v0.1.1612 (2026-10-26): none absorbed yet; #3623 still
has no fix anywhere. At v0.1.1612 upstream made a plugin window's `Used`
a pointer too (#3650); the merge keeps that beside #3487's pointer
`Amount` — one conditional block each, and the aside-text condition
reads both pointers.

| upstream | fix of record (skygunner/magpie) | what it does |
| --- | --- | --- |
| yetone/magpie#3035 | `8f931a28` | recognize SenseNova's guided_grammar compile_grammar refusal and retry the schema as json_object |
| yetone/magpie#3417 | `e75480ec` | Codex expiry-only resets stay inside the expiry window |
| yetone/magpie#3487 | `d4944ace` | quota card: a null amount is not a full allowance; a millisecond span is not charted as seconds |
| yetone/magpie#3635 | `ad3badd3` + `4f7e63a8` | NVIDIA parameter-list repair keeps reasoning effort; duplicate effort controls stay aligned through routing |
| yetone/magpie#3636 | `0d3dca3d` | an explicit unsupported-parameter 400 fails over to the group's remaining members |
| yetone/magpie#3624 | `c931166a` | Usage → View routing reveals the selected trace, not off-screen |

Plugin (not this branch): `@magpie-community/opencode-cloudflare-workers-ai-auth`
0.0.4 tarball (plugins PR #366), installed in the `magpie-config` volume at
`/config/magpie/plugins/` — survives image upgrades; re-verify after each.

Not carried: #3623 (no fix exists on the fork yet; nothing to apply).
Checked again at v0.1.1612: still zero comments, still nothing upstream.
