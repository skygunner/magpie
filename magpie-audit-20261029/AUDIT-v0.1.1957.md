# Magpie v0.1.1957 全面代码审计（只读）

> 审计对象：上游正式发布版 `v0.1.1957`（commit a0ba8e1f2），`yetone/magpie-releases` 的 Latest release（16 个资产，2026-10-28T20:55:28Z）。
> 源码基线：`git archive v0.1.1957` 解出的只读快照（4011 个 `.go` 文件；gateway 79.7k、provider 73.6k、agent 44.3k 非测试 LOC）。
> 构建基线：`go build ./...` 退出码 0；`go vet ./...` 退出码 0（go1.27.3 linux/amd64）。
> 排除：本地工作区 `/home/vincenty/Projects/magpie`（dev 分支 `internal/redact` 有未解决冲突）从未被修改、构建或用于任何结论。

## 0. 结论摘要

终局计数：**3 个 P1**；§3 共 **44 条 P2 编号**（其中 2 条已驳回、4 条降级为 P3，**38 条保留 P2**）；§4 共 **29 行 P3**（其中 1 行含已驳回半边，**28 条有效**，另有 4 条由 P2 降级）。证伪波六组全部完成：**CONFIRMED 61 / DOWNGRADED 8 / REJECTED 5 / ADDED 4**；驳回与降级的具体条目见 §9，编号保留以便对照通道报告。

- 没有发现 P0（无需认证即可远程利用、必然崩溃或必然数据毁损的路径）。
- 最该先处理的三件事：
  1. `internal/pii` 的过期 miss 列表：已用上游 #4118 的合成测试在 v0.1.1957 上**稳定复现**（严格/宽松两种模式都失败），会造成误报 masking 503，属于可用性故障。
  2. WebDAV/S3 备份读取上限 64 MiB：超过上限的完好备份被误判为“损坏”，界面给出的恢复动作会**覆盖远端唯一完好的备份**。
  3. Windows 上 `wslDistros` 持包锁执行最多 30 s/发行版的 WSL 探测：GUI 的 `/api/state`、Sessions/Usage 会被拖住数十秒到数分钟。
- 安全面最值得注意：GUI 的 CSRF 防护只挡跨站不挡**同站不同端口**；HTTPS 页面的登录 Cookie 缺 `Secure`；98 个 `/api` 处理函数没有请求体上限。
- 性能面：组路由本身开销可接受（200 KB 请求约 4.7–5.4 ms，20 核并行 730–780 MB/s）；真正的开销在“每个请求/每次轮询都做的整表扫描、整文件重读、无上限分配”，见第 5 节。

## 1. 方法与限制

- 17 条只读审计通道（网关核心/协议/订阅后端、provider 账户与目录、usage、插件、GUI API/前端、sessions、library/davsync/update、agent 配置与运行时、并发、性能、安全、外围包）+ 6 个证伪校验节点；每条通道先读源码、写带 `file:line` 与代码引用的报告，再由另一节点逐条尝试证伪。
- 主会话对全部 P1 与绝大多数 P2 重新打开源码核对；只有 1 条做了真实复现（P1-1），其余为**静态验证**，文中逐条标注置信度。
- 基准测试在本机 20 核（Intel Ultra 7 265K）上运行，**同时有 4 个审计子代理在工作**，数字应视为上界；命令与原始输出见证据索引。
- 大规模文件有明确的未读区段，逐通道列在第 6 节；未读区段不代表没有问题。

## 2. P1（3）

**P1-1 `internal/pii`：让出模型回合后沿用过期 miss 列表，误报 masking 503（已复现）**
位置：`internal/pii/find.go:184`（`known` → 让出/重取回合 → `missing`，find.go:213/230）。
触发：同一文本在请求 A 让出模型回合期间被请求 B 写入缓存；A 恢复后仍用旧的 `miss` 下标，`missing()` 返回空，严格模式走到 “the local masking model has no text to read for N texts”。
影响：请求被 fail-closed 拒绝（503），本地 PII 遮蔽不可用；宽松模式则直接返回而丢掉已缓存的命中。
复现：上游 #4118 提供的合成测试 `TestCacheFilledWhileYielding`，在发布版副本上 `go test ./internal/pii -run '^TestCacheFilledWhileYielding$' -count=1 -v` **FAIL（strict + bounded 均失败）**，输出见 `evidence-pii-4118.log`。
修复：在重取回合后若 `units` 为空，重新执行 `out, miss = b.known(texts, got)`；只有确实没有 miss 才走空分支，保持分类器失败时的 fail-closed。
核验：已复现（high）。

**P1-2 WebDAV/S3：大于 64 MiB 的备份被截断后误判为“损坏”，恢复动作会覆盖完好备份**
位置：`internal/davsync/dav.go:291`、`internal/davsync/s3.go:234`（读取上限 `64<<20`，不检查超限）；`internal/backup/backup.go:299-300`（前缀像备份 + 读不全 → `ErrCorrupt`）；`internal/davsync/notbackup.go:136`（`Replace: true`，提示上传本机设置覆盖它）。
触发：远端是一份有效的密封备份，但大于 64 MiB（技能文件预算 32 MiB + 每提供商图标 1 MiB，双重 base64 后可达）；写入端从不检查大小。
影响：除写入者外的每台电脑每次同步都报“备份被截断或损坏”，并只提供“用本机设置覆盖远端”的出路；照做会销毁远端唯一完好的备份。写入者因 ETag 未变而毫无察觉，每 3 分钟重试一次。
修复：读取 `64<<20+1` 并在超限时返回明确的“备份大于 64 MiB”错误；同时在推送端拒绝超过上限的备份（GitHub 路径已经这样做，`github.go:21`）。
核验：主会话已核对源码（high）。

