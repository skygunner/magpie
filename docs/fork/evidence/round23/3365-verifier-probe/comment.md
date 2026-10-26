补充样本收到。用探针在当前 main（`19253f67`，redact 包）上把两条可能的来源分开验证了一下，结论：**裸词 `verifier` 本身不会触发规则**，这个样本几乎可以确定是 `known` 传播——与本 issue 主题"示例密码被学习后在后续内容持续被替换"是同一家族。

探针（`internal/redact`，`MaskJSON` 真实引擎，thinking 内容带/不带 `signature` 各测一轮）：

| 前置状态 | `verifier nodes` / `verifier-node tagged droplets` 的结果 |
| --- | --- |
| 无任何前置（未学习） | 原样保留，0 次命中，无任何 `ByRule` |
| 前置 `Preview`/`Scrub` 过包含 verifier 的 URL（不学习） | 原样保留 |
| 前置 `Mask` 过 `postgres://service:verifier@localhost/db`（URL 密码位学到） | 整词替换为 `{{PASSWORD_<hash>}}`，`-node` 中的部分同样替换，`By=ByKnown` |
| 前置 `Mask` 过 `密码：verifier`（赋值右侧学到） | 同上，整词替换 |

也就是说，与你截图完全一致的替换形态，只在同一请求上下文里**更早出现过一次把 `verifier` 当作凭据值**（URL 的密码位、或 password/token/secret 类赋值的右侧，包括代码块里的示例凭据）时发生；学到之后所有出现处都会被替换——含 `verifier-node` 里的部分匹配——这正是当前"已知值全程遮蔽"的设计路径，不是新的规则误判。

两个可以帮忙收口的点：

1. 方便的话在这台电脑的同一条请求里（「对照」界面向前翻，或直接搜原始 JSON）确认是否存在 `:verifier@`、`密码/token/secret…: verifier` 这类更早的出现。有的话，本例即已知的传播行为，是否要限制"学到的值在叙述性文本中的传播"（比如只遮蔽凭据位、或对短词/常见词不传播）是维护者的取舍：真实密码完全可能恰好是个普通英文词，收紧传播就有漏遮的真实风险。
2. 如果确认**没有**任何更早的凭据位出现，那才是新的规则误判路径——那种情况下请把该条请求的原始 JSON（脱敏后）贴出来，我们按原样复现。

探针代码与完整日志：[verifier_test.go / probe.log](https://github.com/skygunner/magpie/tree/dev/docs/fork/evidence/round23/3365-verifier-probe)（dev 分支证据目录）。
