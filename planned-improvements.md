# Magpie Smart Split 路由优化计划

> 更新日期：2026-10-23（Australia/Brisbane）
>
> 状态：**第一阶段已实施、观察中；独立 `long-context` 组已创建并验收。Smart Split 长度规则尚未启用。**
> 运行环境：`docker1` 上的 Magpie 容器  
> 相关排障笔记：[`claude-code-learnt.md`](claude-code-learnt.md)

## 2026-10-23：第一阶段执行记录

本节记录第一阶段；随后「独立长上下文组」节记录用户追加要求后的变更。后面的「一」至「九」保留 **2026-10-22 的历史研究与候选方案**，其中旧配置、额度、统计和未勾选清单不代表现在。第一阶段只调整有实际失败证据的成员，没有全面重排模型。

### 已备份与实际修改

- 完整备份位于 `docker1:/home/hermes/magpie-backups/20261022T173228Z-pre-routing`，包含配置卷除缓存外的全部内容；关键配置文件哈希已核对。
- 本机核心配置副本：`/home/vincenty/.local/state/magpie-audit/20261023/config-backup.tar.gz`，约 5.37 MB，manifest 中 8 个文件的哈希均已核对。该私有目录还保存 `runtime-before.json`、`usage-before.jsonl`、`routes-before.jsonl`、`usage-findings.json`、`routes-aggregate.json` 等证据；不在本文公开凭据、账户身份或原始请求。
- 变更于 `2026-10-22 17:38:42Z` 起，最终在 **`17:39:38.58Z`** 完成，即 Brisbane `2026-10-23 03:38–03:39`。通过运行中的配置接口生效，**没有重启容器**。

| 已修改的组 | 加入 `off` 的精确成员 | 实际依据与收益 |
| --- | --- | --- |
| `auto-dots-3-note-preview-free` | `kilo/dots-studio/dots-3-note-preview:free`；`openrouter/dots-studio/dots-3-note-preview:free` | 重启后 13 次最终 HTTP 400 全来自这两条通道，分别 11、2 次；停用这些路径，目标是减少同类中断。保留 Cline 同模型通道。 |
| `general`、`fast-cheap` | `kilo/minimax/minimax-m3.1-flash:free`；`openrouter/minimax/minimax-m3.1-flash:free` | 上游明确通知免费版本已退役或模型不存在；13:15Z 后 25 次 404、零成功，累计无效尝试 116.667 秒。去掉这两条失效路径，保留其他供应商的 MiniMax。 |

**仅上述三个组的 `off` 列表变化。** 分组策略、成员列表及顺序、上下文、超时、意图规则、分类器和其他设置均保持原值。保留 OpenRouter MiniMax 的 `modelAPIs=chat` 兼容配置，也保留 Hermes 的 `ollama_num_ctx: 0` 绕过。模型退役与协议兼容是两件事；不把退役的免费版本自动换成收费版本。

25 次 MiniMax 404 原本都进入了后备尝试；116.667 秒是历史请求中无效尝试的累计耗时，不是一次任务可节省的时间。Dots 的 13 次硬失败说明这次调整有明确必要性，不能据此承诺未来成功率提升某个百分比。Cline Dots 历史只有 3 个成功样本，并有其他失败尝试，保留后仍需观察。

### 本次基线与保持不动的理由

usage 基线覆盖 `2026-10-21 15:02:08Z–2026-10-22 17:32:10Z`：14,202 条记录，其中外部请求 13,763 条、内部分类器 439 条。路由 trace 有 13,753 个唯一 `id`、14,414 次尝试；两类日志覆盖并不完全相同，不混用分母。trace 的 `seq` 不能用于去重，尝试的实际供应商取 `tries[].id`，不能用顶层 `provider` 代替。

最近一次容器启动为 `2026-10-22 11:21:18Z`。从此时到截点，外部请求 5,555 次：5,485 次无错误成功，54 次客户端取消，13 次 HTTP 400，另有 3 次 HTTP 200 但记录了流内错误。**499 不计作供应商故障，HTTP 200 也不能直接等同完整成功。**

| 路径 | 重启后样本 | 保留理由 |
| --- | --- | --- |
| `general` | 1,371 次；1,322 成功、49 取消，无硬失败、无回退 | 默认 `smart` 策略下正常路径已经工作，不为重新设计而强制改成固定顺序。 |
| `fast-cheap` | 1,053 次；1,052 成功、1 取消，无硬失败、无回退 | 同样保留 `smart`；只清除已退役成员。旧文档中较慢的历史中位数不能代表当前行为。 |
| `smart-split` | 3,127 次；3,107 成功、4 取消、13 个 400、3 个流内错误；成功请求总耗时 P50 8.97s、P95 54.90s | 外层保持 `order` 和 compact / complex / routine / trivial 规则；先消除明确错误，避免同时改动分类与模型排序。 |
| `ultra` | 直接调用样本很少；主要通过 Smart Split 的 complex 规则进入 | 保留质量档及现有成员，不能只靠跨模型总耗时比较裁掉强模型。Step 5 的超时单独列入第二阶段。 |
| `typesafe/jev-latest` | usage 记录 265 次内部分类调用全部成功；路由 trace 273 次分类决策（含缓存）无错误 | 总耗时 P50 约 0.24s、P95 约 5.35s；保持分类器，没有更换依据。 |

