## Verdicts

prov-core/F1 CONFIRMED - provider.Save (:918 -> :950) and takeMatching (autolist.go:229 -> :247) both do read -> mutate -> store(f) with no mutex or file lock, and store replaces the whole file, so the later write drops the earlier edit.
prov-core/F2 CONFIRMED - fetchSubscriptionUsage starts one goroutine per account (:528-534) and no keySlots limiter is taken on the account path; keepLast softens a 429 on a card that has a prior reading but not the burst.
prov-core/F3 DOWNGRADED P3 - os.WriteFile (:256) is non-atomic and readAlertMarks (:237-243) ignores the unmarshal error, but no CLI path calls checkAlerts: only the GUI host's WatchQuotas does (gui/quota_alert_notify.go:174), so only a crash/power loss (or ENOSPC) mid-write triggers it.
prov-core/F4 CONFIRMED - the account branch of Save rebuilds Provider from a whitelist (:891) that omits LimitRestOff, after the handler set it (gui/providers.go:1530-1534), so every account save stores false.
prov-core/F5 DOWNGRADED P3 - partial=true (:159) does keep a vanished card, but a card refresh hides every other provider (subscription_usage.go:439-451) and the sign-out/active-removal path clears the whole cache (logins.go:1358-1362, 1479-1482), so only a removed non-active account of the refreshed provider lingers, under a minute.
prov-models/F1 CONFIRMED - setModelName does settings.Load() (:90) then settings.Save() (:103) with no lock spanning them; Save takes fileMu only for the write (settings.go:1426-1429) and save() re-reads only GatewayConversations (:1743-1745).
prov-models/F2 REJECTED - duplicate of prov-core/F1: the same missing providers.json lock, and store(f) at models.go:334 is one of the write sites prov-core/F1's fix covers; the model-refresh trigger is folded into that finding, no separate defect.
prov-models/F3 REJECTED - the union gap at :396 is real code but the stated effect is refuted: empty Efforts means "levels unknown", the GUI fills provider.Levels (gui/providers.go:785-787), the gateway clamps only when len > 0, and Account.Levels plus effortMate still route high/xhigh (gateway/effort_accounts.go:77, gateway.go:2846-2847).
prov-models/F4 CONFIRMED - codexModels reads 4<<20 with the read error discarded (:51) where CodexLiveList reads 16<<20 and keeps the error (:90-93); a cut or oversized body becomes "ChatGPT listed no Codex models".
usage-stats/F1 CONFIRMED - Period.Known accepts any parseable range or positive last: (period_range.go:80-86), and timeline allocates one point per day (ledger.go:639-654) while buildRequestBlocksIn makes six maps per point (request_page.go:2065-2071) with no cap; 0001-01-01..9999-12-31 is ~522k points.
usage-stats/F2 CONFIRMED - pageKey (:743) carries only the local date, not the rolling window's start, and adopt clears pages only when st.key changes (:1079), so last:24h is served from the 09:00 build until midnight, a log write or a restart; summary_cache.go:77-81 already refuses to cache the same window.
usage-stats/F3 CONFIRMED - LastSeen holds the global mu (:1204) across open/seek/scan of the last 256 KiB (:1206-1222), and agent.Applied calls it per agent (applied.go:226, :236) while Append takes the same mu (:277).
usage-stats/F4 CONFIRMED - OwnPage re-cuts l.Accounts by the current page's rows (:461-465) although the page built that list over every cell in the key-filtered period (request_page.go:1973-1981, :2051).
usage-stats/F5 CONFIRMED - blockPricer and the totals/ledger paths all gate on in+out > 0 (:275, usage.go:759, ledger.go:452), so a cache-only row is priced 0, is not Priced, and is never counted Unpriced even though CostSplit bills cache tokens (catalog.go:336-341).

## Corrected findings

