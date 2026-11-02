# verify-plugin-gui: plugin-host, gui-api, gui-assets falsified against /tmp/magpie-audit-Pzn4fNp7/src

Every input finding was opened at its cited line in the release tree, its quote compared with the
source, and its trigger traced through the real callers (grep/sed/read under `src/internal`).
Read-only: nothing in the tree or in the input lane files was modified.

## Verdicts

plugin-host/F1 CONFIRMED - host.js:1345 caps runs at 128 and host.js:1360 returns null; gateway.go:206-207 collapses that null with "no hook", and the request is served unhooked with no log or counter.
plugin-host/F2 CONFIRMED - store.go:138-151 never exits when OpenFile is EACCES (os.IsPermission continues) and ignores the failing os.Remove; SignOut (providers.go:900-905) takes this path when no host runs. A read-only volume is not this case (EROFS hits the early return at 145); a directory the process cannot write is.
plugin-host/F3 CONFIRMED - gatewayHooks (middleware.go:111-129) calls gatewayResponse for every run; host.js:1432 skips onResponse only after Go has spooled the whole reply and, for a stream, gatewayReply has re-parsed it.
plugin-host/F4 CONFIRMED - gateway_hooks.go:27 sends nil/false to Finish whenever capture.full.cut() is true; capture.go:114 is s.kept < s.size and conversations.go:19 sets the cap at 8 MiB.
plugin-host/F5 CONFIRMED - pluginrt.go:252 skips p.state == Failed on every Reconcile; only dispose (pluginrt.go:338) leaves Failed, so a failed Apply needs an off/on toggle or a restart.
gui-api/F1 CONFIRMED - apiFramed (pluginpage.go:356-368) rejects cross-site but not same-site; a page on another port of the host sends the Lax cookie and a text/plain POST, and the LAN handler decodes {On,NewKey} whatever the content type.
gui-api/F2 CONFIRMED - the only Set-Cookie (web.go:208) has no Secure; the cookie is host-scoped and port-independent, so a plain-HTTP service on the same host receives the master key.
gui-api/F3 DOWNGRADED P3 - api.go:2470 really calls provider.Changed for every non-/api GET, but the claimed 20-30 rebuilds per page load is overstated: the page's scripts are fetched before /api/state on a cold load, so the real cost is lazily fetched icons/plugin assets and concurrent reads.
gui-api/F4 CONFIRMED - backup.go:68-72 decodes an unbounded r.Body and base64-decodes it again; the 7 MaxBytesReader uses are per-handler and the servers set only ReadHeaderTimeout (web.go:108, apppage.go:239).
gui-api/F5 CONFIRMED - mcpauth.go:311 discards filelock's held flag; after Wait (10s, filelock.go:20) a second magpie re-spends the rotating refresh token, and markDead (mcpauth.go:372-378) leaves Dead set.
gui-api/F6 CONFIRMED - archive.go:204-211, backup.go:48-56 and usage.go:230-240 stat-then-rename with no O_EXCL; two concurrent exports pick the same name and the second rename replaces the first.
gui-assets/F1 CONFIRMED - load()'s catch (app.js:4215-4216) only toasts; .loading is removed only in renderAgents (app.js:429), and no timer calls load() again, so a failed first state read leaves the skeleton until focus/visibility changes.
gui-assets/F2 CONFIRMED - the stale guard at app.js:4172 (not 4180) applies only while `shown` is set, i.e. the first-load kept list; a later load's `state = next` at 4180 overwrites a write that landed during the read.
gui-assets/F3 CONFIRMED - library/client.js:2114-2126 polls every 800ms and the catch at 2117 never bounds attempts; a persistent 404 (magpie restart) or transport failure never changes mcpSigning.
gui-assets/F4 CONFIRMED - app.js:5976-5981 captures `was` but guards only `kept && providers !== kept`; kept is null on later loads, so a newer write's list is replaced by the older read.
gui-assets/F5 CONFIRMED - app.js:4306 schedules an unowned 700ms chain per caller; load() (4218), the 15-minute interval (24245) and the install answers (4297/4324/4329) each add one during a download.

## Corrected findings

