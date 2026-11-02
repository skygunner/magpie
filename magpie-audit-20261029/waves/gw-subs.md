## Coverage
- internal/gateway/codex_socket.go — lines 1-585 (whole file). Traced codexSocket.RoundTrip, codexSocketMessage, codexCreate, withClientMetadata, codexSocketHeaders, socket (handshake, timer, prelude, error frame), codexSocketError, codexLimitHeaders, codexMessageTooBig/refusedTooBig/shunCodexSocket, codexSocketBody.frame/Read/Close.
- internal/gateway/codex_images.go — lines 1-601 (whole file). Traced codexImagesFit, fitAll, fitInHand, fitBatch, imageHolds, fitHolds, codexImageRefit, fitCache.get/put/drop/letGo, triangleResize/triangleWeights.
- internal/gateway/continuation.go — lines 1-690 (whole file). Traced streamTranslated (retry loop, held errors, usage summing), continuation.possible/again/request/emit/add/unecho, geminiEnds, saysItsEnd, chatTail.finish, prefillHow/chatPrefill, emptyRetries.
- internal/gateway/codex_titles.go — lines 1-715 (whole file). Traced codexTitlesTo/codexDescriptionsTo/codexSummariesTo/codexPredictionsTo, codexSideTo/codexSideEffort, codexTitle, codexDescription, codexSummary, codexShaped, codexPrediction, titleJSON/summaryJSON/descriptionJSON/predictionJSON, writeTitleReply/writeTitleEnd, sentNowhere/answeredHere.
- internal/gateway/textcalls.go — lines 1-858 (whole file). Traced textCallAt, parseTextCalls, hermesCall/paramCall/glmCall/dsmlCall, callArgs, escapeRawControls, textCallSee.event/settle/release, textCallTidy.write/line/flush, wholeTextCalls.
- internal/gateway/trace.go — lines 1-952 (whole file). Traced trace.begin/beginFor/update/drop/trim, sessionFolder/shortFolder, Trace (long poll, wake/gather), gatewayTrace, OwnerTrace/OwnerLanes, routeUsage.
- internal/gateway/prompt.go — lines 1-1262 (whole file). Traced promptOf and all four body readers (anthropic/responses/chatCompletions/gemini), promptBuilder.add/chat/prompt, splitMemory/splitSkills, callOf/filesOfCommand, userWords, calibrate/calibrated, inspectPrompt, promptCounted.
- internal/gateway/search.go — lines 1-1287 (whole file). Traced searchesItself/searchesModel/modelSearches, searchRank/namedSearcherRank/autoSearcherReady/chosenSearcher, searcherModel/searcherReady/smallReadyOf, webSearch, modelSearch, searchWith, searchBody, searchReply/searchRounds, askTranslated.
- internal/gateway/codex_backend.go — lines 60-390, 900-1419, 1440-2070, 2515-2775, plus line-pinned greps over the whole file. Traced codexBackend (selection to wire), codexUpstream/codexUpstreamOn (retries, credits/reset/edge paths, stream copy, end()), codexFallBack, codexServed, apiKey/hopHeader/codexHeader/copyHeaders/callKind, codexBackendList/askCodexList/codexModels/codexMerged, callItemIDs/plainItemIDs/boundCallIDs/forVendor.
- internal/gateway/claude_subscription.go — lines 60-330, 465-1015, 1427-1930, 2171-2700, 2695-2765, 3555-3625, 3800-3890. Traced subscriptionBridge/subscriptionRun state, randomToken, startRun (launch, room, tmp, callback token), cliControl, claimResume/resume, keepIdle, shelve/unshelve, letGo/rewind/answered/park, readOutput (stream and whole-message paths, result/error/tool handling), cliUsage/gateway/plus/carryUpdate, continueWith, finish, launch.
- Verification-only reads (outside scope, for a guard or a library semantic): internal/gateway/key_models.go 24-53, internal/gateway/lan.go 386-425, internal/provider/codex_zstd.go 84-92, ~/go/pkg/mod/github.com/coder/websocket@v1.8.14/read.go 97-107.
COVERAGE: read 10 files, 15089 non-test LOC in scope; skipped 0 (all ten read; the two over 3k lines through their hot functions plus line-pinned greps rather than every line).

