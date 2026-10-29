# verify-gateway: gw-core, gw-proto, gw-subs falsified against /tmp/magpie-audit-Pzn4fNp7/src

Every input finding was opened at its cited line in the release tree, its quote compared
character for character, and its trigger traced through `rg -n '<symbol>' /tmp/magpie-audit-Pzn4fNp7/src/internal`.
Read-only: nothing in the tree or in the input lane files was modified.

## Verdicts

gw-core/F1 CONFIRMED - WithCancel at 2918 is the only context the try owns; the tree adds no body deadline (no WriteTimeout, ResponseHeaderTimeout covers headers alone, keepaliveLongest 5min ends only the comments), so a stalled reply holds the attempt and its lane slot. HTTP/2 health pings are protocol-level and are answered by a vendor whose application stopped writing, so they do not falsify the trigger.
gw-core/F2 CONFIRMED - affinity.go:707-713 prunes by age (stickKeep 24h) inside the writer's global lock and never bounds len(sticks.m); saveSticks' 512-entry cap applies to the file only. rules.go:349-355 repeats it.
gw-core/F3 DOWNGRADED P3 - affinity.go:256 I/O under sticks.Lock is real, but the os.ReadFile+decodeJSON at 261-266 runs once per mtime change (saveSticks at most every sticksFlushAfter=2s), not on every miss; only map-miss requests take the stat.
gw-proto/F1 CONFIRMED - chatEncoder.event (1554-1596) and renderChat (1613-1658) have no KImage case; saysSomething (gateway.go:6647-6655) counts KImage as content, so the paid image is delivered as an empty success.
gw-proto/F2 CONFIRMED - all four mends are gated by compact byte constants (chattidy.go:79-87); a line with ": " separators matches none but toolCallsField, while line() would mend it by value. Line 12's own rationale quotes the spaced form of WorkBuddy's delta.
gw-proto/F3 CONFIRMED - chat.go:1276-1279 calls giveHeld but not end; d.lead (set at 1106 and 1143) is flushed only by end (1261-1272).
gw-subs/F1 CONFIRMED - codexLists.at (1784-1789) is inserted at 1823 and has no production delete; the key (1804-1811) carries the caller's Authorization/chatgpt-account-id and raw query string.
gw-subs/F2 CONFIRMED - the loop's read error is dropped at 1330-1331; holdWriter.settle returns at once for a passing stream (fallback.go:2474) and nothing else inspects the cut, so end() records 200 with no error and no fallback.
gw-subs/F3 CONFIRMED - codex_socket.go:304 disables coder/websocket's 32768-byte default (-1 is not bumped by SetReadLimit, read.go:97-107) and codexSocketBody.frame's c.Read has no other bound.
gw-subs/F4 REJECTED - the trigger is impossible on this toolchain: crypto/rand.Read "never returns an error" in Go 1.24+ (go doc on go1.27.3; go.mod requires go 1.26.8) and crashes instead, so the fallback branch at 507-509 cannot mint a guessable token.
gw-subs/F5 CONFIRMED - codexDescription passes chose=true unconditionally at 209, where codexTitle (159-161) and codexPredictionsTo (290-291) compare to the setting they came from; codexDescriptionsTo (72-82) returns the caller's model when the setting is empty.

## Corrected findings

