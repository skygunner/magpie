# FIX-MAP — magpie v0.1.1957 审计活条目（生成于 2026-10-29）

覆盖 §2 的 3 条 P1、§3 的 44 条 P2 编号与 §4 的 29 行 P3（含已驳回条目标注）。默认 1 发现 = 1 分支 = 1 issue；同文件同机制的相邻发现并簇。Batch 1 = 六个已定簇；Batch 2/3 为后续批次，开贴前必须过 round28 四道排除门（已交付 / 上游已修 / 他人已认领或贴补丁 / 已指派）。

| 发现 | 一句话 | 簇 | 分支 | 交付方式 | 批次 | 状态 |
|---|---|---|---|---|---|---|
| P1-1 | `internal/pii` 过期 miss 列表导致误报 masking 503 | B1 pii-stale-miss | fix/4118-pii-stale-miss | #4118 已回贴 | 1 | 已交付 |
| P1-2 | WebDAV/S3 64 MiB 读取上限把完好备份误判为损坏 | B2 davsync-read-cap | fix/4137-davsync-read-cap | #4137 已交付 | 1 | 已交付 |
| P1-3 | Windows `wslDistros` 持包锁执行 30 s/distro 探测 | B3 wsl-probe-lock | fix/4136-wsl-probe-lock | #4136 已交付 | 1 | 已交付 |
| P2-1 | 非流式组请求没有首响应故障转移 | gateway 组请求 | fix/<#>-nonstream-firstwait | 回贴 #3788 | 2 | 有效 |
| P2-2 | 回复体没有服务端截止时间 | response-body-deadline | fix/<#>-response-body-deadline | 新 issue（开贴前查重） | 2 | 有效 |
| P2-3 | `holdWriter` 持锁写客户端，看门狗与保活一起被阻塞 | 网关与协议 / `holdWriter` 持锁写客户 | 待定（簇内合并） | 新 issue（开贴前查重） | 不计入 | 已驳回 |
| P2-4 | Codex 自有登录路径：流中途读错误被记成 200 成功 | codex-stream-read-errors | fix/<#>-codex-stream-read-errors | 新 issue（开贴前查重） | 2 | 有效 |
| P2-5 | Chat Completions 静默丢弃 `video_url` / `input_audio` 内容块 | chat 多模态 | fix/<#>-chat-multimodal | 回贴 #4101 | 2 | 有效 |
| P2-6 | Chat/Responses 编码器丢弃模型生成的图片（KImage） | chat-generated-images | fix/<#>-chat-generated-images | 新 issue（开贴前查重） | 2 | 有效 |
| P2-7 | `thread_description` 绕过受限 key 的模型白名单 | codex-description-key-policy | fix/<#>-codex-description-key-policy | 新 issue（开贴前查重） | 2 | 有效 |
| P2-8 | `codexLists` 无淘汰 | codex-model-list-eviction | fix/<#>-codex-model-list-eviction | 新 issue（开贴前查重） | 2 | 有效 |
| P2-9 | affinity 映射只按年龄清理，超过 4096 条后每次写都全表扫描 | affinity-capacity | fix/<#>-affinity-capacity | 新 issue（开贴前查重） | 2 | 有效 |
| P2-10 | `stickOf` 在全局锁内做 `os.Stat`/`ReadFile`/JSON 解码 | affinity-lock-io | fix/<#>-affinity-lock-io | 新 issue（开贴前查重） | 3 | 降级 P3 |
| P2-11 | 搜索 key 冷却的读-改-写竞态 | search-key-cooldown-race | fix/<#>-search-key-cooldown-race | 新 issue（开贴前查重） | 3 | 降级 P3 |
| P2-12 | `providers.json` 读-改-写无锁（约 47 处 `store(f)`） | providers-update-race | fix/<#>-providers-update-race | 新 issue（开贴前查重） | 2 | 有效 |
| P2-13 | `settings.json` 读-改-写无锁（`modelprefs.go` 所有 setter） | settings-update-race | fix/<#>-settings-update-race | 新 issue（开贴前查重） | 2 | 有效 |
| P2-14 | Codex 多账号的 reasoning 级别合并不生效 | 配置与账户 / Codex 多账号的 reasoni | 待定（簇内合并） | 新 issue（开贴前查重） | 不计入 | 已驳回 |
| P2-15 | 超过 128 个并发请求后钩子被静默跳过 | plugin-hook-overload | fix/<#>-plugin-hook-overload | 新 issue（开贴前查重） | 2 | 有效 |
| P2-16 | `lockAuth` 在权限错误时永久自旋 | plugin-auth-lock-errors | fix/<#>-plugin-auth-lock-errors | 新 issue（开贴前查重） | 2 | 有效 |
| P2-17 | 只要装了任何网关钩子，每个回复都整段落盘并重新解析 | plugin-response-capture | fix/<#>-plugin-response-capture | 新 issue（开贴前查重） | 2 | 有效 |
| P2-18 | 超过 8 MiB 的成功回复被报告给钩子为失败 | plugin-response-truncation | fix/<#>-plugin-response-truncation | 新 issue（开贴前查重） | 2 | 有效 |
| P2-19 | CSRF 只挡跨站，不挡同站不同端口 | gui-same-site-csrf | fix/<#>-gui-same-site-csrf | 新 issue（开贴前查重） | 2 | 有效 |
| P2-20 | 页面登录 Cookie 缺 `Secure` | gui-secure-cookie | fix/<#>-gui-secure-cookie | 新 issue（开贴前查重） | 2 | 有效 |
| P2-21 | 每个静态资源请求都清空 provider 目录/文件缓存 | gui-static-cache-invalidation | fix/<#>-gui-static-cache-invalidation | 新 issue（开贴前查重） | 3 | 降级 P3 |
| P2-22 | 98 个 `/api` 处理函数用无上限的 `json.NewDecoder(r.Body)` | gui-request-body-limit | fix/<#>-gui-request-body-limit | 新 issue（开贴前查重） | 2 | 有效 |
| P2-23 | MCP 令牌续期忽略跨进程锁是否拿到 | mcp-refresh-lock | fix/<#>-mcp-refresh-lock | 新 issue（开贴前查重） | 2 | 有效 |
| P2-24 | `?period=` 无上限，单次请求可分配约 50 万个图表点/300 万个 map | usage-period-limit | fix/<#>-usage-period-limit | 新 issue（开贴前查重） | 2 | 有效 |
| P2-25 | `last:24h` 页面按“今天”缓存，一整天不更新 | usage-rolling-page-cache | fix/<#>-usage-rolling-page-cache | 新 issue（开贴前查重） | 2 | 有效 |
| P2-26 | `LastSeen` 持全局 usage 锁扫描 256 KiB | usage-lastseen-lock | fix/<#>-usage-lastseen-lock | 新 issue（开贴前查重） | 2 | 有效 |
| P2-27 | 删除会话失败回滚不检查错误，随后删除仍装着已移动文件的回收目录 | sessions-delete-rollback | fix/<#>-sessions-delete-rollback | 新 issue（开贴前查重） | 2 | 有效 |
| P2-28 | 增量解析读错误后仍发布半成品，下次从旧偏移重放导致重复计数 | sessions-incremental-read-error | fix/<#>-sessions-incremental-read-error | 新 issue（开贴前查重） | 2 | 有效 |
| P2-29 | 删除会话持排他锁跨卷复制 | sessions-delete-lock-io | fix/<#>-sessions-delete-lock-io | 新 issue（开贴前查重） | 2 | 有效 |
| P2-30 | WebDAV 无校验写被当成已校验写 | B6 dav-put-version | fix/4141-dav-put-version | #4141 已交付 | 1 | 已交付 |
| P2-31 | `library.Put` 先把旧技能移走，失败不回滚 | library-put-rollback | fix/<#>-library-put-rollback | 新 issue（开贴前查重） | 2 | 有效 |
| P2-32 | Windows/Linux 的更新包只校验同一响应里的 SHA256 | update-signature-verification | fix/<#>-update-signature-verification | 新 issue（开贴前查重） | 2 | 有效 |
| P2-33 | agent Disconnect 先消费恢复点，再执行可能失败的写 | agent-disconnect-restore | fix/<#>-agent-disconnect-restore | 新 issue（开贴前查重） | 2 | 有效 |
| P2-43 | `stash()`/`unstash()` 丢弃 `stashWrite` 的错误，用户值的唯一副本可被静默丢失 | B4 agent-stash-durability | fix/4258-agent-stash-durability | #4258 已交付 | 1 | 已交付 |
| P2-44 | `stash.json` 的读-改-写没有跨进程锁，CLI 与应用会互相覆盖已存值 | B4 agent-stash-durability | fix/4258-agent-stash-durability | #4258 已交付 | 1 | 已交付 |
| P2-34 | dsh `.env` 会覆盖/删除用户自己的 `MAGPIE_GATEWAY_KEY` | dsh-env-key-preservation | fix/<#>-dsh-env-key-preservation | 新 issue（开贴前查重） | 3 | 降级 P3 |
| P2-35 | `applied.json` 非原子写且丢弃两个错误 | agent-applied-atomic-write | fix/<#>-agent-applied-atomic-write | 新 issue（开贴前查重） | 2 | 有效 |
| P2-40 | 路由回合表按天无界增长，超过 4096 后每次请求全表扫描 | route-turn-capacity | fix/<#>-route-turn-capacity | 新 issue（开贴前查重） | 2 | 有效 |
| P2-41 | `models.json` 原地重写，中断后整份目录为空且 24 小时内不复原 | catalog-atomic-write | fix/<#>-catalog-atomic-write | 新 issue（开贴前查重） | 2 | 有效 |
| P2-42 | 图片/视频命名循环在 Stat 非“不存在”错误时无限自旋 | image-output-stat-errors | fix/<#>-image-output-stat-errors | 新 issue（开贴前查重） | 2 | 有效 |
| P2-36 | 首次 `/api/state` 失败后 Agents 页面永久停在骨架屏 | agents-initial-state-error | fix/<#>-agents-initial-state-error | 新 issue（开贴前查重） | 2 | 有效 |
| P2-37 | 较慢的 state 读取覆盖读取期间完成的写操作 | gui-state-stale-read | fix/<#>-gui-state-stale-read | 新 issue（开贴前查重） | 2 | 有效 |
| P2-38 | `loadProvidersNow` 同样用旧读覆盖新写 | gui-state-stale-read | fix/<#>-gui-state-stale-read | 新 issue（开贴前查重） | 2 | 有效 |
| P2-39 | MCP 登录轮询在接口持续失败时永不结束 | mcp-signin-poll-errors | fix/<#>-mcp-signin-poll-errors | 新 issue（开贴前查重） | 2 | 有效 |
| P3-1 | `SetReadLimit(-1)` 关闭 32 KiB 帧上限 | codex-socket-frame-limit | fix/<#>-codex-socket-frame-limit | 新 issue（开贴前查重） | 3 | 有效 |
| P3-2 | `ownCallback` 在交接窗口内接受任意存活 token 且免 key（randomToken 回退半边已驳回： | callback-address-allowlist | fix/<#>-callback-address-allowlist | 同 P3-21（开贴前查重） | 合并 P3-21 | 重复，不单独交付 |
| P3-3 | 账户分支重建 Provider 时丢掉 `LimitRestOff`（字段在 207） | provider-account-limit-rest | fix/<#>-provider-account-limit-rest | 新 issue（开贴前查重） | 3 | 有效 |
| P3-4 | `usage-alerts.json` 原地写、忽略解析错误 | quota-alert-atomic-write | fix/<#>-quota-alert-atomic-write | 新 issue（开贴前查重） | 3 | 有效 |
| P3-5 | `cacheCards(..., true)` 合并保留已删除账户的卡片 | quota-card-cache-refresh | fix/<#>-quota-card-cache-refresh | 新 issue（开贴前查重） | 3 | 有效 |
| P3-6 | 4 MiB 截断且丢弃 `ReadAll` 错误（同端点 `CodexLiveList:89` 用 16 MiB） | codex-model-list-read-errors | fix/<#>-codex-model-list-read-errors | 新 issue（开贴前查重） | 3 | 有效 |
| P3-7 | `[DONE]` 分支只 `giveHeld`，不 `end` | chat-done-flush | fix/<#>-chat-done-flush | 新 issue（开贴前查重） | 3 | 有效 |
| P3-8 | 完整 zstd 帧解码失败被当作“还没写完”，`end=start`、`next` 不变 | B5 sessions-frame-tail | fix/4144-sessions-frame-tail | #4144 已交付 | 1 | 保守缓解，损坏后内容仍不可恢复 |
| P3-9 | 【已驳回】证伪波 REJECTED（=sessions-store/F5）：`defer in.Close()` 在 W | sessions / 【已驳回】证伪波 REJECTED（ | 待定（簇内合并） | 新 issue（开贴前查重） | 不计入 | 已驳回 |
| P3-10 | 导出名 Stat 后再写，非原子占名 | archive-exclusive-name | fix/<#>-archive-exclusive-name | 新 issue（开贴前查重） | 3 | 有效 |
| P3-11 | `state==Failed` 永不重试，即使配置已改 | plugin-retry-after-config | fix/<#>-plugin-retry-after-config | 新 issue（开贴前查重） | 3 | 有效 |
| P3-12 | `appliedMu` 只在进程内，`applied.json` 无跨进程锁 | agent-applied-process-lock | fix/<#>-agent-applied-process-lock | 新 issue（开贴前查重） | 3 | 有效 |
| P3-13 | 用当前页 `Rows` 再裁剪 `l.Accounts` | usage-account-page-filter | fix/<#>-usage-account-page-filter | 新 issue（开贴前查重） | 3 | 有效 |
| P3-14 | 只按 `in+out>0` 计价 | usage-cache-only-pricing | fix/<#>-usage-cache-only-pricing | 新 issue（开贴前查重） | 3 | 有效 |
| P3-15 | 每次 `renderUpdateBadge` 都启动一条 700 ms 轮询链，句柄不保存 | update-badge-polling | fix/<#>-update-badge-polling | 新 issue（开贴前查重） | 3 | 有效 |
| P3-16 | `saveState` 丢弃 `WriteAtomic` 错误 | davsync-save-state-errors | fix/<#>-davsync-save-state-errors | 新 issue（开贴前查重） | 3 | 有效 |
| P3-17 | 网关看门狗持 `gatewayMu` 调用 `serveGatewayLocked`（含 500 ms 拨号/HTTP  | gateway-watchdog-lock-io | fix/<#>-gateway-watchdog-lock-io | 新 issue（开贴前查重） | 3 | 有效 |
| P3-18 | 基准仍断言 `group/auto-model-499`，发布版对外是 `auto/<model>` | model-list-benchmark-name | fix/<#>-model-list-benchmark-name | 新 issue（开贴前查重） | 3 | 有效 |
| P3-19 | 配对 memo 失效时全量重建；实测 20 万行约 178 ms、111 MiB/次 | usage-pair-cache-rebuild | fix/<#>-usage-pair-cache-rebuild | 新 issue（开贴前查重） | 3 | 有效 |
| P3-20 | 页面登录无失败计数，可无限猜 16 字符 key | gui-login-rate-limit | fix/<#>-gui-login-rate-limit | 新 issue（开贴前查重） | 3 | 有效 |
| P3-21 | 交接窗口内 `ownCallback` 在 allow-list 之前直接放行 | callback-address-allowlist | fix/<#>-callback-address-allowlist | 新 issue（开贴前查重） | 3 | 有效 |
| P3-22 | `.local` 不解析即信任；未见过的名字在请求路径做最多 400 ms DNS | netproxy-local-resolution | fix/<#>-netproxy-local-resolution | 新 issue（开贴前查重） | 3 | 有效 |
| P3-23 | 同一互斥锁覆盖文件追加与整文件读取 | maskrec-lock-io | fix/<#>-maskrec-lock-io | 新 issue（开贴前查重） | 3 | 有效 |
| P3-24 | 保存时整体重新 marshal，未知键丢失 | settings-unknown-fields | fix/<#>-settings-unknown-fields | 新 issue（开贴前查重） | 3 | 有效 |
| P3-25 | 主题刷新持锁跑 fontconfig/hyprctl（合计可达数秒） | theme-refresh-lock-io | fix/<#>-theme-refresh-lock-io | 新 issue（开贴前查重） | 3 | 有效 |
| P3-26 | `wslRun` 无发行版侧停止，与 `wslrun.Tool` 的 cancelable/Stop 不同（证伪降级 = | wsl-command-cancellation | fix/<#>-wsl-command-cancellation | 新 issue（开贴前查重） | 3 | 有效 |
| P3-27 | shell 不应答时 `LoginPath` 每次都在 `login.Lock` 内重跑两次 5 s 询问（=agent | login-path-failure-backoff | fix/<#>-login-path-failure-backoff | 新 issue（开贴前查重） | 3 | 有效 |
| P3-28 | 帧头损坏被当作“还没写完”，`readFrame` 的 `errFrame` 被吞，尾帧永久钉住（ADDED A1，另见 P3-8） | B5 sessions-frame-tail | fix/4144-sessions-frame-tail | #4144 已交付 | 1 | 保守缓解，损坏后内容仍不可恢复 |
| P3-29 | `put` 只接受/返回 ETag，丢弃可用的 Last-Modified（ADDED A2） | B6 dav-put-version | fix/4141-dav-put-version | #4141 已交付 | 1 | 已交付 |