### prov-core/F1 [P2] providers.json is read-modify-written with no lock: two writers silently drop one's edit
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/provider.go:918 (the store is at :950)
- code:
```go
	f, err := read()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(f.Providers, func(q Provider) bool { return q.ID == p.ID })
```
- trigger: two providers.json mutations overlapping inside one `magpie serve` process: a GUI provider Save (read :918, store :950) against the once-a-minute list refresh's takeMatching (autolist.go:229 -> :247), another tab's save, or the background refetch's dropGonePicks (models.go:305 -> :334); a CLI run beside the gateway has no cross-process lock either.
- effect: the later store writes the whole snapshot it read before the other's change, so the earlier edit disappears with no error: an added provider or group, a key switched on or replaced, a model pick. The rest of the file is not corrupted.
- why: store is atomic per write (temp file + rename through writePrivate, logins.go:311-327) but the read -> edit -> write sequence has no mutual exclusion, in-process or cross-process; atomicity of the write is not serializability of the edit.
- fix: hold one process mutex plus a filelock.Mutex on Path()+".lock" around every read-modify-write, as account.go:995 and logins.go:134 already do for their files. Locking store alone is not enough; the read must be inside the lock.
- evidence: no sync.Mutex, filelock or flock in provider.go, while filelock is used at account.go:995, logins.go:134, codex_resets_used.go:60 and chatgpt_api.go:380; store(f) sites at provider.go:950, autolist.go:247 and models.go:334 all follow a fresh read().
- confidence: high (mechanism), medium (how often a user hits it)

### prov-models/F1 [P2] Two model-preference writes at once lose one of them, with the whole settings file
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/modelprefs.go:90 (save at :103)
- code:
```go
	s := settings.Load()
	key := p.ID + "/" + model
	if s.ModelNames[key] == name {
		return false, nil
	}
	if name == "" {
		delete(s.ModelNames, key)
	} else {
		if s.ModelNames == nil {
			s.ModelNames = map[string]string{}
		}
		s.ModelNames[key] = name
	}
	if err := settings.Save(s); err != nil {
```
- trigger: two settings writers in flight in one process: the GUI provider Save's SetModelPrefs (gui/providers.go:1715) against a GUI save of another setting (or a second GUI save), or a CLI model command beside either. Each setter does Load -> mutate -> Save, and SetModelPrefs repeats that pair once per preference (modelprefs.go:522-560), widening the window.
- effect: the later Save writes its whole snapshot (settings.go:1754), so the other change and every other setting changed in that window (theme, model prices, wire names) silently reverts. GatewayConversations is the one field save() re-reads (settings.go:1723-1745), so recording consent is not lost.
- why: Load releases fileMu before returning (settings.go:1224-1227) and Save takes it only for the write (1426-1429); the package already has read-modify-write helpers (EditUsageAliases :1462-1477, EditCheckinOff :1443-1455) that no model-pref setter uses.
- fix: one Edit(func(*Settings) error) under lockFile, used by setModelName, SetModelPrice, setModelEfforts, setModelReasonless, setModelImage, setModelAPI, setModelSame, SetUpstreamName and DropUpstreamName, with SetModelPrefs performing a single edit.
- evidence: modelprefs.go has no Lock anywhere; Save is `defer lockFile()(); return save(s, false)`; save() merges only the stored GatewayConversations before marshalling the passed snapshot.
- confidence: high

