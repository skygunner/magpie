## 2026-10-24 第十轮后续独立执行：#3401 完整补齐、#3440 产品决策、#3447 收口

本节由本轮直接执行并记录，非 zcode 返工；**#3447 不再扩张实现**（上游独立实现 `1d6b72ad` 已被报告者在 Magpie 0.1.1371 + DSH 实测，issue 关闭），`fix/issue-batch-1024pm` 的旧混合补丁不提交上游。

| 项 | 执行动作与实际结果 | 状态 |
|---|---|---|
| R1 #3401 | 新建独立工作树 `../magpie-reasonix-1024`，基于 `dd1e8822`，cherry-pick 仅 Reasonix MCP/skills 的 `fd4bcb54`，另完成原生 `settings.json` Hooks（平面 `hooks.<Event>[]`、`match` 正则、毫秒超时、Reasonix 原生事件）、`~/.agents/skills` 的 `readsShared` 归属。完整交付链 `e3bcacfc -> 0281ada7 -> cf87dc37`。独立分支普通推送及远端 SHA 读回一致。 | 完成；Windows 真机待报告者 |
| R2 #3401 验收 | 真实 Library 写入/保留用户 Hooks/删除 Magpie-owned 项及共享目录测试通过，`go test -tags nogui ./internal/library -count=1` 通过，Linux 和 Windows/amd64 静态 `go vet` 通过。最终独立 head `cf87dc37`，`umask 022` 的全量 nogui：**68 ok / 2 FAIL / 19 no-test，exit=1**；预存失败 #3280/#2207，非全绿。Windows Studio 2.50 原生未执行。 | 本地验收通过，实机待验证 |
| R3 #3440 | 向上游原 issue 发布 A/B/C 产品决策澄清 [6383458761](https://github.com/yetone/magpie/issues/3440#issuecomment-6383458761)，认可 #674 原合同并纠正默认 272K 的作用域扩张；**不自行选 A/B/C，也不请求合并争议补丁**。GitHub 读回作者/正文/时间一致。 | upstream_decision_pending |
| R4 #3401 上游沟通 | 发布独立分支、测试与 Windows Studio 2.50 实机验证请求 [6383557929](https://github.com/yetone/magpie/issues/3401#issuecomment-6383557929)，读回一致；无 PR、无真实账户调用。 | 等报告者原生运行结果 |
| R5 #3447 | 上游已修并由报告者实测；原 fork searchWith 的缺口仅留历史事实，不继续写第三个变体。 | superseded_by_upstream_independent |

原始命令、哈希、评论回执与剩余验证范围见 [round13/closeout.md](docs/fork/evidence/round13/closeout.md)。仅推独立 fork 分支及 dev 维护记录，未删除分支、未 force-push、未触碰另一 agent 的工作树。

## 2026-10-24 第十轮返工复核（ZCode 会话十二）：r11 审查限定收口

r11 独立复核（magpie-r11-review）确认：14c1551 的多搜索累计/URL 去重全绿；8 条
comment_publications 的 author/issue/时间/UTF-8 SHA256 与 GitHub API 8/8 吻合；
selftest 19 负例全拒。实质遗留＝我方分支 searchWith 丢失搜索器真实 cited_text
（reviewer 最小回归已红）。**新上游事实**：yetone 已合入独立实现
1d6b72ada8c9（语义＝按回答文字实际写出的 URL 为对应行附引用，摘录取自搜索模型
回复的 text.citations），上游 main 现为 9d643631（超出审查时 7d9165）；**报告者
HuHong-Tao 已在真实 0.1.1371 上验证 1d6b72ad 覆盖其原路径**（两条路由、流式
citations_delta 先于文字、DSH mapAnthropicResponse 拿到 snippet），**issue 已于
08:38Z 关闭**，报告者明言两个 fork 分支无需再验证。ggbdpq 的 69068ead 与
1d6b72 是两套合同（搜索器摘录直引 vs URL 提及行附引），其 e2e fixture 在
upstream 上 2 红 1 绿（fixture 的模拟回答不含 URL）——如实记录，不当"上游没修"
或"已等价采纳"。边界同前：不发上游评论/PR、不 force-push、不删分支、不碰他人
worktree；fork 分支无产品改动则无需推送。

| M | 目标 | 修改范围 | 验收 | 验证命令 | 状态 |
|---|---|---|---|---|---|
| M0 | 精确重现与上游对照 | 临时树（不提交） | reviewer 的 searchWith 红测试在 14c1551 红（"real searcher cited_text lost"）；绿色控制组并列；upstream tip 自有 H20 测试绿；ggbdpq 三 fixture 在 upstream 上复跑得 2红1绿（语义差异自有证据）；新上游 SHA 与 #3447 关闭状态入档 | go test（临时 archive 树）+ gh api | 完成（红测试于 14c1551 archive 精确复现 "real searcher cited_text lost"；绿色控制 7 项全绿；upstream 9d643631 自有 H20 三测试绿；ggbdpq 三 fixture 于 upstream 复跑 2红1绿；#3447 closed 08:38:42Z、报告者 6382090015 实测 1d6b72ad；证据 evidence/round12/behavior-tests.md） |
| M1 | 裁定：产品补足 or superseded（M3.4 路径） | 无产品提交 | 依 NEXT-PROMPT M3.4：上游已由报告者在真实环境证明覆盖用户原路径 → 我方 #3447 单元标 superseded_by_upstream_independent（非 pass/adopted），不做第三套变体；semantic_comparison_pending 记录两套合同差异 | 行为测试证据 + 台账 | 完成（裁定＝superseded 路径：不做第三套变体；单元 verify.state=superseded_by_upstream_independent、upstream.status=independent_alternative/semantic_comparison_pending，coverage 记完整 SHA、两套合同差异与本单元 searchWith 残余缺口；generate.py+selftest 过） |
| M2 | 台账与两份草稿 | docs/fork | #3447 单元改 superseded 状态、upstream.status=independent_alternative、coverage 引完整 SHA 与差异；#3440 维持 decision_pending；八条评论记录不再改写；**#3447 中文更正草稿单独成文（不发送）**：更正「两路由均代搜」错误、说明上游独立实现与报告者验证、声明不与 ggbdpq 竞争；#3440 草稿原样保留 | generate.py + selftest | 完成（单元口径已改；八条评论记录未再动；docs/fork/comment-draft-3447-correction.md 单独成文未发送——两处更正+合同分歧边界+不竞争声明；#3440 草稿原样） |
| M3 | 最终验收 | 全部 | 冻结产品 head 仍为 14c1551（远端读回一致）；定向套件全绿；全量 nogui 一次（预存红分类）；gofmt/vet/diff --check；dev 推送读回；报告含 SHA/日志/剩余边界 | go test -tags nogui ./... + git | 完成（全量于冻结 14c1551：68 ok / 2 预存 FAIL #3280+#2207 / 19 no-test，零新增；gofmt/vet/diff--check 过；产品树 clean、远端读回 14c1551 一致；dev 推送读回见下） |

### 本轮记录（2026-10-24 完成）

- **裁定：走 NEXT-PROMPT M3.4 的 superseded 路径，不做第三套产品变体。** 依据：
  ①上游 1d6b72ada8c9（v0.1.1369+）已由报告者 HuHong-Tao 在真实 0.1.1371 上端到端
  验证覆盖 issue 原路径（两条路由、流式+整包、DSH mapAnthropicResponse；评论
  6382090015），维护者据此关闭 issue（08:38:42Z）；②ggbdpq 69068ead 与上游是两套
  合同（搜索器摘录直引 vs URL 提及行附引），其 e2e fixture 在 upstream 9d643631
  上 2红1绿（本轮自有复现，非转述）；③我方分支再出一个变体只会制造第四份语义。
- 产品分支 fix/issue-batch-1024pm **无新提交**：冻结 head
  **14c1551e9672758c656ed2fd6b4db69ac1300b15**，远端读回一致（base dd1e8822 共
  7 commits 不变）。e5a05c44+14c1551 保留为「搜索器摘录直引」备选；其 searchWith
  对搜索器 cited_text 的缺口由 r11 探针在 14c1551 上精确复现（临时 archive 树，
  产品树未动），作为 superseded 单元的已记录残余缺口，不修补。
- 行为测试证据：docs/fork/evidence/round12/behavior-tests.md（红复现、绿色控制
  7 项、H20 自有三测试绿、fixture 2红1绿、裁定依据）。
- 台账：#3447 单元 verify.state=**superseded_by_upstream_independent**（非
  pass/adopted、非归档候选）、upstream.status=independent_alternative +
  decision=semantic_comparison_pending，coverage 记完整上游 SHA、报告者验证、
  关闭时间、两套合同差异与本单元残余缺口。generate.py + selftest（19 负例）过。
- 草稿（均未发送）：**docs/fork/comment-draft-3447-correction.md**（单独成文：
  更正「两路由均代搜」、说明上游独立实现与报告者验证、记录合同分歧、声明不与
  ggbdpq 竞争）；docs/fork/comment-draft-3440-r10-rework.md 原样保留
  （#3440 维持 upstream_decision_pending）。八条评论记录本轮未再改写。
- 全量 nogui（冻结 14c1551，串行）：**68 ok / 2 包 FAIL / 19 no-test，exit 1**；
  FAIL＝#3280、#2207（干净 dd1e8822 基线同红），零新增。日志
  evidence/round12/full-suite-14c1551-close.log。未做 Windows/macOS 原生验证
  （仅跨平台 build/vet 历轮口径；本轮无产品改动）。

## 2026-10-24 第十轮返工（ZCode 会话十一）：r10 审查限定范围