**P1-3 Windows：`wslDistros` 持包锁做 WSL 探测，一个慢发行版冻结所有 WSL 读者**
位置：`internal/agent/wsl.go:1094`（`wsl.Lock()` + defer Unlock）→ `wslRunningLocked()` → `wslProbe(n)`（`wslRun(30*time.Second, ...)`，wsl.go:1429）→ `wslSaveLocked()`。
触发：Windows + WSL2，首个运行中发行版尚未探测（首次 GUI 加载、`wsl.json` 不可解析、发行版重启、Rescan），或探测失败后每 10 分钟重试。
影响：`wslAgents()`、`wslHomes()`、`WSLRunning()` 等所有读者串行阻塞；3 个发行版可让 `/api/state` 停顿约 90 s，期间 GUI 的 WSL 行为空。
修复：在锁内只计算待探测列表，释放锁后探测，再取锁写回；或标记 in-flight 让并发读者跳过。
核验：主会话已核对源码（high；仅 Windows）。

## 3. P2（44 条编号；2 条已驳回、4 条降级 P3，38 条保留 P2）

### 网关与协议
- **P2-1 非流式组请求没有首响应故障转移**（`gateway.go:2661`，仅 `streams` 时设 `hw.firstWait`；`upstream_transport.go:41` 首包等待 10 分钟；`provider/group.go:187` 文档也只写流式）。触发：非流式客户端 + 组内成员卡住。影响：客户端只能等到 10 分钟/自身超时，组内其他成员不会被尝试（上游 #3788 的已知缺口）。修复：为首响应设置对所有协议生效的截止时间。核验：主会话（high）。
- **P2-2 回复体没有服务端截止时间**（`gateway.go:2918` 只有 `context.WithCancel`；`fallback.go:2729` 的 `keepaliveLongest` 只停保活；HTTP client 无 `Timeout`，`ResponseHeaderTimeout` 只覆盖首包）。触发：供应商发出响应头后停住。影响：尝试与并发槽位被占到客户端断开为止；`MaxConcurrency=1` 时其他请求排队。修复：为每次尝试设置总/空闲读截止时间。核验：主会话（high）。
- **P2-3 `holdWriter` 持锁写客户端，看门狗与保活一起被阻塞**（`fallback.go:1950` 取 `h.mu`，1998 在锁内 `h.w.Write`；`watch()` 的 tick/deadline 分支与 `keepQuiet` 都要同一把锁）。触发：客户端停止读取、接收窗口填满。影响：首 token 截止时间无法触发、组内转移不发生、keepalive 发不出，goroutine 一直挂起。修复：锁内拷贝待写数据，锁外写；或给响应设写截止时间。核验：主会话（high）。**【已驳回】证伪波 REJECTED（=concurrency/F2）：`h.passing` 时只走该写路径，`started()` 计入 passing，deadline 分支不会置 `h.slow` 或 `stop()`；锁即写锁，见 §9。**
- **P2-4 Codex 自有登录路径：流中途读错误被记成 200 成功**（`codex_backend.go:1310` 丢弃 `Read` 的 `err`；`call.Status=res.StatusCode`，`call.Error` 只在 ≥400 时设置）。触发：SSE 读中断（代理/网络）。影响：Usage 显示 200、无错误，也不会换成员。修复：读错误写入 `call.Error` 并按失败处理。核验：主会话（high）。
- **P2-5 Chat Completions 静默丢弃 `video_url` / `input_audio` 内容块**（`chat.go:198-216` 的 switch 只处理 text/image_url/file；全树无 `input_audio` 处理）。触发：客户端发视频/音频内容块。影响：上游模型从未收到附件，Magpie 仍返回 200 并按文本计费（上游 #4101）。修复：至少显式 400 拒绝，或转换为上游支持的 file/多模态块。核验：主会话（high，静态）。
- **P2-6 Chat/Responses 编码器丢弃模型生成的图片（KImage）**（`chat.go:1554-1595` 无 KImage 分支；`renderChat` 1613-1658 只处理 Text/Thinking/ToolCall；`gemini.go:676` 与 `draw.go:1744` 才有）。触发：CodeAssist/Gemini 在聊天流里返回内联图片。影响：图片被静默丢弃，客户端拿到空成功。修复：为 Chat/Responses 定义图片映射，或明确返回不支持。核验：主会话（high）。
- **P2-7 `thread_description` 绕过受限 key 的模型白名单**（`codex_titles.go:209` 无条件 `chose=true`；`codexSideTo` 在 Magpie 自己的 API 上返回调用方给的模型，`gateway.go:1615`；标题路径正确比较了设置值 158 行）。触发：受限 key 直接请求 `thread_description` 并指定任意 `auto/...` 或 `provider/model`。影响：受限 key 可消耗未被允许的模型/账户额度。修复：仅在 `to == settings.CodexDescriptions` 时豁免。核验：主会话（high）。
- **P2-8 `codexLists` 无淘汰**（`codex_backend.go:1784` 声明，1823 只插入/查询，全树无 delete）。触发：不同 ChatGPT 账号/轮换 token 访问 `/models`。影响：每个凭据一份模型列表永久驻留。修复：按时间或容量淘汰。核验：主会话（high）。
- **P2-9 affinity 映射只按年龄清理，超过 4096 条后每次写都全表扫描**（`affinity.go:707-713`）。触发：大量不同会话 id（客户端可控）后持续写入。影响：`answered()` 在全局锁内 O(n) 扫描，热路径抖动且内存无上限。修复：按容量淘汰 + 增量过期队列。核验：主会话（high）。
- **P2-10 `stickOf` 在全局锁内做 `os.Stat`/`ReadFile`/JSON 解码**（`affinity.go:249-281`，调用点 505/733）。触发：每个请求的 affinity 读取；`answered` 每 2 秒重写文件使 mtime 变化。影响：请求路径被磁盘 I/O 串行化，多请求时锁竞争显著。修复：锁外读文件、锁内合并。核验：主会话（high）。**【降级 P3】见 §9：锁内 I/O 只在 mtime 变化时发生（`sticksFlushAfter=2s`），不是每次 miss。**
- **P2-11 搜索 key 冷却的读-改-写竞态**（`search_keys.go:79-86`，`sync.Map` 无 CAS）。触发：同一 key 上两个搜索同时被拒（如 402 一小时 + 429 三十秒）。影响：较短的冷却覆盖较长者，缺额 key 被重新请求。修复：`CompareAndSwap` 循环或互斥锁。核验：主会话（high）。**【降级 P3】见 §9：load-compare-store 窗口仅数微秒，代价是一次 30 s 重试。**

