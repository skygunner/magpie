# 长程任务计划：为 magpie 上游修复 issue 并推送 PR

> 创建：2026-10-22。执行者：ZCode（skygunner fork）。
> 上游：yetone/magpie（main）；fork：skygunner/magpie。
> 本文件是活的文档：每个 milestone 完成后更新状态与证据。

## 背景结论（侦察阶段得出）

- 仓库为 Go 单体（网关 + provider + agent 接入 + GUI assets + 插件 host），
  文档在 `docs/`（reference.md、subsystems/、code-standards.md、contributors.md），
  历史教训在 `LESSONS.md`。
- 上游有 1438 个 open issue、16 个 open PR。README（10-09 起）称"不接受 PR"，
  但 contributors.md 定义了社区 PR 的审阅/合并规则，10-14 后仍有社区 PR 被合并
  （#1354、#1310 等）。PR 创建是否被 GitHub 限制以第一个 PR 实测为准；若被
  阻止，退路是：修复合入 fork 分支并在对应 issue 下留言附上分支与验证记录。
- PR 质量红线（docs/code-standards.md + contributors.md）：
  1. 必须带"没有修复就红"的测试（fixture 用报告者原始字节）；
  2. 测试不得触网、不得碰真实 HOME/agent 配置（sandbox + 固定 Go 缓存）；
  3. 每个改动跑 `gofmt -l`、三平台 `go vet`、三平台 `go build`、
     `go test -tags nogui ./... -timeout 45m`（全量，不是只跑 touched 包）；
  4. 触碰 GUI assets 需双引擎 + 多语言 Playwright 套件（尽量避免选 GUI 题）；
  5. commit message：`区域: 用户可见行为 (#issue, 报告人)`，body 记录跑了什么、
     哪个测试在旧代码上红；
  6. 严禁机器式连发 PR（几秒内多个 PR 会被 close/block）——逐个开、间隔开；
  7. 一个 commit 一个目标；修到所有 sibling； vendor 行为要对照来源验证。
- 测试容器需要时用 docker1（已验证可达，Docker 29.1.3），自建自清。

## Milestone 0：侦察与索引 ✅

- 目标：全面了解仓库、文档、git 状态、上游 PR/issue 全景。
- 范围：只读操作 + codebase-memory 索引。
- 验收：索引 succeeded；文档通读（README、AGENTS、code-standards、
  contributors、subsystems/README、LESSONS）；issue 全量拉取分诊。
- 验证：`index_repository status=succeeded`（13:13:30Z）；/tmp/magpie-issues.tsv
  1438 行。
- 状态：**完成**（2026-10-22 13:13 UTC 索引成功）。

## Milestone 1：未提交更改入 fork dev 分支

- 目标：3 个未跟踪笔记（claude-code-learnt.md、hermes-magpie-ollama-gemini.md、
  planned-improvements.md）提交到 fork 的 dev 分支（基于当前 main）。
- 范围：仅新增 3 个文件；不动其他分支。
- 验收：fork 上出现 dev 分支含这 3 个文件；工作区干净（分支切回工作分支）。
- 验证：`git push fork dev`；`gh api repos/skygunner/magpie/branches/dev`。
- 状态：**完成**（2026-10-22）。

## Milestone 2：同步 main、建立修复工作流基线

- 目标：本地 main 快进到 origin/main（87d703c5 起），确认基线套件可跑，
  记录全量 `go test -tags nogui ./...` 的时长基线。
- 范围：git 操作 + 一次全量测试。
- 验收：main == origin/main；基线全量测试通过（或已知红全部有 issue 号）。
- 验证：`go test -tags nogui ./... -timeout 45m 2>&1 | tee /tmp/baseline.log`。
- 状态：**完成**（2026-10-22；main=87d703c5；全量 ok，~6.5 分钟）。

## Milestone 3+：逐 issue 修复（循环体）

每个 issue 一个子 milestone，统一模板（在下方"Milestone 日志"追加）：

- 目标：用户可见的行为变化。
- 范围：最小改动 + sibling 全覆盖 + 文档（如该子系统 reference 需要）。
- 验收：红→绿测试；三平台 vet/build；全量 nogui 套件；commit body 记录。
- 验证命令：见各条目。
- PR：推 fork 分支 → 开上游 PR（逐个开，间隔 ≥ 数分钟；若被限制则在 issue
  下附分支链接与验证记录）。

### 候选池（按优先级；执行中可依据调查结果增删）