`smart` 不能理解为根据上述历史 P50/P95 自动学习并选择最快供应商；历史统计用于此次人工决策，不是已部署的延迟优化器。因此也不承诺保留 `smart` 会自动修复 Step 5 的长尾。另保留 `smart-split.firstToken=25s`，它是分组流式请求的首内容回退设置，不能当作 JEV 分类请求的截止时间；历史表格将它放在「分类器」一栏不够准确。

官方付费 DeepSeek 的现状已不同于旧草案：`general`、`ultra` 已将其关闭，`smart-split` 将它放在根组末尾。历史最后一次实际调用为 `04:57:39Z`，重启后为零。源码与路由快照核验显示，当前配置对应的 13:41Z 后 2,617 条候选列表中，即使 DeepSeek 看起来不在字面末尾，排在它后面的也全部是处于 quota / credit / rate 休息期的成员；它是**最后可尝试的候选**。网关不会为等待免费成员解除休息而一直阻塞付费兜底。13:41Z 前曾有不同配置，不能把整段历史当成当前顺序。

### 全部 provider 与组的盘点

运行 API 返回 **20 个 provider**，均有 `ready=true`；这表示配置已具备，并不表示账户有余额、模型未休息或 provider 已开启。以下余额与额度仅是 `17:32Z` 左右的读取值；「历史请求」是整个基线中的最终外部请求数，包含失败，不是每次尝试或质量评分。

| Provider ID | 当前状态、可读额度 | 历史请求与当前角色 |
| --- | --- | --- |
| `opencode-zen` | 开启；quota 休息至约 10-23 05:37Z | 383；免费模型组来源，等待额度恢复。 |
| `opencode-zen-account` | 开启；同样 quota 休息至约 05:37Z | 2,080；另一免费模型来源，不强制提前释放休息。 |
| `kilo` | 开启；没有可用的完整余额读数 | 2,262；保留有效免费模型，尤其已成功承接大输入的 Nemotron Ultra；关闭两条问题通道。 |
| `ollama-cloud` | 开启；余额 $0，月额度 $2.50 / $2.50 已用完 | 521；保留配置，当前不能当成有剩余额度的首选。 |
| `nvidia` | 开启；快照中处于短暂 429 休息 | 393；最近 17 次 DeepSeek 尝试均限流，观察恢复，不贸然全局删除。 |
| `google` | 开启；quota 休息至约 10-23 01:11Z | 29；Gemini / Gemma 候选，保留 Hermes 字段兼容绕过。 |
| `cloudflare-jev` | 开启；未读取统一额度 | 无外部请求；历史有 17 次内部分类调用，当前不是主分类器。 |
| `freellm` | provider 开启；`auto` 已在 general / fast-cheap 内关闭 | 无外部请求；保留既有禁用状态。 |
| `deepseek` | 开启；余额 ¥16.46、$0.00 | 231；官方按量付费最终兜底，重启后没有实际调用。 |
| `vercel-jev` | 已关闭 | 无外部请求；保留关闭。 |
| `vercel` | 已关闭 | 无外部请求；保留关闭。 |
| `openrouter` | 开启；余额 $0 | 38；保留可用免费通道，关闭退役 MiniMax 和失败 Dots 通道。 |
| `typesafe` | 开启；未读取统一额度 | 无外部请求；JEV 为当前主分类器，内部调用单独统计。 |
| `codex` | 开启，prolite；7 天用量读取值 12% | 3,949；当前主要订阅推理通道，保留 Luna / Sol / Astra 分工。 |
| `copilot` | 开启，Free；Chat 0 / 200、Completions 0 / 2000 | 无最终外部请求；最近 14 次 GPT-4.1 尝试因约 301k–316k 输入超过 128k 被拒，属于上下文匹配问题。 |
| `grok` | 开启；7 天用量 72% | 1,695；保留 fast 与常规模型的现有角色，不为耗尽额度而硬切流量。 |
| `workbuddy-ai` | 开启，Free；此快照没有完整可用额度读数 | 1,159；DeepSeek / GLM / HY / Step 候选；Step 5 长尾待单独试验。 |
| `teleagent` | 开启；未读取统一额度 | 无外部请求、未入当前路由组；没有依据增加流量。 |
| `cline` | 开启；余额 $0.50 | 641；保留 MiniMax、Nemotron 与临时保留的 Dots 通道。 |
| `trae-global` | 开启，Free；$1 / $1 额度已用完 | 370；保留配置，等待周期恢复。 |