### usage-stats/F1 [P2] An unbounded `?period=` makes one Requests request allocate ~0.5M chart points and ~3M maps
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usage/ledger.go:639 (allocation at request_page.go:2065)
- code:
```go
	if d := p.Last(); d > 0 {
		since = now.Add(-d)
		if d <= 48*time.Hour {
			bucket = "hour"
			for h := 0; h <= int(d.Hours()); h++ {
				t := since.Add(time.Duration(h) * time.Hour)
				pts = append(pts, Point{Label: t.Format("15"), Time: t})
			}
		} else {
			n = int(d.Hours())/24 + 1
```
- trigger: one loopback request, no key needed (usageRequests admits local(r), gateway/usage_api.go:66-70): `GET /v1/magpie/usage/requests?period=0001-01-01..9999-12-31&limit=1`. Period.Known accepts any parseable range and any positive last: (period_range.go:80-86), so a range past 60 days becomes weekly points; `last:2562047h` gives 106,752 daily points.
- effect: out.Series gets ~521,723 SeriesPoints (request_page.go:2065), each with one outer and five dimension maps (2067-2070), ~3.13M maps and hundreds of MB on an empty log; newCellPass builds and discards the same base again (request_cells.go:143); the finished page stays in requestCache.pages, up to 16 (request_page.go:815-818), after the request returns. Two such periods wedge or OOM the process. ledgerSeriesAt has the same shape at ledger.go:1074.
- why: the chart's cost is linear in the period's calendar length, but nothing bounds the period: limit is capped at 500 and offset clamped (request_page.go:723-727) while the period is only tested for parseability.
- fix: clamp or reject periods in usageapi.PeriodOf to the heatmap horizon, or cap timeline's n and fold older days into the first bucket.
- evidence: no maxPoints, maxSeries or MaxPeriod anywhere in internal/usage; the two allocation sites are request_page.go:2065 and ledger.go:1074; PeriodOf passes Known() through at usageapi.go:197-202.
- confidence: high

### usage-stats/F2 [P2] A rolling period's Requests page is cached for the whole day, so `last:24h` freezes
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usage/request_page.go:743
- code:
```go
	q := pageKey{p, f, offset, limit, now.Format(time.DateOnly)}
	if page, ok := idx.pages[q]; ok {
		idx.Unlock()
		return page
	}
```
- trigger: `period=last:24h` at 09:00 and again at 21:00 with no usage.jsonl append and no session-file change in between: st.key (request_page.go:650-678) is unchanged, and its only clock input is the local date at :668.
- effect: the 21:00 answer is the 09:00 page verbatim: rows, Sum, Through/Direct/Reported and the hourly Series still include calls that are now outside the window and stop 12 hours short of now. It stays stale until midnight, the next log/session write, or a restart.
- why: pageKey's only clock input is now.Format(time.DateOnly) (:510-518, :743); adopt clears pages only when st.key changes (:1079); the same file already refuses to cache rolling cell parts (:779), and summary_cache.go:77-81 refuses rolling summaries for exactly this reason.
- fix: bypass idx.pages and idx.flights when p.Last() > 0, or put p.Since(now) into pageKey.
- evidence: pageKey holds Period, Filter, Offset, Limit and Date only (510-518); the page is served from idx.pages at 744-746; summary_cache.go:77-81 documents the rule this cache breaks.
- confidence: high

### prov-core/F2 [P3] One goroutine and one vendor request per saved account, with no concurrency bound
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
- trigger: a user with several saved accounts of one agent (Codex, Copilot, Claude, Gemini), on a cold reading or after RefreshUsage's forgetLoginReading: perLogin appends one fetch per login (:558-568) and fetchSubscriptionUsage starts them all at once.
- effect: one goroutine and one HTTPS request per account, simultaneously, instead of the 8-at-a-time key path. A cold account's 429 shows as its card's error, while an account with a previous reading keeps it (login_usage.go:223-229), so the reported "every card showing its error" is too strong; the burst itself is unbounded and user-controlled.
- why: keyReadsAtOnce = 8 and keySlots (keys.go:271-292) exist for exactly this and are taken by Balance, PlanQuotas and remote_key_limits; the account path takes none.
- fix: take a keySlots slot per fetch around f() in the loop, as the key and plan paths do.
- evidence: no keySlots or slots.take in subscription_usage.go; one bare goroutine per fetch at 528-534; the slot limiter only at balance.go:1522, planquota.go:691 and remote_key_limits.go:127.
- confidence: high