### 配置与账户
- **P2-12 `providers.json` 读-改-写无锁（约 47 处 `store(f)`）**（`provider.go:918` 读 → 950 写；`autolist.go:229/247` 是第二个写者；`steady.WriteFile` 只保证写入原子，不保证基于新读）。触发：后台 `Fetch`/自动列表与 GUI 保存并发。影响：后写者用自己的旧快照覆盖，删除的提供商/新 key 会“复活”或消失，无错误提示。修复：包级读写锁或“锁内重读+只改一个提供商”。核验：主会话（high，机制确定）。
- **P2-13 `settings.json` 读-改-写无锁（`modelprefs.go` 所有 setter）**（`modelprefs.go:90/103` 等先 `settings.Load()` 再 `settings.Save(s)`；`settings.Save` 只锁写入，settings.go:1426；同文件已有 `EditUsageAliases`/`EditCheckinOff` 的正确模式）。触发：GUI 保存一个设置与 `magpie model name/price` 并发。影响：整个设置文件回退，模型名/价格/主题等一起丢失。修复：提供 `Edit(func(*Settings) error)` 并统一使用。核验：主会话（high）。
- **P2-14 Codex 多账号的 reasoning 级别合并不生效**（`codex_models.go:396-398`：基础列表该模型 `Efforts` 为空时直接 `continue`，与 366-377 的“所有账号的并集”文档矛盾）。触发：主账号列表有模型但无级别，副账号有级别。影响：模型被展示为不支持 effort，代理无法请求 high/xhigh。修复：空级别时直接取副账号的级别再合并。核验：主会话（high）。**【已驳回】证伪波 REJECTED（=prov-models/F3）：空 Efforts 表示“级别未知”，GUI 填 `provider.Levels`，网关仅在 `len>0` 时钳制，`Account.Levels` 与 `effortMate` 仍可路由 high/xhigh。**

### 插件
- **P2-15 超过 128 个并发请求后钩子被静默跳过**（`internal/plugin/host.js:1345/1360` 返回 null；`internal/plugin/gateway.go:206` 把“无钩子/宿主不可用/超限”都折成 nil，无日志无计数）。触发：共享网关 129+ 并发请求。影响：改写提示词、脱敏、限流等钩子对溢出请求不生效，插件毫不知情（200 并发时约 36% 绕过）。修复：把超限作为可区分的拒绝返回并计数。核验：主会话（high）。
- **P2-16 `lockAuth` 在权限错误时永久自旋**（`internal/plugin/store.go:139-151`：`os.IsPermission` 进入循环；超过 `2*authLockStale` 只执行忽略错误的 `os.Remove`；无返回路径）。触发：plugins 目录被 root 所有、只读卷或锁文件 EACCES。影响：`SignOut`/`Import`/`Take`/`Restore` 永久挂起，每秒约 200 次系统调用。修复：超时后返回错误。核验：主会话（high）。
- **P2-17 只要装了任何网关钩子，每个回复都整段落盘并重新解析**（`internal/gateway/middleware.go:123-129` 无条件 `gatewayResponse`；`gateway_hooks.go:15-24` 以 `conversationBodyMax=8 MiB` 建 spool，读完再 `gatewayReply` 重放 SSE）。触发：只实现 `onRequest` 的插件，或任何钩子请求。影响：每请求最多 8 MiB 临时文件写 + 整段读回 + 完整 SSE 解码/重渲染，即使没有 `onResponse`。修复：begin 回复“是否有 onResponse”，没有就跳过捕获。核验：主会话（high）。
- **P2-18 超过 8 MiB 的成功回复被报告给钩子为失败**（`gateway_hooks.go:27` 把 `spool.cut()` 与真实失败并列，`run.Finish(nil,false,body)`）。触发：长工具回合、带图回复超过 8 MiB 且装有响应钩子。影响：钩子看到 `success:false`、`reply:null`，据此做的记账/重试建议全错。修复：截断用独立标志，success 只看状态与已保留字节。核验：主会话（high）。