保存文件中另有休眠的 `gemini` 条目，但它未出现在运行 API 的 20 个 provider 中，不能算成第 21 个可路由来源，也不自动启用它。

共有 **15 个保存组**：5 个入口/用途组（`smart-split`、`cheap-first`、`fast-cheap`、`general`、`ultra`），8 个 OpenCode Zen 免费模型聚合组（Beam、MiMo、Nemotron Super VL、Nemotron Lightning、LongCat、DeepSeek Flash、GLM Flash、MiniMax Flash），以及 Dots 聚合组和 Ollama/Cline MiniMax 聚合组。运行 API 显示 25 个组，是额外包含 10 个自动生成的同模型聚合组，不是又有 10 套人工路由策略。此次只改上述三个保存组的 `off`，`cheap-first` 与其他聚合组保持原值。全部组的具体模型、策略和设置保存在[调整后分组快照](/home/vincenty/.local/state/magpie-audit/20261023/groups-after.json)。

付费兜底实际成功后，既有会话亲和性仍可能继续选择该成员；本次没有制造其他候选全部失败来触发付费调用，也没有修改亲和性。因此上面的最后兜底结论针对本次核验的候选规划，不承诺每个既有会话都重新从免费模型开始。

### 已完成验收与短窗口观察

- Chat Completions SSE：`general`、`fast-cheap`、`smart-split`、`auto-dots-3-note-preview-free` 四组均完成「第一轮工具调用 → 提交工具结果 → 第二轮正确回复」，合计 **8 个请求全部通过**；记录为私有目录的 `smoke-results.jsonl`。
- Anthropic Messages 非流式：`smart-split`、Dots 两组完成同样的两轮工具调用，合计 **4 个请求全部通过**；记录为 `anthropic-smoke-results.jsonl`。这验证了两种已测协议的工具往返，不涵盖所有客户端、图片或长输入。
- 变更前另有 1 次 Cline Dots 直接 SSE 测试成功；它支持暂时保留该成员，但不计入变更后的 12 个验收请求。
- 已检查 Hermes 仍是预期的 custom / `group/smart-split` / Magpie base URL，并保留 `ollama_num_ctx: 0`；没有将此配置检查写成真实 Hermes → Gemini 全链路通过。
- 截至 `2026-10-22 17:47:24Z`，自最终变更时点起约 8 分钟的自然流量为 **65 / 65 成功**（fast-cheap 23、general 30、smart-split 12），上述 12 个验收请求另计；4 条已关闭通道实际尝试为零，PAYG DeepSeek 调用为零。容器 healthy，启动时间仍为 `11:21:18Z`。这个几分钟样本只能证明当前正常路径可用，不能说明 24 小时稳定性或长期延迟已经改善。
- 独立配置核对确认 `providers.json` 只有三个组的 `off` 字段发生变化，其他顶层字段保持不变；递归展开主要组后，4 条问题通道均不可达。
- 远端备份中的 `rollback-stage1.py` 已通过 dry run；`--apply` 仅恢复本次三个组，并在发现并发组编辑时拒绝覆盖。实际回滚未执行。

需要撤销这一阶段时，在本机执行：

```sh
ssh docker1 'sudo -n python3 /home/hermes/magpie-backups/20261022T173228Z-pre-routing/rollback-stage1.py --apply'
```

不加 `--apply` 只检查是否可回滚。该脚本通过配置 API 热恢复三个组，不覆盖账户、凭据、其他组或新增历史。

### 自动路由下一阶段的进入条件

自动改变 Smart Split 分流前，先观察变更后 **至少 24 小时，且至少 1,000 次新的 Smart Split 请求**；两个条件都满足再评估，不根据几次烟雾测试立即改变生产分流。用户随后要求先建独立长上下文组，已按下节实施；这没有改变原入口的流量。没有建立自动调度器或自动后续改配置任务。

下一轮比较同客户端、相近输入长度和任务类型的：最终硬失败、流内错误、回退次数、TTFT、总耗时 P50/P95、实际 PAYG 调用，以及 Agent 是否正常完成工具调用。确认已关闭通道不再被这些组尝试；若替代通道出现新的硬失败，按此次备份恢复对应 `off` 差异或作针对性修正，不整体覆盖其他实时配置。

