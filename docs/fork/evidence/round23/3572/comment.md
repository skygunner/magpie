Root cause is exactly what you pointed at: the one-time recovery only existed for Vertex AI.

`internal/gateway/gateway.go` gated the retry on `p.IsVertex()`, and the matcher in `internal/gateway/vertex.go` only knew `Invalid thought signature`. Antigravity goes through `provider.CodeAssist` and says `Corrupted thought signature`, so the 400 went straight back to the client even though the Code Assist path can send that history unsigned (`skipSignature`).

Fixed on [fix/antigravity-signature](https://github.com/skygunner/magpie/tree/fix/antigravity-signature) (`043636a3`, one commit on today's `main` `01468003`):

- the retry condition is now `p.IsVertex() || to == provider.CodeAssist`;
- the matcher accepts `(?i)(invalid|corrupted) thought signature`;
- `withoutSignatures` now clears signed text parts as well as tool-call signatures, on a copy, so the original request and the stored session history keep them. Signed text parts only reach the vendor from a Gemini/Responses client; the Code Assist envelope already drops thinking parts for an Anthropic client.
- boundaries unchanged: only that 400, only before anything was streamed to the client, one retry, and a request the target accepts keeps its signatures.

RED -> GREEN, re-run independently here:
- clean base `01468003` with only the new test file: `TestAntigravityRetriesCorruptedSignature` fails with your literal error, `400 Antigravity: Corrupted thought signature.`
- at the branch head: that test passes, along with `TestAntigravityDoesNotRetryOtherBadRequests` (another 400 on the same path is the client's - one request only) and the existing `TestVertexRetriesOnlyForVertex`, which checks a same-provider continuation still carries its signature.
- full `go test ./internal/gateway`, `go vet` and `go build` for linux, windows and `-tags nogui`: green. One pre-existing failure, `TestClaudeClientCacheAbsentUnchanged`, is red on the clean base too.

Honest limits: no real Google call. The Antigravity upstream in the test is a local server answering with the refusal in your report, so the retry being *accepted* by the real Code Assist service is not observed here. The text-part clearing is exercised by the same code path but only the tool-call case is asserted end to end.