独立审查（magpie-r10-review，REVIEW.md + NEXT-PROMPT.txt）确认五个交付的定向测试
全绿、全量口径精确（68 ok / 2 预存 FAIL / 19 no-test，基线复现），但确定：①#3447
多搜索 citations 覆盖缺陷（P2，红测试在审查包 tests/ 已验证红：two_searches 与
later_empty_search 的 whole/stream 四检查全失败）；②#3440 属产品默认行为扩张
（ggbdpq 已 downgrade 为诊断并明确留给维护者决定；先认领重叠需纠正）；③台账八条
comment_publications 的 published_at/verified_at 全误填 2026-10-24T16:55:00Z、
body_sha256 全空、meta.checked_at 疑似本地钟写成 UTC；④#3401 需标 partial、
#3392 需记剩余范围。**最新上游事实**：ggbdpq 已于 06:36:17Z 在 #3447 发布完整交付
（hunt76-a-websearch-citations，head 69068ead，并指正我方交付帖「两条路由都走代搜
管线」的说法——deepseek 官方在 searchHosts[Anthropic] 是 passthrough）。边界：
不发上游评论/PR（草稿待授权）、不 force-push、不归档删除、不碰他人 worktree；
fork 分支普通快进推送沿用既有授权。

| M | 目标 | 修改范围 | 验收 | 验证命令 | 状态 |
|---|---|---|---|---|---|
| M0 | 复现与固定 | 产品树 internal/gateway + API | 六提交顺序核实；红测试 4/4 失败于 4f353657；8 条评论真实 UTC 拿到；ggbdpq 最新动作入档 | go test -run TestReviewMultipleSearchCitations；gh api | 完成（六提交与远端读回一致；探针 4/4 红；created_at 与审查所列逐条吻合；ggbdpq #3447 已于 06:36:17Z 交付 69068ead、#3440 06:30:09Z downgrade 为诊断） |
| M1 | #3447 多搜索 citations 累计（P2） | internal/gateway/anthropic.go（流式+whole 两处累计：URL 去重保序、无摘录不造、首个真实 text 后清空） | 审查探针红→绿；单次搜索/无摘要/其他 Anthropic/Responses 回归绿 | go test ./internal/gateway -run 'Citation\|Search\|Anthropic' | 完成 14c1551（accumulateCitations；探针绿；Search\|Anthropic + Custom 422 控制绿；提交已 amend 移除误加的伪造 Co-author） |
| M2 | #3440 产品决策整理 + 草稿 | dev docs + 草稿文件 | 单元状态改 design_sensitive/upstream_decision_pending；覆盖描述改口径；一条谨慎更正草稿（不发送，致谢先认领、承认默认行为扩张、列三备选） | python3 docs/fork/generate.py | 完成（单元 upstream.decision=upstream_decision_pending、coverage/evidence 改口径；草稿 docs/fork/comment-draft-3440-r10-rework.md 含 A/B/C 备选与两处认领重叠更正，未发送） |
| M3 | 台账真实时间与哈希 | docs/fork/upstream-status.json + prior-comments.json + generate.py | published_at=GitHub created_at（8 条各自实际值）；verified_at=本轮新读回时刻；body_sha256 从 API 取 body 计算并验证作者/issue；#3401 标 partial、#3392 剩余范围入 coverage；不碰历史七条 | gh api + python3 json 校验 | 完成（8 条 published_at=created_at 逐条核实、verified_at=2026-10-24T07:14:36Z 实际读回、body_sha256 从 API body 计算并断言作者/issue、body_updated_at 记录；meta.checked_at 改 07:14:36Z；证据 evidence/round11/comment-readback.json；历史条目未动；generate.py+selftest 过） |
| M4 | 最终收口 | 产品树 + dev | 新 head 定向包+全量回归（预存红引用基线）；gofmt/vet/diff --check；推送读回；SHA/日志/剩余范围清单 | go test -tags nogui ./... | 完成（全量于 14c1551：68 ok / 2 预存 FAIL #3280+#2207 / 19 no-test，零新增；分支推送读回 14c1551；dev 台账随本轮提交推送） |

### 本轮记录（2026-10-24 完成）

- 产品提交：**14c1551e9672758c656ed2fd6b4db69ac1300b15**（fix/issue-batch-1024pm，
  dd1e8822..14c1551 共 7 commits：8ecab590/b834f286/e9926d70/e5a05c44/fd4bcb54/
  4f353657/14c1551）。红→绿：审查探针 review_r10_multi_search_test.go 于 4f353657
  4/4 红（two_searches、later_empty_search 的 whole/stream）、14c1551 全绿；
  Search\|Anthropic 与 Custom（含 422 状态控制）绿。已普通推送，读回一致。
- 全量 nogui（14c1551，GOTOOLCHAIN=go1.26.8 umask 022 串行）：**68 ok / 2 包 FAIL /
  19 no-test，exit 1**；FAIL＝#3280 TestClaudeClientCacheAbsentUnchanged 与 #2207
  TestPluginSOCKSProxy，干净 dd1e8822 基线同红（round10 已证），零新增。日志：
  docs/fork/evidence/round11/full-suite-14c1551.log。审查提到的 provider 包 #3284
  偶发在本轮全量与定向跑中未出现。
- 台账：8 条 comment_publications 的 published_at 逐条改为 GitHub created_at
  （05:56:00Z–06:18:36Z），verified_at=2026-10-24T07:14:36Z（本轮真实 API 读回），
  body_sha256 从 API body 计算（作者/issue/时间断言后写入），历史七条未动；
  meta.checked_at 同步为真实 UTC。单元口径：#3440
  upstream_decision_pending/design_sensitive（coverage 与 verify.evidence 写明
  默认行为扩张与 #674 合同冲突）；#3401 partial（hooks 未实现 remain、无原生
  Windows 验证）；#3392 记剩余范围（Claude 400 无错误体，无从修）；#3447 记
  ggbdpq 重叠交付与指正。generate.py + selftest 过。
- 草稿（未发送，待单独授权）：docs/fork/comment-draft-3440-r10-rework.md——
  致谢 ggbdpq 先认领与诊断、承认 e9926d70 默认行为扩张并更正交付口径、列出
  A（维持 #674）/B（扩作用域=现状）/C（仅文案）三备选请维护者定，含 #3447
  「两路由均代搜」说法的更正（deepseek 官方为 passthrough）。
- 未解决/剩余范围：①#3440 等维护者产品决定（草稿未发）；②#3447 上游侧
  ggbdpq 已交付独立修复，我方分支保留为备选，无进一步上游动作；③#3392 的
  Claude 400 半边等报告者错误体；④#3401 hooks 等事件名、原生 Windows 未验证；
  ⑤#3444 真实大图后端未对照。

## 2026-10-24 新一轮修复（ZCode 会话十）：#3375–#3447 巡检后的未认领 issue

上游新开 69 个 open issue（#3375–#3447）。逐条读帖并扫描评论区后：
**已被他人认领/交付**（跳过不撞车）：#3377（Azir-11 websocket 补丁 + ggbdpq 验证）、
#3382/#3388/#3399/#3416/#3418/#3427/#3435（ggbdpq）、#3386/#3421/#3424（JoeyWu 补丁）、
#3397/#3398/#3419/#3420（JoeyWu 自带修法）、#3426（wssz527 自己的实现）；
**报告者自带参考实现**（跳过）：#3423（louisgreen0726 分支）、#3425/#3429（chiperman
提交）、#3430（mdwsk88 参考）、#3437（标题即 patch included）、#3442（xinghui 分支+测试）；
**功能请求/产品决策**（跳过）：#3378 opencode2、#3381（double2tea 自备分支）、#3384/
#3389 ZCode 券与套餐、#3385/#3400/#3428 路由策略增强、#3387 数据屏接口、#3391/#3438/
#3445 UI 建议、#3395/#3396/#3443 新能力；
**维护者域**（跳过）：#3376/#3393/#3402–#3415/#3422 yetone 的 GUI/CI 红名单；
**信息不足/设计使然**（回帖说明或要数据）：#3375（本地回环免鉴权是设计，Anarlog 的
key 探测无法区分）、#3379/#3390（仅截图、无文本可定位）、#3394（仅截图）、#3433
（要 vscode 插件请求头）、#3439（XFF→非本地判定是 #1022 的隧道防护，已有
MAGPIE_TRUST_PROXY=1，回帖指路）、#3441（重试风暴，等更多数据）、#3383（性能，
本轮不动 moejs/记录保留，工作量与风险超一轮）。
**修**：#3392、#3444、#3440、#3447、#3401，弹性项 #3434/#3436。

- 现场：worktree `../magpie-fix-1024`，分支 `fix/issue-batch-1024pm` 基于
  upstream/main `dd1e8822`（= v0.1.1364，含 a2e98cd7 的 codex_socket.go——dev 树
  落后 101 提交，全部调查已在新树复核）。边界沿用：每修一题红→绿、包内全绿、
  三平台 build/vet、全量 nogui 串行、push fork 读回、issue 回帖（中英随题）。