长上下文组的研究依据是：Step 5 重启后 133 次尝试中有 42 次在 30 秒首 token 上限处超时，累计失败回退约 21 分钟；其后 Kilo Nemotron Ultra 多次成功处理超过 600k 的输入。一次约 654k 输入总耗时 92.168 秒，前面六次失败共约 76 秒，最终 Nemotron 仅用 15.417 秒。应先测试能否少走这些错误路径，不能因总耗时高而把真正成功的长窗口模型删掉。

自动接入前必须核实窗口 metadata、路由输入估算与实际请求长度，并评估图片、工具调用、reasoning、流式和客户端 compact；已完成与尚未完成的检查见下节。只在相同任务确实改善完成效果或总耗时后扩大使用；180k–200k 仍是旧草案阈值，不是本次已选定的上线值。暂不改既有组中 Step 5 的超时、增加复杂调度、修改网关源码或承诺减少 compact。

后续观察结果在实际完成后补记，不把第一阶段配置与烟雾测试通过当成 24 小时验收已通过。

## 2026-10-23：独立长上下文组已上线

按用户追加要求，于 `2026-10-22 18:10:04Z`（Brisbane 10-23 04:10）创建 **`group/long-context` / Long Context**。只新增这个组，未加入 `smart-split` 的成员或规则；原 15 个保存组及其他 provider 配置逐项核对保持不变，现在共 16 个保存组。配置热生效，容器仍 healthy，启动时间仍为 `11:21:18Z`。

此次变更前备份：`docker1:/home/hermes/magpie-backups/20261022T180910Z-pre-long-context`，含配置、凭据及哈希 manifest；备份目录只允许所属用户访问。新增组记录为同目录 `long-context-created.json`。若只撤销此次新增，直接在 Groups 删除 `long-context` 即可；目前没有其他组引用它，无需恢复整个备份或覆盖第一阶段修改。

### 已部署的成员与设置

| 顺序 | 精确成员 | 基线中实际输入超过 500k 的成功记录 | 成功尝试自身 TTFT P50 / P95 |
| --- | --- | --- | --- |
| 1 | `kilo/nvidia/nemotron-3-ultra-550b-a55b:free` | 33 次，654,539–683,075 tokens，33 次明确工具调用 | 10.40s / 17.01s |
| 2 | `openrouter/nvidia/nemotron-3-ultra-550b-a55b:free` | 12 次，642,904–649,785 tokens，11 次明确工具调用 | 4.63s / 11.20s |
| 3 | `workbuddy-ai/step-5-preview` | 51 次，616,076–641,162 tokens，51 次明确工具调用 | 27.49s / 29.36s |

这是两个模型、三个渠道，不是三个独立模型。Kilo 样本较多，作为首选；OpenRouter 保留同模型的备用入口，不能仅凭不同工作负载下更低的中位数就断言它更快。Step 5 提供不同模型及目录标记的图片能力，放在最后，避免首先等待它再回退到 Nemotron。没有把仅有目录 1M 声明、缺少 >500k 实测记录的模型塞进新组。

- `routing: order`：以上述顺序为基础，使这轮实验直接检验 Nemotron 优先能否减少回退等待；休眠、限流下沉及已有会话亲和性仍可能改变实际顺序。
- `sink: true`：保留既有组所用的不可用成员下沉行为。
- `firstToken: 30`：流式首内容回退阈值；Step 5 成功样本中位数约 27.5 秒，不采用 25 秒。该值不是总请求截止时间，最后一个可尝试候选不受这一回退计时器限制；未改默认额外重试设置。以后嵌入 Smart Split，长组的 30 秒会覆盖外层 25 秒。
- `context: -1`：由成员中最小已知窗口推导，对外 `/v1/models` 已确认声明 **1,000,000**。`-1` 不是无限上下文，也不表示已经验证满 1M 的任务质量。
- 未强制设置组级 reasoning effort；按请求和模型能力处理。没有加入官方按量付费 DeepSeek。

另按路由准入 token >500k 选出的 175 次历史请求最终均成功；其中 Step 5 有 40 次超时，累计失败尝试约 1,204.6 秒。因此新顺序的目标是减少成功前的等待，不能宣称已经修复长请求最终失败或已实现相同秒数的节省。这一队列与上表按上游实际输入选出的队列不是同一个分母。

### 验收结果

- Chat Completions SSE：新组入口及三个成员分别完成两轮工具往返，**8 个请求全部通过**。
- Anthropic Messages 非流式：新组完成两轮工具往返，**2 个请求全部通过**。
- 新组大文本测试：上游计数 **550,061 输入 tokens**，Kilo Nemotron 返回预期短答案；首文本约 **29.19s**，总耗时 **31.08s**。这是合成容量测试，不是复杂任务质量评测。
- `/v1/models` 实读：`long-context=1,000,000`，`smart-split=272,000`。`settings.json`、caller keys 及所有已有组未变。
- 私有证据：`long-context-evidence.json`、`long-context-smoke-results.jsonl`、`long-context-large-smoke.json`、`long-context-final-check.json`，均在本机 `/home/vincenty/.local/state/magpie-audit/20261023/`。图片、满 1M 输入、复杂 reasoning 质量和真实客户端 compact 行为尚未验证。

