# Future-batch duplicate-search evidence gaps

## Scope and method

Compared `magpie-audit-20261029/FIX-MAP.md` and `docs/fork/2026-10-29-audit-v01957/FIX-MAP.md` byte-for-byte: identical. Batch 2/3 rows marked 有效 or 降级 P3 are live in this local map, not a live upstream status assertion. The audit report supplies the defect references. Reviewed `docs/fork/prior-comments.json` and `docs/fork/upstream-status.json` as historical delivery/comment ledgers, and the audit upstream drafts and evidence; these are not a complete upstream issue/PR index. A documented search means a finding-specific search with query and results, not merely “search before filing” or an issue reference. No finding-specific search record was located for the 63 rows below within these materials. This is a documentation gap, not proof that no duplicate exists. No GitHub search was performed.

Counts: 35 Batch 2 rows and 28 Batch 3 rows, 63 findings / 62 clusters (P2-37 and P2-38 share a cluster). Of these, 61 rows say new issue pending search; P2-1 and P2-5 already point to issues #3788 and #4101, but their mappings alone do not document a duplicate search. P3-2 is an internal duplicate merged into P3-21 and is not an additional live row; rejected items and Batch 1 are excluded.

The two issue references are recorded in the audit report lines 54, 58 and 190–191. Neither #3788 nor #4101 is present in the two historical JSON ledgers. Related, not asserted duplicates: prior-comments #3197 and its upstream-status unit concern severed Chat streams (relevant to P2-4); #3444 and its upstream-status unit concern oversized Codex WebSocket messages (relevant to P3-1). Their presence does not establish a search for either finding.

Queries below are proposed, not executed. Search issues and PRs, open and closed, in `repo:yetone/magpie`; each semicolon separates alternative queries. Examine candidate threads, comments, patches, assignees, and current upstream code before applying the four exclusion gates in the maps.

