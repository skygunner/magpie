Took #3684. Branch: `fix/r25-empty-tool-calls-finish` on [skygunner/magpie](https://github.com/skygunner/magpie/tree/fix/r25-empty-tool-calls-finish), commit `91625dca0bb33ee696be639e659bdf4c8f1b6abd`, base `19253f67bca2310c32739a6eaac88546b3574dbc`.

## Root cause

`chatTidy` (streamed) and `chatWhole` (whole responses) only ever upgraded a `stop` finish to `tool_calls` when a choice delivered calls. The reverse never happened: an upstream that narrated a call and ended `finish_reason=tool_calls` with an empty, null or absent `tool_calls` array had the contradictory finish relayed as-is, and the streamed per-choice "called" state even recorded an absent/null/empty array as a call. This is the narration + phantom-finish sub-shape of the report (6 of 11 occurrences), matching the #636 class.

## Fix (3 files, +113/−4)

- `internal/gateway/chattidy.go`: a choice that delivered no calls now ends `stop` when the upstream ended `tool_calls`, streamed and whole. Streamed state is kept per choice; real native calls keep `tool_calls`, and calls parsed from text (textCallTidy, which runs after chatTidy) still end `tool_calls`. An unreadable `tool_calls` value is treated as unknown and left as it came.
- `internal/gateway/chattidy_phantom_finish_test.go` (new): regression through the real HTTP surface for streamed and whole replies, absent/null/empty/absent-field arrays, with and without tools offered; native and text-parsed calls keep their finish and arguments; per-choice finishes in one chunk follow their own calls.
- `docs/subsystems/gateway-routing.md`: documents the normalization and its limits.

## RED → GREEN

RED at the clean base (production untouched, only the new test present):

```
=== RUN   TestChatPhantomToolFinish/#00/stream
    chattidy_phantom_finish_test.go:29: finish "tool_calls", want stop; text "Checking." calls map[]
--- FAIL: TestChatPhantomToolFinish
--- FAIL: TestChatPhantomFinishPerChoice
FAIL    github.com/yetone/magpie/internal/gateway   0.153s
EXIT_CODE=1
```

GREEN at head (`GOTOOLCHAIN=go1.26.8 go test -race -shuffle=on -count=1 -run "TestChatPhantom|TestChatCallsEndToolCalls|TestTextWritten|TestChatTidy" -v ./internal/gateway`, exit 0, 3.397s).

Also run: `go vet ./...`, `go build ./...`, `GOOS=linux GOARCH=amd64 go build -tags nogui`, `GOOS=windows GOARCH=amd64 go build`, linux/windows `go vet`, `gofmt -l` (all exit 0). Full `internal/gateway -race`: one failure, `TestClaudeClientCacheAbsentUnchanged` (Claude Code client-cache prompt marks) — proven failing at the base with `chattidy.go` restored (exit 1), pre-existing and unrelated. gopls is not installed in this environment, so LSP diagnostics were not run; compile/vet/tests cover the change.

## Limits

- This corrects the finish flag only. It does not implement empty-turn retry/failover or any billing policy; the billed-but-empty sub-shape (5 of 11 occurrences, #3327's class, prior BLOCKED_ARCHITECTURE disposition) is not addressed and the issue stays open for it.
- No real upstream request was made against opencode-zen-free; the fixtures are the wire shapes from the report's log and prior tests, exercised through the gateway's real HTTP stack in a sandbox HOME.