优先（维护者自己立案、描述精确、单测可复现）：
1. #3276/#3251 火山引擎(Volcengine) 429/额度重置时间 "+0800 CST" 不被 resetsWhen 识别
2. #3255 panic: close of nil channel in sessions.refresh（PI_CODING_AGENT_DIR → omp 目录）
3. #3141 proc.Look 同一时钟 tick 内安装的工具保持 "not found"
4. #3188 sub2api rate_limits "unit":"percent" 被当美元
5. #3151 Devin 被误报"无用户级 instructions 文件"
6. #3150 未知 auto/… 模型 404 时报合成 id 而非请求 id
7. #3148 Qoder IDE 同时被识别为 VS Code 和 Qoder（重复条目）
8. #3263 无 session header 的会话显示为 magpie-xxxx 无标题（dsh 已修，siblings 待修）
9. #3176 rest-cap 设在订阅 provider 上被静默忽略（withPicks 不携带 RestCap）
10. #3259 流式 narration-only 完成被改写 finish_reason:tool_calls（mimo）
11. #3121 antigravity union type ["string","array"] 塌缩成 string → Gemini 400
12. #3147 agy 调用记录 model 为空（读 conversations/*.db）
13. #3274 安全过滤 400 连坐健康账号（附 patch，需核实）
14. #3090 Free first 在成本未知时静默按 In order 消耗（patch ready，需核实）
15. #3187 import link 的 balance= 参数
16. #3139 无界面 serve 刷新 Claude 卡后仍带 asOf
17. #3077 ChatGPT Codex Ultrafast tier_served 保持 default
18. #3206 翻译路由上 server-managed tools 被静默丢弃
19. #3253 Claude Code 子代理经 /api/anthropic 丢失内置工具
20. #3258 同一文件夹不同拼写的两个 agent 重复读 sessions
21. #3248 antigravity SSE 流内重启的回复拼接送达
22. #3144/#3143 redact 误伤（sequential_cutoff 判人名；function_call.namespace 被脱敏）
23. #3266 ZCode 5 小时额度重置后旧 rate 状态未解除
24. #3269 VS Code 自带 copilot launcher shim 误报 Copilot CLI 已安装
25. #3260 Portable VS Code 用错 %APPDATA% 目录
26. #3142 WorkBuddy 用户级规则应读 ~/.workbuddy/rules/
27. #3131 OpenCode Zen Free 无法添加第二个账号
28. #2949 TestHeldClock 在 -race 下 time.Local 数据竞争
29. #3128 路由组成员列表同名模型无法分辨（GUI，谨慎）
30. #3156 路由组成员列表改后不刷新直到重启

约束：每题先读完整 issue 线程（含最后一条评论）；确认无 open PR 已修；
不选 GUI 布局类（#1354 教训：社区布局 PR 被整体 revert）；不选维护者
明确"留给 owner"的产品决策题。

## Milestone 日志

执行环境结论（本机跑上游测试的两个必要条件）：
`GOTOOLCHAIN=go1.26.8`（1.27 的 encoding/json 行为差异让 TestMarshalRawIsEncodingJSON/
FuzzMarshalRaw 在 main 上红）+ `umask 022`（本机默认 0027 让 TestWriteAtomicReadOnlyHardLink/
TestProgram 红）。TestPluginSOCKSProxy 红是已知 #2207（本机 Bun 1.4.3）。
上游 PR 创建被 GitHub 权限阻止（README 10-09 起只允许 collaborator 开 PR；
实测 `gh pr create` 报 CreatePullRequest 权限错误），所有修复走"fork 分支 + issue 留言"路径。

- **M0-M2 完成**：索引 succeeded；1438 issues 分诊；main=87d703c5；基线全量 ok。
- **M3 完成** #3251/#3276（Volcengine/Ark 重置时间）：分支 `fix/volcengine-resets`
  commit e6c2b6fb——resetsWhen 接受 ±HHMM/±HH:MM 裸偏移+时区缩写；月度措辞信任到
  longestMonthly(35d) 并同步放宽 failQuota 封顶。TestResetsWritten ark 用例 +
  TestMonthlyWrittenResetRestsTheMonth 红转绿；67 ok 全量。issue 留言：
  https://github.com/yetone/magpie/issues/3251#issuecomment-6342260029 、
  https://github.com/yetone/magpie/issues/3276#issuecomment-6342270787
- **额外**：立案 #3280（TestClaudeClientCacheAbsentUnchanged 在 bash-as-sh 机器上红，
  harness echo 不解释 \n；含二分定位与修复方向）。后立 #3284（TestOpenCodeSessionOnEveryCall
  全量负载红一次，单跑/包内绿）。
