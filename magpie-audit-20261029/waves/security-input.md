# Magpie security-input audit — trust boundaries

Snapshot: `/tmp/magpie-audit-Pzn4fNp7/src` (read-only). Lane: validation, authz, SSRF, traversal, redaction, secret handling.

## Coverage

- internal/gateway/lan.go — 1-665; listenAddr/pinnedHost/publicURL, OnNetwork, Relisten, lanGuard, ownCallback, rebound/fromBrowser/ownHost, callerGuard, identifyCaller, callerKey(s), managedKey, refusedKey, local/proxied/trustProxy.
- internal/gateway/cors.go — 1-223; corsGuard, keyedCaller, foreignPage, localhostSubdomain, hostPort, corsAllowed, corsHeaders.
- internal/gateway/middleware.go — 1-158; serveAgent, triedOn, gatewayHooks, gatewayQuery, gatewayOrigin.
- internal/gateway/allow.go — 1-230; allowList, allowPrefixes, peerAddr, ownAddr, allowedIn, AllowedFrom, KeepsOut, allowed, AllowGuard, refuseNotAllowed.
- internal/gateway/oneport.go — 1-98; Front, ClientPath, FromPagePort, viaPagePort, takePagePort.
- internal/gateway/image_fetch_guard.go — 1-330; fetchAllow/allowFor/addr/host, publicTransport.RoundTrip, guardDialer, dialPinned, fakeAnswer, dropLoon, onThisComputer, guardedClient, PublicClient.
- internal/gateway/rules.go — 1-260, 336-359; ruleFor (turnRules map, 125-355), firstWords, ruleKey, RuleHit.
- internal/gateway/archive.go — 118-195; Call.archiveName, ArchiveName, Archived/ArchivePart, archive queue.
- internal/gateway/handover.go — 290-360; ForwardCallbacks, forwardCallback, forwarding.
- internal/gateway/gateway.go — grepped only (643, 740-840, 7139, 7194); Handler middleware chain, SessionHeader, conversationID.
- internal/netproxy/lan.go — 1-88; onLAN, lanIP, lanLookup, lanWait/lanKeep, lanCache.
- internal/netproxy/unchecked.go — 1-90; WithUncheckedTLS, uncheckedHost, hostKey, uncheckTLS.
- internal/gui/web.go — 150-380, 400-490; webKey, NewWebKey, CheckWebKey, webGuard, localPath, webTurnedAway, webServed/webHandler/webCompressed, onePort, toGateway, loopbackPeer.
- internal/gui/apppage.go — 1-300; appPageConf.addr, loadAppPage, saveAppPage, AppPage, appPage.run/apply, appPageRoutes.
- internal/gui/caller_keys.go — 1-111; keyRow/withLimits, callerKeyRoutes (GET list/models, POST {action}).
- internal/gui/pluginpage.go — 275-380; servePluginPage, apiFramed, pluginPageRoutes.
- internal/gui/archive.go — 1-243; openArchive, saveDownload, archiveRoutes.
- internal/gui/backup.go — 1-178; backupRoutes (export/import, davsync actions), busy, downloads.
- internal/gui/api.go — 2405-2465 only; noFramedAPI, held (the single apiFramed wiring point).
- internal/davsync/restore.go — 1-271; SetAuto, Restore, Upload, Undo, keepHere.
- internal/davsync/archive.go — 1-89; Bucket.at/Put/Get/Open.
- internal/davsync/davsync.go — 760-1060; collect, hashes, orEmpty, take, changed, bring, keepUnseenGatewayKeys, syncOnce.
- internal/provider/icon.go — 20-90; iconName regexp, IconFile, StoreIcon, FetchIcon, iconClient/guardClient.
- internal/provider/codex_request.go — 120-190; codexPromptsPath, codexPrompts, saveCodexPrompts.
- internal/provider/signin.go — grepped 62-740; StartSignIn/StartSignInWith, callback dispatch, state comparison (566, 691), claim, finish.
- internal/provider/claude_signin.go — grepped 238-427; pasted-callback parsing, status handling.
- internal/access/access.go — 60-180, 470-570; ownSecrets, ownSecret, Key/Identity, Path/load/List, save, setLegacyMirror, Authenticate, legacyLANKey, MigrateLegacyLANKey.
- internal/plugin/page.go — 200-285; htmlInside, inside, FileIn.
- internal/library/skillgit.go — 100-230; git source parsing, ref/path validation, gitURL, gitEnv, runGit.
- internal/library/skills.go — 60-140, 280-390; SkillHere, Source, skillsDir/skillDir, readMeta, oldPlace, dropOld, unlink, realDir.
- internal/library/setup.go — 100-260; Snapshot, Holds, Restore.
- internal/pii/serve.go — 1-120; helper, srv, turn, stopCh, timeouts.
- internal/pii/fetch.go — 1-140; client, Download, run, Cancel, install, ensureServe.
- internal/backup/backup.go — 330-510; Parts, Restore (providers/settings/profiles/agents/library branches).
- internal/redact/redact.go — grepped 140-170, 400-430; the API_KEY rule table and slug/token boundary helpers.

