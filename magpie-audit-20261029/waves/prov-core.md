# Provider core: accounts, keys, refresh, quota, routing

Release snapshot audited: /tmp/magpie-audit-Pzn4fNp7/src (read-only). No file in the snapshot was modified; no test or build was run.

## Coverage

- internal/provider/provider.go: 500-599 (Path, load, ErrUnreadable, unreadableError, read, validProviders, FileError, Routings), 596-725 (store, All, clone, allProviders, ownOf), 845-1055 (Save, sameStored, delistedWith, listedWith, Add, add); plus a scripted diff of `type Provider struct` fields against the account-branch field literal in Save.
- internal/provider/group.go: 100-215 (routes, IsOff, Live, Within, pools, Patience constants, Waits) and 153-313 (Group/Member structs).
- internal/provider/account.go: 930-1220 (readJSON, jwtClaims, claimString, codexAuth, codexAccount, codexToken, codexTokenRenewing, codexAccountID, codexRefresh, copilotSession, copilotMu).
- internal/provider/balance.go: 700-810 (balanceShare, balanceAmount, balanceExpr, balanceFuncs, balanceExtreme), 1196-1376 (Balance, balanceRead, balanceAccountWide, openCodeZen).
- internal/provider/keys.go: 40-155 (KeyInfo, KeyRest, RestKey, RestKeys, severalOn, keyID, KeyList, KeysOn, AddKey), 240-330 (AddKeys tail, keyReadsAtOnce, keySlots.take, first, setFirst), 430-644 (findKey, KeyOf, UseKey, promote, SetKeyOn, RemoveKey, RemoveKeys, RenameKey, ReplaceKey, moveKeyed).
- internal/provider/logins.go: 178-333 (loginsPath, parseLogins, validLogins, readLogins, unreadBytes, writeLogins, sortLogins, writePrivate, upsertLogin).
- internal/provider/signin.go: 420-585 (callbackBlocked, listenCodexCallback, SignInStatus, CancelSignIn, SubmitSignInCallback, pastedCallback, ownCallback, pastedReply).
- internal/provider/routing.go: 200-660 (Allowance.OnCredits, Restarts, heldUp, restarted, Limit.applies, For, Fullest, Count, Renewal, Left, Lapse, Full, Pooled, Pace, Allowances), 660-925 (UsageRead, ReadUsageNow, OnRenewed, renewedNow, tellRenewed, AwaitAllowance, forgetAllowance, StaleAllowance, restartedBy, routedAllowance, allowanceOf).
- internal/provider/decide.go: 600-770 (IsDecider, RouteDecider, decideAmbiguous, decideRouteError, decideBadRequest, decideNotFound, DecideRouteStatus, resolveDecideModel, deciderByID).
- internal/provider/planquota.go: 100-140 (readZhipuPlan), 240-330 (readMiniMaxPlan, at()), 494-700 (planWindows, planBody, zhipuKeyRefusal, zhipuRefused, vendorSaid, planKeyWindows, planQuotaCache, PlanQuotas head).
- internal/provider/subscription_usage.go: 228-360 (subscriptionUsageCache, OnSubscriptionUsage, toldUsage, subscriptionTimeout, SubscriptionUsage, SubscriptionUsageReading), 395-570 (fetchSubscriptionUsage, withUser, perLogin), 776-886 (claudeWindows, claudeAskFloor, claudeUsageWait), 1095-1215 (codexWindow.window, quotaDurationName, copilotSubscriptionUsage).
- internal/provider/hold.go: 30-347 (stampOf, heldFreshLocked, environSum, init, Hold, Changed, dropHeld, StampNow, heldOf, heldEntries, keptOf, keptFit, holdingNow, heldPlugins, heldFrom, heldSettings).
- internal/provider/window_holds.go: 1-144 (HoldsFloor, Bounds, Counts, PoolWindow, LineName, CountsIn, PoolMembers).
- internal/provider/quota_refresh.go: 1-205 (cardRefresh, quotaReading, keepReading, refreshing, wantsCard, refreshLogins, forgetLoginReading, RefreshUsage, RefreshReport, refreshCards, cacheCards, mergeCards).
- internal/provider/quota_last.go: 1-213 (passing, keyTag, keepLast, readDeferred, load, reading, reported, lastAllowances).
- internal/provider/quota_alert.go: 1-297 (dueAlerts, dueReminders, sameRun, BalanceNumber, alertKeep, readAlertMarks, checkAlerts, WatchQuotas).
- internal/provider/quota_asked.go: 1-104 (Asked, wasAsked, readsAsked, heldRead, holdUnasked, staleReads, markStaleRead, takeStaleRead, isStaleRead, holding, unheld).
- internal/provider/quota_history.go: 130-150 (windowSpan).
- internal/provider/remote_quotas.go: 20-216 (CachedCards, ReadAllCards, remoteCardCache, remoteCardsID, remoteMagpies, RemoteCards, remoteCards, remoteName, remoteOf), 275-415 (refreshRemote, keepRemoteCards, askRemote, remoteJSON).
- internal/provider/claude_limits.go: 1-125 (claudeKinds, claudeHeard, NoteClaudeLimits, KeptClaudeLimits).
- internal/provider/codex_limits.go: 1-114 (CodexLimit, NoteCodexLimits, heardWindows, heardAllowance).
- internal/provider/remote_key_limits.go: 100-274 (remoteKeys, remoteKeyCards, remoteKeyCardOf, keyLimitCard, keyWindowName, askRemoteKeyLimit).
- internal/provider/sub2api_usage.go: 33-61 (readSub2APIKeyLimits), to cross-check the unit of the windows Balance formats.
- Outside this lane, read only to establish reachability: internal/gui/providers.go:1160-1175, 1485-1535, 1690-1700; internal/gui/assets/app.js:9560-9585, 9915-9927; internal/gateway/draw.go:929-932; internal/gateway/routing.go:1066-1072; internal/gateway/claude_subscription.go:2415-2432, 2666-2700; internal/steady/steady.go:97-122; internal/provider/autolist.go:196-252, 294-330.