### prov-core/F3 [P3] usage-alerts.json is overwritten in place and its parse errors ignored, so a torn write re-sends every alert
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/quota_alert.go:256
- code:
```go
	marks := readAlertMarks(path)
	before, _ := json.Marshal(marks)
	out := dueAlerts(qs, marks, pct, bal, now)
	out = append(out, dueReminders(qs, marks, lead, now)...)
	if after, _ := json.Marshal(marks); string(after) != string(before) {
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, append(after, '
'), 0o600)
	}
```
- trigger: a crash, power loss or full disk while os.WriteFile truncates and writes the marks file. Only the GUI host runs checkAlerts, through WatchQuotas (gui/quota_alert_notify.go:174); quotaAlertMu (quota_alert.go:234-235) serializes in-process callers, and there is no `magpie quota` writer (checkAlerts has one non-test caller at :284), so the input's two-process trigger does not exist.
- effect: readAlertMarks unmarshals a partial or empty file with the error ignored (:237-243), so the marks are empty or truncated and every window and balance is announced again on the next 5-minute run. Duplicate notifications, and the file's marks are lost.
- why: the one "already told" state file is the only account-state file in this package written without a temp-file rename, and it is read without checking the unmarshal error, so partial content becomes "nothing was told yet".
- fix: write through writeFileAtomic (defined kiro.go:464, already used at quota_last.go:95) and keep the previous map when the unmarshal fails; log the write error instead of discarding it.
- evidence: quota_alert.go:237-257; no writeFileAtomic in this file; checkAlerts callers are quota_alert.go:284 and tests only.
- confidence: high (mechanism), low (frequency)

### prov-core/F4 [P3] A signed-in account can never keep "don't rest me on a 429": Save's account branch drops limitRestOff
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/provider.go:891
- code:
```go
		key := ""
		if p.ID == "kiro" {
			key = p.Key
		}
		p = Provider{ID: p.ID, Key: key, Models: p.Models, NoModels: p.NoModels, PickedFrom: p.PickedFrom, Delisted: p.Delisted, Unlisted: p.Unlisted, Off: p.Off, Fallback: p.Fallback, Routing: p.Routing, Sink: p.Sink, Affinity: p.Affinity, KeepLogin: p.KeepLogin, KeepLoginAs: p.KeepLoginAs, SwitchOnly: p.SwitchOnly, Contexts: p.Contexts, Family: p.Family, Proxy: p.Proxy, AccountProxies: p.AccountProxies, AccountModels: p.AccountModels, AccountCaps: p.AccountCaps, AccountWindowCaps: p.AccountWindowCaps, CapStop: p.CapStop, CapCareful: p.CapCareful, AccountNames: p.AccountNames, MaxConcurrency: p.MaxConcurrency, AccountConcurrency: p.AccountConcurrency, QueueLimit: p.QueueLimit, QueueWait: p.QueueWait, MaxRPM: p.MaxRPM, PriceRate: p.PriceRate, QuotaRest: p.QuotaRest, QuotaResetAt: p.QuotaResetAt, RestCap: p.RestCap, PinUpstream: p.PinUpstream, NativeCompaction: p.NativeCompaction, PromptAsWritten: p.PromptAsWritten, Hidden: hiddenAccount(p.ID), Quiet: quietAccount(p.ID), Tucked: tuckedAccount(p.ID)}
```
- trigger: saving a signed-in subscription account after setting "Set it aside when out of quota or rate limited" off, or POSTing `{"id":"codex","limitRestOff":true}` to /api/provider/save. The handler sets in.LimitRestOff (gui/providers.go:1530-1534) and Save's account branch rebuilds p from the literal, which has no LimitRestOff, then store() writes the stripped value.
- effect: LimitRestOff is false forever for every signed-in account, and the gateway rests the account on the first quota refusal or 429 (gateway/draw.go:931, routing.go:1068). The editor also cannot recover it: app.js:9926 sends the field only for custom providers.
- why: a whitelist rebuild of Provider silently accepts and drops any field added later; the handler's preservation of old.LimitRestOff cannot preserve a value that was never stored.
- fix: add `LimitRestOff: p.LimitRestOff` to the literal.
- evidence: the literal's field list has no LimitRestOff; the GUI request struct at gui/providers.go:1172 and its assignment at 1530-1534; draw.go:931 skips the rest only when LimitRestOff is set.
- confidence: high

