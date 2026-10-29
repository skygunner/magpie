## Coverage

- internal/gui/assets/app.js — 100-229 (sameReads/api/notJSON), 372-426 (status, renderAgentsLoading, renderAgents head), 1140-1174 (connectSwitch write path), 1270-1329 (newModels, staleSaid), 3200-3274 (closeAgentModels, openAgentModels), 3700-3799 (openRowMenu, menu listener add/remove), 4130-4370 (load, renderUpdateBadge, downloadingText, restart), 4560-4690 (noteBlocks, mergedNotes, inlineMD, noteLink), 5925-6024 (loadProviders, loadProvidersNow, providersWhileFetching), 8360-8399 (skeleton watchdog interval), 13800-13934 (laneText, laneLimitPill, refreshLanes), 15350-15439 (loadQuotas, loadQuotaHist, keepRefreshedQuotas), 16840-16889 (panel stats interval), 17930-18069 (panelAge, quotaFit, readWhen), 23540-23564 (wag), 24170-24279 (loadBack, drift interval, update/plugin-dot intervals); whole-file greps for innerHTML, insertAdjacentHTML, setInterval, setTimeout, fetch(, AbortController, addEventListener, localStorage, api() call sites, renderUpdateBadge, loadProviders.
- internal/gui/assets/context.js — 195-304 (morph, morphKids, roll, squarify), 600-949 (ctxCard, grid/map/tip listeners, fill, drawRows, ring), 940-1044 (readText, longText, picture, pieceView, openCtxText), 1100-1219 (Usage Context tab, agentCard, fact builders), 1300-1409 (chart, innerHTML gradient); grep of innerHTML sites.
- internal/gui/assets/plugins.js — 780-899 (npm search debounce, mirrorSwitch, draw/drawBody); whole-file grep of title/textContent/href/src/innerHTML writes.
- internal/gui/assets/pluginpages.js — 1-370 (full: open, answer, message listener, panels, fill, load, tabs).
- internal/gui/assets/kernel.js — 1-274 (full: page registry, fail/retry, run, ui kit, magpie.call/start/configure).
- internal/gui/assets/webfonts.js — 1-117 (full: drawn/merge/list/ask/typing).
- internal/gui/assets/fonts.js — 1-42 (full: own/keep/quoted/apply).
- internal/gui/assets/purposes.js — 1-107 (full, generated request-kind table).
- internal/gui/assets/compat.js — 1-41 (full).
- internal/gui/assets/layout.js — 1-22 (full).
- internal/gui/assets/omarchy.js — 1-46 (full: apply/poll).
- internal/gui/assets/start.js — 1-25 (full: keptView, first load).
- internal/gui/assets/theme.js — 1-28 (full: holdStill, boot prefs).
- internal/gui/assets/index.html — 1-123 (head, script order, nav, view-agents), 505-526 (script load order); grep of inline onclick/onload/iframe/script.
- internal/gui/assets/plugins/routing/client.js — 3168-3274 (askedTitles/otherTitles, askUnnamed, refreshSessionNames, 15s timer), 4360-4464 (resume/start/frame/pause), 4470-4689 (inSight, poll loop, backoff, hubText).
- internal/gui/assets/plugins/library/client.js — 2040-2135 (signInBox, start, follow, signOut).
- internal/gui/assets/plugins/sessions/client.js — 150-269 (searchSoon/AbortController, shown, draw, fitObserver).
- internal/gui/assets/plugins/usage/page.js — 895-969 (ledRead flight/abort, loadLedger), 3780-3824 (refreshUsage flight, usageTicks).
- internal/gui/assets/plugins/gateway/keys.js — 60-129 (refreshGatewayKeys, gatewayKeysTicks).
- internal/gui/assets/plugins/redact/page.js — 90-124 (maskPageLoad, 15s visibility-guarded interval).
- COVERAGE: read 20 files, 51581 non-test LOC in scope; skipped 13 (23105 lines of generated i18n tables in i18n.js and plugins/*/i18n.js, and 2340 lines in 6 plugin scripts — gateway/client.js, gateway/settings.js, otel/page.js, redact/client.js, redact/demo.js, usage/client.js — not opened; CSS/icons out of scope).

## Findings

### F1 [P2] A failed first `/api/state` read leaves the Agents page on its skeleton forever
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/assets/app.js:4215
- code: ```go
    if (view === "settings" && !syncOpen) await loadSettings();
  } catch (e) {
    status(e.message, "err");
  }
  renderUpdateBadge();
```
- trigger: first load (`load.done` false) with `/api/state` failing — 500 from an unreadable config, or a proxy's 502/504/524 tuned page (the code itself handles proxies in `notJSON`). Both `api("state?last=1")` and `api("state")` must fail, which is the normal first start. `renderAgentsLoading()` at app.js:4140 has already put 5 skeleton rows and `aria-busy=true` on `#view-agents`.
- effect: the default landing page shows 5 shimmering skeleton rows indefinitely; the only feedback is a toast that `status()` clears after 8 s (app.js:382). No error text, no retry control. Same for the tray panel. Recovery requires the user to focus/alt-tab the window (the only other `load()` caller) — and `load.done` stays false, so every later attempt also never draws the kept-state fallback.
- why: the failure path only toasts; the `.loading` class and the skeleton children are removed exclusively inside `renderAgents()` (app.js:429), which needs a `state` object that never arrived, and there is no retry timer or error renderer for this view.
- fix: in the `catch`, when `!state`, replace the skeleton with an error row whose "Try again" button calls `load()` (and keep a retry timer), instead of leaving `.loading` set.
- evidence: `rg -n 'classList\.remove\("loading"\)' internal/gui/assets/app.js` → `429:  page.classList.remove("loading");` and `6132` (providers view only); app.js:4215-4216 shows the catch does nothing else. verified-by: grep
- confidence: high

### F2 [P2] A slow state read overwrites a write that landed while it was in flight
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/assets/app.js:4180
- code: ```go
    if (shown && state !== shown) next = state;
    // a setting changed while this was on its way (it can take seconds):
    // what came back is from before it, and would put the old theme back
    if (!prefsSettled(since) && load.done) next.settings = state.settings;
    if (visibilitySince !== agentsVisibilityRevision) {
      const { agentOrder, agentsHidden, agentsShown } = state.settings;
      next.settings = { ...next.settings, agentOrder, agentsHidden, agentsShown };
    }
    state = next;
```
- trigger: window regains focus (app.js:24194 `visibilitychange` → `loadBack` → `load()`), which asks `api("state")` — the code's own comment says this read "can take seconds" (app.js:4173). The reader then clicks an agent's Connect switch before it lands: app.js:1165 `state = await api("agents/connect/" + a.id, {})` sets the post-write state, and the older read resolves afterwards and executes `state = next` with the pre-connect snapshot. Same for disconnect (app.js:2265) and Keep (app.js:2297).
- effect: a connect/disconnect the server accepted is drawn as not done — the switch flips back and the row's "Connected" line disappears, contradicting the success toast shown by `connectedSaid` a moment earlier — until the next `load()`/read (up to minutes if the window keeps focus). In the reverse case a removed account is drawn again as present.
- why: the "a write's answer stands" invariant is only implemented for the first load, because `shown` is non-null only when the kept `state?last=1` list was drawn; `prefsSettled`/`agentsVisibilityRevision` cover settings and hide/order only, never agents/providers/accounts.
- fix: capture the state identity before the await (`const was = state`) and, when `state !== was`, keep the newer `state` instead of `next` (the same guard `shown` provides on the first load).
- evidence: app.js:4169-4180 read in full: `let next = await asked;` … `if (shown && state !== shown) next = state;` … `state = next;`, with `shown` assigned only inside the `if (!load.done)` branch at app.js:4157-4161; the write sites at 1165/2265/2297 assign `state` directly. verified-by: read
- confidence: medium

### F3 [P2] MCP sign-in poll never ends when its endpoint keeps failing, and the failure is swallowed
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/assets/plugins/library/client.js:2117
- code: ```go
    async function follow(id) {
      while (mcpSigning?.id === id) {
        await new Promise((r) => setTimeout(r, 800));
        let st;
        try { st = await api("library/mcp-signin/" + id); } catch { continue; }
        if (mcpSigning?.id !== id) return;
        if (st.state === "waiting") continue;
```
- trigger: start a Library server's sign-in (`library/mcp-signin`), then make `GET /api/library/mcp-signin/<id>` fail persistently — e.g. magpie (or the plugin/sign-in host) restarts while the browser flow is open, so the id is unknown and the endpoint answers 4xx/5xx. `catch { continue; }` re-enters the loop with the same `id`, and nothing else changes `mcpSigning`.
- effect: an 800 ms request loop that never terminates (1.25 req/s for the life of the page), while the row keeps saying "Finish signing in in your browser…" and the real error is never shown; the reader has to cancel by hand, and if the sign-in box was closed the loop is invisible.
- why: the only exit conditions are a state transition or the user changing `mcpSigning`; a transport/HTTP error is treated as "keep waiting" instead of a terminal failure, so an unrecoverable poll looks like progress.
- fix: count consecutive errors and, after a few, set `mcpSigning = { name, state: "failed", error: e.message }; draw(); return;`.
- evidence: `rg -n -F 'catch { continue; }' internal/gui/assets/plugins/library/client.js` → `2117:        try { st = await api("library/mcp-signin/" + id); } catch { continue; }` (only occurrence; the surrounding `while` at 2114 has no attempt bound). verified-by: grep
- confidence: high

### F4 [P2] loadProvidersNow lets an older providers read overwrite a newer write
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/assets/app.js:5981
- code: ```go
  const was = providers;
  const next = await fresh;
  // a change saved meanwhile drew the list it was answered with, newer
  // than this read
  if (kept && providers !== kept) return;
  providers = next;
```
- trigger: on the Providers page, a reload is in flight (window focus → app.js:4205 `await loadProviders()`, or `show("providers")` app.js:23303) and the reader flips a provider's on/off switch before it lands: app.js:6419 `providers = await api("provider/" + (on ? "on" : "off"), { id: p.id })` assigns the post-write list; the older read then resolves and assigns `providers = next` with the pre-toggle list. Overlap is possible because `loadProviders()` de-duplicates only while `providers` is null or `.last` (app.js:5930), so any later load starts a second concurrent read.
- effect: the switch the reader just flipped flips back (or an account just removed reappears) even though the server accepted the change, until the next read. It looks like the toggle failed and invites a second click.
- why: the stale-check `if (kept && providers !== kept)` only exists for the first-load kept list (`kept` is null on every later load); `was` is captured at 5976 but never compared, so the write's newer answer is discarded.
- fix: `if (providers !== was) return;` immediately before `providers = next;` (mirror of the `kept` guard).
- evidence: app.js:5929-5937 (`if ((!providers || providers.last) && loadProviders.first) return loadProviders.first;`) and 5976-5981 read in full; `rg -n -F 'providers = next' internal/gui/assets/app.js` → `5981` and `6393`. verified-by: read
- confidence: medium

### F5 [P3] Every caller of renderUpdateBadge starts its own 700 ms poll chain during a download
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/assets/app.js:4306
- code: ```go
  if (pulling) {
    if (u.state === "ready") {
      delete b.dataset.pulling;
      return restart();
    }
    label.textContent = downloadingText(u);
    setTimeout(renderUpdateBadge, 700);
    return;
```
- trigger: an update is downloading (`b.dataset.pulling` set, state checking/downloading), and the reader focuses the window a few times: every `load()` ends with `renderUpdateBadge()` (app.js:4218), the header also polls every 15 min (app.js:24245), and each `api("update/install")` answer calls it again (4297/4329/4350). Each of those invocations sees `pulling` and schedules its own 700 ms continuation; the timer handle is never stored, so chains are never cleared or merged.
- effect: N parallel `GET /api/update` loops at ~1.4 req/s each — five focus events during one download give ~7 req/s of polling plus the backend's update-state work, and they all persist until the download leaves the pulling states. Nothing on screen shows more than one pill.
- why: the recursive poll has no single owner — unlike `providersWhileFetching.timer` (app.js:6005) or `b.waitPoll` (app.js:4328), the handle is not kept and not cleared before scheduling.
- fix: keep one handle: `clearTimeout(renderUpdateBadge.t); renderUpdateBadge.t = setTimeout(renderUpdateBadge, 700);`.
- evidence: `rg -n -F 'setTimeout(renderUpdateBadge' internal/gui/assets/app.js` → only `4306`; `rg -n 'renderUpdateBadge' internal/gui/assets/app.js` → callers at 4218 (`load()`), 4297, 4329, 4350, 4384, 19608, and `setInterval` at 24245. verified-by: grep
- confidence: high

## Checked, no defect

- `api()` read cache and delta merge (app.js:100-205): the `X-Magpie-Had`/`X-Magpie-Same` short-circuit only returns the cached object when the server confirms identity, `X-Magpie-Delta` is applied only when both sides are arrays, and `readTags` is a `WeakMap` guarded by `data && typeof data === "object"`, so no stale-body or null-key crash path was found.
- `pluginpages.js` message boundary (1-370): every `postMessage` is accepted only when `ev.source` is the exact `contentWindow` of a known page/panel frame, frames are `sandbox="allow-scripts"` with no same-origin, and `read` is checked against the plugin's declared `reads` before the API call; no origin/`event.data` confusion found.
- `kernel.js` error boundary (1-274): `run()` catches sync throws and attaches `.catch` to returned promises, so a page hook that fails renders "This page stopped working" with Try again instead of a blank page; `p.ready` rejections are therefore never unhandled (app.js:23302 attaches `.then(back)` to the already-caught promise).
- Render-path escaping: every `innerHTML`/`insertAdjacentHTML` site under assets interpolates constants or numbers only (static SVG glyphs at app.js:253/3307/8387, context.js:1152/1340 with a generated `ctx-waste-N` id, `part.html` in kernel.js:50/97 which is the plugin's own pack), and release notes go through `inlineMD`/`noteLink`, which link only `https?://` and never write HTML; no server- or model-derived string reaches an HTML parser.
- Timers that correctly stop: `refreshLanes` clears its 2 s tick when no `.alane` pill is in sight (app.js:13904), `usageTicks`/`gatewayKeysTicks`/`maskLoad`'s interval all clear on `document.hidden`/view changed (usage/page.js:3804, gateway/keys.js:99, redact/page.js:103), and the routing trace long-poll waits for visibility before re-asking with a 3 s error backoff (routing/client.js:4526-4633).
- Overlapping-read discipline in the newer loaders: `loadQuotas` (app.js:15363) serializes whole-page reads and re-asks while `X-Magpie-Reading` is set, `loadQuotaHist` drops older answers by sequence, `refreshUsage`/`ledRead` queue or abort via `AbortController`, and sessions search discards answers whose `found`/`needle` moved on — no stale-overwrite defect found in those paths.
- Dialog/menu listener pairing: `openAgentModels`/`closeAgentModels` and `openRowMenu`/`closeAgentMenu` add and remove the same capture-phase `mousedown`/`keydown`/`scroll` and `resize` handlers (app.js:3267-3275, 3706-3726, 3795-3799), so repeated open/close does not accumulate listeners.

## Open questions

- Whether `/api/state` and `/api/providers` reads in practice outlast the write calls they race with (F2, F4) was not measured; the source comments claim seconds-long reads after a start or over `\\wsl.localhost`, but a timing run on a machine with several agents/WSL would settle the overlap window.