COVERAGE: read 35 files, 229265 non-test LOC in scope; skipped 1189 non-test Go files (neighboring lanes own them; triaged by grep only)

## Findings

### F1 [P2] Routing-turn table is unbounded for a day and re-scanned on every request
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/rules.go:348
- code:
```go
	turnRules.m[key] = turnRule{turn: turn, use: hit.Use, n: hit.N, at: now, input: input, intent: q.Intent, effort: hit.Pick, wanted: hit.Wanted, answered: answered, size: size, hour: hour}
	if len(turnRules.m) > 4096 {
		for k, tr := range turnRules.m {
			if now.Sub(tr.at) > stickKeep {
				delete(turnRules.m, k)
			}
		}
	}
	return hit
```
- trigger: a group that has rules (`g.Ruled()`) and a caller allowed to send it requests (any enabled gateway key when shared, or any local agent): POST /v1/chat/completions N times with a fresh `X-Magpie-Session: s-<n>` (SessionHeader, gateway.go:7139) and a distinct first user message. Each request is a new `key` (group + session + firstWords, rules.go:132-135), so line 348 inserts a new entry.
- effect: entries younger than `stickKeep` (24h, affinity.go:71) are never removed, so the map grows 1 entry/request (~200 B) with no ceiling. Once it passes 4096, every single request runs the loop at 349-353 over the whole map while holding `turnRules.Lock()` (also taken at 192 and 430, and per request twice), so the hot path becomes O(n) under one global mutex shared by all requests: 200k unique sessions means ~200k map iterations of CPU per request, serialized, plus ~40 MB retained.
- why: the only eviction is age-based and only fires above the cap, so 4096 is a trigger, not a bound — the "recent turns" table is really "every session seen in the last 24 hours", an attacker-sized set, and the cleanup itself is on the request path.
- fix: delete stale entries before inserting (scan a bounded number, or evict the oldest when over the cap) instead of walking the whole map when it is over 4096.
- evidence: `rg -n 'turnRules\.' internal/gateway --glob '!*_test.go'` → 125 (declaration), 192/256/342/430 (locks), 348 (sole insert), 349-353 (sole delete, guarded by `len(turnRules.m) > 4096` and `now.Sub(tr.at) > stickKeep`); `rg -n 'stickKeep =' internal/gateway/affinity.go` → `71: stickKeep = 24 * time.Hour`. verified-by: read, grep
- confidence: high

