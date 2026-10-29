# Magpie GUI HTTP API, middleware, routes, access control and MCP auth — audit

Snapshot: `/tmp/magpie-audit-Pzn4fNp7/src` (read-only). Findings are ordered most severe first.

## Coverage

- internal/gui/api.go:841-1000 — `Handler` (mux assembly, boot.js, /api/state, /api/set, /api/drift); 1195-1480 (route registration incl. `if !isWeb(w) { appPageRoutes(mux) }`, `POST /api/settings` carry-over); 2095-2200 (`/api/settings/lan`, `/cors`, `/allowips`, `/port`); 2330-2480 (`held`, `noFramedAPI`, `state`); 2700-2911 (`writeJSONRows` delta path, `fail`, `tilde`, `editRedactRule`, `countClient`); route table enumerated with `rg 'mux\.HandleFunc\("(GET|POST|PUT|DELETE|PATCH)'` (233 registrations).
- internal/gui/web.go:1-474 — `StartWeb`, `webKey`/`CheckWebKey`, `webGuard` (key exchange, sign-in form, `localPath`), `webAppFiles`, `webServed`/`webHandler`/`webCompressed`, `onePort`, `toGateway`, `isWeb`/`ownWeb`, `NetworkLinks`/`LANAddrs`.
- internal/gui/pluginpage.go:1-535 — `pageToken`, `windowParts`, `pagePolicy`, `pageHost`, `servePluginPage`, `apiFramed`, `pluginPageRoutes`, `pageSetAllowed`, `pageRead`.
- internal/gui/archive.go:1-242 (full) — `openArchive`, `archiveRoutes` (`/api/archive`, `/api/archive/file`, `/api/archive/export`), `saveDownload`, `archiveState`.
- internal/gui/backup.go:1-178 (full) — `backupRoutes` (`/api/backup/export`, `/api/backup/import`, `/api/davsync/{action}`), `busy`, `downloads`.
- internal/gui/apppage.go:1-354 (full) — `loadAppPage`/`saveAppPage`, `AppPage`, `appPage.apply`/`state`, `appPageRoutes`, `appPagePortOK`.
- internal/gui/library.go:130-330 — `libraryRoutes` (`/api/library/skill`, `/market/*`, `/icon`, `/rtk*`, `/projects/choose`, `/reveal`); helpers traced into internal/library: `Icon`/`offered` (market.go:1076-1121), `SkillText`/`SkillPath` (skills.go:2456-2468), `probeSkills` (skills.go:1058-1097).
- internal/gui/sessions.go:218-320, 398-437, 630-670 — `/api/sessions/one`, `/transcript`, `/markdown`, `/export`, `sessionMarkdown`, `tightenGC`.
- internal/gui/page_memory.go:1-82 (full) — `readingPage`, `releaseIfPagesRest`, `debug.FreeOSMemory` release path.
- internal/gui/usage.go:37-63, 195-240 — `periodOf`/`ledgerFilter` aliases, `csvStamp`, `/api/usage/requests.csv`, `/api/usage/requests/export`.
- internal/gui/providers.go:1002-1175 — `providerRoutes` head (`/api/lanes`, `/api/upstream`, `/api/providers` ?wait/?last, `/api/gateway/call`), `/api/icons`, `/icons/favicon`, `GET /api/icons/{name}`.
- internal/gui/context.go:552-606 — `contextRoutesAPI` (`/api/context`, `/api/context/text`).
- internal/gui/api.go:993-1195 — agents routes (`/cli`, `/install`, `/stage`, `/restart`, `/preview`, `/disconnect-offline` revision check, `/{action}/{id}`).
- internal/gui/dev_on.go:60-190 + internal/gui/dev_off.go:1-17 — dev long-poll `/api/dev/wait`, `devPage`, `devListen`, `devBackend`; `dev_on.go` is `//go:build dev`, so none of it is in the release binary.
- internal/mcpauth/mcpauth.go:274-404 (Token/`Renew`/`renew`/`markDead`/`expires`), 770-1040 (`StartWith`, callback, `keep`, `finish`, `end`, `Progress`, `Cancel`).
- internal/access/limit.go:1-222 (full) — `Limit`/`Valid`/`Window`/`sinceOf`/`cycle`, calendar vs cyclic windows.
- internal/access/access.go:20-160, 480-560 — `Managed`/`Named`/`ownSecrets`/`ownSecret`, `Authenticate`, `legacyLANKey`, `MigrateLegacyLANKey`.
- internal/middleware/run.go:40-340 — `Begin`/`End`/`letGo`/`acquire`/`reject`/`Request`/`Tried`/`apply`/`Wrap`/`writer.decide`/`Write`.
- internal/gui/app.go:280-400 and internal/cli/web_cli.go, daemon_cli.go — only the web-serving call sites (`Handler(h, gw)`, `StartWeb`) were read, by grep and the surrounding 30 lines.
- Route/auth map: every `/api/*` route sits behind `noFramedAPI(held(mux))` (api.go:2426) and, on `magpie web`/the app's browser page, behind `onePort(gateway.AllowGuard(webHandler(...)))` → `webGuard` at web.go:341/346. `/_magpie/plugin/` skips the key by design (web.go:215) and is token-gated (pluginpage.go:283-350). `/v1`,`/v1beta`,`/mcp` on the page's port are handed to `gw.Front()` (gateway guards, not the page key). `/manifest.json` and two icons are key-free GETs (web.go:234).
COVERAGE: read 18 files, 28474 non-test LOC in scope; skipped 112 (102 other internal/gui non-test files, 8 other middleware/routes/access/mcpauth files, 2 cli web files — large handlers such as the remaining 1.4k lines of providers.go, sessions.go, plugins.go, update.go, trace.go, groups.go, masking.go, and all of internal/routes: route enumeration plus targeted greps only, per budget).