### prov-core/F5 [P3] A manual refresh can never drop the card of a removed account
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/quota_refresh.go:159
- code:
```go
		c := &subscriptionUsageCache
		c.Lock()
		c.data = cacheCards(c.data, out, seq, true)
		c.Unlock()
```
- trigger: RefreshUsage(id, "") on a provider with several saved accounts after one non-active account of it was removed. The sign-out and active-account removal case is guarded: ForgetLogin calls ForgetAccounts when the account was signed out (logins.go:1358-1362), which nils subscriptionUsageCache (1479-1482).
- effect: the removed account's card survives that provider's refresh and is served from the cache until the next full timed read (subscription_usage.go:293-307 commits with partial=false), about a minute. Other providers are hidden during a card refresh (subscription_usage.go:439-451), so nothing else is kept stale.
- why: refreshCards always commits with partial=true (:159), even when the read covered the provider's whole account group; the partial branch is contractually "keep the rest as they were". The input's premise that this path reads every card is wrong: a refresh hides every other provider, and with user=="" it reads only the requested provider's accounts.
- fix: in the user=="" (whole-provider) case, drop that provider's cards that out no longer has before merging; keep partial=true for a single-card refresh, since partial=false there would drop every other provider's cards.
- evidence: cacheCards' contract at 167-184; refreshCards 116-165 (cardRefresh set at 135-136, fetch at 153); ForgetAccounts 1473-1483.
- confidence: high (mechanism), low (impact)