| M | 目标 | 修改范围 | 验收 | 验证命令 | 状态 |
|---|---|---|---|---|---|
| M0 | 现场固定 + 候选甄别 | 本节 | 上表；refs 对齐 | git/worktree/gh 扫描 | 完成 |
| M1 | #3392：xAI 422 "unknown variant `custom`" 不在 customToolRefusal 内，翻译兜底不触发 | internal/gateway/custom_tool.go | 报告者错误原文红→绿；DeepSeek 原例不回归 | go test ./internal/gateway -run CustomTool | 完成 8ecab590（红→绿实拍；Custom* 21 测试绿） |
| M2 | #3444：codex websocket Write 遇 StatusMessageTooBig 关闭帧 → 回退 HTTP 而非 502 | internal/gateway/codex_socket.go | 假 ws 服务器回 1009 关闭帧，测试断言走 HTTP 完成 | go test ./internal/gateway -run CodexSocket | 完成 b834f286（红＝报告者 502 原文复现；CodexSocket|CodexTurn 8 绿） |
| M3 | #3440：官方订阅原生模型不套全局 272K 压缩（CodexNativeWindow 不读 settings.Compact） | internal/gateway/codex_backend.go + internal/provider/codex_models.go（ETag 标记） | 475000 条目在默认/自定义阈值下被压到 272K/自定义并保留 max；用户自设窗口优先；FullContext 不压；tag 变化触发重取 | go test ./internal/gateway ./internal/provider -run 'Native.*Window|Compact' | 完成 e9926d70（默认 272K/自定义 300K/FullContext/用户窗口四断言；provider+codexcat+agent 全绿；两处 ETag 期望随 CodexListTag 公式更新） |
| M4 | #3447：magpie 自搜管线的 Anthropic 回复 text 块无 citations[]，下游只拿 url+title | internal/gateway ir.go/search_api.go/search.go/anthropic.go | Hit 带 Cited；renderAnthropic 与流式编码器在 text 块给 citations（web_search_result_location，cited_text≤150 字符）；无 snippet 时不造 | 同上 -run 'Citation|Search' | 完成 e5a05c44（whole+streamed 两路断言；Search|Anthropic 221 绿；仅回退编码器时红） |
| M5 | #3401：Reasonix Studio 进 Library（MCP [[plugins]] TOML、skills、项目目录） | internal/library/ | `library mcp/skill` 对 reasonix 可用；现有 agent 不回归 | go test ./internal/library/... | 完成 fd4bcb54（fmtReasonix 数组表写入/读回/删除保留用户项；skills+项目目录；library 410 绿；hooks 未接——事件名未知，已在回帖说明） |
| M6 | 回帖类：#3439/#3375 指路（TRUST_PROXY/本地信任模型）、#3433 要数据 | 上游评论 | 草稿→发布→读回 | gh | 完成（#3439 comment 6380614794；#3375 comment 6380622846；#3433 comment 6380630767） |
| M7 | 弹性：#3434 路由按 native turn 续判（保新 task 语义） | internal/gateway/rules.go | 报告者 4 步协议测试 1–4 全过 | -run 'Turn|Classif' | 放弃：报告者已有本地原型与完整协议语义（delegation 仍须立新 task 等），改动横跨 rules.go 键控与原生元数据管道，回归面大且撞车风险高（其将自行交付） |
| M8 | 全量回归 + 推送 + 回帖发布 + 台账 | 全部 + docs/fork | 全量 nogui 一次（预存红引用基线）；分支读回；修复回帖；台账 JSON/MD 重建 | go test -tags nogui ./...；python3 docs/fork/generate.py | 完成（见本轮记录） |

### 本轮记录（2026-10-24 完成）

- 交付分支 `fix/issue-batch-1024pm`（base dd1e8822 = v0.1.1364，6 commits＝5 修复
  + 1 文档，head 4f353657；产品 head fd4bcb54）：#3392 8ecab590、#3444 b834f286、#3440 e9926d70、#3447 e5a05c44、
  #3401 fd4bcb54。每题红→绿实录见各 milestone 状态列。已推 skygunner/magpie 并
  读回一致。
- 回帖 8 条全部以 skygunner 发布并经 API 读回验证（5 修复交付 + #3439/#3375
  指路 + #3433 要数据）：3392→6380779544、3444→6380789247、3440→6380798386、
  3447→6380807714、3401→6380817606、3439→6380614794、3375→6380622846、
  3433→6380630767。
- 全量 nogui（串行，head fd4bcb54，GOTOOLCHAIN=go1.26.8 umask 022）：
  **68 包 ok / 2 包 FAIL / 19 no-test**，exit 1。FAIL＝TestClaudeClientCacheAbsentUnchanged
  （gateway，#3280）与 TestPluginSOCKSProxy（plugin，#2207）；同环境在干净
  dd1e8822 基线工作树上复跑此两测试**同样失败**——零新增失败。日志与基线说明：
  docs/fork/evidence/round10/（full-suite-fd4bcb54.log + manifest.json）。
- 三平台 build+vet（windows/amd64、darwin/arm64、darwin/amd64，nogui）干净。
- 台账：upstream-status.json 增 5 units（全 40 位 SHA + 交付评论）、92 分支记录、
  comment_publications 15 条、deliveries 增 fix/issue-batch-1024pm（6 commit 全链，
  head 4f353657＝产品 fd4bcb54 + 文档提交）；prior-comments.json 增 5 行（77 行）；
  REQUIRED_SET 扩至 77 项（product 60）；generate.py 校验过（65 units /
  69 unit-covered / 90 remote heads）+ selftest 负例全拒。UPSTREAM_STATUS.md 已再生。
- 文档义务：五个修复的语义变更已写入子系统页（gateway-routing 三处、
  providers-accounts 一处、library 一处，commit 4f353657），随分支推送读回。
- 放弃项与理由：#3434（见 M7 行）；#3383（moejs/记录保留的性能重构超出单轮
  风险预算）；#3436（Kelivo 快照匹配需重写 conversation 续接判定，报告者已具
  备完整离线复现，留待其交付或下轮）；#3441（8 分钟重试风暴——数据不足以定位
  单一可修点）；#3379/#3390/#3394（仅截图无文本）。

## 2026-10-24 第三轮返工（ZCode 会话五）：R1–R4 + 台账收口

独立审查（Magpie-v2-independent-review，REVIEW.txt）在精确 head 上复现 R1–R4、
测试/文档缺陷与台账缺口。本节为逐 milestone 计划；状态实时更新。

- 现场确认：三 v2 head + dev 与审查 refs 一致；upstream/main 已前移 a2e98cd7→
  b437e4ad（3 commits，未触碰 redact/decide/rename）；r3-batch/r3-union/r3-rests
  干净 rebase 到 b437e4ad，无冲突。边界不变：仅普通推送本轮分支、读回 SHA；
  不发上游评论/PR/tag；评论更正仅草稿；无真实供应商/生产操作。

| M | 目标 | 修改范围 | 验收 | 验证命令 | 状态 |
|---|---|---|---|---|---|
| M0 | 固定范围/版本/口径 | PLAN.md | 本表；refs 对齐 | git rev-parse/remote/worktree | 完成 |
| M1 | R1：凭据例外收紧（24 矩阵） | internal/redact/ | 24 项绿 + 原三输入/良性样例/B/缓存绿 + 全包 | go test ./internal/redact（含 TestReviewV2*） | 完成（a0b2faa8；其后 R1 新回归见第四轮 M1） |
| M2 | R2/R3/R4：provider 身份与决策语义 | internal/provider/ (+gateway 测试) | 三探针绿 + 原 C/D 测试绿 + 撞名正控 + Catalog 不丢 + 文档 | go test ./internal/provider ./internal/gateway -run 'TestReviewV2\|TestPluginPicked\|TestPluginServes' | 完成（完成（5b4e0d5f；其后续见第四轮 M2）） |
| M3 | 测试证据/文档修正 | gateway 测试 + docs | relayReply 合法 JSON + G 断言加强 + 无冲突标记 + E/F 链 full SHA | go test ./internal/gateway -run 'TestAgentMessage\|TestChatEndings'; git diff --check | 完成（完成（113867ad）） |
| M4 | 72 条逐项对账 + 分支归属 | docs/fork/ (dev) | 每项 permalink/SHA/fix_id 可追溯；85+ 远端 heads 归属；指定集合一致 | python3 docs/fork/generate.py（严格校验器） | 完成（完成（dev 台账重建）） |
| M5 | 校验器负例 + 归档候选规则 | docs/fork/generate.py + JSON | 五类坏数据全拒；火山三层准确；候选 read-only | 校验器 + 负例测试 | 完成（完成（--selftest 五负例；九负例见第四轮 M4）） |
| M6 | 最终回归 + 推送核对 | 全部 | 全量 nogui 串行 + 基线证据 + 推送读回 | go test -tags nogui ./... | 完成（见下） |

### M6 记录（2026-10-24）

- 环境：Linux 6.18 amd64（CachyOS，bash as /bin/sh），go1.26.8（GOTOOLCHAIN），umask 022，Bun 1.4.3，隔离 XDG。Windows/macOS 原生未执行（无环境）；跨平台 build/vet 仅静态检查（三平台过）。真实供应商/真实守护进程未调用。
- 全量 nogui（r3-batch head 885c1b8b，串行，45m 超时）：**67 包 ok，2 失败**，exit=1。失败＝TestClaudeClientCacheAbsentUnchanged（gateway，#3280）与 TestPluginSOCKSProxy（plugin，#2207）。同环境基线：干净 upstream/main（b437e4ad）上同两测试同样 FAIL——**无新增失败**；"全套全绿"未达成（这两个预存红仍在）。#3284 本轮未复现。
- 推送读回：r3-batch 885c1b8b、r3-union 275b7a2e、r3-rests d639c8d7、dev——local=remote 全一致。远端 heads 现 89（85+3 r3+1 origin 杂散，见台账分支表）。本轮远端删除 0、无 force-push。
- 评论草稿：docs/fork/comment-drafts-2026-10-24.md（5 条，覆盖 R1-R4 修正与新 head，未发送）。

## 2026-10-24 第四轮返工（ZCode 会话六）：r3 验收剩余项

独立审查（magpie-r3-review，REVIEW.md）在精确 head 上复现三项代码缺陷与四处台账缺口。
边界不变：r3-batch/dev 向前普通提交，r3-union/r3-rests 本轮无产品缺陷不重建；无 force-push、
无上游交互、无真实供应商；远端删除 0。上游 main 已前移（审查时 a2189463，现场 a93190ae），
但触及文件（redact/provider.decide/rename/provider.go/gui.providers）自 b437e4ad 无漂移，
r3 树追加提交有效。环境：go1.26.8（GOTOOLCHAIN），Bun 1.4.4（1.4.3→1.4.4 变化记录），
umask 022 用于测试。