COVERAGE: read 24 files, ~5300 non-test LOC of the 17317 in scope; skipped 1 (internal/provider/quotas.go, 267 LOC of card assembly) plus the unread remainder of provider.go/group.go/balance.go/logins.go, for budget.

## Findings

### F1 [P2] providers.json is read-modify-written with no lock: two writers silently drop one's edit
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/provider.go:918
- code:
```go
	f, err := read()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(f.Providers, func(q Provider) bool { return q.ID == p.ID })
```
- trigger: two providers.json mutations overlapping inside the same `magpie serve` process — e.g. pressing Save on provider A in the GUI (Save: read at :918, store at :950) while the once-a-minute list refresh stores provider B's new list (autolist.go takeMatching: read at :229, store at :247), or two GUI tabs saving at once. Same for every group/key mutator, and for the CLI running beside the gateway (no file lock on providers.json at all).
- effect: the later `store` writes the whole snapshot it read before the other's change, so the earlier edit disappears with no error shown: an added provider/group, a key turned on or replaced, a model pick. Lost work, not corruption of the rest of the file.
- why: `store` is atomic per write (temp file + rename) but the read→edit→write sequence it belongs to has no mutual exclusion, in-process or cross-process; atomicity of the write does not give serializability of the edit.
- fix: wrap every read-modify-write of providers.json in one process mutex plus a `filelock.Mutex` on Path()+".lock", as access.go already does for access.json.
- evidence: `rg -n 'store\(f\)' internal/provider/*.go | grep -v _test | wc -l` -> 44 write sites; `rg -c 'sync\.Mutex|filelock' internal/provider/provider.go internal/steady/steady.go` -> no match (exit 1), so nothing serializes them. verified-by: grep
- confidence: high (mechanism), medium (how often a user hits it)

### F2 [P3] One goroutine and one vendor request per saved account, with no concurrency bound
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/subscription_usage.go:530
- code:
```go
	out := make([]SubscriptionQuota, len(fetches))
	var wg sync.WaitGroup
	for i, f := range fetches {
		wg.Add(1)
		go func() { defer wg.Done(); out[i] = f() }()
	}
	wg.Wait()
```
- trigger: a user with several saved accounts of one agent (Codex, Copilot, Claude, Gemini) opening the Usage page, pressing Refresh, or any routing read that reaches Allowances: `perLogin` adds one fetch per login and `fetchSubscriptionUsage` starts them all at once.
- effect: N simultaneous HTTPS requests to the vendor's usage endpoint inside one 20s window instead of 8 at a time — the endpoint the code itself notes is aggressively rate limited (subscription_usage.go:271-275). With 20 saved accounts that is 20 parallel requests per refresh, whose 429s then leave every card showing its error.
- why: key reads are bounded (keyReadsAtOnce = 8) for exactly this reason, but account reads are not; the fan-out is bounded only by the number of saved accounts, which is user-controlled.
- fix: take a `keySlots` slot per fetch around `f()` in the loop, as KeyBalances and PlanQuotas already do.
- evidence: `rg -n 'var slots keySlots' internal/provider/*.go` -> only planquota.go:691, remote_key_limits.go:127, balance.go:1522 (none in the account path); `rg -n 'for i, f := range fetches' -A2 internal/provider/subscription_usage.go` -> one bare goroutine per fetch. verified-by: grep
- confidence: high