### prov-models/F4 [P3] The Codex model list is truncated at 4 MiB where the same endpoint is read to 16 MiB, and the read error is thrown away
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/provider/codex_models.go:51
- code:
```go
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ChatGPT models: %s", resp.Status)
	}
	saveCodexPrompts(b)
	ms := parseCodexModels(b)
	if len(ms) == 0 {
		return nil, errors.New("ChatGPT listed no Codex models")
	}
```
- trigger: `GET {CodexBase}/models` answering a body above 4 MiB (CodexLiveList reads the same URL at 16<<20, :90), or any error after the 200 that cuts the body short (a reset proxy, a cancelled stream).
- effect: the truncated body fails JSON decoding (parseCodexModels, :249-251), so the caller reports "ChatGPT listed no Codex models" rather than a read or size problem, saveCodexPrompts receives the cut body, and codexFetchSaved keeps the previous saved list (:431-437).
- why: two readers of one endpoint disagree on its size, and the only signal that the body was not read whole (io.ReadAll's error) is discarded before the parse, making truncation and an empty list the same state.
- fix: read with 16<<20 as CodexLiveList does, and return the io.ReadAll error instead of `_`.
- evidence: codex_models.go:51 (4<<20, b, _ :=) and :90-93 (16<<20, error returned); parseCodexModels returns nil on any unmarshal error.
- confidence: high (code), the >4 MiB body itself unmeasured (the input's open question)

### usage-stats/F3 [P3] `LastSeen` holds the global usage-log mutex across a per-agent 256 KiB file scan
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usage/usage.go:1204
- code:
```go
	mu.Lock()
	defer mu.Unlock()
	f, err := os.Open(Path())
	if err != nil {
		return last
	}
	defer f.Close()
	const tail = 256 << 10
```
- trigger: any Agents or manage-page render that calls agent.Applied, which calls lastSeen(a.ID) per agent (agent/applied.go:226 and :236) while the gateway serves calls.
- effect: N sequential open/seek/scan/JSON-decode passes over the last 256 KiB of usage.jsonl, all under mu; with 20 agents that is 20 scans of up to ~1,000 records each, blocking Append (usage.go:277-296) and so the accounting write of every served call for tens of milliseconds per render. The input's "a few ms each" per unmarshal is too high; the aggregate is what blocks.
- why: mu exists to serialize writers, but LastSeen only reads an append-only file whose partial trailing line it already tolerates, so it does not need the writer lock across I/O.
- fix: drop mu in LastSeen, or read the tail first and take the lock only for the seen lookup.
- evidence: LastSeen at 1199-1224, mu.Lock at :1204, the tail scan at :1211-1222; mu is the same sync.Mutex Append takes at :277; call sites applied.go:226 and :236.
- confidence: high

### usage-stats/F4 [P3] A held gateway key's Account filter lists only the accounts on the page being served
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usageapi/usageapi.go:463 (the cut is at :465)
- code:
```go
	accounts := map[string]bool{}
	for _, r := range l.Rows {
		accounts[r.Account()] = true
	}
	l.Accounts = slices.DeleteFunc(l.Accounts, func(a Account) bool { return !accounts[a.ID] })
```
- trigger: a shared magpie and a held gateway key (heldKey, gateway/usage_api.go:29-31) whose calls in the period used two subscription accounts, with the second account's only calls beyond the current page: `GET /v1/magpie/usage/requests?offset=0&limit=100`.
- effect: l.Accounts is built over every cell of the key-filtered period (request_page.go:1973-1981, :2051) but OwnPage re-cuts it to the accounts in the current page's at-most-limit rows, so the Account filter changes as the reader pages and an account used only later in the period never appears.
- why: it conflates "accounts these rows used" with "accounts this caller's calls in the period used"; the per-key cut the comment intends is already made by f.CallerKey = who.KeyID (gateway/usage_api.go:80) before the page is built.
- fix: keep the period-wide l.Accounts as built and do not re-cut it by page rows.
- evidence: account collection over all cells at request_page.go:1973-1981, assignment at :2051; the re-cut at usageapi.go:461-465; f.CallerKey at gateway/usage_api.go:80.
- confidence: high

### usage-stats/F5 [P3] Calls with no input/output tokens are priced as free and never counted as unpriced
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/usage/block_price.go:274 (the guard is at :275)
- code:
```go
		p.Cost = 0
		if got.price != nil && p.Tokens[0]+p.Tokens[1] > 0 {
			p.Flags |= 1
			p.Cost = got.price.CostSplit(int(p.Tokens[0]), int(p.Tokens[1]), int(p.Tokens[2]), int(p.Tokens[3]), int(p.Tokens[5]))
		}
```
- trigger: a row whose only tokens are cache traffic (in = out = 0, cache_read/cache_write > 0): a cancelled or failed call whose usage block names cache, or a reported record posted to /v1/magpie/usage/requests, which the ingest accepts (in, out, cache_read and cache_write are independent fields, usage_ingest.go:311).
- effect: the row is not priced (Priced stays false, Cost 0), and both totals paths return before their Unpriced branch (usage.go:759-764, ledger.go:452-453), so the cache cost is missing from the page's Cost, the ledger's Sum and the CSV, and the row is not reported unpriced either. A reported row's own cost survives only on the row (request_page.go:324-330).
- why: in+out > 0 is used as a proxy for "this call has a billable cost", but cache reads and writes are billable on their own and CostSplit already prices them (catalog.go:336-341).
- fix: price when in+out+cache_read+cache_write > 0 and count Unpriced on the same test in addFrom, addRow and ledgerMatched.
- evidence: block_price.go:274-278; usage.go:759-764; ledger.go:452-453; CostSplit at catalog.go:336-341; the ingest field list at usage_ingest.go:311.
- confidence: medium (a real vendor row is unproven; the reported path is certain)

## Added findings

none

## VERDICT

VERDICT: CONFIRMED 10, DOWNGRADED 2, REJECTED 2, ADDED 0; reviewed 30 files