### GUI 与安全
- **P2-19 CSRF 只挡跨站，不挡同站不同端口**（`internal/gui/pluginpage.go:356-372` 只对 `Sec-Fetch-Site: cross-site` 返回拒绝；`web.go:208` Cookie `SameSite=Lax`，Cookie 不区分端口；处理函数用 `json.NewDecoder`，忽略 Content-Type）。触发：同主机任意 HTTP 端口上的页面用 `text/plain` 简单请求盲写 `/api/*`。影响：可开启 LAN 共享、轮换页面 key、改写设置。修复：对非 GET 的 `same-site` 也拒绝。核验：主会话（high）。
- **P2-20 页面登录 Cookie 缺 `Secure`**（`web.go:208` 只设 HttpOnly/SameSite=Lax；TLS 页面路径见 web.go:89）。触发：HTTPS 页面 + 同主机明文 HTTP 服务。影响：主密钥经明文端口泄露，拿到后可读取所有 provider key/账户。修复：`Secure: r.TLS != nil || X-Forwarded-Proto==https`。核验：主会话（high）。
- **P2-21 每个静态资源请求都清空 provider 目录/文件缓存**（`gui/api.go:2470-2473`：`!(api && GET)` 对 `/app.js`、图标等全部成立，`defer provider.Changed()` → `filememo.Forget()`+`dropHeld()`）。触发：任何页面加载（20-30 个资源请求）。影响：目录在页面加载期间被反复重建，代码自述“30 个提供商时占一次请求 CPU 的 1/5”。修复：只有 `api && 非 GET` 才 `Changed()`。核验：主会话（high）。**【降级 P3】见 §9：静态资源确实清缓存，但“每页 20–30 次重建”被高估。**
- **P2-22 98 个 `/api` 处理函数用无上限的 `json.NewDecoder(r.Body)`**（`gui/backup.go:68` 典型：先解出 string 再 base64 解码；`MaxBytesReader` 全包只有 7 处；HTTP server 只设 ReadHeaderTimeout）。触发：`POST /api/backup/import` 1 GiB 请求体。影响：单请求可占 2.5-3 GiB，OOM 或整机换页。修复：mux 统一 `MaxBytesReader`，导入路由给更小上限。核验：主会话（high）。
- **P2-23 MCP 令牌续期忽略跨进程锁是否拿到**（`internal/mcpauth/mcpauth.go:311` `unlock, _ := filelock.Lock(...)`；`filelock.go` 10 s 后返回 `held=false` 并打印“going on without it”）。触发：两个 magpie 同时续期同一个轮换 refresh token。影响：第二次 spend 触发 `invalid_grant`，记录被标死，所有使用该 MCP 的代理报 `ErrExpired`。修复：未拿到锁就返回可重试错误。核验：主会话（high 机制）。

### Usage
- **P2-24 `?period=` 无上限，单次请求可分配约 50 万个图表点/300 万个 map**（`usageapi/usageapi.go:197-201` 只检查 `Known()`；`usage/period_range.go:80-86` 接受任意 `last:` 时长和任意日期范围；`usage/ledger.go:639` 每个小时/天/周建一个点，`request_page.go:2065` 及维度 map）。触发：本机请求 `GET /v1/magpie/usage/requests?period=last:2562047h&limit=1`（loopback 无需 key）。影响：空日志也分配约 250-350 MB，页面还进 `requestCache.pages`（最多 16 份），可 OOM。修复：在 `PeriodOf` 夹到热图 53 周上限。核验：主会话（high）。
- **P2-25 `last:24h` 页面按“今天”缓存，一整天不更新**（`usage/request_page.go:511-519` 的 `pageKey` 只有 Period/Filter/Offset/Limit/Date；743 行命中缓存前没有滚动窗口判断；779 行对 partCache 已正确跳过 `p.Last()!=0`）。触发：09:00 与 21:00 各请一次 `last:24h`，中间无日志/会话变化。影响：21:00 返回 09:00 的页面，窗口既不前移也不含新数据。修复：滚动窗口不进 `pages`/`flights`，或把 `Since(now)` 放进 key。核验：主会话（high）。
- **P2-26 `LastSeen` 持全局 usage 锁扫描 256 KiB**（`usage/usage.go:1199-1224` 取 `mu` 后 open/seek/scan/JSON 解码；`Append` 用同一把锁 265-278）。触发：每次 Agents/manage 页面渲染对每个 agent 调一次。影响：20 个 agent 的顺序扫描会让所有在途调用的记账写入停顿数十毫秒。修复：读尾部不需要写锁，移出临界区。核验：主会话（high）。