### gw-core/F1 [P1] No per-attempt or total deadline: a reply that stalls mid-stream hangs the turn, its lane slot and the client forever
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/gateway.go:2918
- code:
```go
			// a stream that fails before anything is said is let go at
			// once (hw.stop), not read on until the vendor hangs up
			ctx, stop := context.WithCancel(withWireAt(r.Context(), at))
			if isGroup {
				ctx = groupTry(ctx)
			}
```
- trigger: an upstream that answers 200 and one event, then keeps the connection open and sends nothing more (or only SSE comments), on any key or account with MaxConcurrency/QueueWait, or a non-streaming body that stalls after the headers.
- effect: attempt sits in the body read, so the deferred release() (gateway.go:2976-2983) never runs and the lane slot stays taken; with MaxConcurrency=1 every other request to that key queues and is then turned away 429 with Retry-After (gateway.go:2987-3014). Keepalives stop at 5 minutes (keepaliveEvery/keepaliveLongest = 15s/5min, gateway.go:6823) and nothing replaces them: keepQuiet returns with "left for a timeout to end" (fallback.go:2729-2731) and no such timeout exists. The agent sees a reply that never ends until its own client timeout.
- why: every try is derived from the client's context with WithCancel only, and hw.stop (the slow-start cancel) fires before first content alone: watch's deadline (fallback.go:2692-2704) is disarmed once started() is true. "A try ends in bounded time" therefore rests on the vendor.
- fix: give each try its own context.WithTimeout(r.Context(), tryLongest) (or a body-read deadline) instead of WithCancel, so a stalled reply ends as the next candidate's turn.
- evidence: gateway.go:2916-2921 read verbatim; rg -n 'context.WithTimeout|context.WithDeadline' internal/gateway -g '!*_test.go' -> 26 hits, none in serve, attempt, forward, forwardTranslated; gateway.go:643's http.Server sets ReadHeaderTimeout and IdleTimeout but no WriteTimeout; upstream_transport.go:40,65 ResponseHeaderTimeout=10min covers headers only; netproxy.PingAfter/PingWait (20s/10s, netproxy.go:44-46) are transport-level HTTP/2 pings a stalled vendor still acknowledges, so they are not a reply deadline. verified-by: read, grep
- confidence: medium (the trigger needs a vendor that keeps its connection alive; the missing bound is certain)

### gw-core/F2 [P2] The affinity map grows without bound and every write past 4096 walks all of it under the global lock
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/affinity.go:707
- code:
```go
	sticks.m[key] = stick{rest: c.rest, who: c.who(), model: c.model, effort: c.effort, turn: turn, at: now, cacheRead: cacheRead, cacheWrite: cacheWrite, hour: hour}
	delete(sticks.opening, key)
	if len(sticks.m) > 4096 {
		for k, st := range sticks.m {
			if now.Sub(st.at) > stickKeep {
				delete(sticks.m, k)
			}
		}
	}
```
- trigger: a client that varies any of the eight session headers per request (gateway.go:7117-7121, read by conversationID at 7188-7195) or whose body-derived conversation id differs each turn, so answered inserts a fresh key per request.
- effect: stickKeep is 24h (affinity.go:71), so the prune only removes entries older than a day: every distinct id seen in 24h stays resident, each carrying who/model/hourCache. Past 4096, every answered request walks the whole map while holding sticks.Lock (699-714), so N answers cost O(N^2) comparisons and every affinity lookup (affine locks the same mutex at 505) waits behind that walk. Nothing bounds len(sticks.m) by count; saveSticks' sticksKept=512 cap (306-307) applies to the file, not the map. rules.go:349-355 repeats the same age-only prune for turnRules.m.
- why: "per-conversation state is bounded" is enforced by age only, and eviction is charged to the next writer on the request path.
- fix: evict by count (drop the oldest past a hard cap) instead of scanning for entries older than 24h, keeping each write O(1).
- evidence: affinity.go:705-714 and 251-281 read; stickKeep=24h at 71, sticksKept=512 at 73; rg -n 'sticks.m|4096' affinity.go -> the only count trigger is 707 (and 822 for refusedSeals), nothing deletes by count; gateway.go:7117-7121 lists the eight caller-supplied headers. verified-by: read, grep
- confidence: high