## Findings

### F1 [P2] The backend model-list cache is keyed by the caller's credential and never pruned
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/codex_backend.go:1784
- code:
```go
var codexLists = struct {
	sync.Mutex
	at map[string]*codexListed
	// asking is the asks going, for tests to wait on
	asking sync.WaitGroup
}{at: map[string]*codexListed{}}
```
- trigger: any client that can reach `GET /v1/codex/models` with a different credential each time — `Authorization` (or `chatgpt-account-id`) or `?query`, since `codexListKey` (line 1804) builds the map key from `provider.CodexBase + "\x00" + who + "\x00" + r.URL.RawQuery`. A Codex whose token rotates, a key with many accounts, or a script with one gateway key all produce distinct keys.
- effect: line 1823 `codexLists.at[key] = e` creates an entry that nothing ever deletes, so the map grows for the life of the process. Each entry keeps either the backend's whole model list (read under a `16<<20` limit, `askCodexList`) or the 401/403 body, so a few thousand distinct keys is tens to hundreds of MiB of live heap that is never reclaimed; each new key also costs one outbound chatgpt.com request that the first caller waits on for `codexModelsWait` (3 s).
- why: a per-credential cache with no eviction turns client-supplied key material into an unbounded map — server state that the caller chooses the size of.
- fix: evict on read like the rest of the file does for lists (`if time.Since(e.got) > codexListFor` drop others), or cap the map and delete the oldest entry when it is over.
- evidence: `grep -rn 'codexLists' --include=*.go /tmp/magpie-audit-Pzn4fNp7/src` → 8 hits in codex_backend.go, all reads/assignments plus `e.asked = nil`; the only `codexLists.at = map[...]` reset is codex_models_shared_test.go:24. No production `delete`. verified-by: grep
- confidence: high