### 会话、同步与更新
- **P2-27 删除会话失败回滚不检查错误，随后删除仍装着已移动文件的回收目录**（`sessions/manage.go:383-390`：回滚 `move` 的返回值被丢弃，然后 `os.RemoveAll(dir)`）。触发：多路径删除中后一个 `move` 失败（ENOSPC/EACCES/9p），且回滚也失败（原路径已被代理重建）。影响：用户会话文件被永久删除，`Restore` 所需清单也一起没了。修复：回滚失败时保留 `dir` 并报错。核验：主会话（high）。
- **P2-28 增量解析读错误后仍发布半成品，下次从旧偏移重放导致重复计数**（`sessions/sessions.go:1703-1708` 只在成功时更新 `s.Off`，但 `s` 已被逐行修改并返回发布）。触发：EIO/ESTALE/EACCES/FUSE 卸载等非 EOF 读错误。影响：token/活跃时长/成本翻倍，直到文件重写或缓存版本变化；`call_cache.go:350` 同样形状。修复：出错时返回旧状态。核验：主会话（high）。
- **P2-29 删除会话持排他锁跨卷复制**（`sessions/manage.go:319` 取 `storeMu`+`mu` 后进入 move/copyAll）。触发：WSL/9p 或第二个挂载上的大工具结果目录。影响：所有 session 读（List/Stats/Manage）阻塞整个复制时长，1 GB/30 MB/s 约 35 s。修复：锁内收集路径，锁外移动。核验：主会话（high）。
- **P2-30 WebDAV 无校验写被当成已校验写**（`davsync/dav.go:62-76` 无 ETag/Last-Modified 不足 1 s 时返回空 version；`dav.go:318-323` 只在 etag 非空时发 `If-Match`，返回的 version 里 `Blind` 仍为 false；只有 `s3.go:302` 会设 `Blind: true`）。触发：不返回 ETag 的 WebDAV 服务器/代理，两台电脑同秒写入。影响：后写覆盖先写，双方都以为写入被校验；下一次同步把输家的改动回退且无提示。修复：未发 `If-Match` 时返回 `Blind: true`。核验：主会话（high）。
- **P2-31 `library.Put` 先把旧技能移走，失败不回滚**（`library/carry.go:356-397`：aside 目录建好后逐个 `move`，后续任何步骤失败直接返回；`library.go:266-275` 出错不保存 `library.json`）。触发：恢复/同步中途失败（Windows 文件锁、磁盘满、EPERM）。影响：`library.json` 仍列出技能，但技能目录已不在；下一次 `Collect` 把不可读目录当空技能推送，其他电脑上的真实技能被空目录替换。修复：记录已移动项，失败时反向移动。核验：主会话（high）。
- **P2-32 Windows/Linux 的更新包只校验同一响应里的 SHA256**（`internal/update/update.go:993` 与 feed 的 `Asset.SHA256` 同源；`rg authenticode|signtool` 无结果；只有 macOS 的 `sameSigner` 校验签名）。触发：能控制 `usemagpie.ai` 响应或完成 TLS 中间人。影响：可安装任意代码。修复：发布 detached 签名，用编译进二进制里的公钥校验。核验：主会话（high，静态）。
- **P2-33 agent Disconnect 先消费恢复点，再执行可能失败的写**（`agent/applied.go:774` `unstash(a.ID+".connect.was")` 会删除，778 `f.Set(w)` 可能失败；`stash.go:92-101` 读后即删）。触发：只读配置/磁盘满/rename 失败。影响：用户接入 Magpie 前的 provider/model 永久丢失，重试 Disconnect 也无法恢复。修复：读取不删除，全部写成功后再 forget。核验：主会话（high）。
- **P2-43 `stash()`/`unstash()` 丢弃 `stashWrite` 的错误，用户值的唯一副本可被静默丢失**（`agent/stash.go:78/90/99`；`stashWrite` 在 41-65 返回 error）。触发：磁盘满/只读目录/rename 失败时 Connect/Disconnect 暂存或恢复。影响：内存里删了键、磁盘没写成，原值永久丢失。修复：返回并传播错误，写成功后再从内存删除。核验：证伪 ADDED 1（P2）。
- **P2-44 `stash.json` 的读-改-写没有跨进程锁，CLI 与应用会互相覆盖已存值**（`agent/stash.go:67`；`stashMu` 仅进程内，27 行）。触发：两个 Magpie 共用配置目录时同时 Connect/Disconnect。影响：一方的 stash 条目被另一方旧快照覆盖，Disconnect 恢复不到原值。修复：`filelock` 包住读-改-写（与 P3-12 applied.json 不同文件、不同缺陷）。核验：证伪 ADDED 2（P2）。
- **P2-34 dsh `.env` 会覆盖/删除用户自己的 `MAGPIE_GATEWAY_KEY`**（`agent/dsh.go:1030` 删除条件 `(k == dshKeyRef || ourKey(v))` 对当前名字恒真；写入路径 1022-1025 无 stash，而 `.credentials.yaml` 路径 1072/1091 有）。触发：用户按文档在 `.env` 放了另一台 Magpie 的共享 key。影响：key 被无备份删除，dsh 报“no credential for provider route magpie”。修复：删除前加 `ourKey(v)`，覆盖前 stash。核验：主会话（high）。**【降级 P3】见 §9：仅在用户把密钥放在 magpie 自己的 `MAGPIE_GATEWAY_KEY` 名下时成立。**
- **P2-35 `applied.json` 非原子写且丢弃两个错误**（`agent/applied.go:60-66` 用 `os.WriteFile`，忽略 MkdirAll/WriteFile 错误；`appliedLoad` 41-47 解析失败当空）。触发：进程被杀/磁盘满/只读目录。影响：整份漂移台账失效，所有 agent 的“配置被外部改动”告警消失，Apply 仍报成功。修复：`edit.WriteAtomic` 并返回错误。核验：主会话（high）。

### 安全、目录与图片（外围）
- **P2-40 路由回合表按天无界增长，超过 4096 后每次请求全表扫描**（`gateway/rules.go:348-353`：插入即建新 key；只在 `len>4096` 时按 24h 年龄清理，清理本身在 `turnRules.Lock()` 内全表遍历）。触发：带规则组 + 每次新 `X-Magpie-Session`。影响：20 万唯一会话 ≈ 40 MB 常驻，且热路径 O(n) 串行。修复：按容量淘汰或插入前清理。核验：主会话（high）。
- **P2-41 `models.json` 原地重写，中断后整份目录为空且 24 小时内不复原**（`catalog/catalog.go:780` `os.WriteFile`；读取 563-570 解析失败即跳过；`Stale()` 只比较 mtime）。触发：同步过程被杀/断电/磁盘满。影响：所有模型列表、定价、Provider 列表为空，Usage 显示 $0，直到 24h 检查或手动 `magpie sync`。修复：`steady.WriteFile` + 读取校验。核验：主会话（high）。
- **P2-42 图片/视频命名循环在 Stat 非“不存在”错误时无限自旋**（`imagemcp/imagemcp.go:676-681`：只认 `ErrNotExist`，其余错误也继续递增名字）。触发：目录不可 stat（EACCES/EPERM/EIO、断开的网络盘）。影响：goroutine 满 CPU 永久生成 hero-2/3/…，已付费的图片永不落盘，`tools/call` 永远“still being made”。修复：其他错误立即返回。核验：主会话（high）。

### 前端状态
- **P2-36 首次 `/api/state` 失败后 Agents 页面永久停在骨架屏**（`gui/assets/app.js:4140` 先渲染骨架，`catch` 4215-4217 只弹 toast；`.loading` 只在 `renderAgents()` 429 行移除）。触发：首次加载遇到 500/502/504。影响：5 行闪烁骨架永久存在，无错误文案、无重试按钮。修复：`!state` 时渲染错误行并提供重试。核验：主会话（high）。
- **P2-37 较慢的 state 读取覆盖读取期间完成的写操作**（`app.js:4157-4180`：`shown` 只在首次加载时设置，后续加载 `state = next` 直接覆盖；connect/disconnect/keep 在 1165/2265/2297 直接写 `state`）。触发：窗口重获焦点触发慢读，用户同时点 Connect。影响：开关弹回、刚移除的账户又出现，直到下一次加载。修复：比较 `state` 身份后再赋值。核验：主会话（high）。
- **P2-38 `loadProvidersNow` 同样用旧读覆盖新写**（`app.js:5976-5981`：捕获了 `was` 却只检查 `kept`）。触发：Providers 页面刷新与新写并发。影响：开关/删除被回退。修复：赋值前加 `providers !== was` 判断。核验：主会话（high）。
- **P2-39 MCP 登录轮询在接口持续失败时永不结束**（`gui/assets/plugins/library/client.js:2117` `catch { continue; }`）。触发：登录过程中 Magpie/插件宿主重启，接口持续 4xx/5xx。影响：每 800 ms 一次请求直到页面关闭，界面永远“Finish signing in…”。修复：连续错误计数后进入 failed 状态。核验：主会话（high）。