| M | 目标 | 范围 | 验收 | 状态 |
|---|---|---|---|---|
| M0 | 现场/证据固定 | PLAN + 工作树 | refs 对齐、漂移记录、探针就位 | 完成 |
| M1 | R1 新回归：紧凑/缩写/后缀凭据名（apikey/APIKey/api_key_v2/password_primary，8 项） | internal/redact | 8 项红→绿；原 24 项、三输入、良性控制、B/cache 绿；全包绿 | 完成 bbd6e228（strongCredential 改名字包含语义；40 项定向+全包绿） |
| M2 | R4 补遗：无 key 无 Name 本地 decision provider 被隐藏；R3 补遗：DecisionModels 漏显式 Jev（GUI deciders 元数据） | internal/provider + gui | 两探针红→绿（4e136a4b）；全部控制组绿；provider 全包绿；gui deciders 契约测试绿 | 完成 4e136a4b（accountOrPluginID 按 id 归属；DecisionModels 并入显式 picks；providerInfo 元数据断言） |
| M3 | 台账 r3 关系：heads/fix_ids/pushed 状态、union/rests 真实 r3 链、PENDING 替换、base 补齐、草稿 v2/r3 混用修正 | docs/fork JSON+MD+草稿 | 校验器过；链与远端读回一致；历史头另存 | 完成（deliveries 对象+真链+PENDING=0+base 全补；草稿 #3370/#2008 改引 r3 真链并记录被替代关系） |
| M4 | 校验器：新增 9 类负例 + MD 保留 fix_ids + %(symref) 判据 + totals 计算 + 候选从图派生 + evidence receipt | generate.py + evidence/ | 九负例拒、正确数据过 | 完成（--selftest 14 负例全拒=原 5+新 9；merge_branch_records 把 JSON 归属合入实时行，head 漂移显式标注；totals 重算比对；候选三方一致校验；evidence/round4/manifest.json） |
| M5 | 最终回归与收口 | 全部 | 最终 head 全量 nogui 串行（失败有同环境基线）、diff/gofmt/vet、推送读回、PLAN/草稿一致 | 完成（见 M6 记录与本节 evidence） |

## 2026-10-24 第五轮返工（ZCode 会话七）：r4 验收限定范围

r4 独立验收（magpie-r4-review）确认三项上轮修复已接收；本轮限定范围＝两个新代码回归
（R4-A token 复合 key 字段泄漏 P1；R4-B 公开 -plugin ID 签出后 picks 被当独立 provider P2）
+ 两个维护校验问题（R4-C local-only 历史记录缺 where 在异 clone KeyError；R4-D 四个归档
门槛未独立生效）+ 文档/草稿/manifest 订正。边界不变：普通快进推送、读回 SHA、无上游交互、
无归档删除、不碰 fix/codex-expiry-window。上游 main 现场为 dd1e8822（与审查抓取一致），
触及文件自 b437e4ad 无漂移，r3-batch 追加提交有效。环境 go1.26.8/Bun 1.4.4/umask 022。

| M | 目标 | 范围 | 验收 | 状态 |
|---|---|---|---|---|
| M0 | 版本/边界固定 | PLAN+工作树 | 四 refs 与审查一致；漂移记录 | 完成 |
| M1 | R4-A：token_signing_key/token_encryption_key 6 项泄漏 | internal/redact | 新 6 项红→绿；旧 8+24+三输入+良性+B/cache 绿；全包绿 | 进行中 |
| M2 | R4-B：openai-plugin 签出后 picks 被当独立 provider | internal/provider | 新签出探针红→绿；正控/原 C/D/V2/R3、撞名、remote refresh、gui deciders 全绿；providers-accounts.md 修正 | 待做 |
| M3 | R4-C：local-only 记录缺 where 异 clone KeyError | docs/fork/generate.py | 记录契约统一（merge/count/render）；原机与缺席模拟都可生成 | 待做 |
| M4 | R4-D：四个归档门槛独立生效 | docs/fork/generate.py + selftest | 新 4 负例逐条拒绝；原 14 负例保持；正确数据过 | 待做 |
| M5 | 文档/草稿/manifest 订正 + 最终全量 + 推送读回 | 维护分支 + 产品分支 | 订正七项落地；manifest started/范围收窄；全量串行+基线；读回 | 完成（见下） |

## 2026-10-24 第六轮返工（ZCode 会话八）：r5 验收限定范围

r5 独立验收确认 R4-A/C/D 与原 openai-plugin 签出场景已接收；剩余＝R5-PROVIDER-ID
（P2：cline-plugin 独立 provider 被误归属隐藏）+ R5-DELIVERY-CHAIN（P2：deliveries
commits 漏本轮 3 个新提交）+ 草稿/PLAN 文字订正。边界不变。现场 refs 与审查一致
（r3-batch 1952329f，upstream/main 现场值见 git）。

| M | 目标 | 范围 | 验收 | 状态 |
|---|---|---|---|---|
| M0 | 版本固定 | PLAN+工作树 | refs 一致；探针就位 | 完成 |
| M1 | R5-PROVIDER-ID：cline-plugin 独立 provider 被隐藏 | internal/provider | 新探针红→绿；R4 签出/V2/R3/R4 全家绿；provider 全包绿 | 完成 820eb896（-plugin 归属仅当 PluginID 契约真会后缀 raw id；15 探针全绿、provider 全包绿） |
| M2 | R5-DELIVERY-CHAIN：r3-batch commits 补全（25 条，含本轮 820eb896）+ 完整性校验 | docs/fork + generate.py | delivery-check 三条 exit0；新增截断负例共 19 负例全拒 | 完成（commits 由 rev-list 重生成；完整性 gate 拒截断） |
| M3 | 草稿/PLAN 文字订正 | 草稿+PLAN | #3365 绿 head、#3370 首段、#2008 三链分开、本轮表 | 完成 |
| M4 | 最终全量 nogui + evidence + 推送读回 | 全部 | 串行全量；失败分类；读回 | 完成（820eb896 全量；evidence/round6。注：该表在 cbd4ed9b 提交时被遗漏，本轮补交） |

## 2026-10-24 第七轮返工（ZCode 会话九）：r6 验收限定收口

r6 独立验收接收 delivery 链/19 负例/便携脚本/草稿订正/原 R5 反例；唯一剩余产品项＝
R5-PROVIDER-ID 的组合场景（raw cline 已保存 Decide picks 时，空闲的 cline-plugin 仍被
误认领）。审查已提供最小候选 patch（stored-endpoint 条件补 !accountPicksEntry）与配套
反例测试，均经定向验证。边界不变。

| M | 目标 | 范围 | 验收 | 状态 |
|---|---|---|---|---|
| M0 | 版本固定（820eb896/cbd4ed9b） | PLAN+工作树 | refs 一致；候选/测试哈希核对 | 完成 |
| M1 | stored-endpoint 补 !accountPicksEntry | internal/provider | 新反例红→绿；R3/R4/R5/V2/撞名/refresh/SystemOne 全绿；provider 全包 | 完成 7f927133 |
| M2 | 台账同步 + 三处记录订正 | docs/fork + PLAN | delivery/portable/selftest 三检查过；66 包口径/isolated-3284 身份/PLAN 遗漏补交 | 完成 |
| M3 | 最终全量 nogui 一次 + evidence + 推送读回 | 全部 | 串行全量；失败分类精确；读回 | 完成（7f927133 全量：66 ok/3 FAIL/19 no-test，exit1；#3280/#2207 基线引用，#3284 本 head 单测绿（日志带 head）；evidence/round7） |

# 长程任务计划：为 magpie 上游修复 issue 并推送 PR

## 2026-10-25 第十二轮（ZCode 会话十四）：三题独立分支 + M0/M1 维护

接第十一轮。按 ROUND12-CONTINUE 执行：一 issue 一独立 fork 分支（不再 batch）。

| M | 目标 | 结果 |
|---|---|---|
| M0 | round15 回执修复 + PLAN 口径 | 完成：comment-readback.json 重写为 7 条真实回执（author/issue/UTC/UTF-8 SHA256，checked_at 2026-10-25T01:20:13Z）；round15 表格 M2/M4/M6/M8/M10 残留「待做」、M7/M14 误标「完成」均已按真实边界改列；#3470/#1105 剩余边界如实标注 |
| M2-A | #3117 Desktop 3P 会话归属 | 完成：`fix/claude-desktop-3p-sessions` 58a0f39c（base b027ca26），entrypoint 前缀判定三处（calls/traces/list-state），双 RED→GREEN，推送读回，中文回帖 [6392132800](https://github.com/yetone/magpie/issues/3117#issuecomment-6392132800) |
| M2-B | #3043 Registry 20s 超时 | 完成：`fix/mcp-registry-timeout` 11d176bd，Registry 专用 60s client（skills.sh/图标不动），基线探针红（报告者错误原文）→绿，对照（超自身预算仍报指名错误），推送读回，中文回帖 [6392132968](https://github.com/yetone/magpie/issues/3043#issuecomment-6392132968) |
| M2-C | #3470 Stats.Days follow-up | 完成：`fix/sessions-day-cache-unknown` 4264a4b7（base=cherry-pick fe330ea1 d6c12abb，依赖已写明），NativeSessionUnknownByDay 按日 join + page.js 累加，RED→GREEN，推送读回，英文回帖 [6392134728](https://github.com/yetone/magpie/issues/3470#issuecomment-6392134728)。**同 issue 补齐，不计新题** |
| M1 | 历史对账 | 完成：commenter:skygunner open 83 vs 台账 72 units——差异全部为有意分类（PR 评论 3、已关 4、upstream_fixed 5、other 2、回帖类 5、我方立案 1、本轮新交付 2），无失联线程；sprints/2026-10-25/followup.json |
| 只读 | #3456 要数据 | 完成：中文索取上游脱敏请求原文+最小 JSON diff [6392141495](https://github.com/yetone/magpie/issues/3456#issuecomment-6392141495) |
| M3 | 扩池 | 完成：零评论池 622 扫 4 页 + 上轮全量分诊复核。新排除（带据）：#3430/#3442/#2918（他人完整补丁 OTHER_PATCH_READY）、#3537（需真实订阅）、#3190（2b80641d 故意行为，需真实 Codex 后端验证）、#3156（快照语义产品决策）、#2951/#2952/#3118/#3477（信息不足）、#3108（defer）。candidates.json 全账 |
| M5 | 账本 | 完成：sprints/2026-10-25/{deliveries.jsonl,candidates.json,followup.json}；upstream-status.json 75 units/95 heads，REQUIRED_SET 86 项，generate.py+--selftest 过；全量 nogui 于合并树 bc193d08（三题无冲突合并，证据 evidence/round16-full-suite-combined.log） |