### F3 [P3] usage-alerts.json is overwritten in place and its parse errors ignored, so a torn write re-sends every alert
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/quota_alert.go:256
- code:
```go
	marks := readAlertMarks(path)
	before, _ := json.Marshal(marks)
	out := dueAlerts(qs, marks, pct, bal, now)
	out = append(out, dueReminders(qs, marks, lead, now)...)
	if after, _ := json.Marshal(marks); string(after) != string(before) {
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, append(after, '\n'), 0o600)
	}
```
- trigger: the file is written with os.WriteFile while the gateway's WatchQuotas timer and a `magpie quota` CLI run both hold copies of the marks (quotaAlertMu is in-process only), or the machine goes down mid-write; then `readAlertMarks` unmarshals a truncated file and ignores the error, leaving an empty map.
- effect: every window and balance is announced again — duplicate notifications the "each is said once" design exists to prevent — and the other process's marks are dropped on its next write.
- why: the one file that keeps "already told" state is the only account-state file in this package written without the temp-file rename (steady.WriteFile/writeFileAtomic) and without an unmarshal check, so partial content becomes "nothing was told yet".
- fix: write it through `writeFileAtomic` (quota_last.go:95) and keep the previous map when the unmarshal fails.
- evidence: `rg -n 'os.WriteFile\(path' internal/provider/quota_alert.go` -> 256; `rg -n 'writeFileAtomic|writePrivate' internal/provider/quota_last.go internal/provider/quota_alert.go` -> quota_last.go:95 only. verified-by: grep
- confidence: high

### F4 [P3] A signed-in account can never keep "don't rest me on a 429": Save's account branch drops limitRestOff
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/provider.go:891
- code:
```go
		key := ""
		if p.ID == "kiro" {
			key = p.Key
		}
		p = Provider{ID: p.ID, Key: key, Models: p.Models, NoModels: p.NoModels, PickedFrom: p.PickedFrom, Delisted: p.Delisted, Unlisted: p.Unlisted, Off: p.Off, Fallback: p.Fallback, Routing: p.Routing, Sink: p.Sink, Affinity: p.Affinity, KeepLogin: p.KeepLogin, KeepLoginAs: p.KeepLoginAs, SwitchOnly: p.SwitchOnly, Contexts: p.Contexts, Family: p.Family, Proxy: p.Proxy, AccountProxies: p.AccountProxies, AccountModels: p.AccountModels, AccountCaps: p.AccountCaps, AccountWindowCaps: p.AccountWindowCaps, CapStop: p.CapStop, CapCareful: p.CapCareful, AccountNames: p.AccountNames, MaxConcurrency: p.MaxConcurrency, AccountConcurrency: p.AccountConcurrency, QueueLimit: p.QueueLimit, QueueWait: p.QueueWait, MaxRPM: p.MaxRPM, PriceRate: p.PriceRate, QuotaRest: p.QuotaRest, QuotaResetAt: p.QuotaResetAt, RestCap: p.RestCap, PinUpstream: p.PinUpstream, NativeCompaction: p.NativeCompaction, PromptAsWritten: p.PromptAsWritten, Hidden: hiddenAccount(p.ID), Quiet: quietAccount(p.ID), Tucked: tuckedAccount(p.ID)}
```
- trigger: setting "Set it aside when out of quota or rate limited" off for a subscription account (codex, claude, copilot, cursor...), or POSTing `{"id":"codex","limitRestOff":true}` to /api/provider/save: the field is not in the field list Save rebuilds the account provider from, so it is discarded while the call returns success.
- effect: the setting the gateway honours for accounts (`c.p.Account == nil || c.p.LimitRestOff`, gateway/draw.go:931) is false forever for every signed-in account, and the account is rested on the first quota refusal or 429 with no way to turn that off from the GUI either, since the editor only sends the field for custom providers (app.js:9926).
- why: a whitelist rebuild of Provider silently accepts and drops any field added later; an input the API accepts is discarded rather than refused.
- fix: add `LimitRestOff: p.LimitRestOff` to the literal.
- evidence: scripted diff of `type Provider struct` fields vs the literal at :891 -> "missing from whitelist: [... 'LimitRestOff' ...]"; `rg -n 'body.limitRestOff' internal/gui/assets/app.js` -> `9926: if (custom) body.limitRestOff = ...`; `sed -n '929,932p' internal/gateway/draw.go` -> account rest skipped when LimitRestOff. verified-by: grep
- confidence: high