## 4. P3（29 行；1 行含已驳回半边，28 条有效）

| # | 位置 | 问题 | 影响/修复 | 核验 |
|---|---|---|---|---|
| P3-1 | `gateway/codex_socket.go:304` | `SetReadLimit(-1)` 关闭 32 KiB 帧上限 | 恶意/异常上游可用超大帧耗尽内存；设合理上限 | 主会话 |
| P3-2 | `gateway/lan.go:398-412` | `ownCallback` 在交接窗口内接受任意存活 token 且免 key（randomToken 回退半边已驳回：Go 1.24+ `crypto/rand.Read` 不返回错误） | 见 P3-21：先做地址检查，仅跳过 key 要求 | 主会话 |
| P3-3 | `provider/provider.go:891` | 账户分支重建 Provider 时丢掉 `LimitRestOff`（字段在 207） | 账户永远无法启用“限额不休息”；补进白名单 | 主会话 |
| P3-4 | `provider/quota_alert.go:256` | `usage-alerts.json` 原地写、忽略解析错误 | 崩溃时告警记录损坏/重复；`edit.WriteAtomic` | 主会话 |
| P3-5 | `provider/quota_refresh.go:159` | `cacheCards(..., true)` 合并保留已删除账户的卡片 | 删除账户后卡片残留约 1 分钟；整页读用 false | 主会话 |
| P3-6 | `provider/codex_models.go:51` | 4 MiB 截断且丢弃 `ReadAll` 错误（同端点 `CodexLiveList:89` 用 16 MiB） | 大列表/断流被报成“没有模型”；统一上限并返回错误 | 主会话 |
| P3-7 | `gateway/chat.go:1276` | `[DONE]` 分支只 `giveHeld`，不 `end` | 流末尾被暂存的最多 9 字节文本丢失；补 `d.end(emit)` | 主会话 |
| P3-8 | `sessions/frames.go:228` | 完整 zstd 帧解码失败被当作“还没写完”，`end=start`、`next` 不变 | 该帧之后的内容永不出现，每次轮询重复解码；跳过坏帧 | 主会话 |
| P3-9 | `sessions/manage.go:596` | **【已驳回】证伪波 REJECTED（=sessions-store/F5）**：`defer in.Close()` 在 WalkDir 回调体内，逐文件关闭，无 fd 累积 | 见 §9 | 主会话 |
| P3-10 | `gui/archive.go:204` | 导出名 Stat 后再写，非原子占名 | 并发导出互相覆盖，备份“看起来有两份实际一份”；`O_EXCL` | 主会话 |
| P3-11 | `pluginrt/pluginrt.go:252` | `state==Failed` 永不重试，即使配置已改 | 修好配置也只能关开插件或重启；区分永久/可重试失败 | 主会话 |
| P3-12 | `agent/applied.go:71` | `appliedMu` 只在进程内，`applied.json` 无跨进程锁 | 两个 Magpie 共用配置目录时丢条目，漂移检测漏报 | 通道（medium） |
| P3-13 | `usageapi/usageapi.go:463` | 用当前页 `Rows` 再裁剪 `l.Accounts` | 账户筛选随分页变化；保留整期列表 | 通道（medium） |
| P3-14 | `usage/block_price.go:274`、`usage.go:759`、`ledger.go:452` | 只按 `in+out>0` 计价 | 仅缓存读写的调用成本记 0 且不标“未定价”；把 cache 计入条件 | 主会话 |
| P3-15 | `gui/assets/app.js:4306` | 每次 `renderUpdateBadge` 都启动一条 700 ms 轮询链，句柄不保存 | 下载中多次聚焦会并行多条轮询；保留单一 timer | 主会话 |
| P3-16 | `davsync/davsync.go:555` | `saveState` 丢弃 `WriteAtomic` 错误 | 同步报成功但基线未记录，下轮合并可能丢另一台电脑的改动；返回错误 | 主会话 |
| P3-17 | `gui/backend.go:96` | 网关看门狗持 `gatewayMu` 调用 `serveGatewayLocked`（含 500 ms 拨号/HTTP 探测） | 重启/接管被网络探测拖慢；锁外探测 | 主会话 |
| P3-18 | `gateway/key_models_list_test.go:254` | 基准仍断言 `group/auto-model-499`，发布版对外是 `auto/<model>` | `BenchmarkHeldKeyListsModels` 首轮即失败，#1762 的性能护栏失效；更新断言 | 主会话（实测） |
| P3-19 | `usage/request_pairs.go:68` | 配对 memo 失效时全量重建；实测 20 万行约 178 ms、111 MiB/次 | 活跃期每次追加都会让 memo 失效；考虑增量配对或降低上限 | 主会话（基准） |
| P3-20 | `gui/web.go:238` | 页面登录无失败计数，可无限猜 16 字符 key | 唯一管理凭据可被在线爆破；复用 gateway 的失败桶返回 429 | 通道（medium） |
| P3-21 | `gateway/lan.go:398-412` | 交接窗口内 `ownCallback` 在 allow-list 之前直接放行 | 被排除的地址可探测前任 token；先做地址检查 | 通道（路径 high） |
| P3-22 | `netproxy/lan.go:57` | `.local` 不解析即信任；未见过的名字在请求路径做最多 400 ms DNS | 绕过用户代理、地址可被恶意解析；解析并只信私有地址 | 通道（medium） |
| P3-23 | `maskrec/maskrec.go:201/218/441` | 同一互斥锁覆盖文件追加与整文件读取 | 请求结算被 Privacy 页读取阻塞；锁外做 I/O | 通道（medium） |
| P3-24 | `settings/settings.go:1754` | 保存时整体重新 marshal，未知键丢失 | 新版本/分叉写入的键被静默删除；保留未知顶层键 | 主会话（机制 high） |
| P3-25 | `omarchy/omarchy.go:380-392` | 主题刷新持锁跑 fontconfig/hyprctl（合计可达数秒） | 所有读主题的调用排队；锁外刷新 | 通道（medium） |
| P3-26 | `agent/wsl.go:1050-1057` | `wslRun` 无发行版侧停止，与 `wslrun.Tool` 的 cancelable/Stop 不同（证伪降级 =agent-runtime/F2） | 真正卡死的命令/挂载会活过 30 s；用可取消的 Stop | 通道（降级） |
| P3-27 | `agent/path_other.go:89-101` | shell 不应答时 `LoginPath` 每次都在 `login.Lock` 内重跑两次 5 s 询问（=agent-runtime/F3） | GUI 重试上限 3 次、间隔 4 s，行挂起数分钟；缓存失败并退避 | 通道（降级） |
| P3-28 | `sessions/frames.go:212-218` | 帧头损坏被当作“还没写完”，`readFrame` 的 `errFrame` 被吞，尾帧永久钉住（ADDED A1，另见 P3-8） | 之后的消息/token 永不出现；记 corrupt 偏移或跳到下一个 magic | 证伪新增 |
| P3-29 | `davsync/dav.go:319-323/353` | `put` 只接受/返回 ETag，丢弃可用的 Last-Modified（ADDED A2） | 无 ETag 的服务器写入无版本、下一轮 GET 无 304 重下整份；收发 `If-Unmodified-Since` | 证伪新增 |