## 2026-10-25 第十三轮（ZCode 会话十五）：#3530、#3529、#3531 + M0 证据重做

按 ROUND13-CONTINUE 执行（base = upstream/main `641b1217`）。

| M | 目标 | 结果 |
|---|---|---|
| M0 | round16 HandedOver 证据 | 完成：干净 b027ca26 与合并树各 3 bursts 原始日志（evidence/round17/handedover-rerun-*.log，每次含 HEAD/命令/逐行输出/exit）——基线 1/3 红、合并 2/3 红，flake 在未触碰基线复现，预存归因成立；README 修正日志路径（round16-full-suite-combined.log 在上级目录，SHA256 7f1be897… 与原始日志吻合）+ manifest.json；partial 的 handedover-baseline-check.txt 保留为当时原样 |
| M1-A | #3530 Codex 多 HOME | 完成：`fix/codex-multi-home-sessions` 8b9022cd——per-folder 标题缓存（pass-freeze #3493 语义保留）、codexStateDBs 尊重各 folder 的 sqlite_home（相对该 folder），CODEX_SQLITE_HOME 限定主 folder；双 RED→GREEN；英文回帖 [6392564071](https://github.com/yetone/magpie/issues/3530#issuecomment-6392564071)。**披露**：工作树操作失误使 #3531 提交落进同一条链（8b9022cd → 7597f816），不 force-push；单题取用 cherry-pick 各自 SHA（两题文件不相交），已在两处回帖写明 |
| M1-B | #3529 Trae credits | 完成：`fix/trae-checkin-credits` 0a2a0d74——claim credits 单取（extra_credits 仅在 credits 未说时），绝不取和；synthetic fixtures（如实声明无线上事故）；RED→GREEN；英文回帖 [6392564252](https://github.com/yetone/magpie/issues/3529#issuecomment-6392564252) |
| M2 | #3531 侧栏端口 | 完成：`fix/sidebar-gateway-line` 7597f816——save 应答画侧栏行；portSays 移入 i18n.js 腾 605 字节（预算不动，TestCoreScriptBudget 1,321,347/1,321,424）；Playwright 双引擎 en+zh RED→GREEN；sidebar/api-key-theme 的 12 红经干净树对照为预存；英文回帖 [6392783800](https://github.com/yetone/magpie/issues/3531#issuecomment-6392783800)（含披露） |
| M2 甄别 | #3536/#3539/#3528/#3522/#3526/#3511 | #3536 需逐 agent 外部源码验证 startup timeout 语义（无本机证据不猜）；#3539 的 413 failover 是 #700 设计（per-vendor 字节上限），报告者亦声明不能归因 magpie——不修；#3528/#3522 产品语义待维护者；#3526（Edward4226 06ebeb75 已交叉验证）、#3511（ggbdpq 认领）不重复 |
| M3 | 全量分页 | 完成：零评论 open 623 条 7 页到穷尽（zero-comment-audit.json 每条含 author/title）；169 条 yetone 域；116 条 bug 标题初筛；下轮候选 16 条入 candidates.json coverage |
| 合并回归 | 3530+3529 合并树 | 完成：69 ok / 2 包 FAIL（#3280、#2207 已知预存）/ 19 no-test，零新增（evidence/round17-full-suite-combined.log）；三平台 build/vet、gofmt 干净 |

累计真实修复：9（前两轮）+ 本轮 #3530、#3529、#3531 三个新 issue = **12 个不同 issue**（#3470/#3470-follow-up 不重复计）。

## 2026-10-25 第十四轮（ZCode 会话十六）：M0 standalone refs + #2866、#2777、#3303

按 ROUND14-CONTINUE 执行（base = upstream/main `42976f9a`）。主仓库他人未提交的
internal/gateway/fallback.go 修改全程未触碰。

| M | 目标 | 结果 |
|---|---|---|
| M0 | 独立分支交付形状修正 | 完成（无 force-push）：`fix/codex-multi-home-sessions-standalone-1025` = 8b9022cd（单提交、仅 internal/sessions）；`fix/sidebar-gateway-line-standalone-1025` = 71d5d7c0（7597f816 cherry-pick 到干净 641b1217，仅 gui/i18n/测试），新 head 上重验：TestCoreScriptBudget 过、Playwright 4/4、sidebar/api-key-theme 12 预存红对照一致。两条原帖评论以 Pickup note 编辑增补（保留原文，API 读回 author/UTC/SHA256，evidence/round18/pickup-note-readback.json）；#3529 未动 |
| M1-A | #2866 Atom 中文分段 | 完成：`fix/atom-zh-sections` 7aa04408——zhLed 块级切分（Han≥Latin）落在所属标题下；RED=报告者混排 fixture（zh 半边只剩安装段/en 混入中文条目）→GREEN；site 32 测试（原 21）+internal/update 47 绿；英文回帖 [6393086996](https://github.com/yetone/magpie/issues/2866#issuecomment-6393086996) |
| M1-B | #2777 签到账户代理 | 完成：`fix/checkin-account-proxy` 69eaf094——CheckinVia（ViaLogin）包全部五个 runner 构造（Trae/MiniMax/Qoder/WorkBuddy+插件 AccountCheckin）；契约测试 RED→GREEN（账户自有出口/无自己代理跟 provider/全无留空）；provider 42s 全绿；英文回帖 [6393185272](https://github.com/yetone/magpie/issues/2777#issuecomment-6393185272)；边界如实：未对真实厂商压签到 |
| M1-C | #3303 DSH npx | 完成：`fix/dsh-npx-config-writes` b8f2a4c8——userDirs 补 ~/.npm/_npx/*/node_modules/.bin（仅按文件发现，缓存内不执行）；proc+agent 双层 RED→GREEN；卸载残留闸门保持；回帖 [6393262850](https://github.com/yetone/magpie/issues/3303#issuecomment-6393262850)；边界：Linux 测试验证，无原生 Windows/macOS 运行（Windows FindTool 本就覆盖） |
| M2 | #3191 网络题 | 评估后留独立一轮：写阶段无界是 ResponseHeaderTimeout 之后的真缺口，但修复涉 TryNet.Send 结构+failover 语义+stall 后连接复用三件事，报告者已给出可复现指引（可控 h2 服务+大 body 填流控窗口），下轮首要候选。#3330/#2892 OTHER_PATCH_READY；#2822/#2836/#2736 产品语义待维护者 |

合并回归：三题合并树串行全量 nogui（evidence/round18-full-suite-combined.log）。
台账：81 units / 103 远端 heads，REQUIRED_SET 92 项，generate.py+--selftest 过。

累计真实修复：12（前三轮）+ 本轮 #2866、#2777、#3303 = **15 个不同 issue**。

## 2026-10-25 第十五轮（ZCode 会话十七）：#2741、#3561、#3567 + #3327/#3539 甄别

按 ROUND15-CONTINUE 执行（base = upstream/main `1b039a3c`）。他人未提交的
internal/gateway/fallback.go 全程未触碰。

