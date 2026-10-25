# Round 22 evidence (2026-10-25/26)

Six one-issue branches off upstream/main `3173440c`, each with its own
delivery comment, all read back through the API:

| issue | branch | head | comment |
|---|---|---|---|
| 3597 | fix/redact-hex-count | 50f6e8b5dea715e59f400a89541be728eb9ed1a5 | 6400908682 |
| 3601 | fix/redact-repeat-secret | 08352789de1b17e35e05f374adf159ec2d914c5f | 6400908955 |
| 3598 | fix/library-instructions-no-agent | 8977ba980e7206aea8383fe863c630fd7928ebf8 | 6400912188 |
| 3619 | fix/sophnet-model-429 | 466a153cba7865a6b5e974298177ff5a630756bd | 6400912438 |
| 3629 | fix/masking-unit-and-bytes | 9e906567f5be2f3f6852f23125ec2c3116284d6c | 6400915163 |
| 3595 | fix/agy-wsl-sessions | 96311a8ae26114360275f2a95f8b8bb549eeb227 | 6401147159 |

- comment-readback.json: six receipts, author/issue/UTC/UTF-8 body SHA256
  each, read back 2026-10-25T17:37-17:59Z.
- Per-package (nogui, GOTOOLCHAIN=go1.26.8, umask 022): internal/redact
  82 ok at 50f6e8b5 and 83 ok at 08352789/9e906567; internal/cli +
  internal/library 789 ok at 8977ba98; internal/gateway 5172 ok with the
  known #3280 red at 466a153c; internal/sessions 649 ok at 96311a8a.
  gofmt clean (skillgit_test.go's upstream pre-existing unformatted state
  untouched); go vet + go build clean on linux (plain and nogui) and
  windows at each head. Darwin cross-build fails in wails v3.0.0-beta.24
  itself, identically on the clean base ef3b9f85 — environmental, left
  out of the claim.
- full-suite-agy-head.log: full `-tags nogui ./...` at 96311a8a —
  **67 ok / 3 package FAIL**. All three named and attributed: gateway
  TestClaudeClientCacheAbsentUnchanged (#3280, red on a clean base this
  machine), gui TestCoreScriptBudget (red on clean ef3b9f85, verified
  this round), plugin TestPluginSOCKSProxy (#2207, local Bun 1.4.3). A
  provider synctest-bubble crash appeared in one loaded run of both the
  head and the clean base and did not appear in either later run — a
  load flake, not a regression.
- Boundary: no real vendor account was called from here; #3629's bare
  `systemctl status name@inst.service` form (no path) is recorded as
  left for the maintainer; #3595 was not tried against a real WSL agy.

Addendum (same session, an hour later): #3628, fix/pii-upgrade-race
80cea91a8d58df163c98f3ec27f9749486a3fc17, comment 6401392845 (read back,
receipt in comment-readback.json). The reporter's command
`go test -tags nogui -race -count=2 ./internal/pii/` was red on this box
on the clean branch point (three DATA RACE warnings,
TestOldServeRunsUntilUpgraded and TestWarm failing) and is green at the
head, -count=3 with it; plain suites and linux+windows vet clean.

Round-22 scope note: another live session owned #3035/#3635/#3636/#3624
and their uncommitted dev-tree draft; this round did not touch those
branches or issues.
