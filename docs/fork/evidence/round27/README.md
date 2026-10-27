# Round 27 — twenty upstream issues, 2026-10-27

Base for every branch: `6eb510f7` (upstream `main` when work started). One branch, one commit, one comment in the issue's own language (16 en / 4 zh: #2789, #3137, #3821, #3826). No pull request: this repository takes no external PRs.

| issue | branch | head | comment |
|---|---|---|---|
| #1903 | `fix/1903-setaside-leftbehind` | `76dc5c2e` | https://github.com/yetone/magpie/issues/1903#issuecomment-6438165549 |
| #2099 | `fix/2099-risk-wrap` | `68bbfcf5` | https://github.com/yetone/magpie/issues/2099#issuecomment-6438166156 |
| #2541 | `fix/2541-queue-test-budget` | `7a327208` | https://github.com/yetone/magpie/issues/2541#issuecomment-6438166795 |
| #2746 | `fix/2746-unlisted-by-group` | `e678dbc5` | https://github.com/yetone/magpie/issues/2746#issuecomment-6438167437 |
| #2755 | `fix/2755-led-format-tokens` | `b20f17d5` | https://github.com/yetone/magpie/issues/2755#issuecomment-6438168091 |
| #2761 | `fix/2761-filememo-clock-seam` | `de2c6563` | https://github.com/yetone/magpie/issues/2761#issuecomment-6438168723 |
| #2789 | `fix/2789-textcall-xml` | `1734b21d` | https://github.com/yetone/magpie/issues/2789#issuecomment-6438169361 |
| #3137 | `fix/3137-proxy-advice-address` | `a9213cae` | https://github.com/yetone/magpie/issues/3137#issuecomment-6438169999 |
| #3278 | `fix/3278-day-window-budget` | `1386690f` | https://github.com/yetone/magpie/issues/3278#issuecomment-6438170609 |
| #3738 | `fix/3738-free-model-rest-id` | `499f0249` | https://github.com/yetone/magpie/issues/3738#issuecomment-6438171195 |
| #3785 | `fix/3785-pool-tooltip` | `a5f315a3` | https://github.com/yetone/magpie/issues/3785#issuecomment-6438171807 |
| #3792 | `fix/3792-chat-error-no-start` | `b16a0651` | https://github.com/yetone/magpie/issues/3792#issuecomment-6438172447 |
| #3793 | `fix/3793-as-sent-spawn-agent` | `f0d7a9cd` | https://github.com/yetone/magpie/issues/3793#issuecomment-6438173089 |
| #3805 | `fix/3805-ready-not-models` | `848a87ef` | https://github.com/yetone/magpie/issues/3805#issuecomment-6438173724 |
| #3812 | `fix/3812-workdir-enterable` | `e2958536` | https://github.com/yetone/magpie/issues/3812#issuecomment-6438174287 |
| #3814 | `fix/3814-inline-object-bytes` | `6276ec7d` | https://github.com/yetone/magpie/issues/3814#issuecomment-6438174941 |
| #3816 | `fix/3816-plan-quota-readat` | `3c7725cb` | https://github.com/yetone/magpie/issues/3816#issuecomment-6438175575 |
| #3821 | `fix/3821-hooks-discover` | `38b8f9b9` | https://github.com/yetone/magpie/issues/3821#issuecomment-6438176207 |
| #3826 | `fix/3826-zcode-levels-order` | `f26769d7` | https://github.com/yetone/magpie/issues/3826#issuecomment-6438176803 |
| #3828 | `fix/3828-opencode-bin-path` | `127fee62` | https://github.com/yetone/magpie/issues/3828#issuecomment-6438177401 |

Two branches carry more than one commit, because the first version's own red caught a regression the fix itself introduced:

- `fix/3814-inline-object-bytes` — `ceb41d44` (the reported shape), `0bbb64ee`, `6276ec7d`. The span rule that keeps an inline object inline
  also caught the file's **root**, so a config magpie writes from nothing came out on one line (`TestZCodeKeepsRemoteAddress`,
  `TestRemoveServers`); the root keeps the file's style, and the span rule holds only for an object added inside another object.
- `fix/3738-free-model-rest-id` — `a97b86a2`, `499f0249`. Asking `FreeModel` for every candidate reordered the searcher's model pick and
  broke `TestAutoVisionDoesNotBuildListsAgain`; it is now asked only when there is a window to override.

## What was measured

- **Targeted red-green** per issue: the command and its failing line are in each commit body and quoted in each comment; every one was re-run
  at the pushed head before the comment went out.
- **Full suite** at the merge of all twenty branches plus base (`suite-r27-merged`, head `f8ed0033`):
  `go test -tags nogui ./... -count=1 -timeout 45m` → 69 packages ok, 6 red:
  `internal/agent` `TestAgyShell`, `internal/edit` `TestWriteAtomicReadOnlyHardLink`,
  `internal/gateway` (`TestClaudeClientCacheAbsentUnchanged`, `TestCodexFittedImagesPastTheBudget`, `TestMarshalRawIsEncodingJSON`,
  `TestServeWSLListensBesideLoopback`, `FuzzMarshalRaw`), `internal/plugin` `TestPluginSOCKSProxy`,
  `internal/provider` `TestUnreadLoginsWriteErrorsKeepCredentials` (a synctest fatal), `internal/testenv` `TestProgram`.
  The same six packages fail with the same test names at base `6eb510f7` on this machine (umask 0027), run from the same worktree:
  `suite.log` is the merged run, `base-small.log` and `base-gateway.log` the base runs of exactly those packages. `internal/gui` is green.
- **GUI tests** in a throwaway container on `docker1` (`mcr.microsoft.com/playwright:v1.62.1-noble`, the repo's Playwright version),
  assets exported from the branch, never the working tree:
  - #2099 `risk-wrap.test.cjs` (new): red at 360 px in German (the warning 45 px under its line), 18/18 green on the branch across engines, languages and widths.
  - #2746 `unlisted-models.test.cjs`: red (the note blamed the provider), 4/4 green; the i18n-keys check passes with the new wording in all five languages.
  - #2755 `usage-ttft-rounding.test.cjs`: red on base assets, 8/8 green on the branch.
  - #3785 `routing-pool-tooltip.test.cjs` (new): red then green 4/4; the neighbouring routing tests still pass.
  - #3805 `routing-provider-off.test.cjs`: red (a ready member greyed), 20/20 green.
  - #3278 day-window test: red with `.rt-req{min-height:66px}` injected (what the mixed state load produced), 30/30 green without it.
  - `TestCoreScriptBudget` unchanged on every GUI branch.
- `gofmt -l` silent, `go vet -tags nogui` clean, `go build -tags nogui ./...` ok on every branch.

## Not done

- **#3808** was dropped: a working branch is already posted in its thread.
- **#3810** and **#3464** were already fixed upstream (`3ec3cb70`, `3a47e377`).
- **#3834** and **#3835** need designs of their own rather than a patch.
- **#3820, #3760, #3822, #3206, #3166, #2585, #2282, #2101, #2665, #2945, #3449** are product calls for the maintainer, not reports with a
  single right fix.
