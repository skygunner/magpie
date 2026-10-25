# Round 20 evidence (2026-10-25)

Issue #3392's fourth round — bf84dd4f on fix/issue-batch-1024pm, the
reporter's 10:08Z retest still answering the same `/required: null` —
plus the closeout the third round (c172b65d, comment 6395007475) was
missing.

- red-at-base.log: TestXAITakesANullRequiredToolSchema, extended with two
  MCP-flavoured tools (one schema's null under an anyOf branch, one under
  a tuple items), run at c172b65d — red with the reporter's wording
  verbatim, the two nulls visibly surviving in the body xAI is sent; the
  root and properties nulls of the earlier tools are already gone there,
  which is why the walk had to reach arrays.
- suite-head.log / suite-base.log: full `-tags nogui` internal/gateway at
  bf84dd4f and at c172b65d, same sandbox HOME, same session, ~254s each.
  Four reds on each side, the same four on both (TestClaudeClient
  CacheAbsentUnchanged, TestCodexFittedImagesPastTheBudget,
  TestMarshalRawIsEncodingJSON, FuzzMarshalRaw) — the set c172b65d's own
  commit message recorded this morning. Zero new failures from the walk.
- Targeted: `go test ./internal/gateway -run 'TestXAI|TestXai|TestCustom'
  -count=1` → 23 ok at bf84dd4f. `gofmt -l` clean; `go vet ./...`,
  `GOOS=linux go vet -tags nogui ./...`, `GOOS=windows go vet ./...` clean;
  `go build`, `GOOS=linux go build -tags nogui`, `GOOS=windows go build`
  clean.
- comment-readback.json: both delivery comments read back through the API
  (author, UTC, UTF-8 body SHA256). 6395007475 was posted by the earlier
  session today and never recorded; recorded here with this round's
  6397710434.
- Push read back: `git ls-remote origin fix/issue-batch-1024pm` =
  bf84dd4f86d236771ef3e8a3751a8b91f4a9c1a8 after a plain push from
  c172b65d (no force, no branch touched but this one).

Boundary: the real xAI upstream was not called from here; the reporter is
asked to confirm which commit their build carries and, should the error
persist at bf84dd4f, to paste the failing tool's input_schema.
