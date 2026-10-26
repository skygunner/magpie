Confirmed: `kimiSearch`, `wbSearch` and `askSearchAPI` send their own HTTP request, never go through `s.serve`, and so never reach the row writer - Usage > Requests had nothing for a search a user's plan or search key paid for.

Fixed on [fix/3528-search-usage-rows](https://github.com/skygunner/magpie/tree/fix/3528-search-usage-rows) (`f6064d5d`, on `c0985122`): each of the three paths writes one row through a shared `logSearchCall` - `Kind: web_search`, the provider's id (or the search API's `Name()`), host, status, milliseconds and the error when it failed - on success and on failure alike, via a `defer`.

On the decision the issue leaves open: the row is counted as a call, like the model-search `web_search` row, but its tokens, cache and cost are all zero, so no total changes; nothing in the counting path was altered. One consequence worth naming: a turn that searched now has one more ledger row, exactly as a turn with a model search always had, so `usage_cache_zero_split_test.go`'s expectation now looks at the turn's own row.

RED -> GREEN, re-run independently here:
- clean base `c0985122` with only the new test: all three subtests fail with `0 rows written, want 1` (Kimi Code's search service, WorkBuddy's search service, a search API).
- at the head: `TestSearchesWriteARequestRow` passes and the surrounding search tests still pass (62 in the filtered run). Full `go test ./internal/gateway` (261.8s) fails only `TestClaudeClientCacheAbsentUnchanged`, red on the clean base too; `go vet`, the three builds and `gofmt -l` are clean.

Limits: the Recent-calls ring is not written (it wants an agent protocol and a model, which these calls have not); a path that fails before sending (masking, WorkBuddy's `Sign`) leaves a row with status 0; only in-package fakes were used - no real Kimi, WorkBuddy or search-API key was called.