### plugin-host/F1 [P2] Request hooks are silently skipped once 128 runs are open, and refusal looks like success to Go
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/plugin/gateway.go:206
- code:
```go
	err = h.call(start, "gatewayRequest", map[string]any{"operation": "begin", "body": json.RawMessage(body), "context": info}, &out)
	if err != nil || out.ID == "" || out.Token == "" || len(out.Body) == 0 || out.Body[0] != '{' || !json.Valid(out.Body) {
		return nil
	}
	// the run ends when its owner finishes or closes it, never when the
	// client goes: Codex hangs up on response.completed, and the reply it
	// got is still the exchange to observe
	return &GatewayRun{Body: out.Body, h: h, id: out.ID, token: out.Token}
```
- trigger: 129 concurrent agent requests while any installed, enabled plugin has a magpie.hooks gateway entry. host.js:1345 sets GATEWAY_RUN_LIMIT = 128 and host.js:1360 returns null for the 129th begin; Go's out is then all zero values and line 207 returns nil.
- effect: the overflow request is served with the original body and no onResponse call, with no log line in internal/plugin/gateway.go and no per-hook counter (gatewayStates never sees the refusal). A hook that rewrites the prompt, pins a model or scrubs secrets does not run for that request. At 200 in-flight hooked requests, 72 (36%) bypass their hooks.
- why: "no hook", "host down" and "too many runs" are one nil return, so a resource limit at the plugin boundary turns into a silent policy gap instead of a per-request error or a fallback.
- fix: make host.js answer an over-limit begin with a distinguishable error (or a refused flag), and log/count it in GatewayBegin instead of returning nil.
- evidence: read gateway.go:174-213 and host.js:1344-1362; grep -n 'GATEWAY_RUN_LIMIT\|gatewayRuns.size' internal/plugin/host.js -> 1345, 1360; grep -n 'log\.' internal/plugin/gateway.go -> no output. verified-by: read, grep
- confidence: high

### plugin-host/F2 [P2] `lockAuth` spins forever when the lock file cannot be created or removed
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/plugin/store.go:138
- code:
```go
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			return func() { os.Remove(lock) }
		}
		if !os.IsExist(err) && !os.IsPermission(err) {
			return func() {} // no folder to lock in: no file to change either
		}
		if fi, err := os.Stat(lock); err == nil && time.Since(fi.ModTime()) > authLockStale || time.Since(start) > 2*authLockStale {
			os.Remove(lock)
		}
		time.Sleep(5 * time.Millisecond)
	}
```
- trigger: SignOut (providers.go:900-905) with no host running, on a plugins folder the process cannot write (EACCES: a folder left root-owned by an earlier sudo magpie, or a lock the user cannot remove). os.IsPermission(err) is true, so the loop neither returns nor breaks; the os.Remove at 148 is unchecked and keeps failing. A read-only volume (EROFS) is not this trigger: it fails the !IsExist && !IsPermission test and returns at 145.
- effect: the sign-out never returns (no error, no timeout); the GUI request handler or CLI blocks forever while burning ~200 open/stat/remove iterations a second, and the pending sign-in/out work behind it never happens.
- why: the retry loop treats a permanent permission failure as "someone else holds the lock" and has no exit after 2*authLockStale; its final escape hatch (os.Remove) discards its error.
- fix: after time.Since(start) > 2*authLockStale, return an error (make lockAuth return (func(), error)) instead of removing-and-retrying.
- evidence: read store.go:127-153; grep -n 'lockAuth()' internal/plugin -> providers.go:905 and store.go:135; grep -n 'authLockStale' internal/plugin/store.go -> 129, 147. verified-by: read, grep
- confidence: high