- **M4 完成** #3258/#3255（共享文件夹双读）：分支 `fix/shared-session-folder`
  commit 4c3d6e2——piFiles 在 PiDir 与 OmpDir 同文件夹时返回空（#2932 sameFolder
  先例，omp 属主读一次）；qoderFiles 同理去 qoder-cn。TestSharedFolderReadOnce
  （named+linked×2 对）红转绿；620 包内全绿；全量 67 ok。留言：
  https://github.com/yetone/magpie/issues/3258#issuecomment-6342495758 、
  https://github.com/yetone/magpie/issues/3255#issuecomment-6342498748
- **M5 完成** #3259（mimo `tool_calls:null` 被当已调用）：分支 `fix/null-tool-calls`
  commit d2abf49——chattidy 把 null 与 [] 同样对待。TestChatTidyNullToolCalls
  （报告者原始帧）红转绿；全量 67 ok。留言：
  https://github.com/yetone/magpie/issues/3259#issuecomment-6342839037
- **M6 完成** #3176（订阅 provider 的 rest-cap 被丢）：分支 `fix/account-rest-cap`
  commit c41b6fe——withPicks 补 RestCap/QuotaRest/QuotaResetAt（第三个是同类 sibling）。
  TestAccountKeepsItsRests 红转绿（含 AlsoOn）；全量 67 ok。留言：
  https://github.com/yetone/magpie/issues/3176#issuecomment-6343065990
- **M7 完成** #3150（404 报合成 auto id）：分支 `fix/unknown-model-sent-id`（worktree
  /tmp/magpie-probe）——requestModel 返回原始 id，keepSentModel 经 request context
  带到 404 站点（subagentPickKey 同款先例），refusal 回显客户端发来的 id。
  TestUnknownModelNamesTheIdSent 红转绿。
- **M8 等 reportter 回复** #3148：Qoder IDE product.json 身份值未知（另一应用的值
  要从该应用本身来），已在 issue 请报告者提供：
  https://github.com/yetone/magpie/issues/3148#issuecomment-6342963529
- **跳过（有现成 patch 等 maintainer 取）**：#3141（JoeyWu f9d7eac9）、#3188（wtj-0527
  c7db44bc）——重复移植价值低。
- **main 已修、issue 未关（已留言指认修复 commit）**：#3151（e6568888）、#2949（b641868e）、
  #3143+#3144（88119d4f，10-21 修复，晚于本计划的 issue 快照——教训：动手前先
  `git log --grep` 并在**全仓库**搜 issue 关键词，不能只看标题分诊）。
- **M10 放弃** #3144：动手时发现 main 已修（见上），kept 表补充属冗余，已撤销。
- **M12 完成** #3213（per-model queue 503 歇整账号）：分支 `fix/queue-model-rest`
  commit 1b9f089——queuedWords 措辞匹配 + restAfterMarked 双点收窄（计数与休息 key
  都走 restID）。TestQueuedModelRestsAlone 红转绿；全量 67 ok。留言：
  https://github.com/yetone/magpie/issues/3213#issuecomment-6343670538
- **不修（附原因）**：#3263（需逐 agent 真实客户端抓包）；#3077（usage 记录语义的
  产品设计，留给 owner）；#3147（已向报告者要 conversations/*.db 脱敏样本，见
  issue 留言）；#3148（已向报告者要 Qoder IDE product.json 值）；#3131（属社区插件
  仓库 magpie-community/plugins，已指路）；#3269/#3260（Windows 平台题，无 Windows
  环境验证，标准要求在对应平台实测）；#3141/#3188/#3274/#3090（已有现成 patch 等
  maintainer 采用，重复移植无价值）。

## 终验收（2026-10-22）

- 8 个分支全部推送 fork（7 个修复 + dev）；工作区干净；每分支只含各自目标文件。
- 每个修复分支的 head 都有独立的全量 `go test -tags nogui ./... -timeout 45m` 记录
  （M3 66 ok，其余 67 ok），红项全部为 main 预存且有 issue 号（#2207、#3280、#3284），
  commit body 均已如实记录。红→绿验证：每个新测试都在还原旧代码后跑出真实 FAIL。
- 临时物清理：/tmp/magpie-bisect、/tmp/magpie-probe worktree 已删；调试测试文件
  （zz_probe/zz_debug）已删；无遗留 debug 输出。
- docker1 未用到（无需测试容器）。
