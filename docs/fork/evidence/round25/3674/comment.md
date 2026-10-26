Reproduced and fixed on branch [`fix/r25-usage-dedup-session-file`](https://github.com/skygunner/magpie/tree/fix/r25-usage-dedup-session-file) (base 19253f67bca2310c32739a6eaac88546b3574dbc, single commit 18006848027b956553615e5dd462a2e6580cb92d).

## Root cause

Correlation was gated on Codex in two places, although dsh's session files contain a per-call id:

1. **The DSH parser dropped the id** (`internal/sessions/dsh.go`): `dshResponse` decoded only `model`/`provider`/`responseModel`. There was no field for `data.message.source.replayState.response.responseId`, so `sessions.Call.ResponseID` remained empty.
2. **The ledger retained ids only for Codex and paired only Codex rows** (`logRecord` in `internal/usage/ledger.go` copied `ResponseID` only when `c.Agent == "codex"`; `codexResponseMatches` and `codexChunkResponsePairs` in `internal/usage/codex_identity.go` both required the agents on both sides to be `codex`). dsh sends no session header, so the session/token fallback could not match it. The same physical call therefore appeared twice on the Requests page: once through, once direct.

## Changes

- Added ``ResponseID string `json:"responseId"` `` to `dshResponse` and carried it into `sessions.Call.ResponseID` (`internal/sessions/dsh.go`, `calls_dsh.go`).
- `logRecord` retains `ResponseID` for any call that supplies one; the compaction marker remains Codex-only.
- Extended unique response-id pairing in `codexResponseMatches` and `codexChunkResponsePairs` to same-agent dsh calls alongside codex, retaining the existing rejected, reported-source and bidirectional uniqueness guards. Calls dsh made directly, which magpie never saw, still count as direct; other sources such as ccs-* are unaffected.

## RED→GREEN

Added `TestDshCallThroughMagpieCountedOnce` (`internal/usage/dsh_match_test.go`), and updated `TestLogRecordKeepsAResponseIDAnyAgentNamed` and the responseId fixture assertion in `TestDshCalls`. On a clean base copy (`git archive <base>` plus only the test files):

- `TestDshCallThroughMagpieCountedOnce`: FAIL — `the call the gateway served is counted again from dsh's session file` and `the request page paired 0 rows by response id, want 1` (exit 1)
- `TestLogRecordKeepsAResponseIDAnyAgentNamed`: FAIL — `dsh's response id was lost from its usage row` (exit 1)
- `TestDshCalls`: FAIL — `dsh-gw's own call: [{ResponseID: ...` (exit 1)

All three PASS on the fix commit (exit 0).

## Package verification

- `go test ./internal/sessions/ ./internal/usage/ -count=1`: ok 6.135s / ok 10.856s, exit 0
- `gofmt -l internal/sessions internal/usage`: no output
- `GOTOOLCHAIN=go1.26.8 go vet ./...`: exit 0
- `go build ./...`, `GOOS=linux GOARCH=amd64 go build -tags nogui ./...`, `GOOS=windows GOARCH=amd64 go build ./...`: all exit 0
- `git rev-list --count <base>..HEAD` = 1; the remote hash from `git ls-remote` matches the local hash

## What was not tried

- The optional `{tokens, model, ±2 s}` fallback pairing in issue point 3, for agents that report no id, was not implemented; this fix uses the exact id path.
- No live dsh + workbuddy account run was performed. Verification used synthetic fixtures (including responseId pairs constructed from the report) and the existing Codex regressions to check that the guards did not regress.
- The provider-name skip logic in `dshGatewayProvider` was left unchanged: matching names are still skipped directly; pairing covers cases the names cannot identify.
