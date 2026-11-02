# Gateway protocol translation and streaming — audit of /tmp/magpie-audit-Pzn4fNp7/src

Scope: vendor JSON -> IR -> client format, and upstream SSE -> client SSE, in the release snapshot.
Read-only; no file in the snapshot was modified, no repo-wide command run.

## Coverage

- `internal/gateway/chat.go` 980-1659 — `cUsage.usage`, `Usage.chat`, `chatDecoder.text/mdCode.find/markedThought/end/decode/toolCall/toolArgs/giveHeld`, `chatChoiceIndex`, `stopFromChat/stopToChat`, `chatEncoder.chunk/start/event/keepalive/finish`, `renderChat`; line 127 (`reasoning_content` read).
- `internal/gateway/responses.go` 1350-1700, 1740-2110 — `responsesTool`, `rUsage.usage`, `Usage.responses`, `responsesDecoder.argsDelta/endCall/jsonWhole.feed/pickArgs/decode`, `requestRefused`, `saidError`, `callTo` tail, `itemPrefix`, `responsesEncoder.send/write/release/response/start/closeItem/closeAs/openItem/event/keepalive/finish`.
- `internal/gateway/anthropic.go` 1150-1420, 1490-1720 — message pairing tail, `resultSaid`, `anthropicDecoder.decode`, `decodeAnthropic`, `aUsage.usage`, `Usage.anthropic`, `stopFromAnthropic/stopToAnthropic`, `safeCallID/rawCallID`, `anthropicEncoder.start/close/openBlock/thinking/delta/sayCited/event/keepalive/finish`.
- `internal/gateway/gemini.go` 537-722 — `stopToGemini`, `Usage.gemini`, `geminiParts`, `geminiEncoder.chunk/keepalive/start/flushTool/event/finish`, `renderGemini`.
- `internal/gateway/codeassist.go` 1154-1330, 1540-1591 — `codeAssistDecoder.decode`, `toolOfCall`, `textCall` hold path, `hasResponse`, `antigravityRound`.
- `internal/gateway/chattidy.go` 1-360 — rationale header, `chatTidy.write/flush/lines/needs/line`, `needsTidy`, `mayMendUsage`, `openAIUsage`, `withoutEmptyNames`, `streamCallIndexes`, `openCall`.
- `internal/gateway/sse.go` 1-143 (read in full) — `readSSE`, `readSSEAlive`, `sseWriter.begin/flush/quiet/comment/event`.
- `internal/gateway/whole_stream.go` 1-330 (read in full) — `wholeAsStream`, `wholeEvents`, `chatWholeEvents`, `responsesWholeEvents`, `anthropicWholeEvents`, `orNull`.
- `internal/gateway/upstream.go` 1-213 (read in full) — `upstreamOf`, `upstreamStop`, `replyEnds`, `eventType`, `carriedError`, `streamErrorText`, `carries`.
- `internal/gateway/upstream_transport.go` 1-169 (read in full) — `upstreamTransport`, `vendorTransport`, `uploadTransport`, `uploading`, `drawsApart.RoundTrip`, `lostPing`, `replayable.RoundTrip`.
- `internal/gateway/ir.go` 1-460, 454-780 — `Part/Message/Request/Event/Usage`, `Usage.add/billed/prompt/cacheNotCounted`, `collector.add/grow/flush/closeTool/finish`, `saidAnything`, `argsOf/argsString/parseArgs`, `stringOrText`, `toolOutput` head.
- `internal/gateway/fastjson.go` 1-120, 182-456, 456-1050, 1050-1200 — `decodeJSONIn`, `holdBody`, `jsonFieldsOf`, `fillJSONIn`, `eachMember(At|In)`, `eachElement`, `elementsOf`, `inputList`, `skipJSONString/Value`, `unquoteJSON`, `hex4`, `longBody`/`validMemo`, `jsonValid`/`jsonCheck`, `validJSONValue/String`, `validJSONNumber`, `topMember`, `withoutTopMember`, `memberOf`, `keyIs`.
- `internal/gateway/video.go` 233-400, 440-600 — `readFilming`, `parseSeconds`, `decimalNumber`, `pixelResolution`, `volcengineVideoBody`, `videoID`, `signer`, `idPart`, `parseVideoID`, `videoState`, `videoObject`, `videoMaker`, `remoteVideoID`, `theirVideoID`.
- `internal/gateway/draw.go` 640-700, 1760-1800 — `readDrawing`, `readPart` call sites, the `n` clamp (`maxDrawings`), `eachDrawing`.
- `internal/gateway/continuation.go` 70-140, 340-690 (supporting; the SSE re-encoder) — `geminiEnds`, `geminiKin`, `saysItsEnd`, `continuation.unecho`, `streamTranslated` end to end.
- `internal/gateway/gateway.go` 5060-5330, 5355-5445, 6350-6700, 6781-6860 (supporting) — the passthrough relay and its tidy pipeline plus exit-path flushes, `streamFailure`, `eventStream`, `fits/usable`, `translate`, `emptyFails/repliesCounted/uncounted/answersNothing`, `pathOf/upstreamPath/parse`, `decoderFor`, `keepThroughSilence`.

