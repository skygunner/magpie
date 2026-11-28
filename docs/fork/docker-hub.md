# magpie-patched

[yetone/magpie](https://github.com/yetone/magpie) — the local AI gateway —
built from each upstream **release tag** plus a small, reviewed set of
fork differences retained after upstream compatibility review. Maintained by
[skygunner](https://github.com/skygunner/magpie); the manifest of carried
patches lives in [`PATCHES.md`](https://github.com/skygunner/magpie/blob/integration/nightly/PATCHES.md)
on the `integration/nightly` branch.

## Tags

- `latest` — the newest production integration (same as the newest
  `X.Y.Z-pN` tag below).
- `X.Y.Z-pN` — one image per deployed integration round: upstream release
  `vX.Y.Z` plus the carried patches (e.g. `0.1.1670-p1` = `v0.1.1670` +
  the patches below). Every image corresponds to an annotated git tag
  `deployed/<version>` on the fork for releases since 0.1.3007-p2; older `prod/*` tags can also be failed candidates.

## Current production image

`0.1.4984-p1` and `latest` point to the same accepted image. The upstream base is [v0.1.4984](https://github.com/yetone/magpie/releases/tag/v0.1.4984). [Exact build source](https://github.com/skygunner/magpie/commit/ced88b9d5567db1cd7580cee677aac5eb0a03377).

### Hotfixes carried in this image

| Issue or change | Behavior |
| --- | --- |
| [#8028](https://github.com/yetone/magpie/issues/8028) · [Fork PR #4](https://github.com/skygunner/magpie/pull/4) | Skip the Raccoon GLM Flash group member for Chat or Responses JSON schema output. Forward the original schema to another eligible member; JSON object requests can still use Raccoon. |
| [#7867](https://github.com/yetone/magpie/issues/7867) · OpenAI reasoning temperature | GPT-6 Luna and GPT-6.1 Terra API-key attempts omit a custom temperature during reasoning. Explicit none and other providers keep the requested temperature. |
| [#7864](https://github.com/yetone/magpie/issues/7864) · Tool call IDs on Chat relay | Restore the original ID on assistant tool calls and matching tool results when forwarding carried Responses reasoning to Chat. This avoids a 400 from the upstream 64-character ID limit. |
| [#7668](https://github.com/yetone/magpie/issues/7668) · Raccoon reasoning replay | Collapse duplicate reasoning aliases on SenseNova Flash streaming Chat replay while retaining distinct reasoning and tool history. |
| [#3035](https://github.com/yetone/magpie/issues/3035) · Structured output fallback | Retry an explicitly refused guided-grammar format using the compatible output format. |
| [#3635](https://github.com/yetone/magpie/issues/3635) · NVIDIA parameter compatibility | Repair explicitly rejected optional parameters and keep equivalent reasoning settings in sync. |
| [#3636](https://github.com/yetone/magpie/issues/3636) · Group fallback on parameter refusal | Let another member serve a request when the previous member rejects an explicit list of parameters, without resting that account. |
| [#3794](https://github.com/yetone/magpie/issues/3794) · Tool-count limit | Pass the complete tool list to the next group member when an upstream rejects the number of tools. |
| [#3948](https://github.com/yetone/magpie/issues/3948) · Request protocol diagnostics | Check bounded non-streaming error envelopes and preserve the HTTP status already sent to the client. Usage identifies protocol failures. |
| [#3623](https://github.com/yetone/magpie/issues/3623) · Recorded routing attempts | Show the actual ordered upstream attempts in request details, including canceled requests. |
| [#3624](https://github.com/yetone/magpie/issues/3624) · Usage navigation | Reveal the selected trace and its routing story when opening a request from Usage. |
| [#4385](https://github.com/yetone/magpie/issues/4385) · Interrupted wheel scrolling | Restore the full interrupted pixel-wheel step from its original position in the WebKit UI. |
| [Fork PR #2](https://github.com/skygunner/magpie/pull/2) · Slow first token | Back off the affected model on the same account or key from its first missed streaming deadline. The first rest is one minute, doubling up to one hour. Provider opt-outs and lower caps still apply. |
| [Fork commit b3cbe9e7c](https://github.com/skygunner/magpie/commit/b3cbe9e7c) · Patched build versions | Compare a fork build such as `0.1.4984-p1` with upstream releases using its base version, including release-note ranges. This fork-specific change is tracked by commit. |

The separate Cloudflare Workers AI plugin is installed in the persistent volume. Its [plugin PR #366](https://github.com/magpie-community/plugins/pull/366) is tracked separately from the image. The [fork manifest](https://github.com/skygunner/magpie/blob/integration/nightly/PATCHES.md) records retained, suspended and absorbed patches.

## Release process (summary)

`integration/nightly` retains reviewed fork differences and merges one pinned upstream release per round. Absorbed, overlapping or conflicting patches are removed or suspended. `production/stable` and `deployed/*` advance only after production acceptance and observation succeed. Old `prod/*` tags may represent failed candidates.

Unknown snapshot differences or failed gates block production. Rollback is part of the detached deployment transaction. Full manifest and evidence: [integration/nightly/PATCHES.md](https://github.com/skygunner/magpie/blob/integration/nightly/PATCHES.md).

## Deployment

```sh
docker run -d --name magpie \
  --restart unless-stopped --user nonroot \
  -p 3425:3425 -p 3430:3430 \
  -v magpie-config:/config \
  -e MAGPIE_PUBLIC_URL=http://<this-host>:3425 \
  bigshort/magpie-patched:latest web --addr 0.0.0.0:3430 --no-open
```

- **3425** — the agent API (OpenAI/Anthropic-compatible, `/v1`).
- **3430** — the browser console; the first run prints a `?k=…` link
  (also `docker logs magpie`) that signs you in to set up providers,
  routing groups, and keys.
- `magpie-config` volume keeps everything: config, sign-ins, keys,
  plugins. Back it up before upgrades
  (`docker run --rm -v magpie-config:/from:ro -v $PWD:/to alpine tar czf /to/magpie-config.tgz -C /from .`).
- Add `:z` to the volume mount on SELinux hosts.
- Upgrading: pull the new tag, stop and rename the old container, run the
  new one with the same flags — the volume carries over.

Upstream project and docs: [yetone/magpie](https://github.com/yetone/magpie).