| M | 目标 | 结果 |
|---|---|---|
| M1-A | #3327 Mistral empty_response | **BLOCKED_ARCHITECTURE**：#667 的空回复重问只存在于翻译 continuation 回路；OpenCode→mistral 的 Chat passthrough 完全没有 re-ask 机制——移植属架构改动而非本轮。response_error 标记本身真实且 vendor 特定，已入 candidates.json 供后续/维护者 |
| M1-B | #2741 Command Code replay | 完成：`fix/commandcode-replay-reasoning` 741237cf——`chatReplaysReasoning` 对 commandcode 全域匹配（插件 id + `api.commandcode.ai`），#1462 式；不造 reasoning，只回放 turn 实际携带的。双 RED→GREEN；英文回帖 [6394163102](https://github.com/yetone/magpie/issues/2741#issuecomment-6394163102) |
| M1-C | #3539 嵌套 413 | **TRIAGED_TO_PLUGIN**：结构化 `param.statusCode=413/isRetryable=false` 在 cline 插件 `failure()/errorResponse()` 内被拍扁成 message 字符串，magpie 只见 502 带字——按消息正则猜不安全（会误伤普通 502/429）；一行修在插件仓库。中文调查帖（含链路证据与插件侧修法）[6394204236](https://github.com/yetone/magpie/issues/3539#issuecomment-6394204236)，typo 编辑回执 evidence/round19/3539-typo-fix.json |
| M2 | #3561 下载器 proxy | 完成：`fix/pii-download-proxy` 55dfe0a5——报告者诊断与 `proxied()` 方案（已致谢）：transport 走 `netproxy.Func`+`Dispatch`；行为 RED（假发布主机仅有代理一条路，旧代码直连且照常成功）→GREEN；pii 17 绿。**披露**：提交又落在 #2741 工作分支后拆出（无 force-push），远端 2741 分支保持单提交；文件不相交、各自可 cherry-pick，回帖已写明。英文回帖 [6394265586](https://github.com/yetone/magpie/issues/3561#issuecomment-6394265586)。#3191 留独立轮 |
| 新发现 | 上游 main 红测试 | 回归归因时发现 `TestNoCommandBypassesProc` 在干净 `1b039a3c` 稳定红（-count=2）——上游当日 `75e78a5a`（keep.go）用 os/exec 弄红自家守卫；= 维护者自报 [#3567](https://github.com/yetone/magpie/issues/3567)（零评论，CI 40166079788，issue 自述修法）。完成：`fix/tui-keep-proc-command` 64bd29fa——`proc.Command` 一行修 + os/exec 移除；proc/tui 双包绿；英文回帖 [6394559364](https://github.com/yetone/magpie/issues/3567#issuecomment-6394559364) |
| 回归 | 合并树 | 三修复合并树 9e7f5275：pii/proc/tui 全绿、守卫 -count=3 绿；package-suites 日志三红全部带据归因（#3280 已知、HandedOver 已知 flake、NoCommand=上游 #3567 已修），见 evidence/round19/README.md |

台账：84 units / 106 远端 heads，REQUIRED_SET 95 项，generate.py+--selftest 过。

累计真实修复：15（前四轮）+ 本轮 #2741、#3561、#3567 = **18 个不同 issue**。

扩池补充审计（同轮晚些）：#2968（Cowork 配置目录未知，需 macOS 真机）、#3379/#3107（仅截图）、#3149（产品诉求）——均已带据入 candidates.json。

## 2026-10-25 #3392 跟进轮（issue follow-up）：xAI null required 的数组嵌套缺口 + 第三轮补记

报告者 10:08Z 复测：codex 已通，claude 仍 400 同一条 `/required: null is not of type "array"`。重读 c172b65d（其交付会话未做台账收口，本轮补记）发现真实缺口：`dropNullRequired` 只沿对象递归、不进数组——zod union 的 `anyOf` 分支或元组 `items` 里挂的 null 原样穿到 xAI。

| M | 目标 | 结果 |
|---|---|---|
| M1 | 数组递归修复 | 完成：`fix/issue-batch-1024pm` bf84dd4f——`dropNullRequired` 增 `[]any` 分支，对象与数组同样递归；TestXAITakesANullRequiredToolSchema 扩 anyOf+items-tuple 两工具，c172b65d 上红（报告者原文逐字复现，null 在 anyOf/items 里可见）→ bf84dd4f 绿；定向 23 项绿 |
| M2 | 全量与静态 | 完成：gateway 包 nogui 修复头/基线同 sandbox HOME 各一次（~254s），4 红两侧逐个相同（TestClaudeClientCacheAbsentUnchanged、TestCodexFittedImagesPastTheBudget、TestMarshalRawIsEncodingJSON、FuzzMarshalRaw，与 c172b65d 提交信息同晨记录一致）＝零新增；gofmt、vet（linux/linux-nogui/windows）、build（linux/linux-nogui/windows）干净 |
| M3 | 交付与读回 | 完成：普通 push c172b65d→bf84dd4f，ls-remote 读回一致；中文回帖 [6397710434](https://github.com/yetone/magpie/issues/3392#issuecomment-6397710434)（含「先确认构建包含 c172b65d」与「再失败就贴 input_schema 原文」两个可执行请求）；两条评论 API 读回 SHA256 入 evidence/round20/comment-readback.json |
| M4 | 台账补记 | 完成：unit 追加 c172b65d+bf84dd4f、coverage/verify/deliveries/branch head 更新；上一会话遗漏的 6395007475 一并补记为 delivery_followup；prior-comments.json 同步；generate.py --write + --selftest 过（84 units / 106 远端 heads） |

边界：真实 xAI 未调用；报告者复测与 input_schema 原文待回。本轮不算新 issue（#3392 已在台账），不占 sprint 第十六轮编号。

累计真实修复：第十一轮 7 题 + 本轮 2 题新 issue（#3117、#3043）+ #3470 follow-up（同题补齐）= **9 个不同 issue**。距 30–60 目标尚远，候选池审计后的真实可得率是约束（见 candidates.json 的 15 条带据排除）；继续按轮推进。

## 2026-10-25 第十一轮（ZCode 会话十三）：#3448–#3520 巡检 + 未认领 issue 修复

### 巡检结论（本轮开头完成，2026-10-25）

- 上游 main 前移到 `7d67910f`（10-24 一天约 40+ 提交，yetone 修复了
  #3464（3a47e377）、#3493（三连）、#3383（4b9694c3）、#3394（v0.1.1400/1424/1425/1428）、
  #2554、#3022、#2973、#2297、#2328、#2656 等）。
- 新增 open issue #3448–#3520 共 70 个：约 20 个是 yetone 自己的 GUI/CI 红名单
  与 run record（维护者域，跳过），约 12 个功能请求/产品决策（#3461/#3462/#3463/
  #3459/#3486/#3482/#3490/#3504/#3454/#3458/#3457/#3466 等，跳过），2 个仅截图
  （#3506/#3507），Windows/macOS 专平台题（#3495/#3467/#3472/#3318/#3238，无
  对应真机，跳过）。
- **认领排除**（本轮最重要事实）：ggbdpq 10-24 一天内认领并交付了新批次中的
  #3510/#3517/#3452/#3453/#3508；JoeyWu-tech 认领 #3480；Totopo27 交付 #3471；
  Yaka88（报告者）自带 #3460 修复；Edward4226 自带 #3172 修复；#3520/#3481/
  #3381 报告者自带分支。旧积压（#2471/#2887/#2873/#2833/#2826/#2809/#2798/#2591/
  #2526/#2725/#3351/#3373/#2622/#3346/#1475/#3380/#3055/#2969 等）全部已被
  ggbdpq 认领或在飞行中。**这些一律不做，避免重复劳动。**
- 我们参与过但仍开放的 issue 共 65 个（台账 69 减去已关闭的 3173/3276/3285/3447），
  全部已有交付分支与回帖，等待上游吸收，本轮不重复动作。
- 本轮可做的未认领、未修目标（见下表 M1 起）。

### 本轮记录（2026-10-25 完成；2026-10-25 round16 口径订正）

- round16 订正：①上表 M2/M4/M6/M8/M10 曾勾选但状态列残留「待做」，M7/M14 曾误标
  「完成」而实未动手——已按真实提交/验证边界改列；②round15 的
  comment-readback.json 曾是空 comments 数组（脚本按行解析 gh api 的多行 JSON
  失败），已用 GitHub API 逐条 GET 重写真实回执（作者/issue/UTC/UTF-8 SHA256，
  checked_at 2026-10-25T01:20:13Z），七条评论未改动、未重发；③#3470 的
  native Stats.Days 按日剩余与 #1105 的 edits multipart 边界在上表如实标注。
- 交付分支 `fix/issue-batch-1025a`（base upstream/main `7d67910f`，7 commits，head
  `b78a4f07`，产品提交 d4944ace/fe330ea1/91fc2733/3053f99f/3e3fd41a/fb29f08c/b78a4f07）：
  #3487、#3470、#3519、#3514、#1105（GEEKCX 半边）、#3513、#3455。全部已推
  skygunner/magpie 并读回一致。
- 回帖 7 条全部以 skygunner 发布并经 API 读回验证（语言随原帖：3487/3470/1105/3455
  英文，3519/3514/3513 中文）：3487→6389261270、3470→6389269893、3519→6389270118、
  3514→6389275410、1105→6389275629、3513→6389279389、3455→6389279618。
  #3519/#3514 的报告者修法被原样移植并在回帖中声明致谢与可替代。
- 全量 nogui（head b78a4f07，串行）：**68 ok / 2 包 FAIL / 19 no-test**；FAIL＝#3280、
  #2207（历轮同款预存红，基线引用）。日志 evidence/round15/full-suite-b78a4f07.log。
  三平台 build+vet（windows/amd64、darwin/arm64、darwin/amd64）干净；gofmt 干净
  （skillgit_test.go 上游预存未格式化，未动）。
- 本机新装 Playwright（chromium+webkit，用户级 icu74/flite 解包进浏览器目录），GUI
  测试自此可跑：quota-amount-unknown（新，双引擎 en/zh 8/8 绿）、quota-credits、
  quota-count-wrap、sessions-overview（4 个预存本机 locale 红在基线资产上同样出现，
  非本轮改动所致）。
- 台账：upstream-status.json 增 7 units（72 总）、comment_publications 27 条、
  deliveries 增 fix/issue-batch-1025a、branches 92 远端 heads；prior-comments.json
  增 7 行（84）；REQUIRED_SET 扩至 84 项；generate.py + selftest 过。
- 文档义务：provider-plugins.md（#3470 的覆盖与剩余）、gateway-routing.md（#3455 的
  url_citation 语义）已更新，随分支推送。
- 放弃项与理由：#3474（yetone 自立且标明"fix needs a way to tell which session's
  main model"，机制选型留给维护者）；#3449（网关侧 WebSocket 大功能）；#3516
  （GUI 资产纪律本轮不扩）；#3489/#3461 等 FR（产品决策）。留作下轮已分诊候选：
  #3456（报告者已补诊断，待读全）、#3475、#3117、#3043、#3512、#3417。

### 边界（沿用）

现场 worktree `../magpie-fix-1025`，分支 `fix/issue-batch-1025a` 基于
upstream/main `7d67910f`。每修一题：读全 issue 线程 → 确认无人认领/上游未修 →
红测试（报告者原始数据）→ 修 → 绿 → 包内全绿 + 三平台 build/vet → 全量 nogui
（预存红 #2207/#3280 基线引用）→ push fork 读回 → 原帖回报（原 issue 什么语言
就回什么语言）→ 台账。commit message `区域: 行为 (#issue, 报告人)`。

| M | 目标 | 修改范围 | 验收标准 | 验证命令 | 状态 |
|---|---|---|---|---|---|
| M0 | 巡检 + 候选甄别 | 本节 | 上表；refs 对齐 | gh / git | 完成 |
| M1 ✅ | #3487 quota 卡：插件 span 毫秒被读成秒（窗口起点 2024）+ 未知用量渲染成满额 | internal/usage 或 provider quota 解析 | 报告者 JSON 红→绿；正常秒/百分比不回归 | go test 定向 | 完成（见本轮记录） |
| M2 ✅ | #3470 会话文件命中率把 cache_unknown 记为 miss | internal/sessions + tui/cli/gui 展示 | Kiro 0 缓存读会话不计入 miss；普通会话率不变 | go test ./internal/sessions ./internal/tui ./internal/cli | 完成（fe330ea1；剩余：native Stats.Days 按日仍未排除，见 round16 follow-up） |
| M3 ✅ | #3519 纯 IPv6 主机被 tcp4 回退锁 30 分钟 | internal/provider 或网关 dialer | AAAA-only 主机不进 30min 惩罚；双栈主机仍可回退 | go test 定向 | 完成（见本轮记录） |
| M4 ✅ | #3514 dsh 把 Claude Code 订阅的 Claude 模型写进 openai-completions 路由 | internal/agent/dsh* | Claude 模型进 magpie-anthropic；deepseek/chat 模型不回归 | go test ./internal/agent -run Dsh | 完成（3053f99f） |
| M5 ⏸ | #3474 /model 切换后 haiku/opus/subagent tier 环境变量要等 GUI Follow | internal/agent/claude.go | 写路径与 Follow 同步 tier 值；classifier 先例不回归 | go test ./internal/agent -run Claude | 待做 |
| M6 ✅ | #1105 images-API-only 模型在非 openrouter 主机仍被发 chat/completions（gitee z-image-turbo） | internal/gateway/draw.go viaFor | gitee 案例 404 红→绿；openrouter 路由不变 | go test ./internal/gateway -run Draw | 完成（3e3fd41a；边界：仅 generations 方言已测，edits multipart 未映射未测） |
| M7 ⏸ | #3456 华为云 InferHub 400 参数无效（读全诊断后定） | internal/gateway 翻译 | 报告者载荷红→绿 | go test 定向 | 暂缓：未动手——缺上游脱敏请求原文，只向报告者索取关键参数与模型名最小 JSON diff，不假修 |
| M8 ✅ | #3513 ChatGPT 账号 gpt-image-2.5 忽略 size | internal/gateway/draw | size 翻译断言；不回归既有 draw 测试 | go test ./internal/gateway -run Draw|Image | 待做 |
| M9 ⏸ | #3475 Copilot 拒绝 tool-search-tool beta 名 | internal/gateway/anthropic | beta 名映射/剥离后 400 消失（协议级测试） | go test ./internal/gateway -run Anthropic | 待做 |
| M10 ✅ | #3455 search 管线的 Responses/Chat 回复缺 url_citation annotations | internal/gateway responses/chat 编码器 | Responses output_text 带 url_citation；Chat 同源；单搜索/无命中不回归 | go test ./internal/gateway -run 'Citation|Search' | 待做 |
| M11 ⏸ | #3117 Claude Desktop（3P 网关模式）会话被归为 Claude Code | internal/sessions 归属 | Desktop 3P 会话归属正确；Claude Code 会话不变 | go test ./internal/sessions | 待做 |
| M12 ⏸ | #3043 MCP Registry 20s 硬超时导致搜索 0 结果 | internal/library | 超时不再吞结果/可重试；正常路径不变 | go test ./internal/library | 待做 |
| M13 ⏸ | #3512 全新进程首次写 Claude Code 配置把用户代理当中转（surelyMagpie 未等探测） | internal/agent/reach.go | 慢代理（0.3s/1.5s）首写不改写 base URL | go test ./internal/agent -run Reach|Claude | 待做 |
| M14 ⏸ | #3417（我方立案）「重置卡到期前使用」隐藏额度恢复判断提前消耗卡片 | internal/provider 或 routing | 独立开关语义恢复；主路径 rest 逻辑不回归 | go test 定向 | 暂缓：未动手——旧修复 fix/codex-expiry-window (e75480ec) 已在 fork，等维护者跟进，不用真实卡测试 |
| M15 | 全量回归 + 推送 + 回帖发布 + 台账 | 全部 + docs/fork | 全量 nogui 一次（预存红基线）；分支读回；回帖（语言随题）；台账重建 | go test -tags nogui ./...；python3 docs/fork/generate.py | 完成（68 ok/2 预存红；台账校验过） |

弹性项（时间允许才做）：#3489 腾讯云 WSA 搜索源（纯增量，FR）、#3449 网关侧
WebSocket（大功能，默认不做）、#3516 apple-touch-icon（GUI 资产纪律，默认不做）。
若调查发现某题已被上游修/已被认领（以动手时最新评论为准），记录证据后跳过并从
表中标注，不硬做。

### 第五轮订正注记（2026-10-24）

- 第三轮报告"远端 heads 89 = 85 + 3 r3 + 1 origin 杂散分支"中对 origin 的解释不成立：`refs/remotes/origin/HEAD` 是符号引用不是分支，当时快照里也没有 `refs/heads/origin`。历史数字不改写；r4 审查时 89 heads 含新加入的 fix/codex-expiry-window（另一 agent 的工作）。
- manifest 的 run.started 无有效 UTC（原为 "~05:55Z" 占位）：改为 unknown，并收窄第三失败的复验声明（保留一份整包重跑日志，撤回未存档的"第二次重跑/单测日志"表述）。
- r3-union/r3-rests 的 "not pushed" 旧 note 已更新为已推送读回；local-only 历史快照记录（3 条）的 "not pushed" 是当时机器状态的事实，保留不改写。


> 创建：2026-10-22。执行者：ZCode（skygunner fork）。
> 上游：yetone/magpie（main）；fork：skygunner/magpie。
> 本文件是活的文档：每个 milestone 完成后更新状态与证据。

## 2026-10-24 返工轮（ZCode 会话四）：六项返工 + 总账

独立审查（公开 refs：fix/issue-batch-1023pm 007b8b9b、union-items b2336968、
persist-gateway-rests 4d0b0b75、web-signal-shutdown c5334b59、dev 269f6fe8、
upstream/main 50f52b54）确认了六项缺陷。本轮全部返工，交付树重建于新的
upstream/main（现场实测已前移至 fa22bc64，含上游新 redact 缓存 sentKnownSpans）：

- **v2 交付树**：`fix/issue-batch-1023pm-v2`（base fa22bc64，15 commits：
  #3183 两 commit 移植 + 旧批次 7 commit 的 rebase + 返工 6 commit）；
  `fix/union-items-v2`（base fa22bc64，3 commits：d4e6998f→99d6e6ed→cbc6c3b3）；
  `fix/persist-gateway-rests-v2`（base fa22bc64，3 commits：047520f0→ae193b86→5035eb0b）。
  旧分支保留原 SHA 不动（追溯关系），未 force-push、未删除。
- **A**（9c720bc0）：secretValue 版本/repo 例外改为字段感知——password/api_key 等
  凭据字段不再因值形似放行（审查三个输入红→绿）。
- **B**（fde6695f）：learned 资格表——占位密码升级为真实凭据后开始学习；
  reindexKnown 只重建 learned（审查两条转移红→绿；与上游 sentKnownSpans 缓存
  共存，测试全用新文本避开缓存假绿）。
- **C**（683547f0）：账号 picks 的 Decide 不再触发 customIDs/phantom provider
  （PluginID("cline") 回归红→绿）。
- **D**（1e7644d1）：显式 picks 的决策模型经 RouteDecider→/v1/systemone 全链
  （红＝审查报错原文；真实供应商行为仍是报告者的验证，本轮只验本地路由与协议构造）。
- **E**（cbc6c3b3）：union fold 按最终类型统一清理，["array","object"] 顺序
  不再漏（红→绿）；依赖 6ae89820→b2336968→本轮修正，旧评论"可独立 cherry-pick"
  说法错误（草稿待发）。
- **F**（5035eb0b）：restSlow 落盘（锁释放后 saveRests，审查确认的遗漏）；
  模拟重启恢复绝对截止时间/原因/状态/次数。
- **G**（95537ceb）：#3358 正向覆盖——text/tool-call/length/filter 各结束态在
  /v1/responses（流式+非流式）保持各自语义。
- **统一**（97a8a0c2）：#3374 的裸 agent_message 改写退役，采纳 #3183 的
  PlainAgentMessage（保留 From/To 与图片）；组合测试走 /backend-api/codex/responses；
  sealed 项 400 拒绝（旧测试断言错误已改）。
- **台账**：docs/fork/upstream-status.json（唯一数据源，19 units）+
  UPSTREAM_STATUS.md（generate.py 生成，校验重复/缺失 SHA/共享补丁/误归档）+
  comment-drafts-2026-10-24.md（四条待发草稿，未发送）。火山修复入账：
  adopted_with_changes@8278fd9b/v0.1.1325；三测试在上游实现上绿（fa22bc64）；
  fix/volcengine-resets 列归档候选（最早复查 2027-01-23，本轮删除 0 分支）。
- **验证**：redact 包 67 绿、provider 包 1752 绿、union/rests 的 gateway 包
  全绿（唯一红 #3280 预存）；三平台 build+vet 过。该轮全量 nogui：69 包 ok + 4 本机已知预存红（#2207/#3280/umask 两项），运行于分支 head；原始日志当时未归档（已丢失），此后各轮的全量记录见对应轮次节与本轮 evidence 目录。

## 2026-10-23 午后轮（ZCode 会话三）：巡检 + 新 issue 修复

### Milestone 日志（本轮）

- **巡检完成**：65 条留言/60 issue 的反馈全部正面（ggbdpq 逐项 verify ~20 个分支，
  JoeyWu 认可 #3141，CNife 确认 #3251）。上游 main 前移 65 commit（→0815c918），
  窗口内 0 个吸收；77 个修复 commit 中仅 3 组不再干净套用。
- **M-3370 完成**（#3370, yxr1995-maker）：`fix/union-items` rebase 到最新 main 并加强
  （b2336968）——union 取 array 半边时丢掉 object 半边关键字（properties/
  minProperties/maxProperties）。报告者 codex_apps payload 逐字节红→绿。已回帖。
- **M-3345 完成**（#3345, Sumeriasns）：与 #2932 同一 bug（refresh publish close nil
  channel，栈与红测试行号 891/905 完全一致），572a1757 已在 v0.1.1320 修复，
  报告者只测到 1319。回帖指认 + 请升级验证。
- **M-3374 完成**（#3374, bigTsingLung）：根因＝deepseek preset 双端点（Chat+Responses）
  → Responses 原生**透传**，`agent_message`（ChatGPT 后端私有项）被 DeepSeek 静默丢弃。
  修：passthrough 对无账号 provider 把未密封 agent_message 改写成 user message
  （67b57000，与 chat 翻译同语义）；codex 账号/grok/zed 插件自己的处理不动。
- **M-3339 完成**（#3339/#3027）：fetchAvailableModels 过滤 `-disabled` 后缀占位 id
  （8b4ecbac）。普通 id 的 disabled:true 保持列出。
- **M-3352 完成**（#3352, 1234fsdc）：balance-only 读数（有 balance 无 windows）在
  live 读数（Allowances）和磁盘回退（lastAllowances）两处都记为"已知、空 Allowance"
  （4fb09e6b）。错误读数仍未知。
- **M-3349 完成**（#3349, Arcadia822）：decide= 落在账号 provider 的 picks 白名单
  （Save）+ withPicks 携带 + allProviders 分类修正（账号 id 的存储条目不做独立
  provider）（b2497dd8 + 007b8b9b follow-up：插件自己声明的 decide 不被空 pick 覆盖）。
- **M-3365 完成**（#3365, Ethereal49）：redact 三处——URL 占位密码（密码==用户名）
  打码但不进 known；版本号（三段点分数字）不做 SECRET；owner/name(#n) 不做 SECRET
  （4b6d93d9）。真密码/真 key 控制组不回归。
- **M-3358 完成**（#3358, Latias94）：普通 Chat 上游"啥也没说就 stop"判为 empty-reply
  失败（600fa01c）——emptyFails 加 Chat（AIGO 案例有 usage 也覆盖，判据是说了什么
  而不是数了什么）；流式先按 #667 重问再 response.failed；magpie 自己的标题请求豁免。
- **重建失效分支**：fix/web-signal-shutdown（3 commit 链：#3285 → fixtures → signals，
  新 head c5334b59）与 fix/persist-gateway-rests（2fc84b93 → 4d0b0b75），测试通过，
  分别在 #1258/#2008 回帖新 head。
- **分支**：本轮全部在 `fix/issue-batch-1023pm`（7 commit，基于 origin/main 0815c918；#3370 在独立分支 fix/union-items），
  外加 rebase 后的 `fix/union-items`。全部已推 fork，全部已在对应 issue 回帖
  （英文题英文、中文题中文）。
- **验证纪律执行情况**：每个修复红→绿真实还原验证（三子修复的 #3349 逐项红）；
  gateway/provider/redact 三包全量绿（唯一红 #3280 预存）；三平台 build+vet 干净；
  全量 `go test -tags nogui ./...` 跑于分支 head（10-23 当轮记录：69 包 ok +
  4 个本机已知预存红；10-24 返工轮的复验见下方"2026-10-24 返工轮"）。
- **不修（附原因）**：#3367/#3357/#3350/#3362（维护者自领）；#3372（报告者准备提 PR）；
  #3361（自带修复分支）；#3337（上游限额行为不定，根因存疑）；#3353（数据局限+
  产品设计）；#3356（复现重、参与历史 rebasing 设计敏感）；#3373（复现条件不明，
  待报告者数据）；#3366/#3340/#3341（Windows 平台题）；#3363/#3364/#3335/#3334/#3333
  （功能请求类）。


### 巡检结论（本轮开头完成）

- fork 现有 82 个远程分支、60 个 issue 上的 65 条留言；最近一轮交付 45+ 组修复。
- 上游 main 前移到 0815c918（65 个新提交），窗口内 **0 个** skygunner co-author 提交（尚未被吸收）。
- ggbdpq 对 ~20 个我的适配分支做了逐项 verify（cherry-pick 到 main、ablation、跑测试），
  全部正面，无需返工；JoeyWu 认可 #3141；CNife 确认 #3251 已修复。
- 维护者 yetone 今天上午自立案并自领：#3367（app.js 预算红）、#3357（CI race 红）、
  #3362（partners 刷新）——跳过不撞车；#3350 是他顺手发现的 1 行测试期望，跳过。
- #3372（XDG autostart）报告者本人准备提 PR——跳过。

### 本轮候选池（按优先级；调查后可删）

1. #3370 Antigravity `["object","array"]`+items 400 —— 疑似 fix/union-items (c638203) 已覆盖，
   验证后回帖指认分支。
2. #3345 Windows 无 WSL 时 GUI 启动 2 秒必 panic（sessions.go:891 close of nil channel）——
   严重且确定性；代码路径可在 Linux 上测。
3. #3374 Codex `agent_message` 输入项被翻译丢弃（deepseek chat）——确定性直连复现。
4. #3339 Antigravity 模型列表暴露 `*-disabled` 占位模型（404）。
5. #3352 routing.go 跳过 balance-only 读数 → 「剩余额度未知」（报告者给了精确代码位置）。
6. #3349 插件 provider 上 `decide=` 显示已保存但不持久化（commandcode-plan）。
7. #3358 Chat→Responses 翻译静默完成 Codex 轮次（5 起真实事故）。
8. #3365 redact：普通 key/版本号误判 SECRET + known 状态跨请求污染 paperclip。
9. #3346 Z.ai 流中错误对象被当正常 chunk 转发（读版本依赖后定）。
10. （低优先，可能放弃）#3337 prompt-too-long 会话钉死；#3353 add-on 窗口 0%；#3356 库同步覆盖。

### 验证纪律（沿用）

每题：读全 issue 线程 → `git log --grep` + 全仓库搜关键词确认未修 → 红测试（报告者原始数据）
→ 修 → 绿 → 三平台 vet/build → 全量 `GOTOOLCHAIN=go1.26.8 umask 022
go test -tags nogui ./... -timeout 45m`（预存红：#2207/#3280/#3284）→ push fork →
issue 留言（英文题英文、中文题中文）。commit message `区域: 行为 (#issue, 报告人)`，body 记录验证。

### 分支命名

沿用 batch 堆叠模式，但基于最新 origin/main（0815c918）开新链：
`fix/issue-batch-1023pm`、`-b`、`-c`…每 commit 一个 issue。

---
（以下为 2026-10-22 首轮计划与日志，保留为历史）

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

## 2026-10-24 上游交付说明同步（用户明确授权）

本轮仅同步 #3349/#3365/#3370/#3374/#2008 的既有交付留言及 fork 维护记录。
用户本轮授权更新上游留言，覆盖此前返工阶段“草稿不发送”的限制；不涉及 PR、产品修改或归档删除。
当前产品头：r3-batch 7f927133、r3-union 275b7a2e、r3-rests d639c8d7；维护起点 c1e0d235。

| M | 目标 | 修改范围 | 验收标准 | 验证方式 | 状态 |
|---|---|---|---|---|---|
| S0 | 固定身份、版本与待发布正文 | 5 个线程、7 条本人留言；原文备份与更正正文 | 账号 skygunner；SHA 读回一致；语言、范围、依赖及证据准确 | GitHub GET；本地 git/既有证据核对 | 完成 |
| S1 | 更新旧交付说明 | 5 条主更正 + 2 条历史转向提示 | 原 permalink 不变；原文折叠保留；逐条读回正文一致 | GitHub 更新及 GET 读回 | 完成：7/7 正文一致 |
| S2 | 持久化发布记录 | docs/fork 台账、发布记录、摘要及 PLAN | receipts/body hashes 可复核；台账校验过；普通推送 dev 后 SHA 一致 | generate.py --selftest/--write；git diff --check；远端读回 | 完成：7/7 回执、19 项负例及 diff 检查通过；dev 维护提交 1fbd97acb27d4875e0c650e3fe6ed5cd2a97f3ac 已普通推送，GitHub ref 与回执文件读回一致 |

发布完成：5 个线程、7 条既有留言已编辑并逐条读回一致；新建留言 0，删除留言 0。
原文、最终正文、GitHub updated_at 和 SHA256 回执：docs/fork/evidence/upstream-sync-2026-10-24/comments.json。
最后一次 GitHub 更新为 2026-10-24T04:46:54Z；本轮读回核对完成于 2026-10-24T04:47:56Z。
本轮不重跑产品测试；沿用对应 exact head 的已有验证，明确保留全量 exit=1 与未执行环境。
