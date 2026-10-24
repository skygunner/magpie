# r11-review rework: behavior tests, 2026-10-24 (ZCode session 12)

Environment: Linux amd64, CachyOS, go1.26.8 (GOTOOLCHAIN), umask 022, temp git-archive trees only.

## Trees
- fork product (frozen): 14c1551e9672758c656ed2fd6b4db69ac1300b15 (branch fix/issue-batch-1024pm, unchanged this session; remote read back identical)
- upstream main tip: 9d643631 (beyond the review's 7d9165; fetched 2026-10-24)
- ggbdpq alternative: 69068eadfd60b98bb4f98b190243df6791759b8e (public branch hunt76-a-websearch-citations, cloned read-only to /tmp, never pushed to)

## 1. Reviewer's searchWith red test on fork 14c1551 (temp archive)
go test -tags nogui ./internal/gateway -run '^TestReviewR11SearcherExcerptSurvivesParsing$' -count=1 -v
FAIL — review_r11_searcher_excerpt_test.go:45: real searcher cited_text lost: {Title:Go URL:https://go.dev/dl/ PageAge: Cited:}
=> the fork branch's searchWith drops a searcher's real cited_text (known; superseded, see below)

## 2. Green controls on fork 14c1551 (product worktree, committed tests)
go test -tags nogui ./internal/gateway -run 'TestReviewMultipleSearchCitations|TestSearchCitedOnTheTextAfterIt|TestCustomToolRefusalWordings|TestCodexSocketMessageTooBigGoesOverHTTP|TestCodexModelListNativeCompactedAtTheThreshold|TestReasonixPluginsMCP' -count=1
ok — 7 passed (multi-search two_searches/later_empty_search, single search, custom refusal 400+422, socket too-big fallback, native compaction, reasonix MCP)

## 3. Upstream H20 own tests on 9d643631 (temp archive)
go test -tags nogui ./internal/gateway -run 'TestSearchCitedAsAnthropics|TestNoSearchNoCitations|TestCitesOnlyTheWholeURL' -count=1 -v
PASS / PASS / PASS

## 4. ggbdpq's three e2e fixtures on upstream 9d643631 (their test file copied into the temp tree)
TestWebSearchCitationsCarriedToText  FAIL
TestWebSearchCitationsWholeReply     FAIL
TestWebSearchWithoutExcerptsGivesNoCitations PASS
=> reproduced the reviewer's result on a newer upstream tip: the two contracts differ.
   H20 (1d6b72ada8c93ca05bb70d02ef77cfadf853361f) cites lines whose text names a
   whole URL, excerpts from the search model's own text.citations; ggbdpq 69068ead
   (and the fork's e5a05c44+14c1551) cite the searched pages from the searcher's
   excerpts regardless of whether the answer names the URL. On the issue's real
   traffic the model names URLs, and the reporter (HuHong-Tao, comment 6382090015,
   2026-10-24T08:38:27Z) verified 1d6b72ad on a real 0.1.1371 install end to end
   (both routers, stream + whole, DSH mapAnthropicResponse); the issue was closed.
   The fixtures' shape (answer without any URL) is where the two contracts part.

## Decision (NEXT-PROMPT M3.4)
Upstream covers the user's original path (reporter-verified on the real install).
No third product variant is made; the fork's #3447 unit is marked
superseded_by_upstream_independent (not pass/adopted), with the semantic
difference recorded as semantic_comparison_pending. The fork branch keeps
e5a05c44+14c1551 as a coherent alternative for anyone on that tree; its
searchWith excerpt gap stands documented by test 1, not fixed.