## Findings

### F1 [P2] The API's only CSRF guard rejects cross-site, not same-site: any page on another port of the same host can POST to the API
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/pluginpage.go:360
- code:
```go
	if r.Header.Get("Origin") == "null" {
		return true
	}
	switch r.Header.Get("Sec-Fetch-Dest") {
	case "iframe", "frame", "embed", "object":
		return true
	}
	return r.Header.Get("Sec-Fetch-Site") == "cross-site" && r.Header.Get("Sec-Fetch-Dest") != "document"
```
- trigger: `magpie web` (or the app's Browser-access page) is on, the browser holds the sign-in cookie, and a page served by any other service on the same host — another port, plain HTTP, e.g. `http://<host>:8000` — runs `fetch('http://<host>:3430/api/settings/lan',{method:'POST',mode:'no-cors',headers:{'Content-Type':'text/plain'},body:'{"On":true,"NewKey":true}'})`. Cookies ignore ports and `SameSite=Lax` treats another port as same-site, so the cookie is sent; `text/plain` makes it a CORS "simple" request (no preflight), and `json.NewDecoder(r.Body)` decodes it whatever the Content-Type says.
- effect: that page changes magpie's state with no key of its own: turn LAN sharing on (the gateway becomes reachable from the network), rotate the page key (the user is signed out), rewrite settings. Responses can't be read (no CORS headers), so it is blind writes — which is enough for the routes that matter.
- why: `webGuard`'s cookie and `apiFramed`'s header test both assume "another page" means another site; the browser's real boundary for a host-scoped cookie is origin (scheme/host/port), and `Sec-Fetch-Site: same-site` passes here untouched.
- fix: in `apiFramed`, also return true when `r.Method != http.MethodGet && r.Header.Get("Sec-Fetch-Site") == "same-site"`.
- evidence: `rg -n 'same-site|"same-origin"' internal/gui/*.go` → only test files and a comment match; `rg -n 'apiFramed' internal/gui/*.go` → defined pluginpage.go:356, used only at api.go:2433. So no release code rejects a same-site write. verified-by: grep
- confidence: high

### F2 [P2] The page's sign-in cookie is not `Secure`, so any plain-HTTP service on the same host receives the master key
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/web.go:208
- code:
```go
		// Priority=High (Chrome's): the page's host is often an address
		// other services share on other ports, and a browser keeps a
		// limited number of cookies for one host, all ports together;
		// past it Chrome drops the low and medium ones first (#2033)
		rw.Header().Add("Set-Cookie", (&http.Cookie{Name: cookie, Value: key, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: int(keep / time.Second)}).String()+"; Priority=High")
```
- trigger: `magpie web --lan --tls-cert ... --tls-key ...` (StartWeb wraps the listener with `gateway.ServeTLS`, web.go:89), the user signs in over HTTPS, and a plain-HTTP service is reachable on the same host at another port. The browser sends the host-scoped cookie (no `Secure` attribute, `Path=/`) to that service too.
- effect: that service receives `magpie_web_3430=<the key>` in cleartext, and with it can open the page and read every provider key and account, or change any setting. The same holds for an operator's access log or a proxy in front of a plain-HTTP service on that host.
- why: a cookie that is the whole credential for an HTTPS-administered page must be `Secure` whenever the page was served over TLS; the attribute is absent from the only place cookies are minted.
- fix: set `Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")` on that cookie.
- evidence: `rg -n 'Secure' internal/gui/web.go` → no match; the only Set-Cookie construction in the package is web.go:208. verified-by: grep
- confidence: high

### F3 [P2] Every static asset request drops the shared catalog and file memo, so a page load rebuilds them once per asset
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/api.go:2470
- code:
```go
		if !(api && r.Method == http.MethodGet) {
			// anything that may have written drops what was held, once it
			// has answered with it
			defer provider.Changed()
		}
		h.ServeHTTP(rw, r)
```
- trigger: any page load — `GET /`, `/app.js`, `/styles`, `/icons/*`, `/boot.js`, or a plugin page's files under `/_magpie/plugin/` — interleaved with the page's own `GET /api/state`/`/api/providers`. `held` wraps the whole mux (api.go:2426), so `api` is false for every non-`/api` path and the condition is true.
- effect: each asset finish calls `provider.Changed()` → `filememo.Forget()` + `dropHeld()` (provider/hold.go:145-148), clearing the built catalog. The page issues ~20-30 asset requests during a load, so the catalog is rebuilt that many times while its own API reads run, plus every memoised file read is repeated. The code's own measurements put this at "a fifth of a request's CPU with 30 providers" and `/api/state` at 4s with a few hundred models — that cost is paid per asset instead of once.
- why: the guard tests "this request may have written" as `!(api && GET)`, which classifies every non-API GET — all of which are immutable embedded files — as a write, defeating the memo `held` exists to provide.
- fix: `if api && r.Method != http.MethodGet { defer provider.Changed() }`.
- evidence: `rg -n 'noFramedAPI\(held\(mux\)\)' internal/gui/api.go` → `2426: return noFramedAPI(held(mux))`; `rg -n 'defer provider.Changed' internal/gui/api.go` → `2473`; `rg -n 'func Changed' -A3 internal/provider/hold.go` → `filememo.Forget(); dropHeld()`. The static handler is `mux.Handle("/", ...)` at api.go:851, inside the wrapped mux. verified-by: read
- confidence: high

### F4 [P2] 98 `/api` handlers decode `r.Body` with no size limit; `/api/backup/import` then base64-decodes it into a second copy
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/backup.go:68
- code:
```go
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			fail(rw, err)
			return
		}
		data, err := base64.StdEncoding.DecodeString(in.Data)
		if err != nil {
			fail(rw, err)
			return
		}
```
- trigger: `POST /api/backup/import` with a body of a few hundred MiB-1 GiB (or a 100 MB `POST /api/settings`, which decodes the body into a `json.RawMessage` and then `json.Unmarshal`s it again, api.go:1241-1249). No server-wide body cap exists: the http.Servers built at web.go:106 and apppage.go:237 set only `ReadHeaderTimeout`.
- effect: a 1 GiB body costs ≥1 GiB for the JSON string, ~750 MiB for the decoded bytes, and another copy inside `backup.Open`/`Restore` — roughly 2.5-3 GiB resident from one request, enough to OOM a 2 GiB container or swap the desktop app to a standstill; disk-backed swap then makes every other handler slow.
- why: request bodies are trusted to be small, but the only gate in front of these handlers is a key (or, in the app, nothing but the frame check), so body size is attacker-chosen — and 98 decode sites against 7 `MaxBytesReader` uses in the package shows the limit was applied per handler, not at the boundary.
- fix: wrap the mux once — `http.MaxBytesReader` in `held` (or a `MaxBytesHandler`) with a per-route ceiling for `/api/backup/import`.
- evidence: `rg -o 'json\.NewDecoder\(r\.Body\)' internal/gui/*.go | wc -l` → 98; `rg -n 'MaxBytesReader' internal/gui/*.go | wc -l` → 7; `rg -n 'MaxBytesReader|LimitReader' internal/gui/backup.go` → no match. verified-by: grep
- confidence: high

### F5 [P2] MCP token renewal ignores whether the cross-process lock was taken, so two magpies can spend one rotating refresh token
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/mcpauth/mcpauth.go:311
- code:
```go
	mu.Lock()
	defer mu.Unlock()
	// one renewal at a time across magpies as well: a server that rotates
	// its refresh tokens takes each once
	unlock, _ := filelock.Lock(path()+".renew.lock", filelock.Wait)
	defer unlock()
	r, ok := Get(name)
```
- trigger: the app and `magpie serve` (or two magpie versions during a handover) both hold the same `mcp-signins.json` and both need to renew the same MCP server's token while its token endpoint is slow: process A takes the file lock and spends >10s in `tokenRequest`; B waits out `filelock.Wait` (10s, filelock.go:20/55), which logs "another magpie held it … going on without it" and returns `held=false`. B ignores that (`_`), re-reads the still-old record and spends the same refresh token.
- effect: the rotating server refuses the second spend (`invalid_grant`), and `renew` treats that as death: `markDead(name, r.Access)` → the record is dead and every agent using that server is told `ErrExpired` until the user signs in again. A's own successful write is then refused too (`cur.Access != r.Access`), so its new tokens are dropped.
- why: the invariant "a rotating refresh token is spent once" needs the lock to actually be held; `filelock`'s timeout return is a soft failure meant for callers that can proceed safely, and this caller cannot.
- fix: check `held` and return a retryable error instead of renewing without the lock: `if unlock, held := filelock.Lock(...); !held { return "", errBusy }`.
- evidence: `rg -n 'renew.lock' internal/mcpauth/mcpauth.go` → single global lock path at 311, taken with `_,`; `sed -n '20,80p' internal/filelock/filelock.go` → `Wait = 10 * time.Second` and "held is false when it went on without the lock". verified-by: read
- confidence: medium

### F6 [P3] Export files are named by check-then-write, so two exports at once silently overwrite one another
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/archive.go:204
- code:
```go
	name := filepath.Join(dir, stem+ext)
	for i := 2; ; i++ { // never over an earlier one
		if _, err := os.Stat(name); err != nil {
			break
		}
		name = filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, i, ext))
	}
	if err := os.Rename(tmp.Name(), name); err != nil {
```
- trigger: two exports of the same artifact in flight at once — e.g. `POST /api/backup/export` from the panel and the main window, or the page retrying a slow request — with the same day/stem. Both `os.Stat` see no file, both pick the same name, and `os.Rename`/`edit.WriteAtomic` replaces it. The same shape is at backup.go:46-56 and in `POST /api/usage/requests/export`.
- effect: the first file is replaced by the second, and both callers answer with the same path, so the user believes they have two dated artifacts and has one. For `/api/backup/export` the survivor contains every provider key, so losing one silently means a backup the user thought existed does not.
- why: "never over an earlier one" needs the name to be reserved atomically with creation (`O_EXCL`/`linkat`), not decided by a stat a concurrent writer can race.
- fix: create the chosen name with `os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)` in the loop and rename over only the file it just created.
- evidence: `rg -n 'O_EXCL' internal/gui/backup.go internal/gui/archive.go internal/gui/usage.go` → no match; the three call sites all stat-then-write (archive.go:204-211, backup.go:46-57). verified-by: grep
- confidence: high

## Checked, no defect

- `tightenGC` (sessions.go:637-670) — GOGC/memory-limit tightening is refcounted under one mutex with `sync.OnceFunc` releases, so overlapping `/api/sessions` polls cannot leave the process at GOGC 25 permanently (the naive save/restore bug this shape usually has is not present).
- `readingPage`/`releaseIfPagesRest` (page_memory.go:36-82) — a single `due` timer is created under the same lock that counts in-flight reads, and a late release re-arms for the remaining rest, so a poll every 15s cannot stack timers or trigger a `FreeOSMemory` per request.
- `servePluginPage` (pluginpage.go:283-350) — the `webGuard` bypass for `/_magpie/plugin/` is safe: GET/HEAD only, 32-hex token compared with `hmac.Equal`, folder resolved from `windowParts()` (never from the path), and the file from `plugin.FileIn(dir, rel)`; no request-supplied path reaches `os.Open`.
- `localPath` (web.go:287-293) — the sign-in form's `back` is parsed and rejected on scheme/host/`//`/`/\`/`/_magpie/sign-in`, so the post-sign-in 303 cannot be pointed at another site.
- `mcpauth.renew` in-process path (mcpauth.go:305-360) — per-name `lockOf` mutex, `still(R)` recheck after taking it, and an `update()` that writes only while `cur.Access == r.Access`: a concurrent renewal inside one process returns the new token instead of spending the refresh token twice.
- `POST /api/settings/allowips`/`/lan` (api.go:2100-2180) — the allow list is cleaned by `settings.CleanAllowIPs`, `gateway.KeepsOut` refuses a list that would shut the caller out, and `MAGPIE_ALLOW_IPS` overrides the page entirely; the LAN guard cannot be widened blind.
- `provider.Hold` (provider/hold.go:113-141) — takes the mutex only to build/drop and to bump a refcount; no lock is held while a handler runs, so a slow `/api/*` request cannot serialise the server (the `held` wrapper is safe to apply to long reads).
- `provider.FetchNewSoon` (provider/models.go:475-488) — CAS plus a minimum interval, so the `/api/state` poll cannot fan out one goroutine per request; `FetchNewBehind` shares the same flag.
- `GET /api/icons/{name}` (providers.go:1145-1158) — `provider.IconServed` gates on `iconName` (provider/icon.go:40-45) before `filepath.Join(IconDir(), name)`, and the mux wildcard is one segment, so no traversal; SVG is served with `CSP: default-src 'none'`.
- `GET /api/library/icon?u=` (library.go:181-190) — `library.Icon` consults a cache keyed by a hash, then `offered(u)` restricts fetches to `https://github.com/<owner>.png?size=96` and previously seen market icons, so the query cannot be used as a general SSRF.
- `POST /api/agents/disconnect-offline/{id}` (api.go:1094-1126) — the plan's `Revision(a.ID)` is recomputed and compared before `ExecuteOffline`, and `DisallowUnknownFields` is set, so a stale preview cannot be replayed onto a changed config.
- `StartWeb`/`appPage.apply` (web.go:79-118, apppage.go:200-260) — the page is only ever served with `webGuard`, the key is kept out of `settings.json`/backups (web-page.json, 0600), and `appPageRoutes` is registered only when `!isWeb(w)`, so a browser cannot turn browser access off or read the key it came in with.

## Open questions

- Does WebKitGTK/WebView2 send `Sec-Fetch-Site: same-site` (rather than `none`) on the desktop window's own POSTs? If it can, F1's proposed `same-site` rejection must be scoped to requests that also carry the web cookie, or the app's own page would be locked out. Needs a real webview capture.
- How long a catalog rebuild actually takes on this snapshot's default config (number of providers × models)? F3's magnitude is quoted from the code's own comments (4s `/api/state`, "a fifth of a request's CPU with 30 providers"); a timed run against a populated config would size it exactly.
