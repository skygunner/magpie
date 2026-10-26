# Round 26 — twenty upstream issues, 2026-10-27

Base for every branch: `6afbe3f3` (upstream `main` when work started). One branch, one commit, one comment in the issue's own language. No pull request: this repository takes no external PRs.

| issue | branch | head | comment |
|---|---|---|---|
| #3188 | `fix/3188-sub2api-percent` | `0817d2e9` | https://github.com/yetone/magpie/issues/3188#issuecomment-6429785328 |
| #3308 | `fix/3308-panel-subagent-effort` | `2a75e5d3` | https://github.com/yetone/magpie/issues/3308#issuecomment-6429785833 |
| #3320 | `fix/3320-rtk-verdict` | `ff8cdd64` | https://github.com/yetone/magpie/issues/3320#issuecomment-6429784843 |
| #3324 | `fix/3324-history-viewer-day` | `0765619b` | https://github.com/yetone/magpie/issues/3324#issuecomment-6429780051 |
| #3399 | `fix/3399-kimi-thinking` | `d19b48b8` | https://github.com/yetone/magpie/issues/3399#issuecomment-6429783755 |
| #3555 | `fix/3555-unrest-owner` | `1fe501d2` | https://github.com/yetone/magpie/issues/3555#issuecomment-6429784291 |
| #3669 | `fix/3669-minimax-cn-url` | `da00218e` | https://github.com/yetone/magpie/issues/3669#issuecomment-6429777197 |
| #3690 | `fix/3690-claude-room-freed` | `56b4f317` | https://github.com/yetone/magpie/issues/3690#issuecomment-6429778471 |
| #3698 | `fix/3698-prefix-off` | `6d7c04d0` | https://github.com/yetone/magpie/issues/3698#issuecomment-6429783205 |
| #3701 | `fix/3701-held-clause` | `5253cad3` | https://github.com/yetone/magpie/issues/3701#issuecomment-6429777819 |
| #3705 | `fix/3705-remote-key-limit` | `6c8bd9bc` | https://github.com/yetone/magpie/issues/3705#issuecomment-6429778980 |
| #3708 | `fix/3708-agy-settle` | `1c8f94de` | https://github.com/yetone/magpie/issues/3708#issuecomment-6429781699 |
| #3713 | `fix/3713-canonical-slug` | `129d94ae` | https://github.com/yetone/magpie/issues/3713#issuecomment-6429776643 |
| #3716 | `fix/3716-options-field` | `62bb7d58` | https://github.com/yetone/magpie/issues/3716#issuecomment-6429779510 |
| #3721 | `fix/3721-siwc-quota` | `20ac3a6c` | https://github.com/yetone/magpie/issues/3721#issuecomment-6429782671 |
| #3723 | `fix/3723-import-stamps` | `e232a777` | https://github.com/yetone/magpie/issues/3723#issuecomment-6429781209 |
| #3742 | `fix/3742-late-interrupt` | `fd442d4f` | https://github.com/yetone/magpie/issues/3742#issuecomment-6429780633 |
| #3751 | `fix/3751-codex-meta-length` | `7e061eda` | https://github.com/yetone/magpie/issues/3751#issuecomment-6429782208 |
| #3775 | `fix/3775-token-count` | `c4fd6c4b` | https://github.com/yetone/magpie/issues/3775#issuecomment-6429791341 |
| #3776 | `fix/3776-jp-phone` | `ab40c7ea` | https://github.com/yetone/magpie/issues/3776#issuecomment-6429790548 |

## What was measured

- **Targeted red-green** per issue: the command and its failing line are in each commit body and quoted in each comment; every one was re-run at the pushed head before the comment went out.
- **Full suite** at the merge of all twenty branches plus base (`suite-r26-merged`): `go test -tags nogui ./... -count=1 -timeout 45m` →
  70 packages ok, 5 red: `internal/edit` `TestWriteAtomicReadOnlyHardLink`, `internal/gateway` (`TestClaudeClientCacheAbsentUnchanged`,
  `TestCodexFittedImagesPastTheBudget`, `TestMarshalRawIsEncodingJSON`, `TestServeWSLListensBesideLoopback`, `FuzzMarshalRaw`),
  `internal/plugin` `TestPluginSOCKSProxy`, `internal/provider` (`TestUnreadLoginsWriteErrorsKeepCredentials`, a synctest fatal),
  `internal/testenv` `TestProgram`. The same five packages fail identically at base `6afbe3f3` on this machine (umask 0027);
  `internal/gui` is green.
- **GUI tests** in a throwaway container on `docker1` (`mcr.microsoft.com/playwright:v1.62.1-noble`, the repo's Playwright version),
  assets exported from the branch, never the working tree:
  - #3320 `library-rtk-gain.test.cjs`: red on base assets (the old verdict wording), 8/8 green on the branch (chromium + webkit, en/zh).
  - #3308 `claude-subagent-effort-row.test.cjs`: red (4 of 24, the no-model cases) on base, 24/24 green on the branch.
- `gofmt -l` silent, `go vet -tags nogui` clean, `go build -tags nogui ./...` ok on every branch.

## Not done

- **#3511** was dropped: a working patch (`hunt79-h-3511`, head `6c99a27f`) is already posted in its thread, and the base still carries the wall-clock read it fixes.
- **#3323** (Sessions tab in the viewer's zone) was not attempted: the day/hour buckets are computed at parse time in `internal/sessions/sessions.go`
  and the cache is keyed by one zone, so plumbing `?tz=` through `stats.go` and the GUI would make rows outside the viewer's range disappear instead
  of moving them. A correct fix means storing absolute times (or UTC-hour buckets) and letting the cache hold more than one zone — its own branch, not a slice of this round.