### gw-proto/F1 [P2] A model's drawn image never reaches a Chat client: the Chat encoder and renderer have no image case
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/chat.go:1594
- code:
```go
		saidError(failed, ev.Said)
		e.w.event("", map[string]any{"error": failed})
	}
	e.col.add(ev)
}
```
- trigger: a Chat Completions request whose upstream reply carries an image part: a Chat client routed to a CodeAssist/Gemini provider (usable() returns that protocol, gateway.go:5428-5443, and translate() is called with it at 4112) whose reply holds inlineData, so codeassistDecoder emits Event{Kind: KImage, ...} (codeassist.go:1237).
- effect: the client gets a successful, empty turn. Streamed: the role chunk, no content chunk, finish_reason "stop", usage, [DONE]. Whole: "content": null with finish_reason "stop" (renderChat keeps content nil, chat.go:1614,1619-1637). The image stays in the collector's Result.Parts (ir.go:653-656) and is never written.
- why: the IR has a first-class image event and saysSomething counts it (gateway.go:6647-6655), so the reply is non-empty and delivered as success while its only content is discarded; chatEncoder.event's switch (1554-1596) covers KText, KThink, KToolStart, KToolArgs, KError only, and gemini.go:676 is the one encoder that handles KImage.
- fix: add a `case KImage:` to chatEncoder.event (content as a data: URI or an image part) and the same part to renderChat, as geminiEncoder.event does.
- evidence: rg -n 'KImage' -g '!*_test.go' internal/gateway -> codeassist.go:1237, draw.go:1744, gateway.go:6650, gemini.go:676, ir.go:324/655, translated_probe.go:31; no hit in chat.go or responses.go. chat.go:1554-1596 and 1613-1658 read. verified-by: read, grep
- confidence: high on the drop; medium on how often a Chat route lands on an image model (a routing question outside this lane)