COVERAGE: read 16 files, 15507 non-test LOC in scope; skipped ~9600 (the request-construction halves of chat/responses/anthropic/gemini/codeassist/draw/video and fastjson 1200-1591, which no stream path in this lane touches), plus continuation.go and gateway.go read only around the relay/geminiEnds points above; gw-core and gw-subs files excluded by lane.

## Findings

### F1 [P2] A model's drawn image never reaches a Chat client: the Chat encoder and renderer have no image case
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/chat.go:1594
- code:
```go
		saidError(failed, ev.Said)
		e.w.event("", map[string]any{"error": failed})
	}
	e.col.add(ev)
}
```
- trigger: a Chat Completions request (`"stream":true` or not) whose upstream reply carries an image part — i.e. an image model decoded by `codeAssistDecoder`, which emits `Event{Kind: KImage, Name: p.InlineData.MimeType, Text: p.InlineData.Data}` (codeassist.go:1237), or Gemini's, translated to a Chat client through `translate(from=Chat, to=CodeAssist)`.
- effect: the client receives a normal, successful turn with nothing in it. Streamed: `chat.completion.chunk`s with only the empty role delta, then `finish_reason:"stop"`, a zeroed usage chunk and `[DONE]`; whole: `content: null` with `finish_reason:"stop"`. The generated image (a paid image generation) is dropped entirely — `collector` keeps it in `Result.Parts`, but neither `chatEncoder.event` nor `renderChat` writes it.
- why: the IR has an image event (`KImage`, ir.go:324) and `saidAnything` counts a non-thinking part as "something said" (ir.go:687-693), so the reply is treated as non-empty and delivered as success while its only content is discarded. Every encoder but Chat's handles the event (gemini.go:676, and the drawing route's collector at draw.go:1744); `case KImage` appears nowhere in chat.go or responses.go.
- fix: add `case KImage: e.chunk(map[string]any{"content": "![" + ev.Name + "](data:" + ev.Name + ";base64," + ev.Text + ")"}, nil, nil)` to `chatEncoder.event`, and the same part as text in `renderChat`.
- evidence: `cd src/internal/gateway && grep -rn 'KImage' *.go | grep -v _test` → only codeassist.go:1237, draw.go:1744, gateway.go:6650, gemini.go:676, ir.go:324/655, translated_probe.go:31; no hit in chat.go or responses.go. verified-by: grep
- confidence: high on the drop; medium on how often the Chat route is pointed at an image model (that routing table is gw-core's).

### F2 [P2] chatTidy's mends are byte-exact on compact JSON, so a relay that writes `": "` separators is never mended
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/chattidy.go:80
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
- trigger: the upstream writes its chunks with a space after the colon — `data: {"choices": [{"delta": {"reasoning_content": ""}, "finish_reason": ""}]}` — as a Python `json.dumps` default server does. `needsTidy(line)` (chattidy.go:128-136) then finds none of its four compact patterns, `chatTidy.line` returns the line untouched (chattidy.go:224), and only the separator-agnostic `tool_calls` term still catches tool-call chunks.
- effect: the three client breakages the mends exist for come back for such an upstream: Qoder takes each empty `reasoning_content` for the thinking restarting and reads the reply one word to a line; the AI SDK rejects every chunk that carries `"finish_reason": ""` (a strict client's whole reply, since most chunks carry it); the grok CLI merges a repeated `{"name": ""}` and fails the tool call with "Tool not found". Magnitude is per chunk, so a streamed reply can be unusable end to end.
- why: the mends are content tests written against one serialization form, while the shape they match is a JSON value — `line()` already compares values separator-agnostically (`string(delta["reasoning_content"]) == `""``, chattidy.go:261), so only the gate is too narrow. The file itself proves the gap is an oversight: `contentParts`/`contentPartsSpace` cover both forms of one pattern and no other pattern got the spaced variant.
- fix: make `needsTidy` separator-tolerant for the three string shapes (`"reasoning_content"`, `"finish_reason"`, `"name"` matched on the key alone and decided by value in `line`), as `toolCallsField` already does.
- evidence: `cd src/internal/gateway && grep -rn 'reasoning_content": "' *.go` → chattidy.go:12, the rationale comment, and ollama_chat_test.go:118; no production constant holds the spaced form. `sed -n '79,87p;126,136p' chattidy.go` shows the seven constants and the gate. verified-by: read, grep
- confidence: medium — the defect is certain, the reach depends on which real relays pretty-print (WorkBuddy's and Qoder's wire form is not in this tree).

### F3 [P3] Held-back text is dropped when a Chat stream ends at `[DONE]` instead of a finish_reason
- file: /tmp/magpie-audit-Pzn4fNp7/src/internal/gateway/chat.go:1276
- code:
```go
func (d *chatDecoder) decode(data string, emit func(Event)) error {
	if strings.TrimSpace(data) == "[DONE]" {
		d.giveHeld(emit)
		return nil
	}
```
- trigger: the reply's text begins with something that could be the start of a thought tag (`<`, `<t`, `<th`, `<thi`, `<think` without the close) or, inside a `<think>` block, ends mid-close-tag or mid-backtick-run, and the stream's last data line is `[DONE]` with no chunk carrying `finish_reason` — e.g. a relay that only sends `finish_reason` on some turns, or a cut-then-closed stream that ends cleanly.
- effect: the bytes `chatDecoder` holds in `d.lead` (a thought tag's open prefix, at most 9 for `</thought>`, or a run of backticks) are never emitted: the client's reply is silently short by that text, with no error and no retry, and what was already reported as thinking stays as it was.
- why: `d.end(emit)` is the only thing that flushes `d.lead` (chat.go:1263-1273) and the `[DONE]` branch — the end of a Chat stream — is the one exit that does not call it; the finish_reason path (chat.go:1380-1384) and the tool-call path do. `giveHeld` releases held *tool calls* only.
- fix: call `d.end(emit)` next to `d.giveHeld(emit)` in the `[DONE]` branch.
- evidence: `grep -n 'd.end(emit)' chat.go` → chat.go:1371, 1381 only (both inside `decode`'s choice loop, after a tool-call delta and after `FinishReason != ""`); the `[DONE]` branch at 1276-1279 has neither. verified-by: read, grep
- confidence: high on the mechanism, low on frequency (needs text shaped like a tag prefix plus a stream without a finish_reason).

## Checked, no defect

- `eventStream` and non-2xx bodies (`gateway.go:5388-5406`, call sites 5092/6504, `continuation.go:523`): both stream relays test `res.StatusCode >= 400` and read at most 1 MiB before any decoding, so an error body is never parsed as a stream; a non-SSE 2xx body is read whole and reported as "did not stream".
- `keepThroughSilence` (`gateway.go:6855-6895`): `stop` closes `done` and waits on `over`, so the keepalive goroutine is out of `keep()` before the post-loop `enc.finish()`, even though those writes do not take the loop's mutex — no concurrent write to `sseWriter`, no goroutine leak.
- `readSSEAlive` (`sse.go:22-60`): `bufio.Scanner` splits on newlines only, so a read that ends mid-frame or mid-rune is buffered, not split; the final `flush()` before `sc.Err()` delivers a last event that arrived without its blank line; comments reset the client's quiet clock via `alive`.
- `chatTidy` exit paths (`gateway.go:5244-5290`): after the read loop breaks — EOF, read error, write error — `tidy.flush()`, `written.flush()`, `whole.flush()`, `search/sigs/ints/ids/phases.flush()` all run, so a final line with no newline and every held event still reach the client.
- `anthropicDecoder` state maps (`gateway.go:6803`): `decoderFor` builds it as `&anthropicDecoder{server: map[int]bool{}, whole: map[int]bool{}}`, so the unconditional `d.server[ev.Index] = …` in `decode` cannot panic on a nil map.
- `wholeAsStream` input (`whole_stream.go:17-42, 66-79`): `wholeEvents` returns nil unless the body is that protocol's own reply object (`object:"response"` / `type:"message"` / non-empty `choices` with a `message`), so an arbitrary body is never re-framed as a stream.
- `videoMaker` id handling (`video.go:460-477`): `idPart = ^[A-Za-z0-9_-]+$` is required of every id part that goes into a vendor URL, so a forged `video_…` id cannot traverse or redirect the path; `n` in a drawing request is clamped to `maxDrawings = 4` before `eachDrawing` fans out (draw.go:726, 1770-1786).
- `fastjson` string reader (`fastjson.go:675-780, 1006-1048`): surrogate pairs, lone surrogates and invalid UTF-8 become U+FFFD as `encoding/json` does, `\uXXXX` is validated in the grammar pass, and any shape it declines falls back to `json.Unmarshal` with the destination zeroed — the `\'` escape `unquoteJSON` accepts cannot arrive, since `jsonCheck` rejects it first.

## Open questions

- `wholeAsStream` sets `sw.begun = true` without writing the status line or headers (`whole_stream.go:22-26`, "the 200 and its headers went out with the keepalives"). Whether `holdWriter` can reach it before any keepalive began — which would send `data:` lines with a sniffed `text/plain` Content-Type, no `X-Accel-Buffering` and no flush — needs fallback.go (gw-core lane) plus a live cache-miss request.
- F1's severity depends on whether any Chat route can select a model whose reply carries an image (Antigravity/CodeAssist image models). That routing decision lives in gw-core, which this lane did not audit.
- Whether the real WorkBuddy/Qoder relays emit `": "` separators (F2) cannot be settled from this snapshot; it needs one captured upstream chunk.