### 如何与 fast-cheap / general / ultra 共存

**当前方案是独立入口，自动长度规则未启用。** 已知大输入的任务可以直接选择 `group/long-context`，普通使用继续选择 `group/smart-split`。这能先验证长池的效果，保留既有三个档位。

若后续接入自动路由，较合理的首轮候选是 **准入输入 tokens ≥600,000**，而不是立即采用 500,000。原因是历史中 GPT-6.1-Sol 已有 **68 次实际输入 501,311–549,986 的成功请求，零重试**，其中 67 次路由准入值也超过 500k。直接在 500k 切走这些任务，尚无速度或质量收益证据。600k 只是避开这段已成功负载的候选边界，不是已经证明的最优值。

概念上的规则顺序如下，**仅为提案，未写入运行配置**：

1. `tokens: 600000 → group/long-context`。
2. 原 `compact → fast-cheap`。
3. 原 complex / routine / trivial → ultra / general / fast-cheap。

长度规则应在 compact 之前，因为摘要请求本身也需要装下被摘要的输入。接入时应把新组加入根组成员、置于官方 DeepSeek 之前；原三组的成员无需删除重复模型，同 provider / model / effort 的候选会去重。

须明确的兼容语义：

- `tokens` 实际表示 **≥阈值**，使用当前估算与既有上游实际输入等信息，不是配置窗口大小，也不含输出预算。工具会话还可能沿用此前命中的规则，不保证每一轮刚越线就立即重选。
- 阈值以下保留原三档；阈值以上让容量优先，**不再同时保证 ultra 的模型质量档位**。把新组设为 high reasoning 也不能使 Nemotron 等价于原 ultra 模型。当前没有证据支持再拆三套长上下文组。
- 命中规则是优先选择，不是排他白名单；长组失败后仍可进入根组其他候选及既有付费兜底。容量筛选也不是绝对禁止尝试小窗口模型，不能把这条规则描述成硬隔离。
- `smart-split` 仍声明 272k，部分客户端会在到达 500k / 600k 前自行 compact。只加长度规则不能改变这一点；提高入口声明窗口会改变客户端行为，应作为后续单独实验。直接选新组则可取得其 1M 声明。

---

以下为 **2026-10-22 历史研究**，用于保留当时的依据与未实施候选；当前执行状态以上节为准。

## 一、目标与判断标准

1. 尽可能优先使用**稳定、足够聪明的免费模型**；订阅模型有额度且即将重置或过期时，合理优先消耗订阅额度。
2. 官方 DeepSeek 按量付费 API 作为**真正的最后兜底**。必须检查所有嵌套分组，不能只看 Smart Split 外层顺序。
3. 避免偶发错误、协议不兼容、长时间无首 token 的供应商使 Claude Code、OpenCode、Codex 等 Agent 卡住或整轮中断。
4. 根据意图和输入上下文大小选模型，适度利用大上下文模型，验证是否能减少不必要的 compact。
5. 优化目标是**端到端任务完成效果**，不是单纯降低模型标价或某一次响应时间。核心指标：任务成功率、总耗时、TTFT（首 token 延迟）、P50/P95/P99、失败率、实际费用及 compact 次数。

设计约束：先调整已有分组和规则；只有数据证明必要时才增设一个长上下文分组。不要先造多个子系统、后台状态机或复杂调度器。

## 二、当前配置快照与可疑点

以下是前次对 `docker1` 运行中配置的**只读检查结果**；随着后续操作，现状可能变化，实施前必须重新核对。

| 位置 | 当时观察到的配置 | 研判 |
| --- | --- | --- |
| `smart-split` | `routing=order` | 外层按成员顺序尝试，但嵌套分组仍有自己的选择策略。 |
| 外层成员顺序 | `group/general` → `group/fast-cheap` → `group/ultra` | 仅此顺序无法保证付费 DeepSeek 最后才调用。 |
| 意图规则 | `compact → fast-cheap`；`complex → ultra`；`routine → general`；`trivial → fast-cheap` | 已有可用的意图骨架；值得先保留、逐条验证。 |
| 分类器 | `typesafe/jev-latest`；effort `auto`；firstToken `25s`；context `-1` | 应从真实分类器延迟与错误率判断是否调整；不能无证据地把 25 秒改成 10 秒。 |
| `general` / `fast-cheap` | routing 未显式设置（使用默认行为），context 均为 `272000` | 需要确认默认路由策略、成员健康度与大上下文声明是否属实。 |
| `ultra` | context `-1` | 需要核实每个模型的真实窗口与客户端可见的 metadata。 |
| 官方付费 DeepSeek | `deepseek/deepseek-flash` 同时出现在 `general` 和 `ultra` | **首要配置问题：即使外层顺序合理，命中 general 也可能提前使用付费 API。** |
| OpenRouter MiniMax | `settings.json` 的 `modelAPIs` 已将 `openrouter/minimax/minimax-m3.1-flash:free` 固定为 `chat` | 这是已验证的兼容性规避配置，备份与调整时必须保留。 |
| 容器 | `ghcr.io/yetone/magpie:latest`；检查时 healthy | `latest` 不代表后续代码及配置行为始终相同，改动前记录镜像 digest/版本。 |

