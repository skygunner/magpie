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