## 5. 性能专项（实测）

运行方式（发布版快照，20 核，**有 4 个审计子代理并行，数字为上界**）：
`go test -run '^$' -bench . -benchtime=100ms -count=1 ./internal/gateway ./internal/usage ./internal/sessions ./internal/provider ./internal/plugin ./internal/redact ./internal/middleware`

关键数字（原始输出 `bench.log`，共 119 个基准项）：

- 组路由开销（200 KB 请求，单次）：直连 p0/m1 **4.7–4.9 ms**；12×20 目录的组 **5.0–5.4 ms**；30×60 目录的组 **6.6–7.3 ms**（+40%–55%）。
- 并行吞吐（200 KB）：12×20 **约 450–480 µs/op，730–780 MB/s**；30×60 **约 716–870 µs/op，400–490 MB/s**。
- 2 MB 请求：单次约 **32–40 ms**（90–100 MB/s）。
- `BenchmarkEvent` 998 ns/op、10 allocs；`BenchmarkHistoryLook` 34.5 µs。
- 高分配项：`BenchmarkPairRows` **178 ms、111 MiB/op**（20 万对）；`BenchmarkExportAmbiguousSession/8000` 10.2 ms、18 MiB；`BenchmarkFetchBodyProd/16MiB` 22 MB/op（整段读入）。
- 基准套件在发布版上 **FAIL**：`BenchmarkHeldKeyListsModels`（断言过时，见 P3-18），使 gateway 包的基准运行整体失败；`go test`（非 benchmark）未受影响。

与静态审查相互印证的热点：每请求整段落盘/重解析（P2-17）、静态资源清空目录缓存（P2-21）、`stickOf` 锁内磁盘 I/O（P2-10）、`LastSeen` 锁内 256 KiB 扫描（P2-26）、无上限 period 分配（P2-24）、配对全量重建（P3-19）。

## 6. 覆盖与未覆盖

| 通道 | 读取 | 明确跳过 |
|---|---|---|
| gw-core | 17 文件 / 19.6k LOC | 0（大文件读热函数+定点 grep） |
| gw-proto | 16 / 15.5k | 约 9.6k（请求构造半部、fastjson 1200-1591） |
| gw-subs | 10 / 15.1k | 0（两个 3k+ 文件读热函数） |
| prov-core | 24 / 约 5.3k（范围内 17.3k） | 约 12k（provider/group/balance/logins 未读部分、quotas.go） |
| prov-models | 16 / 7.1k（范围内 11.4k） | 4.4k（预设字面表、models.go 1701-2979 等） |
| usage-stats | 36 / 12.5k | 10 个辅助文件、request_page.go 部分区间 |
| plugin-host | 39 / 10.6k | 约 2.3k（安装/更新/registry/CA） |
| gui-api | 18 / 28.5k | 112 个文件（大部分 GUI handler、全部 routes 只做路由枚举+grep） |
| gui-assets | 未声明 COVERAGE（通道报告缺陷） | 未知（按最大 JS 模块优先） |
| sessions-store | 19 / 27k | 67 个（其他 agent 解析器、转录/搜索等） |
| library-sync | 20 / 32.2k | 76 个（library 的 mcp/hooks/targets/rtk、update 提权/COM、davsync/usage.go 等） |
| agent-wiring | 9 / 13.5k | 约 8.2k（各 agent 的目录/通知/CLI 主体） |
| agent-runtime | 50 / 10.8k | 143 个（配置写入归 agent-wiring；平台专属 proc/netproxy/autostart） |
| concurrency | 33 文件 / 22.5 万 LOC 扫描 | 674/707 文件（全树模式扫描+热点深读） |
| perf-hot | 18 / 23.7k | 4 个（无基准指向的大文件主体） |
| security-input | 35 / 22.9 万 LOC 模式扫描 | 1189 个非测试 Go 文件仅做 grep 分诊（邻近通道所有） |
| peripheral | 未声明 COVERAGE（通道报告缺陷） | 按包读取，未逐文件声明 |

