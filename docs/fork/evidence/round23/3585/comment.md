`cued` lowercased the 40 bytes before a value and asked `strings.Contains`, so any substring counted: `tool_call`/`callback` cued a phone ("call"), `telemetry`/`hotel` cued "tel", `monkey`/`keyboard` cued a secret ("key").

Fixed on [fix/redact-cue-words](https://github.com/skygunner/magpie/tree/fix/redact-cue-words) (`e60d14a8`, one commit on `01468003`): a cue now has to start a word (start of the window, a non-word byte, or a camelCase join) and end one (end of the window, a non-word byte, or a camelCase/underscore join), matched on the text as written. Because a cue that follows an underscore is inside the identifier, `api_key` and `apikey` were added as cues of their own, so `api_key: …` and `apiKey = …` still cue.

RED -> GREEN, re-run independently here:
- clean base `01468003` with only the new test: all six false cues are reported, `"tool_call " cued true, want false`, `"callback "`, `"telemetry "`, `"hotel "`, `"monkey "`, `"keyboard "`.
- at the head: `TestModelCueIsAWord` passes, with the true cues the issue listed (`phone:`, `tel:`, `api_key`, `apiKey`, `手机`) still cuing.
- whole `internal/redact` package passes; `go vet` and `go build` for linux, windows and `-tags nogui` pass; `gofmt -l` is empty.

One consequence worth naming: a snake_case name whose cue is only its last segment - `OPENAI_API_KEY`'s `key` - no longer cues the local model, because that `key` follows an underscore and is not a word of its own. Its `KEY=` context still masks it through the rule, and `api_key`/`apikey` are cues, so the case in your issue is unaffected. Not checked: the real local model over the 3 MB corpus, and a cue cut by the window's start byte.