### F2 [P3] Page key has no failure accounting: unlimited guesses against the one credential for every setting
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/web.go:238
- code:
```go
		if r.URL.Path == webSignInPath && r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(rw, r.Body, 4096)
			back := localPath(r.PostFormValue("back"))
			if !same(r.PostFormValue("key")) {
				webTurnedAway(rw, r, back, "That key is not this magpie web's.")
				return
			}
			signIn(rw, r, back)
```
- trigger: the page port reachable by someone else — `web-page.json` mode `lan` (apppage.go:62-69 binds 0.0.0.0) or any tunnel/forward to it — then repeated `POST /_magpie/sign-in` with `back=/&key=<guess>`. A wrong guess is answered 401 with a distinct body; a right one sets the cookie.
- effect: guesses are unlimited at HTTP speed and per-peer stateless; the key may be as short as 16 characters of `[A-Za-z0-9-._~]` (`webKeyMin = 16`, web.go:149, CheckWebKey 184-193), and a human-chosen MAGPIE_WEB_KEY is the case where 16 characters are guessable. A guessed key is every setting, every provider credential and every gateway key (the page's whole /api, apppage.go:71-73).
- why: the page key is the only credential for the entire admin API, and its check has constant-time comparison but no bucket: the gateway refuses to be an authentication oracle once failures pile up (`tooManyAuthFails`, lan.go:502-509, 429 + Retry-After), while the page port answers every wrong guess forever.
- fix: reuse the gateway's failed-auth bucket keyed by peer address in webGuard (401 → 429 without comparing once over).
- evidence: `rg -n 'tooManyAuthFails|Retry-After|throttle|rate.?limit' internal/gui --glob '!*_test.go'` → no Go matches (only UI strings in assets/app.js, assets/app.css); webGuard body 201-268 has no counter, only `subtle.ConstantTimeCompare`. `rg -n 'webKeyMin' internal/gui/web.go` → `149: const webKeyMin = 16`. verified-by: read, grep
- confidence: medium

### F3 [P3] Allow list and key requirement are skipped on the Claude-helper path while a predecessor is set
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/lan.go:398
- code:
```go
func ownCallback(r *http.Request) bool {
	if r.Method != http.MethodPost || r.URL.RawPath != "" {
		return false
	}
	token, ok := strings.CutPrefix(r.URL.Path, claudeMCPPath)
	if !ok || token == "" {
		return false
	}
```
- trigger: `predecessorCallbacks.sock` is set (gateway handed over to this process during an app restart, handover.go:296-343) and magpie is shared (0.0.0.0). Then from any address: `POST /<claudeMCPPath><any non-empty token>`. `ownCallback` returns true via `if forwarding() { return true }` (lan.go:410-412), which runs before `refuseNotAllowed` (328) and before the `remote || managedKey` identification (351), so neither `MAGPIE_ALLOW_IPS` nor an enabled gateway key is consulted.
- effect: for the length of the handover window, a route family answers remote callers that the allow list excludes, and every such POST is relayed over the predecessor's unix socket (handover.go:308-341). The predecessor still checks its own tokens, so the observable is a token oracle ("unknown or expired Claude run" vs a tool result) reachable from addresses the operator deliberately excluded, plus unauthenticated work done on the predecessor's single-threaded socket.
- why: the bypass exists because the helper's token is unknown to this process, but it is applied before the address check that has nothing to do with tokens, so a leaked key's uselessness outside the allow list is lost on this path.
- fix: run `refuseNotAllowed` (allow list) before returning true for a forwarded callback, keeping only the key requirement skipped.
- evidence: read lan.go 314-363 (order 316 ownCallback → 328 refuseNotAllowed → 337 remoteKeyed → 351 identifyCaller) and handover.go 308-343 (`forwarding()` is `sock != ""`, cleared only when the socket is gone). verified-by: read
- confidence: high on the path, medium on impact

### F4 [P3] "Is this host on my network?" costs a blocking DNS lookup on the request path, and `.local` is trusted without resolving
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/netproxy/lan.go:57
- code:
```go
	if strings.HasSuffix(host, ".local") {
		return true
	}
	lanCache.Lock()
	s, ok := lanCache.m[host]
	lanCache.Unlock()
	if ok && time.Since(s.at) < lanKeep {
		return s.lan
	}
	ctx, cancel := context.WithTimeout(context.Background(), lanWait)
```
- trigger: a request whose content names hosts magpie then fetches — image/video URLs in a caller's body (publicTransport.RoundTrip calls `netproxy.Func(req)` to choose the proxy, image_fetch_guard.go:205) or a provider base URL — with names never seen this minute. Each unseen name reaches `lanLookup` here and waits up to `lanWait` (400 ms, line 30); a `*.local` name takes the early `return true`.
- effect: attacker-chosen image URLs add up to 400 ms of resolve per distinct name on the request's own goroutine, and the cache is cleared wholesale at 256 entries (lines 75-79), so a handful of images flush it and every name after that pays again. The `.local` branch never resolves, so a name under that suffix is dialled **direct** regardless of the user's proxy settings — the user's egress proxy is bypassed and their address exposed to whatever that name resolves to at dial time (a hostile or captive resolver can answer `.local`).
- why: the trust decision "this is the user's own network" is made from the name alone for `.local`, and from a synchronous resolver call inside the request path otherwise, so both the decision and its cost are influenced by data in the request.
- fix: resolve `.local` like any other name and require every answer to be private/link-local; make the lookup non-blocking for the caller (or keep a persistent negative cache) so an unseen name cannot stall a request for 400 ms.
- evidence: read netproxy/lan.go 1-88 in full (`lanWait = 400 * time.Millisecond` at 30, `if len(lanCache.m) >= 256 { clear(lanCache.m) }` at 75-79); `rg -n 'netproxy.Func' internal/gateway/image_fetch_guard.go` → `205: proxy, err := netproxy.Func(req)`; `rg -n 'WithUncheckedTLS|InsecureSkipVerify' internal/netproxy` → only unchecked.go:81-83, opt-in per host (checked separately, not reported). verified-by: read, grep
- confidence: high on the code, medium on severity

## Checked, no defect

- `access.Authenticate` / `MigrateLegacyLANKey` (access.go:492-560): key secrets compared with `subtle.ConstantTimeCompare`, a key with `Off` returns false on match, a removed default key cannot resurrect (`revokedLANPrefix` mirror plus the `LANKeyID` marker), and `List` (access.go:176) strips secrets — the page only gets one from `access.Update` (`gui/caller_keys.go:96-110`, `Cache-Control: no-store`).
- `lanGuard`'s identity rewrite (lan.go:314-363, 493-532): caller keys are read from Authorization/x-api-key/x-goog-api-key/`?key=` in order, a wrong key from another machine is 401 (429 once the failed-auth bucket is over), and on success every one of those headers plus the query is replaced with magpie's own token before any handler or proxy sees it — a client's gateway key cannot reach a vendor or the request archive.
- CORS/rebinding matrix (cors.go:44-179, lan.go:435-477): a listed origin still needs an enabled gateway key for the real request (`keyedCaller`), preflights only echo header names that are token-safe; `Origin: null` and `localhost` subdomains need a key; and a browser reaching loopback under a foreign Host is refused by `rebound`/`ownHost` (own names only: IPs, localhost, container host names, MAGPIE_PUBLIC_URL's host, hostname).
- `image_fetch_guard.publicTransport` (image_fetch_guard.go:1-330): scheme restricted to http(s), every resolved address must be public, the checked address is pinned into the URL, `guardDialer`'s `Control` re-checks the address actually dialled, each redirect re-enters the same transport (≤5), and the fake-IP exemption is refused through a proxy that is not on this computer.
- One listener, one guard (gateway.go:643, oneport.go:15-19, gui/web.go:421-441): the shared listener serves `Front()` = `dumpFirst(lanGuard(Handler()))`, and client routes arriving on `magpie web`'s page port go through the same `Front()` with `FromPagePort`, so a page port behind a raw TCP forward still needs an enabled key while sharing — no non-loopback bind serves an unauthenticated client route.
- `plugin.FileIn`/`inside` (plugin/page.go:221-260) is a correct containment check (absolute paths, backslash, `..`, and an `EvalSymlinks` re-check against the real parent), and plugin page tokens are compared with `hmac.Equal` (gui/pluginpage.go:299-306) — the pattern the library code below is held to.
- Library restores cannot invent entries: `Library.Restore` (setup.go:196-250) only touches names it already has and reports the rest as `Missing`, so a synced/imported setup cannot add a skill or MCP server; the git installer (skillgit.go:100-135, 221-230) refuses a URL password, a host starting with `-`, and a ref/path with `..`, and runs git with `-c protocol.ext.allow=never -c core.symlinks=false` and `--` before the repo.
- Request-archive names (gateway/archive.go:140-152) parse the date strictly and accept only `[0-9a-z-]{1,64}` as the id, so `?date=`/`?id=` cannot escape `archive/` nor inject into the `Content-Disposition` built at gui/archive.go:166; provider icon names are matched against `^[0-9a-f]{16}\.(png|jpg|gif|webp|ico|svg)$` before any join (provider/icon.go:31-46), and `appPage.saveAppPage` pre-creates `web-page.json` 0600 and re-`Chmod`s it (apppage.go:94-108).