**整体缺口**：GUI 的大型 handler（providers.go 剩余 1.4k 行、sessions.go、plugins.go、update.go、trace.go、groups.go、masking.go）与 `internal/routes` 只做了路由枚举和定点 grep；provider 的 group/balance/logins 主体、library 的 mcp/hooks/rtk、update 的平台提权/COM 路径未逐行读；Windows/macOS 平台专属文件大多未读。这些区域可能仍有缺陷，本报告不对其作“无问题”结论。

## 7. 上游已知问题对照

- **#4118**（pii 过期缓存 → 误报 masking 503）：在 v0.1.1957 上**已复现**（P1-1）。
- **#4101**（Chat 静默丢弃 video_url/file/input_audio）：代码仍无 `input_audio` 处理，`video_url` 被 switch 丢弃（P2-5）。
- **#3788**（非流式首响应超时）：`firstWait` 仍只对流式生效（P2-1）。
- **#4103**（MiniMax 2013 结构化输出绕过 JSON 回退）：本次未独立复核。
- 基准腐化（P3-18）未在上游 issue 中检索到对应记录。

## 8. 证据索引

- 发布确认：`gh api repos/yetone/magpie-releases/releases/latest` → `v0.1.1957`（16 资产）。
- 构建/vet：`baseline.log`（BUILD_EXIT=0, VET_EXIT=0）。
- PII 复现：`cache_yield_regression_test.go`（审计目录内）、`evidence-pii-4118.log`（TEST_EXIT=1）。
- 基准：`bench.log`（BENCH_EXIT=1，含 `BenchmarkHeldKeyListsModels` 失败）；`perf-hot` 通道的原始测量记录在 `waves/perf-hot.md`（`.bench.txt` 为临时输出，未保留）。
- 17 条通道原始报告与 6 份证伪报告：`waves/`（终局 6/6，见 §9：CONFIRMED 61 / DOWNGRADED 8 / REJECTED 5 / ADDED 4）。

---
审计过程只读：产品源码未修改、未提交；本地冲突工作区未触碰。报告与证据已复制到 `magpie-audit-20261029/`，提取的 129 MB 源码快照与编译缓存属临时资源，按计划清理（复核时可从 tag `v0.1.1957` 重新解出）。


## 9. 证伪终局（6/6 组，追加于交付后）

六份 `verify-*.md` 全部完成并已复制到 `waves/`，合计 **CONFIRMED 61 / DOWNGRADED 8 / REJECTED 5 / ADDED 4**。§0 的 44/29 已按正文对账（38 条保留 P2，P3 表含 28 条未驳回记录；跨表重复另见 FIX-MAP）。

**驳回（不计入）**
- `concurrency/F2` → **P2-3**（holdWriter 写锁饿死看门狗）：引文属实，但 `h.passing` 时只走该写路径，`started()` 也计入 passing，截止分支本就不会置 `h.slow` 或 `stop()`；该互斥锁就是写锁，无缺陷。
- `sessions-store/F5` → **P3-9**（`defer in.Close()` 在循环中泄漏）：证伪组判定不成立。
- `prov-models/F2`：与 `prov-core/F1`（providers.json 读-改-写无锁）重复，合并，不单列。
- `prov-models/F3` → **P2-14**（efforts union 缺口）：空 Efforts 表示“级别未知”，GUI 会填 `provider.Levels`，网关仅在 `len>0` 时钳制，`Account.Levels` 与 `effortMate` 仍可路由 high/xhigh，效应被驳。
- 另有 1 条 `randomToken` 回退（Go 1.24+ `crypto/rand.Read` 不会返回错误），已在 §0 说明。

**降级为 P3（8 条，按 P3 阅读）**
- `agent-wiring/F2`：SSH 私钥丢失仅在用户把密钥放在 magpie 自己的 `MAGPIE_GATEWAY_KEY` 名下时成立（文档保留该名字）。
- `agent-runtime/F2`：`wslRun` 无发行版侧停止，但触发器需真正卡死的命令/挂载，非 `sh -lc` 可复现。
- `agent-runtime/F3`：`LoginPath` 在 shell 不应答时确实重跑两次 5 s 询问，但 GUI 重试上限 3 次（间隔 4 s），队列有界，表现是行挂起数分钟。
- `concurrency/F1`：`searchRests` 的 load-compare-store 窗口仅数微秒，代价是一次 30 s 重试，不是每小时多 120 次拒绝。
- `prov-core/F3`：alert marks 非原子写，仅 GUI `WatchQuotas` 路径可触发。
- `prov-core/F5`：`partial=true` 保留消失卡片，但刷新会隐藏其他 provider、登出清空整缓存，残留不超过一分钟。
- `gw-core/F3`：`stickOf` 锁内 I/O 只在 mtime 变化时发生（`sticksFlushAfter=2s`），不是每次 miss。
- `gui-api/F3`：静态资源清目录缓存确实存在，但“每页 20–30 次重建”被高估；真实成本是懒加载图标/插件资源与并发读。

**新增（4 条，均已并入正文编号）**
- P2-43（`verify-agents` ADDED 1）：`stash`/`unstash` 丢弃 `stashWrite` 的错误，唯一副本可静默丢失。
- P2-44（`verify-agents` ADDED 2）：`stash.json` 读-改-写无跨进程锁；与 P3-12 的 `applied.json` 是不同文件、不同缺陷。
- P3-28（`verify-stores` A1）：帧头损坏被当作未写完，尾帧永久钉住；与 P3-8 同函数、不同出错路径。
- P3-29（`verify-stores` A2）：`put` 丢弃可用的 Last-Modified，写入无版本、下一轮重下整份。