| Finding | Batch | Current reference (both FIX-MAPs) | Proposed searches |
|---|---:|---|---|
| P2-1 | 2 | gateway 组请求; 回贴 #3788; 有效 | "non-streaming" failover timeout; "first response" group timeout |
| P2-2 | 2 | response-body-deadline; 新 issue（开贴前查重）; 有效 | "response body" timeout stalled; "ResponseHeaderTimeout" stream |
| P2-4 | 2 | codex-stream-read-errors; 新 issue（开贴前查重）; 有效 | Codex "stream" "read error" usage; "codex_backend" interrupted |
| P2-5 | 2 | chat 多模态; 回贴 #4101; 有效 | "input_audio" "video_url" chat; "multimodal" silently dropped |
| P2-6 | 2 | chat-generated-images; 新 issue（开贴前查重）; 有效 | "KImage" chat response; "generated images" Gemini inline |
| P2-7 | 2 | codex-description-key-policy; 新 issue（开贴前查重）; 有效 | "thread_description" model key; "description" allowed models |
| P2-8 | 2 | codex-model-list-eviction; 新 issue（开贴前查重）; 有效 | "codexLists" cache; Codex model list eviction |
| P2-9 | 2 | affinity-capacity; 新 issue（开贴前查重）; 有效 | "affinity" 4096 cache; affinity session memory growth |
| P2-10 | 3 | affinity-lock-io; 新 issue（开贴前查重）; 降级 P3 | "stickOf" lock; affinity disk read mutex |
| P2-11 | 3 | search-key-cooldown-race; 新 issue（开贴前查重）; 降级 P3 | "search_keys" cooldown race; search key rest concurrent |
| P2-12 | 2 | providers-update-race; 新 issue（开贴前查重）; 有效 | "providers.json" concurrent update; provider settings lost update |
| P2-13 | 2 | settings-update-race; 新 issue（开贴前查重）; 有效 | "settings.json" concurrent update; "modelprefs" lost settings |
| P2-15 | 2 | plugin-hook-overload; 新 issue（开贴前查重）; 有效 | "onRequest" hook 128; plugin hook concurrent limit |
| P2-16 | 2 | plugin-auth-lock-errors; 新 issue（开贴前查重）; 有效 | "lockAuth" permission; plugin auth lock infinite |
| P2-17 | 2 | plugin-response-capture; 新 issue（开贴前查重）; 有效 | plugin gateway hook response spool; onRequest response capture performance |
| P2-18 | 2 | plugin-response-truncation; 新 issue（开贴前查重）; 有效 | plugin response hook 8 MiB; gateway response truncated success |
| P2-19 | 2 | gui-same-site-csrf; 新 issue（开贴前查重）; 有效 | CSRF "same-site" port; "Sec-Fetch-Site" localhost |
| P2-20 | 2 | gui-secure-cookie; 新 issue（开贴前查重）; 有效 | cookie Secure HTTPS; web login insecure cookie |
| P2-21 | 3 | gui-static-cache-invalidation; 新 issue（开贴前查重）; 降级 P3 | static assets provider cache invalidation; "provider.Changed" static |
| P2-22 | 2 | gui-request-body-limit; 新 issue（开贴前查重）; 有效 | API request body limit; backup import oversized memory |
| P2-23 | 2 | mcp-refresh-lock; 新 issue（开贴前查重）; 有效 | MCP refresh token lock; "mcpauth" concurrent refresh |
| P2-24 | 2 | usage-period-limit; 新 issue（开贴前查重）; 有效 | usage period huge allocation; "last:" range limit |
| P2-25 | 2 | usage-rolling-page-cache; 新 issue（开贴前查重）; 有效 | usage "last:24h" cache stale; rolling period requests cache |
| P2-26 | 2 | usage-lastseen-lock; 新 issue（开贴前查重）; 有效 | "LastSeen" usage lock; usage scan mutex |
| P2-27 | 2 | sessions-delete-rollback; 新 issue（开贴前查重）; 有效 | sessions delete rollback lost files; sessions trash restore failure |
| P2-28 | 2 | sessions-incremental-read-error; 新 issue（开贴前查重）; 有效 | sessions incremental read error duplicate; session offset token counts |
| P2-29 | 2 | sessions-delete-lock-io; 新 issue（开贴前查重）; 有效 | sessions delete lock cross filesystem; sessions copy blocking |
| P2-31 | 2 | library-put-rollback; 新 issue（开贴前查重）; 有效 | library skill put rollback; library restore move failure |
| P2-32 | 2 | update-signature-verification; 新 issue（开贴前查重）; 有效 | update signature verification SHA256; Linux Windows update authenticity |
| P2-33 | 2 | agent-disconnect-restore; 新 issue（开贴前查重）; 有效 | agent disconnect unstash failure; restore original config lost |
| P2-34 | 3 | dsh-env-key-preservation; 新 issue（开贴前查重）; 降级 P3 | "MAGPIE_GATEWAY_KEY" dsh env; dsh .env overwrite key |
| P2-35 | 2 | agent-applied-atomic-write; 新 issue（开贴前查重）; 有效 | "applied.json" atomic write; agent drift write error |
| P2-40 | 2 | route-turn-capacity; 新 issue（开贴前查重）; 有效 | "turnRules" 4096; routing session memory growth |
| P2-41 | 2 | catalog-atomic-write; 新 issue（开贴前查重）; 有效 | "models.json" atomic write; catalog corrupted empty sync |
| P2-42 | 2 | image-output-stat-errors; 新 issue（开贴前查重）; 有效 | "imagemcp" stat infinite; image filename permission loop |
| P2-36 | 2 | agents-initial-state-error; 新 issue（开贴前查重）; 有效 | agents page skeleton state error; "api/state" loading retry |
| P2-37 | 2 | gui-state-stale-read; 新 issue（开贴前查重）; 有效 | "loadState" stale state connect; GUI state refresh race |
| P2-38 | 2 | gui-state-stale-read; 新 issue（开贴前查重）; 有效 | "loadProvidersNow" stale; providers refresh concurrent save |
| P2-39 | 2 | mcp-signin-poll-errors; 新 issue（开贴前查重）; 有效 | MCP sign in polling errors; "Finish signing in" endless |
| P3-1 | 3 | codex-socket-frame-limit; 新 issue（开贴前查重）; 有效 | "SetReadLimit" Codex websocket; websocket unlimited frame |
| P3-3 | 3 | provider-account-limit-rest; 新 issue（开贴前查重）; 有效 | "LimitRestOff" account provider; account rest limit lost |
| P3-4 | 3 | quota-alert-atomic-write; 新 issue（开贴前查重）; 有效 | "usage-alerts.json" atomic; quota alerts corrupt |
| P3-5 | 3 | quota-card-cache-refresh; 新 issue（开贴前查重）; 有效 | "cacheCards" deleted account; quota cards stale refresh |
| P3-6 | 3 | codex-model-list-read-errors; 新 issue（开贴前查重）; 有效 | Codex model list "ReadAll" limit; models list truncated read error |
| P3-7 | 3 | chat-done-flush; 新 issue（开贴前查重）; 有效 | "giveHeld" "DONE"; chat stream done flush |
| P3-10 | 3 | archive-exclusive-name; 新 issue（开贴前查重）; 有效 | archive export filename race; export exclusive create |
| P3-11 | 3 | plugin-retry-after-config; 新 issue（开贴前查重）; 有效 | plugin failed retry config change; plugin state Failed restart |
| P3-12 | 3 | agent-applied-process-lock; 新 issue（开贴前查重）; 有效 | "applied.json" process lock; agent applied concurrent processes |
| P3-13 | 3 | usage-account-page-filter; 新 issue（开贴前查重）; 有效 | usage accounts pagination filter; usage account list current page |
| P3-14 | 3 | usage-cache-only-pricing; 新 issue（开贴前查重）; 有效 | usage cache only pricing; cached tokens zero input output cost |
| P3-15 | 3 | update-badge-polling; 新 issue（开贴前查重）; 有效 | "renderUpdateBadge" polling; update badge multiple timers |
| P3-16 | 3 | davsync-save-state-errors; 新 issue（开贴前查重）; 有效 | "davsync" "saveState" error; sync state atomic write failed |
| P3-17 | 3 | gateway-watchdog-lock-io; 新 issue（开贴前查重）; 有效 | "gatewayMu" watchdog; gateway watchdog lock blocking |
| P3-18 | 3 | model-list-benchmark-name; 新 issue（开贴前查重）; 有效 | "group/auto-model-499" benchmark; model list benchmark auto |
| P3-19 | 3 | usage-pair-cache-rebuild; 新 issue（开贴前查重）; 有效 | usage pair cache rebuild; usage pairing large logs memory |
| P3-20 | 3 | gui-login-rate-limit; 新 issue（开贴前查重）; 有效 | web login rate limit; login brute force gateway key |
| P3-21 | 3 | callback-address-allowlist; 新 issue（开贴前查重）; 有效 | "ownCallback" allowlist; callback token address allow list |
| P3-22 | 3 | netproxy-local-resolution; 新 issue（开贴前查重）; 有效 | ".local" DNS proxy; local address hostname resolution SSRF |
| P3-23 | 3 | maskrec-lock-io; 新 issue（开贴前查重）; 有效 | "maskrec" lock; masking records read write blocking |
| P3-24 | 3 | settings-unknown-fields; 新 issue（开贴前查重）; 有效 | settings unknown fields lost; settings json preserve keys |
| P3-25 | 3 | theme-refresh-lock-io; 新 issue（开贴前查重）; 有效 | theme refresh lock fontconfig; hyprctl theme blocking |
| P3-26 | 3 | wsl-command-cancellation; 新 issue（开贴前查重）; 有效 | "wslRun" cancellation; WSL process stop cancel |
| P3-27 | 3 | login-path-failure-backoff; 新 issue（开贴前查重）; 有效 | "LoginPath" shell timeout; login path retry lock |

References are the map rows and the audit report’s matching finding IDs; source line numbers in the report describe the audited snapshot and are not a fresh code verification.