### F2 [P2] A reply cut mid-stream on Codex's own sign-in is recorded as a success and never reported to Codex
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/codex_backend.go:1310
- code (the loop whose `n, err := res.Body.Read(buf)` is at line 1310):
```go
				if _, werr := out.Write(buf[:n]); werr != nil {
					break
				}
				if f != nil {
					f.Flush()
				}
			}
			if err != nil {
				break
			}
		}
```
- trigger: the connection carrying the relayed turn dies mid-SSE (network drop, an idle/proxy cut of the kind codex_socket.go's header describes at ~120 s) on the route that relays Codex's own sign-in — `rest == "/responses"` with `on == nil`.
- effect: `err` from `res.Body.Read` is dropped; the handler falls through to `end(call.Status, call.Error, …)` with `res.StatusCode == 200` and `call.Error == ""`, so the Routing and Usage views record a finished turn with partial token counts, `codexFallBack`/`askAgain` never run (they only fire for busy/edge statuses), and Codex sees an SSE stream that stops after some `event:` frames with no error event and no reason. The same condition elsewhere in the gateway is turned into a client-visible cut: continuation.go:674 `failed = cutMidReply(p.Name, serr)`.
- why: the loop treats a read error as "the reply ended" and only guards the *client* write; the turn's recorded outcome stops matching the reply the agent actually got, so the one case a fallback exists for is silently a 200.
- fix: keep `err`, and when the stream ended short of its terminal event, record it and tell Codex (failWith/streamError with `cutMidReply` semantics) instead of returning 200 with an empty error.
- evidence: read of codex_backend.go 900-1419; `grep -n 'res.Body.Read(buf)' codex_backend.go` → single hit at 1310, and the only examinations of the result are `if n > 0` and `if err != nil { break }`, with no reference to `err` afterwards. `grep -n 'cutMidReply' continuation.go` → 674, the contrast case. verified-by: read
- confidence: high (mechanism); magnitude medium

### F3 [P3] The socket's message-size defence is removed and not replaced
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/codex_socket.go:304
- code:
```go
	c.SetReadLimit(-1)
	if err := c.Write(ctx, websocket.MessageText, msg); err != nil {
```
- trigger: the ChatGPT backend sends one WebSocket text frame far larger than memory headroom — e.g. a `response.completed` carrying the whole reply plus the sealed reasoning of a long turn, or any intermediary that folds frames.
- effect: coder/websocket's default limit is 32768 bytes and `SetReadLimit` only bumps `n` when `n >= 0`, so -1 means *no limit* (websocket@v1.8.14/read.go:97-107). `frame()` then reads the whole message, `sseEvent` copies it, and `body.pending = append(body.pending, …)` (line 359) holds it until the client copies it out: one turn's peak heap becomes whatever one frame is, uncapped, with every concurrent turn on that host paying it.
- why: the transport's only bound on a single message was that default, and the socket path lifted it without substituting a ceiling it knows the largest legitimate event fits.
- fix: `c.SetReadLimit(64 << 20)` (or whatever bound the largest real event needs) instead of -1.
- evidence: read of codex_socket.go 284-365; `grep -n 'SetReadLimit' codex_socket.go` → 304 only; `grep -n 'func (c \*Conn) SetReadLimit' -A 12 ~/go/pkg/mod/github.com/coder/websocket@v1.8.14/read.go` → `if n >= 0 { n++ }` and `defaultReadLimit = 32768`, i.e. -1 disables it. verified-by: read
- confidence: high on the semantics; the peer is chatgpt.com, so the trigger is low-probability

### F4 [P3] The callback token — the only thing letting the MCP path in without a gateway key — falls back to a clock value
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/claude_subscription.go:507
- code:
```go
func randomToken() string {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
```
- trigger: `crypto/rand` failing while a Claude subscription turn starts (`startRun`, line 708).
- effect: the run's callback token becomes its start time in nanoseconds, and `ownCallback` (lan.go:398-410) admits `POST /_magpie/claude-mcp/{token}` "without a gateway key from wherever it comes" on any token in the live set — so a local process, or anything that can reach `MAGPIE_ADDR` on a LAN address, guessing a ~19-digit timestamp can hand that run fake tool results: text injected into the model's conversation as tool output, and the client's pending calls for that run consumed.
- why: a credential must fail closed; a fallback to a guessable value converts a rare entropy failure into an authentication bypass on a path that is deliberately exempt from key auth.
- fix: return an error from randomToken when `rand.Read` fails and fail the turn in startRun (`if err != nil { cleanup(); return nil, nil, err }`), as `claudeBinary` does for a missing CLI.
- evidence: read of claude_subscription.go 505-514 and lan.go 386-410; `grep -n 'randomToken\\|validate callback'` — the fallback is the only failure branch, and ownCallback's check is `subtle.ConstantTimeCompare` against live tokens only. verified-by: read
- confidence: high on mechanism, low on reachability

### F5 [P2] Thread-description calls skip the calling key's model holds even when the caller chose the model
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/codex_titles.go:209
- code:
```go
	s.codexShaped(w, r, body, to, "description", true, descriptionJSON, noDescription)
```
```go
	ctx := context.WithValue(r.Context(), titleCheckKey{}, check)
	if chose {
		ctx = magpieChose(ctx)
	}
```
- trigger: a restricted gateway key (`key_models.go:34` `keyHolds`) whose Codex sends a `thread_description` request naming a model the key may not use. `codexSideTo` returns `modelOf(body)` whenever the id contains "/" — a magpie model id — so `to` is the *caller's* model, not one chosen in magpie's settings.
- effect: `magpieChose` makes `keyHolds` return `false` (key_models.go:40-42), so the description is routed on a model the key is held from and spends that account's allowance. Titles (codex_titles.go:161) and predictions (290) only set the exemption when `to == settings.Load().CodexTitles`/`CodexPredictions`; descriptions assert it unconditionally, so 209 is the one side call where the caller's own pick is priced as magpie's. Account holds still apply (`accountHolds`, key_models.go:47), so the widening is the key's model list only.
- why: "magpie picked this model, so the key's model list doesn't hold it" is asserted on a path where the caller may have picked it; the exemption has to follow who chose, as it does for titles.
- fix: pass a flag from `codexDescriptionsTo` (true only when the setting named the model), as `codexTitle` compares to `settings.Load().CodexTitles`.
- evidence: read of codex_titles.go 196-240 and key_models.go 24-53; `grep -n 'if chose {\\|ctx = magpieChose' codex_titles.go` → 161, 236-237, 290, and `209`/`218` passing `true`/`false`. verified-by: read
- confidence: medium

## Checked, no defect
- codex_socket.socket handshake and fallback: every failure before the first reply frame returns `errCodexSocketFallBack` so the same body goes over HTTP; a 1009 marks the host too-big (`refusedTooBig`) and a refused handshake shuns it for 10 minutes; a dial that raced its `AfterFunc(codexSocketDial)` timer is detected with `stopped` and the conn closed, so no half-open socket is left, and 401/429/5xx are not shunned (they stay HTTP's to report).
- codexSocketBody: `frame()` sets `done` on the terminal event and closes through a goroutine, `Read` maps a normal close to `io.ErrUnexpectedEOF` so a socket that ends before the reply's end reads as a cut rather than a clean end, and `Close` is idempotent via `closed.Swap` — the lifecycle the "socket ended before any of the reply" fallback depends on holds.
- continuation.streamTranslated: retries are capped by `streamRetries`, a retry's error is held rather than emitted (`cut`/`failed`), and `again()` refuses to continue for a vendor refusal code, 400/422/404, a begun tool call (`possible()`), an exhausted count, or a cancelled client — so no double retry and no error-then-continue in the same reply.
- trace.go retention: `trim` keeps `traceKeep` routes and protects ones still `Done == false`; `keepBody` (prompt_text.go:47) drops past `promptBodies`/`promptBodiesMax`; `sessionFolder` is capped by `foldersKeep`; `Trace`'s long poll returns on deadline or context, so neither the history nor a waiting page grows without bound.
- claude bridge shelf/idle maps: every `shelve` caller holds `b.mu` (claude_room.go:337, claude_rss.go:115, keepIdle at claude_subscription.go:1463), and shelve creates `s.timer` before its caller unlocks, so `unshelve`'s `s.timer.Stop()` cannot see a nil timer; runs are capped by `idleMost`/`parkedMost` and each state transition resets the run's own deadline (`endIn`).
- codex_backend id sanitising: `safeIDChars` is `^[A-Za-z0-9_-]+$` (anthropic.go:1509) and `safeID` its allocation-free equivalent, so `plainItemIDs`' pre-check and its rewrite loop agree; `boundCallIDs` maps a call and its output through the same `provider.BoundCallID`, so a truncated id cannot split a call from its result.
- codex_images: `imageHolds` derives bytes from the declared header and format before any decode, `fitHolds` never decodes, `fitInHand` batches to `fitBudget`, and the cache honours `fittedBudget`/`fittedMost` with idle eviction — the header-bomb and 4×-concurrency cases the comments cite are handled.

## Open questions
- F2's user-visible magnitude: whether Codex surfaces a cut mid-SSE as a failed turn or accepts the truncated reply needs a live stream cut against `codexUpstreamOn`; no experiment was run.
- F4's reachability: whether `crypto/rand` ever errors on this platform in practice. The fallback is unreachable by construction in tests, so no test covers the guessable-token path.