### gw-proto/F2 [P2] chatTidy's mends are byte-exact on compact JSON, so a relay that writes ": " separators is never mended
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/chattidy.go:79
- code:
```go
var (
	emptyReasoning    = []byte(`"reasoning_content":""`)
	emptyFinishReason = []byte(`"finish_reason":""`)
	emptyToolName     = []byte(`"name":""`)
	emptyToolCalls    = []byte(`"tool_calls":[]`)
	contentParts      = []byte(`"content":[`)
	contentPartsSpace = []byte(`"content": [`)
	toolCallsField    = []byte(`"tool_calls"`)
)
```
- trigger: an upstream Chat stream written with a space after the colon (Python json.dumps' default), e.g. data: {"choices": [{"delta": {"reasoning_content": ""}, "finish_reason": ""}]}, relayed as it is through the passthrough path where chatTidy is installed (gateway.go:5111-5114).
- effect: needsTidy (128-136) matches none of the four string shapes, so lines() returns the line untouched (109-114) and line()'s value comparison (248-262) never runs; only the separator-agnostic tool_calls term still catches tool-call chunks. The breakages the mends exist for come back for such an upstream: each empty reasoning_content restarts Qoder's thinking block, a strict client rejects every chunk with "finish_reason": "", and a merge-by-overwrite client assembles a tool call named "".
- why: the gate is a byte match written against one serialization form while the value it looks for is JSON; line() already decides by value, and the file covers both forms of exactly one pattern (contentParts/contentPartsSpace), which shows the gap is an oversight. The rationale comment at line 12 quotes WorkBuddy's delta as `"reasoning_content": ""`, i.e. the spaced form the gate cannot match.
- fix: make needsTidy separator-tolerant for the three string shapes (match the key alone, decide the value in line), as toolCallsField already does.
- evidence: chattidy.go:11-13 (rationale quoting the spaced frame), 79-87 (constants), 128-136 (gate), 240-262 (value mends) read; rg -n 'reasoning_content": "' internal/gateway -> chattidy.go:12 and ollama_chat_test.go:118 only, so no production constant holds the spaced form. verified-by: read, grep
- confidence: medium-high (the gate defect is certain and line 12 corroborates the spaced wire form; no captured upstream frame is in the tree)

### gw-subs/F1 [P2] The backend model-list cache is keyed by the caller's credential and never pruned
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
- trigger: any client that can reach GET /v1/codex/models with a different credential or query each time: codexListKey (1804-1811) builds provider.CodexBase + "\x00" + (chatgpt-account-id or Authorization) + "\x00" + r.URL.RawQuery, so a rotating token, several accounts, or an arbitrary ?query per request all mint distinct keys.
- effect: entry creation at 1823 is the only write and nothing ever deletes: the map grows for the process's life. Each entry keeps the backend's whole list (read under a 16<<20 limit, 1890) or the 401/403 body, so thousands of distinct keys are tens to hundreds of MiB of live heap never reclaimed, plus one outbound chatgpt.com request per new key that the first caller waits on for codexModelsWait.
- why: a per-credential cache with no eviction turns client-supplied key material into an unbounded map the caller sizes.
- fix: evict on read (drop entries past codexListFor, as the rest of the file does for lists) or cap the map and delete the oldest entry when over it.
- evidence: rg -n 'codexLists' internal/gateway -> 16 hits, all reads/assignments plus asking.Wait(); the only map reset is codex_models_shared_test.go:24, i.e. test-only. No production delete. verified-by: grep, read
- confidence: high

### gw-subs/F2 [P2] A reply cut mid-stream on Codex's own sign-in is recorded as a success and never reported to Codex
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/codex_backend.go:1310
- code (the loop whose body starts at line 1310):
```go
			n, err := res.Body.Read(buf)
			if n > 0 {
				...
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
- trigger: the connection carrying the relayed turn dies mid-SSE (network drop, or the ~120s cut codex_socket.go:30-38 describes) on the route that relays Codex's own sign-in (rest == "/responses", on == nil).
- effect: err from res.Body.Read is dropped at 1330-1331. holdWriter.settle (fallback.go:2474-2477) returns at once when the stream already passed, so hw.failure stays 0; the busy/edge retry block (1335-1355) and codexFallBack never run, and the handler falls through to end(call.Status, call.Error, ...) (1360-1378) with res.StatusCode 200 and call.Error "", recording a finished turn with partial token counts. Codex's SSE stream simply stops after some frames, with no error event.
- why: the loop treats a read error as "the reply ended" and only guards the client write; the same condition elsewhere is a client-visible cut (continuation.go:674, failed = cutMidReply(...)).
- fix: keep err and, when the stream ended short of its terminal event, record it and tell Codex (failWith/streamError with cutMidReply semantics) instead of returning 200 with an empty error.
- evidence: codex_backend.go:1305-1335 read verbatim (line 1310 is the only res.Body.Read(buf)); no later reference to err; rg -n 'cutMidReply' continuation.go -> 674, the contrast case. verified-by: read, grep
- confidence: high on the mechanism; magnitude depends on how often the vendor-side cut happens

### gw-subs/F5 [P2] Thread-description calls skip the calling key's model holds even when the caller chose the model
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
- trigger: a restricted gateway key (key_models.go:34-43, keyHolds) whose Codex sends a thread_description request naming a model the key may not use. codexDescriptionsTo (72-82) returns the caller's model when Settings -> CodexDescriptions is empty, via codexSideTo (57-61, modelOf(body) when the id contains "/"), and codex_backend.go:128-129 / gateway.go:1615-1616 pass it to codexDescription unchanged.
- effect: magpieChose makes keyHolds return false (key_models.go:40-42), so the description is routed on a model the key is held from and spends that account's allowance. Titles (159-161) and predictions (290-291) set the exemption only when to equals their setting; descriptions assert it unconditionally, so 209 is the one side call where the caller's own pick is priced as magpie's. Account holds still apply (accountHolds, key_models.go:47-50), so the widening is the key's model list alone.
- why: "magpie picked this model, so the key's list does not hold it" is asserted on a path where the caller may have picked it; the exemption must follow who chose, as it does for titles.
- fix: pass a flag that is true only when settings.Load().CodexDescriptions named the model, as codexTitle compares to CodexTitles.
- evidence: codex_titles.go:196-240 read (209 is the call, 236-237 the chose gate); rg -n 'magpieChose|keyHolds' key_models.go -> 24-50; codexDescriptionsTo at 72-82 and codexSideTo at 57-61 read. verified-by: read, grep
- confidence: high on the mechanism; medium on how often a restricted key's Codex names a held model for a description

### gw-proto/F3 [P3] Held-back text is dropped when a Chat stream ends at [DONE] instead of a finish_reason
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/chat.go:1276
- code:
```go
	if strings.TrimSpace(data) == "[DONE]" {
		d.giveHeld(emit)
		return nil
	}
```
- trigger: the reply's text begins with something that could open a thought tag (<, <t, <thi, ... without the close) or, inside a <think> block, ends mid-close-tag or mid-backtick-run (d.lead set at chat.go:1106 and 1143), and the stream's last data line is [DONE] with no chunk carrying finish_reason, as a relay that only sends it sometimes, or a stream cut and closed cleanly, produces.
- effect: the bytes held in d.lead are never emitted: the reply is silently short by that text, with no error and no retry, and what was already reported as thinking stays unclosed.
- why: d.end (chat.go:1261-1272) is the only flush of d.lead; the finish_reason path (1379-1384) and the tool-call path (1370-1371) call it, and the [DONE] branch calls giveHeld, which releases held tool calls only (1446-1458).
- fix: call d.end(emit) beside d.giveHeld(emit) in the [DONE] branch.
- evidence: chat.go:1261-1279, 1370-1384, 1446-1458 read; rg -n 'd.end(emit)' chat.go -> 1371 and 1381 only. verified-by: read, grep
- confidence: high on the mechanism, low on frequency

### gw-subs/F3 [P3] The socket's message-size defence is removed and not replaced
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/codex_socket.go:304
- code:
```go
	c.SetReadLimit(-1)
	if err := c.Write(ctx, websocket.MessageText, msg); err != nil {
```
- trigger: the ChatGPT backend sends one WebSocket text frame far beyond memory headroom, e.g. a response.completed carrying a whole long reply plus its reasoning, or an intermediary that folds frames.
- effect: coder/websocket's default limit is 32768 bytes and SetReadLimit only bumps n when n >= 0, so -1 means no limit (websocket@v1.8.14 read.go:97-107). frame() then reads the whole message (codex_socket.go:520-524) and body.pending holds it until the client copies it out, so one turn's peak heap is whatever one frame is, paid by every concurrent turn on the host.
- why: the transport's only bound on a single message was that default, and the socket path lifted it without a ceiling it knows the largest legitimate event fits. codexMessageTooBig (89-114) bounds only outgoing turns.
- fix: c.SetReadLimit(64 << 20) (or the bound the largest real event needs) instead of -1.
- evidence: rg -n 'SetReadLimit' codex_socket.go -> 304 only; read.go:97-107 and defaultReadLimit=32768 at 107; frame at 520-524. verified-by: read, grep, library source
- confidence: high on the semantics; low probability, since the peer is chatgpt.com

### gw-core/F3 [P3, downgraded] stickOf stats and JSON-decodes affinity.json while the global affinity mutex is held, on the request path
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/affinity.go:256
- code:
```go
func stickOf(key string) (stick, bool) {
	if st, ok := sticks.m[key]; ok {
		return st, true
	}
	path := sticksPath()
	fi, err := os.Stat(path)
	if err != nil || path == sticks.from && fi.ModTime().Equal(sticks.mod) {
		return stick{}, false
```
- trigger: a request whose conversation is not in memory (a new session, or the first after a restart/handover) while other requests are in flight: affine takes sticks.Lock (505) then calls stickOf (506), as do strictKeep (400-401) and unanswered (733-734). Every such miss pays os.Stat(256) under the lock; a miss after the file's mtime moved also pays os.ReadFile (261) and decodeJSON of up to 512 sticks (266-277).
- effect: affinity decisions of concurrent misses serialise behind another request's syscall, and one request per mtime change pays the whole file read and decode inside the mutex every request needs. Downgraded from P2: the read+decode happens once per mtime change (saveSticks runs at most every sticksFlushAfter=2s, affinity.go:165-178, 189-203), not per miss, and only map misses take the slow path, so on a warm page cache the common cost is one stat.
- why: the invariant "no blocking I/O under a lock every request needs" is broken; the file is read for one conversation's key while holding the mutex guarding all of them.
- fix: stat/read/decode outside sticks.Lock and merge the decoded snapshot under the lock, leaving only map access inside.
- evidence: affinity.go:251-281, 299-301, 333-335, 396-403, 501-510, 727-739 read; rg -n 'sticks.Lock' internal/gateway -g '!*_test.go' -> 10 sites in 2 files (affinity.go 299, 333, 400, 505, 628, 699, 727, 733; codex_backend.go 397, 500), the block at 505 enclosing stickOf's I/O. verified-by: read, grep
- confidence: high on the mechanism, lower on severity

## Added findings

none

## VERDICT

VERDICT: CONFIRMED 9, DOWNGRADED 1, REJECTED 1, ADDED 0; reviewed 18 files