**最先要纠正的原则：** 每个普通分组都不能偷偷包含按量付费兜底模型。先对每个组递归列出实际成员、供应商账户、计费类型与失败回退顺序，再决定把官方 DeepSeek 移到哪里。

## 三、运行历史提供了哪些证据

前次分析读取了约 **8,133 条** usage JSONL 记录，时间大致覆盖 **2026-10-21 15:02 UTC 至 2026-10-22 10:37 UTC（约 19.5 小时）**。下列统计是该时间范围内进一步选取的最近约 6 小时观测值，样本短，**不宜直接外推长期 SLA**。

| 分组 | 样本请求数 | P50 总耗时 | P95 总耗时 | 初步判断 |
| --- | ---: | ---: | ---: | --- |
| `smart-split` | 1,137 | 约 7.1s | 约 26s | 仍有较明显的长尾。 |
| `general` | 919 | 约 5.0s | 约 67s | 中位数尚可，尾延迟偏高。 |
| `fast-cheap` | 721 | 约 17.7s | 约 56s | 名称含 fast，但历史样本并没有体现稳定快速。 |

**注意：** 上表是**整次请求的总耗时**，不是 TTFT。不同分组收到的任务复杂程度、输出长度、工具调用比例可能不同，不能仅凭分组 P50 断言某个模型更快。

`smart-split` 最近约 6 小时内，部分成功请求的观察值：

| 实际通道/模型 | 成功样本数 | P50 总耗时 | P95 总耗时 | 目前角色 |
| --- | ---: | ---: | ---: | --- |
| WorkBuddy / DeepSeek V4.1 Flash SG | 376 | 约 5.1s | 约 12.0s | 稳定性、速度的优先复测候选。 |
| Kilo / MiniMax M3.1 Flash Free | 215 | 约 6.2s | 约 54.0s | 中位数不错，但存在明显长尾。 |
| WorkBuddy / GLM 5.3 Flash | 204 | 约 8.8s | 约 22.9s | 可作为常规与复杂任务的候选。 |
| Cline / MiniMax M3.1 Flash | 约 170 | 约 8.3s | 约 24.9s | 值得与 Kilo 同模型、同任务交叉比较。 |

这些是**初筛名单**，不能因为单次快或免费就马上提高权重。仍需补全失败、限流、超时、任务输出质量和 Agent 后续是否顺利完成的记录。

对 `smart-split` 历史请求，以「输入 token + cache read token」做粗略大输入分桶，之前约 24 小时统计为：`≥128k` **1,160** 次；`≥200k` **399** 次；`≥272k` **164** 次；`≥500k` **5** 次。它提示长上下文值得研究，但这些数字**未验证等于模型真实上下文占用**；应先确认 usage 字段含义、cache 计数和工具消息如何计入。

## 四、阻断 Agent 的已知故障

[`claude-code-learnt.md`](claude-code-learnt.md) 记载了一次关键兼容性问题：经 OpenRouter 调用 MiniMax 时，原生 Anthropic `/v1/messages` 路径返回 HTTP **400 `Invalid Anthropic Messages API request`**。Magpie 对此类响应的 `retryable()` 判定为 false，导致所在分组不继续 fallback，Agent 的该轮工作直接中断。

- 当前已检查到针对该模型的 `modelAPIs = chat` 固定配置。把它作为现有防线保留，验证不同 Agent 客户端的最终实际协议。
- 如果之后仍有同类错误，考虑在公共错误处理层**仅对明确可识别的协议不兼容 400/422**允许回退，并加入真实报错回归测试。**不得把所有 400 当作可重试错误**；错误请求、参数不合法等情况仍应明确失败。
- 流已经开始向客户端发送后，切换提供商往往不能做到无缝回退。优先在首 token 前发现超时或兼容性错误，并避免无上限多级重试。
- 整体历史故障中，`claude-code-learnt.md` 还提到约 **36 次 auto-mode 分类后端超时**、约 **29 次 safety policy 拒绝**。这些事件需要区分来源，不能直接算作当前 Smart Split 的 Jev 分类器故障。
- 笔记另有一次 Bash 输出延迟约 **3.5 小时**以及 hook timeout 调整到 **10 秒**的记录，可能发生在客户端/hook 层。Smart Split 的路由优化不能替代该链路的独立排查。

