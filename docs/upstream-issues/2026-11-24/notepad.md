# Magpie upstream follow-up notepad — incremental sweep 2026-11-24

Scope: the same 357 upstream issues tracked in the 2026-11-17 sweep.
The baseline watermark was `2026-11-20T03:00:34Z`. The fresh fetch in
`/tmp/magpie-issue-sweep.fsdUzXVi` produced all 1,071 JSON files (issue,
comments and timeline for each issue), with zero missing. Fetch started at
`2026-11-23T19:01:16Z` and finished at `2026-11-23T19:11:00Z`.

For the next sweep, use **`2026-11-23T19:01:16Z`** as the conservative
coverage/safety watermark, preserving overlap for events during fetch.

## Summary of Activity

- 13 new comments across tracked issues.
- 17 timeline events (closures, cross-references, commits).
- 2 new/updated fix branches pushed.
- 6 issue replies delivered and verbatim-verified via API readback.

## Delivered Fix Branches & Replies

- **#2785 — macOS window position restoration.**
  - **Issue/Root cause**: Reporter @pluo confirmed that after update/restart, `windowAt` was saved as `[2113, 44]` but overwritten by `[1065, 398]` (centered). On macOS startup, Wails' screen discovery can run after the first `putBack()`, causing `openAt()` to fail finding any screen and fall back to centering.
  - **Branch**: `fix/2785-window-place-before-screens` (`3e93c30ea`) rebased on `upstream/main` (`8d0226d9b`).
  - **Fix**: When Wails' screen cache is empty, read AppKit `visibleFrame` directly on the main thread and convert to logical points from primary top-left.
  - **Reply**: [skygunner comment 7111626119](https://github.com/yetone/magpie/issues/2785#issuecomment-7111626119) (SHA256 `1452f81335e28d69d6c10efd68163d8538b5af9bed422139baf5a999acc3f856`).
  - **Evidence**: `evidence/2785/red-fallback-ignored.log`, `evidence/2785/green-at-head.log`, `evidence/2785/baseline-main-red.log`.

- **#4953 — Codex translated turns prompt duplication on gpt-6.2.**
  - **Issue/Root cause**: Claude Code over `/v1/messages` onto `gpt-6.2-*` places system prompt in `input` developer item instead of `instructions`. `codexBody()`'s first-line pass-through (`firstLine(own) == firstLine(codex)`) didn't run because `own` was empty, adding duplicate official prompt (5,426 tokens vs 62 tokens on 5.6).
  - **Branch**: `fix/4953-translated-first-line` (`58f2c396d`) rebased on `upstream/main` (`8d0226d9b`).
  - **Fix**: When `instructions` is empty, check if input's first developer message starts with Codex's base prompt first line; if so, skip adding duplicate official prompt.
  - **Reply**: [skygunner comment 7112009303](https://github.com/yetone/magpie/issues/4953#issuecomment-7112009303) (SHA256 `9c4d13614fb3be62edda9da6a0ed18fb4b55975158c1b47c98368f145a9f1c96`).
  - **Evidence**: `evidence/4953/red-firstline-removed.log`, `evidence/4953/green-at-head.log`, `evidence/4953/baseline-claude-signin-red.log`.

- **#2729 — DeepSeek Search API & Flash search routing.**
  - **Reply**: [skygunner comment 7111673799](https://github.com/yetone/magpie/issues/2729#issuecomment-7111673799) (SHA256 `803d11b2a5d20e80...`). Confirmed #7426 evidence; branch `fix/2729-gateway-deepseek-s-search-api` (`a8de723e`) remains valid.

- **#3035 — SenseNova structured output refusal matcher.**
  - **Reply**: [skygunner comment 7111674365](https://github.com/yetone/magpie/issues/3035#issuecomment-7111674365) (SHA256 `44b341ee5fe350b1...`). Explained generic `formatRefusal` matcher and production integration line inclusion.

- **#3282 — Pi Google model integration.**
  - **Reply**: [skygunner comment 7111674783](https://github.com/yetone/magpie/issues/3282#issuecomment-7111674783) (SHA256 `d0c726cb9219165c...`). Addressed 4 points regarding session state requirement for KTextSig.

- **#3513 — ChatGPT Image model resolution per provider.**
  - **Reply**: [skygunner comment 7111675290](https://github.com/yetone/magpie/issues/3513#issuecomment-7111675290) (SHA256 `a59216033e39a18c...`). Clarified provider prefix routing behavior.

- **#5359 — Provider account header fields.**
  - **Reply**: [skygunner comment 7111675603](https://github.com/yetone/magpie/issues/5359#issuecomment-7111675603) (SHA256 `a6275558683be23f...`). Pointed to `fix/5359-plugin-account-header-fields` (`aca455708`).

## Closed & Upstream Absorbed Status

- **#3331**: Closed upstream by `yetone` with commit `5df6132e2` (co-authored by fork patch authors). Verified in `origin/integration/nightly`.
- **#4945**: Issue closed by reporter after upstream `562214933` / `91e814672`. Fork branch `fix/4945-note-apart` (`39c5413b3`) remains active for the `withNoteAnthropic` placeholder note placement.
- **#5150**, **#2541**, **#4603**: Fork branches remain open and active awaiting maintainer pickup.