### F5 [P3] A manual refresh can never drop the card of a removed account
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/quota_refresh.go:159
- code:
```go
		noteDailyCredits(out, now)
		noteQuotaHistory(out, now)
		c := &subscriptionUsageCache
		c.Lock()
		c.data = cacheCards(c.data, out, seq, true)
		c.Unlock()
```
- trigger: press Refresh on the Usage page (RefreshUsage -> refreshCards) after signing an account out or removing it, or after a provider is deleted while its cached card is the older one.
- effect: the removed account's card survives the refresh that was supposed to replace all cards, and is served on to other magpies by CachedCards, until the next full timed read replaces the cache (the cache's own one-minute freshness window bounds it to about a minute).
- why: `fetchSubscriptionUsage` reads every card here, but the result is committed with partial = true, whose contract is "keep the rest as they were" — the branch that drops cards no longer present is never taken on this path.
- fix: pass `partial` as false when `refreshing(ctx)` reported no single-card refresh.
- evidence: `rg -n 'cacheCards\(c\.data, out, seq' internal/provider/quota_refresh.go internal/provider/subscription_usage.go` -> quota_refresh.go:159 `true` (whole page) vs subscription_usage.go:307 `false` (timed read). verified-by: grep
- confidence: high

## Checked, no defect

- keys.go:285 keySlots.take: the slot is released by `defer slots.take(ctx, p.ID)()` (evaluated at defer time, so the slot is taken immediately) and the ctx-done path returns a no-op release only when no slot was taken; the map is keyed by provider id, so it cannot grow with requests.
- account.go:1063-1160 codexTokenRenewing/codexRefresh: refresh runs under a cross-process file lock and under `context.WithoutCancel` + 30s, so a cancelled request cannot drop a rotated refresh token; `exp` is read as Unix seconds and compared with a 5-minute lead (no ms/s mix-up); the new sign-in is written by `st.write` (temp + rename) and a failed write is logged without failing the request that already holds a usable token.
- hold.go:113-278 Hold/heldOf/keptFit: no lock is held across `build()` — released before building and retaken to store behind a generation check; the release is `sync.OnceFunc`, so a double release cannot under-count holds.
- subscription_usage.go:270-330: the `pending` channel is created, cleared and closed under the mutex, waiters select on it with ctx, and a stale copy is returned while a read is in flight, so a slow vendor cannot block the request path.
- routing.go:290-560 For/Fullest/Count/Renewal/Pace: a window whose reset has passed is zeroed only in the loop copy, so ranking never mutates the shared Allowance; hours-until-reset is floored at 1h, so no division by zero produces an Inf pace.
- balance.go:1214-1330 balanceRead: the key is masked when it appears in a URL carried into the error text, the body is capped at 1 MiB and closed, and a refusal body is preferred to the bare status.
- remote_quotas.go:364-415 / remote_key_limits.go:230-274: every status class mapped (404/405 no share, 403 held vs not shared, 401 key refused, other non-2xx vendor message), 8s timeout, body closed and capped.
- signin.go:555-575 ownCallback: a pasted callback must match this flow's port, a loopback host, the redirect path and the OAuth state before its code is exchanged.

## Open questions

- How often a GUI save and the scheduled list refresh actually overlap in one process: F1's window is milliseconds, so its real frequency needs a runtime count (e.g. instrument `store` with a mutex and log overlaps) rather than a code reading.
- Whether multiple saved accounts of one agent is a common state (it gates F2's magnitude): the code supports it and `perLogin` exists for it, but the audit could not observe a real account count.