改进顺序应是：先去掉会**硬中断 Agent**的已证实错误通道，再优化几十秒级的长尾，最后微调平均延迟。

## 五、Smart Split 与长上下文机制

相关代码入口包括 `internal/gateway/rules.go`、`internal/gateway/compacting.go`、`internal/gateway/fallback.go` 和 `internal/provider/grouprule.go`。前次源码研究的关键点：

1. `ruleFor` 可以匹配与 token 相关的规则，并保留会话内既有的路由选择（turn-sticky 行为）。规则优先级和实际触发时机都需要用真实请求验证。
2. `outgrown()` 在请求规模约触及当前模型窗口 **95%** 时尝试重新选更大上下文模型，但它看到的窗口取决于声明和可查到的 metadata。
3. `memberContexts()` 对嵌套组采用已知成员上下文中的**最小值**推导，使一个大窗口模型所在的混合分组未必被识别成大上下文分组。
4. `compacting()` 通过 Agent 请求文本特征识别 compact；某些 Codex `compaction_trigger` 请求路径可能匹配不到，相关讨论见 [issue #3106](https://github.com/yetone/magpie/issues/3106)。部署前需要真实复现测试。

**核心限制：** 服务端按 token 数选一个 1M 窗口模型，不一定能阻止客户端主动 compact。若 Claude Code、Codex 或 OpenCode 仍认为当前模型只有较短窗口，其客户端阈值可能会先触发压缩。必须同步检查客户端看到的模型 context、输出 token 预留、cache 与消息序列，且用实测确认宣称的大窗口真正可用。

大上下文不是越大越好：很长的输入可能让 TTFT、吞吐、费用、限流和输出质量恶化。评估应比较**相同 Agent 任务**的 compact 次数、cache 命中、任务总耗时、成功率和真实成本，而非只看窗口标称值。

## 六、候选路由方案（配置草案，未部署）

建议先让意图规则保持精简，只试**一个长上下文专用组**：

| 优先级 | 条件 | 候选目标 | 原因/验证点 |
| ---: | --- | --- | --- |
| 1 | 明确检测到 Agent compact 请求 | 已验证可处理压缩的快速、稳定模型（可先复用 `fast-cheap`） | compact 响应要可靠且快，不能单凭免费优先。 |
| 2 | 输入 token 达到约 `180k–200k` | `group/long-context`（仅当实验确认支持时） | 一个大窗口分组承载实际需要的大上下文。阈值是实验初值。 |
| 3 | `complex` 意图 | `group/ultra` | 优先质量、推理能力和工具调用可靠性。 |
| 4 | `routine` 意图 | `group/general` | 稳定、均衡、较低延迟。 |
| 5 | `trivial` 意图 | `group/fast-cheap` | 低成本、快响应；应清理高长尾成员。 |
| 兜底 | 意图分类器失败或未命中 | `group/general` | 默认路径应可靠、可解释。 |

概念上的外层候选成员顺序可设为：

```text
group/general
group/fast-cheap
group/long-context        # 通过测试后才创建
group/ultra
deepseek/deepseek-flash   # 官方按量付费，最终兜底
```

**注意：上述顺序只是草案，不保证实际调用顺序。** Smart Split 的规则匹配、组内 routing、分组嵌套、账号额度和失败分类都可能改变执行路径。正式调整时须逐层审计，并确认 `general`、`fast-cheap`、`ultra`、`long-context` 的所有可达路径中，官方按量付费 DeepSeek 只在最后兜底位置出现。若现有系统不支持某种跨组优先级，需要依据源码和测试调整草案。

订阅通道可以在同等质量和可靠性下优先使用临近周期重置的剩余额度，但必须实际有可读的 quota/reset 数据，且不应让临时限流或快到期账号引发重试风暴。免费渠道的高失败率可能比付费调用更贵，因为浪费 Agent 上下文、重复推理与人工恢复时间。

先不创建 `long-context-ultra`、`long-context-general`、`long-context-fast-cheap` 三套并行组；除非后续数据确实表明需要按质量档再细分。

## 七、分阶段实施与验证

### P0：保全、修正兜底链路和真实故障

- [ ] 备份生产完整路由配置、供应商/模型映射、相关 settings 与 usage 历史；记录镜像版本或 digest。备份应安全保存凭据，文档中不记录密钥。
- [ ] 重新读取生产配置，递归审计 Smart Split 所有分组及成员的实际路由顺序、计费属性、模型上下文和账户可用性。
- [ ] 让官方 DeepSeek 按量付费 API 真正只在最后使用；为有必要的分组显式设定清晰且可验证的顺序。
- [ ] 核实并保留 OpenRouter MiniMax `modelAPIs=chat` 兼容性配置；复测 Anthropic Messages API 400 情况。
- [ ] 根据充分样本找出导致硬失败或 P95/P99 过高的通道，必要时降权、移到后备或暂时剔除。
- [ ] 若模型 API 协议转换依旧触发特征明确的 400/422，考虑最小范围修改 `retryable()` 并加入针对性的回归测试；不扩大到全部 4xx。
- [ ] 对改动后 Agent 的正常请求、失败回退、流式返回、工具调用进行验收。

### P1：补齐测量，逐步优化免费与订阅顺序

- [ ] 按**提供商账户 × 模型 × Agent 客户端 × 输入长度段**统计成功率、状态码与响应质量；至少区分 400/422 协议错误、429 限流、402/余额、超时、服务端错误及 499/客户端取消。
- [ ] 分开记录 TTFT、总耗时和 token 生成速度，并展示 P50/P95/P99；对输出长度和任务类型分桶，避免将不同负载直接横比。
- [ ] 先用真实 Agent 任务 A/B 对比 WorkBuddy DeepSeek V4.1 Flash、WorkBuddy GLM 5.3 Flash，以及 Kilo/Cline 的 MiniMax Flash；记录最终任务完成、工具调用成功、重试和恢复成本。
- [ ] 只在订阅额度与重置日期可验证时，考虑用 `pace` 或现有排序能力消耗即将失效的订阅额度；付费 PAYG 账户继续放在最终兜底。
- [ ] 连续多天观察不同工作时间段的质量与尾延迟，再调整路由成员优先级，避免按一个小时的波动反复改配置。

### P2：验证一个长上下文分组是否真的有收益

- [ ] 选 **1–3 个**已实际验证长窗口、稳定性和价格的模型，必要时创建 `group/long-context`。
- [ ] 对 `180k` 和 `200k` 左右的路由阈值做对照实验；确认 token 数与缓存字段的统计口径。
- [ ] 确认路由器、嵌套组计算和 Codex/Claude Code/OpenCode 的客户端 metadata 对窗口认知一致；实际验证长会话、工具调用、reasoning 与 compact 请求。
- [ ] 同任务比较 compact 次数、错误率、cache、端到端耗时和费用；只有有明显收益才常驻该规则与分组。
- [ ] 若没有收益，删去新组并回归原有简单规则。

## 八、验收标准与待确认事实

改进应达到：官方 DeepSeek 真实保持**最终兜底**；协议不兼容不再轻易导致 Agent 整轮中断；分组 P95/P99 与端到端任务成功率优于基线；长上下文方案确实减少无谓 compact 或总耗时/费用。每次只调整少量变量，并留可恢复的配置备份。

尚不确定、必须实测：

- 19.5 小时 usage 样本的代表性不足；历史 `499` 可能只是客户端取消，并不能直接等同供应商宕机。
- 现有各供应商的免费额度、订阅周期、负载稳定性和长上下文上限会随时间改变。
- `typesafe/jev-latest` 的当前分类器 TTFT、P95/P99 和可用率是否优于其它候选，仍需单独采样。
- 客户端在大窗口模型后是否减少 compact，以及服务器是否真的接收了相应长度的请求，尚无结论。
- 当前服务器配置可能已在上次检查后改变；实施前必须重新核对，而不能把本文当作实时状态。

## 九、参考资料

- 官方意图路由文档：<https://usemagpie.ai/docs/zh/intent>
- 上游讨论：<https://github.com/yetone/magpie/issues/2363>、<https://github.com/yetone/magpie/issues/2120>、<https://github.com/yetone/magpie/issues/1386>
- Codex 压缩检测相关讨论：<https://github.com/yetone/magpie/issues/3106>
- Magpie 介绍与第三方资料：<https://codexapp.cc/reviews/magpie/>、<https://aiberm.com/docs/zh/magpie/>
- 视频线索：<https://www.youtube.com/watch?v=Gsg1qzsbbAo>（尚未逐项核实视频中的说法）
- 本地源码：`internal/gateway/rules.go`、`internal/gateway/compacting.go`、`internal/gateway/fallback.go`、`internal/provider/grouprule.go`
- 本地复盘：[`claude-code-learnt.md`](claude-code-learnt.md)

> 本文是依据截至 2026-10-22 的配置、源码和有限历史数据形成的优化计划。后续每个阶段先验证、再修改，保留数据对照与可回滚备份。