### plugin-host/F3 [P2] Every request with any gateway hook spools, parses and re-renders the whole reply, even when no hook has onResponse
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/gateway_hooks.go:15
- code:
```go
	capture := &conversationWriter{captureResponseWriter: &captureResponseWriter{
		ResponseWriter: w, full: &spool{limit: conversationBodyMax, pattern: "magpie-hook-*"},
	}}
```
- trigger: any agent request while a gateway plugin implements onRequest only. host.js:1432 skips onResponse when the function is absent, but Go has no flag for it: gatewayHooks (middleware.go:111-129) calls gatewayResponse whenever GatewayBegin returns a run.
- effect: per request the whole reply is written to a temp spool (up to conversationBodyMax = 8 MiB), read back whole and sent to the host, which discards it when no hook has onResponse. A streamed reply is also re-parsed by gatewayReply (readSSE + decode + finish + render); a whole-JSON reply is only validated. The reply is spooled a second time only when GatewayConversations is on and a session id is named (conversations.go:61-70); the first spool and the send happen regardless.
- why: the capture path is gated on "some plugin has a magpie.hooks file" instead of "some loaded hook has a response hook", so the expensive half of the exchange runs for a child that cannot use it.
- fix: have the begin answer report whether any participating hook has onResponse, and skip gatewayResponse/gatewayReply/Finish when it does not.
- evidence: read gateway_hooks.go:15-36 and middleware.go:106-129; grep -n 'onResponse' internal/plugin/host.js -> 1313, 1432, 1439, none in Go; grep -n 'conversationBodyMax' internal/gateway/*.go -> conversations.go:19. verified-by: read, grep
- confidence: high

### plugin-host/F4 [P2] A reply over 8 MiB is reported to the hooks as a failed exchange with no body
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/gateway_hooks.go:27
- code:
```go
		output, err := capture.full.read()
		// a client that hangs up once it has the whole reply (Codex, on
		// response.completed) still got it: gatewayReply judges the bytes
		if err != nil || capture.full.cut() || capture.status >= 400 {
			run.Finish(r.Context(), nil, false, body)
			return
		}
```
- trigger: a successful streamed reply larger than conversationBodyMax (8 MiB) with a gateway onResponse hook installed, or a spool that cannot create its temp file. capture.go:114 cut() is s.kept < s.size, so any reply past the cap takes this branch.
- effect: onResponse receives reply: null and success: false for a request the client received in full and without error. Anything the hook does with success (logging, ledger, retry hints, usage accounting, "did the turn work") sees a failure that never happened; the reply is dropped silently.
- why: "the body was truncated for storage" is conflated with "the exchange failed", so a storage cap decides the plugin's correctness verdict.
- fix: compute success from status and the bytes kept independently of cut(), and pass truncation as its own flag.
- evidence: read gateway_hooks.go:24-35 and capture.go:100-116; grep -n 'conversationBodyMax =' internal/gateway/conversations.go -> 19:const conversationBodyMax = 8 << 20. verified-by: read, grep
- confidence: high

### gui-api/F1 [P2] The API's only CSRF guard rejects cross-site, not same-site: any page on another port of the same host can POST to the API
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/pluginpage.go:356
- code:
```go
func apiFramed(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		return false
	}
	if r.Header.Get("Origin") == "null" {
		return true
	}
	switch r.Header.Get("Sec-Fetch-Dest") {
	case "iframe", "frame", "embed", "object":
		return true
	}
	return r.Header.Get("Sec-Fetch-Site") == "cross-site" && r.Header.Get("Sec-Fetch-Dest") != "document"
}
```
- trigger: magpie web (or the app's Browser-access page) is on, the browser holds the sign-in cookie, and a page served by another service on the same host, another port and the same scheme, runs fetch('http://host:3430/api/settings/lan',{method:'POST',credentials:'include',headers:{'Content-Type':'text/plain'},body:'{"On":true,"NewKey":true}'}). Cookies ignore ports and SameSite=Lax treats another port as same-site; text/plain makes it a CORS simple request (no preflight), and the handler decodes it whatever the Content-Type says.
- effect: that page changes magpie's state with no key of its own: turn LAN sharing on (the gateway becomes reachable from the network), rotate the page key (the user is signed out), rewrite settings. Responses cannot be read without CORS headers, so it is blind writes, which is enough for the routes that matter.
- why: webGuard's cookie and apiFramed's header test both assume "another page" means another site; the browser's boundary for a host-scoped cookie is origin (scheme/host/port), and Sec-Fetch-Site: same-site passes here untouched.
- fix: in apiFramed, also return true when r.Method != http.MethodGet && r.Header.Get("Sec-Fetch-Site") == "same-site".
- evidence: read pluginpage.go:349-369 and api.go:2100-2125 (POST /api/settings/lan decodes struct{ On, NewKey bool }); grep -rn 'same-site\|"same-origin"' internal/gui/*.go -> test files and a comment only; grep -rn 'apiFramed' internal/gui/*.go -> pluginpage.go:356 and api.go's noFramedAPI. verified-by: read, grep
- confidence: high

### gui-api/F2 [P2] The page's sign-in cookie is not Secure, so any plain-HTTP service on the same host receives the master key
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/web.go:208
- code:
```go
		// Priority=High (Chrome's): the page's host is often an address
		// other services share on other ports, and a browser keeps a
		// limited number of cookies for one host, all ports together;
		// past it Chrome drops the low and medium ones first (#2033)
		rw.Header().Add("Set-Cookie", (&http.Cookie{Name: cookie, Value: key, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: int(keep / time.Second)}).String()+"; Priority=High")
```
- trigger: magpie web --lan --tls-cert ... --tls-key ... (StartWeb wraps the listener with gateway.ServeTLS, web.go:89), the user signs in over HTTPS, and a plain-HTTP service is reachable on the same host at another port. The browser sends the host-scoped cookie (no Secure, Path=/) to that service too.
- effect: that service receives magpie_web_3430=<the key> in cleartext, and with it can open the page and read every provider key and account, or change any setting. The same holds for an operator's access log or a proxy in front of a plain-HTTP service on that host.
- why: a cookie that is the whole credential for an HTTPS-administered page must be Secure whenever the page was served over TLS; the attribute is absent from the only place cookies are minted.
- fix: set Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") on that cookie.
- evidence: read web.go:196-260; grep -n 'Secure' internal/gui/web.go -> no match; grep -n 'Set-Cookie' internal/gui/web.go -> 208 only. verified-by: read, grep
- confidence: high

### gui-api/F4 [P2] 98 /api handlers decode r.Body with no size limit; /api/backup/import then base64-decodes it into a second copy
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
- trigger: POST /api/backup/import with a body of a few hundred MiB to 1 GiB, or a 100 MB POST /api/settings that decodes into a json.RawMessage and then json.Unmarshal's it again (api.go:1241-1249). No server-wide body cap exists: the http.Servers built at web.go:106-108 and apppage.go:237-239 set only ReadHeaderTimeout.
- effect: a 1 GiB body costs at least 1 GiB for the JSON string and ~750 MiB for the decoded bytes, plus the copies inside backup.Open/Restore, enough to OOM a 2 GiB container or swap a desktop to a standstill. The only gate in front of these handlers is the page key (or, in the app, the frame check), so body size is caller-chosen.
- why: request bodies are trusted to be small, but the limit was applied per handler (7 MaxBytesReader uses) rather than at the mux boundary; 98 decoders have none.
- fix: wrap the mux once with http.MaxBytesReader (or MaxBytesHandler), with a per-route ceiling for /api/backup/import.
- evidence: grep -o 'json\.NewDecoder(r\.Body)' internal/gui/*.go | wc -l -> 98; grep -n 'MaxBytesReader' internal/gui/*.go | wc -l -> 7; grep -n 'MaxBytesReader\|LimitReader' internal/gui/backup.go -> no match; grep -n 'ReadHeaderTimeout' internal/gui/web.go internal/gui/apppage.go -> web.go:108, apppage.go:239. verified-by: grep, read
- confidence: high

### gui-api/F5 [P2] MCP token renewal ignores whether the cross-process lock was taken, so two magpies can spend one rotating refresh token
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
- trigger: the app and magpie serve (or two magpie versions during a handover) both hold the same mcp-signins.json and both need to renew the same server's token while its endpoint is slow: process A takes the file lock and spends >10s in tokenRequest; B waits out filelock.Wait (10s, filelock.go:20/55), logs "another magpie held it ... going on without it" and returns held=false. B ignores that (_), re-reads the still-old record and spends the same refresh token.
- effect: the rotating server refuses the second spend (invalid_grant) and renew treats that as death: markDead(name, r.Access) sets Dead on the record whose Access is unchanged. When B's markDead lands before A's update (likely, since A is still in its >10s request), A's later update passes the cur.Access check and stores the fresh tokens but does not clear Dead, so every Token call returns ErrExpired until the user signs in again. If A's update lands first, B's markDead no longer matches and the record survives; the bad ordering is the slow-endpoint case above.
- why: the invariant "a rotating refresh token is spent once" needs the lock actually held; filelock's timeout return is a soft failure meant for callers that can proceed safely, and this caller cannot.
- fix: check held and return a retryable error instead of renewing without the lock: if unlock, held := filelock.Lock(...); !held { return "", errBusy }.
- evidence: read mcpauth.go:305-360 and 179-193 (update keeps Dead); read filelock.go:15-60; grep -n 'renew.lock' internal/mcpauth/mcpauth.go -> 311; grep -n 'func markDead' -A8 internal/mcpauth/mcpauth.go -> Dead=true only, Access untouched. verified-by: read, grep
- confidence: medium

### gui-assets/F1 [P2] A failed first /api/state read leaves the Agents page on its skeleton forever
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/assets/app.js:4214
- code:
```js
    if (view === "settings" && !syncOpen) await loadSettings();
  } catch (e) {
    status(e.message, "err");
  }
  renderUpdateBadge();
```
- trigger: first load (load.done false) where api("state") fails (a 500 from an unreadable config, or a proxy's 502/504/524). If api("state?last=1") also returns null (no kept state), shown stays unset and no renderAgents happens before the catch. renderAgentsLoading() at 4140 has already put 5 skeleton rows and aria-busy=true on #view-agents.
- effect: the default landing page shows 5 shimmering skeleton rows indefinitely; the only feedback is a toast that status() clears after 8 s (382). No error text and no retry control. Recovery requires the user to focus/alt-tab the window (loadBack at 24185-24194, the only other load() caller apart from a privacy-model catch at 22004), and load.done stays false, so every later attempt also never draws the kept-state fallback.
- why: the failure path only toasts; the .loading class and skeleton children are removed exclusively inside renderAgents() (429), which needs a state object that never arrived, and no retry timer or error renderer exists for this view.
- fix: in the catch, when !state, replace the skeleton with an error row whose "Try again" button calls load(), instead of leaving .loading set.
- evidence: read app.js:401-420, 4138-4218, 24183-24194; grep -n 'classList.remove("loading")' internal/gui/assets/app.js -> 429 (and 6132 for providers only); grep -n 'setInterval(' internal/gui/assets/app.js -> no load() retry. verified-by: read, grep
- confidence: high

### gui-assets/F2 [P2] A slow state read overwrites a write that landed while it was in flight
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/assets/app.js:4172
- code:
```js
    if (shown && state !== shown) next = state;
    // a setting changed while this was on its way (it can take seconds):
    // what came back is from before it, and would put the old theme back
    if (!prefsSettled(since) && load.done) next.settings = state.settings;
```
- trigger: window regains focus (app.js:24194 visibilitychange -> loadBack -> load()), which asks api("state"); the code's own comment says this read can take seconds. The reader then clicks an agent's Connect switch before it lands: app.js:1159-1166 sets state = await api("agents/connect/" + a.id, {}) and renders; the older read then resolves and executes state = next at 4180 with the pre-connect snapshot. Same for disconnect (2265) and Keep (2297).
- effect: a connect/disconnect the server accepted is drawn as not done: the switch flips back and the row's "Connected" line disappears, contradicting the success toast shown by connectedSaid a moment earlier, until the next load/read. In the reverse case a removed account is drawn again as present.
- why: the "a write's answer stands" invariant is implemented only for the first load, because shown is non-null only when the kept state?last=1 list was drawn (4157-4161); prefsSettled/agentsVisibilityRevision cover settings and hide/order only, never agents/providers/accounts.
- fix: capture the state identity before the await (const was = state) and, when state !== was, keep the newer state instead of next (the same guard shown provides on the first load).
- evidence: read app.js:4155-4182, 1147-1172, 2258-2270, 2297; grep -n 'if (shown && state !== shown)' internal/gui/assets/app.js -> 4172; grep -n 'state = await api("agents/connect' -> 1165 and 1759. verified-by: read, grep
- confidence: medium

### gui-assets/F3 [P2] MCP sign-in poll never ends when its endpoint keeps failing, and the failure is swallowed
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/assets/plugins/library/client.js:2117
- code:
```js
    async function follow(id) {
      while (mcpSigning?.id === id) {
        await new Promise((r) => setTimeout(r, 800));
        let st;
        try { st = await api("library/mcp-signin/" + id); } catch { continue; }
        if (mcpSigning?.id !== id) return;
        if (st.state === "waiting") continue;
```
- trigger: start a Library server's sign-in (POST /api/library/mcp-signin), then make GET /api/library/mcp-signin/<id> fail persistently: magpie restarts while the browser flow is open, so the id is unknown and the endpoint answers 404 (library.go:358-363 http.NotFound), or the connection is refused until the server is back. api() throws on both, and catch { continue; } re-enters the loop with the same id; nothing else changes mcpSigning.
- effect: an 800 ms request loop that never terminates (1.25 req/s for the life of the page), while the row keeps saying "Finish signing in in your browser..." and the real error is never shown. The reader has to cancel by hand; if the editor was closed or the row was redrawn out of sight, the loop is invisible.
- why: the only exit conditions are a state transition or the user changing mcpSigning; a transport/HTTP error is treated as "keep waiting" instead of a terminal failure, so an unrecoverable poll looks like progress.
- fix: count consecutive errors and, after a few, set mcpSigning = { name, state: "failed", error: e.message }; draw(); return;.
- evidence: read library/client.js:2045-2130 and library.go:358-364; grep -n -F 'catch { continue; }' internal/gui/assets/plugins/library/client.js -> 2117 (only occurrence); grep -n 'while (mcpSigning?.id === id)' -> 2114. verified-by: read, grep
- confidence: high

### gui-assets/F4 [P2] loadProvidersNow lets an older providers read overwrite a newer write
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/assets/app.js:5976
- code:
```js
  const was = providers;
  const next = await fresh;
  // a change saved meanwhile drew the list it was answered with, newer
  // than this read
  if (kept && providers !== kept) return;
  providers = next;
```
- trigger: on the Providers page, a reload is in flight (window focus -> app.js:4205 await loadProviders(), or show("providers") at 23303) and the reader flips a provider's on/off switch before it lands: app.js:6419 providers = await api("provider/" + (on ? "on" : "off"), { id: p.id }) assigns the post-write list and renders; the older read then resolves and assigns providers = next at 5981 with the pre-toggle list. Overlap is possible because loadProviders() de-duplicates only while providers is null or .last (5929-5937), so a later load starts a second concurrent read.
- effect: the switch the reader just flipped flips back (or an account just removed reappears) even though the server accepted the change, until the next read. It looks like the toggle failed and invites a second click.
- why: the stale-check if (kept && providers !== kept) only exists for the first-load kept list (kept is null on every later load); was is captured at 5976 but never compared, so the write's newer answer is discarded.
- fix: if (providers !== was) return; immediately before providers = next; (mirror of the kept guard).
- evidence: read app.js:5925-6024 and 6405-6425; grep -n 'providers = next' internal/gui/assets/app.js -> 5981 and 6393; grep -n 'const was = providers' -> 5976. verified-by: read, grep
- confidence: medium

### gui-api/F3 [P3] Every static asset request drops the shared catalog and file memo, so a read after it rebuilds them
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/api.go:2470
- code:
```go
		if !(api && r.Method == http.MethodGet) {
			// anything that may have written drops what was held, once it
			// has answered with it
			defer provider.Changed()
		}
```
- trigger: any GET of a non-/api path: /app.js, /styles, /icons/*, /boot.js, a plugin page's files. held wraps the whole mux (api.go:2426) and the static handler is mux.Handle("/", ...) at api.go:851, so api is false and the condition is true.
- effect: each asset finish calls provider.Changed() -> filememo.Forget() + dropHeld() (provider/hold.go:140-157), clearing the built catalog and every memoised file. A concurrent or following /api read rebuilds them (heldOf/hold.go:187-203 refuses to cache a build whose gen moved). The page's own scripts are fetched before /api/state on a cold load, so the per-load rebuild count is not the claimed 20-30; lazily fetched icons/plugin assets and concurrent reads still pay it.
- why: the guard tests "this request may have written" as !(api && GET), which classifies every non-API GET, all immutable embedded files, as a write, defeating the memo held exists to provide.
- fix: if api && r.Method != http.MethodGet { defer provider.Changed() }.
- evidence: read api.go:2436-2475 and provider/hold.go:100-203; grep -n 'func Changed' -A3 internal/provider/hold.go -> filememo.Forget(); dropHeld(); grep -n 'noFramedAPI(held(mux))' internal/gui/api.go -> 2426; grep -n 'mux.Handle("/"' internal/gui/api.go -> 851. verified-by: read, grep
- confidence: high

### gui-api/F6 [P3] Export files are named by check-then-write, so two exports at once silently overwrite one another
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
		os.Remove(tmp.Name())
		return "", err
	}
```
- trigger: two exports of the same artifact in flight at once, e.g. POST /api/backup/export from the panel and the main window, or the page retrying a slow request, with the same day/stem. Both os.Stat calls see no file, both pick the same name, and os.Rename (backup.go:53 uses edit.WriteAtomic's rename) replaces it. The same shape is at backup.go:48-56 and usage.go:230-240.
- effect: the first file is replaced by the second, and both callers answer with the same path, so the user believes they have two dated artifacts and has one. For /api/backup/export the survivor contains every provider key, so a backup the user thought existed does not.
- why: "never over an earlier one" needs the name to be reserved atomically with creation (O_EXCL/linkat), not decided by a stat a concurrent writer can race.
- fix: create the chosen name with os.OpenFile(name, O_CREATE|O_EXCL|O_WRONLY, 0600) in the loop and rename over only the file it just created.
- evidence: read archive.go:191-213, backup.go:20-58 and usage.go:224-241; grep -n 'O_EXCL' internal/gui/backup.go internal/gui/archive.go internal/gui/usage.go -> no match. verified-by: read, grep
- confidence: high

### plugin-host/F5 [P3] A plugin that fails Apply while wanted is never applied again, even after its config changes
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/pluginrt/pluginrt.go:252
- code:
```go
		for _, id := range ids {
			p := rt.plugins[id]
			if !p.wanted || p.state == Active || p.state == Failed {
				continue
			}
```
- trigger: a built-in plugin whose Apply returns an error or panics once, then the user corrects the setting with magpie plugin set-config <id> ... or edits plugins.yml; KeepFresh/Refresh runs Reconcile again with the same wanted=true (config.go:363-395).
- effect: the plugin stays Failed (and every point it serves stays unserved) for the life of the process; the window shows the original error while the fixed config sits unused. Only turning the plugin off and on (Set twice) or restarting magpie recovers it: dispose sets Disposed (pluginrt.go:338) and the next on applies it.
- why: Failed is used both as "not applicable" (an API mismatch, permanent) and "failed to start" (retryable on a config change), so Reconcile cannot tell the two apart.
- fix: skip only the permanent case (!Compatible) and let a Failed plugin be re-applied when its composed config changed.
- evidence: read pluginrt.go:206-340 and config.go:334-395; grep -n 'p.state == Failed' internal/pluginrt/pluginrt.go -> 252 (the only occurrence); grep -n 'p.state, p.why, p.ctx = Failed' -> 324. verified-by: read, grep
- confidence: medium

### gui-assets/F5 [P3] Every caller of renderUpdateBadge starts its own 700 ms poll chain during a download
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gui/assets/app.js:4306
- code:
```js
  if (pulling) {
    if (u.state === "ready") {
      delete b.dataset.pulling;
      return restart();
    }
    label.textContent = downloadingText(u);
    setTimeout(renderUpdateBadge, 700);
    return;
```
- trigger: an update is downloading (b.dataset.pulling set, state checking/downloading), and the reader focuses the window a few times: every load() ends with renderUpdateBadge() (4218), the header also polls every 15 min (24245), and each api("update/install") answer calls it again (4297, 4324, 4329, 4384). Each invocation sees pulling and schedules its own 700 ms continuation; the timer handle is never stored, so chains are never cleared or merged.
- effect: N parallel GET /api/update loops at ~1.4 req/s each; five focus events during one download give ~7 req/s of polling plus the backend's update-state work, and they all persist until the download leaves the pulling states. When the state becomes ready, every chain that computed pulling before the first deletes b.dataset.pulling also calls restart(), fanning out the install request.
- why: the recursive poll has no single owner; unlike providersWhileFetching.timer (6005) or b.waitPoll (4328), the handle is not kept and not cleared before scheduling.
- fix: keep one handle: clearTimeout(renderUpdateBadge.t); renderUpdateBadge.t = setTimeout(renderUpdateBadge, 700);.
- evidence: read app.js:4267-4310; grep -n 'setTimeout(renderUpdateBadge' internal/gui/assets/app.js -> 4306 only; grep -n 'renderUpdateBadge' internal/gui/assets/app.js -> 4218, 4267, 4297, 4306, 4324, 4329, 4384, 24245. verified-by: read, grep
- confidence: high

## Added findings

none

## VERDICT

VERDICT: CONFIRMED 15, DOWNGRADED 1, REJECTED 0, ADDED 0; reviewed 28 files
