# Magpie Smart Split 路由优化计划

> 记录日期：2026-10-22  
> 状态：研究结论与候选改进方案，**尚未实施**  
> 运行环境：`docker1` 上的 Magpie 容器  
> 相关排障笔记：[`claude-code-learnt.md`](claude-code-learnt.md)

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
