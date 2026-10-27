两个点都对，补一下现状，免得重复造轮子：

**“对照界面标出命中来源”这条，当前 main 已经有了**，只是不在字面上：请求记录里每个占位符悬停/聚焦时的提示会写 `种类 · 谁发现的`（`maskByName`，`MASK_BY` 五个值：Rules / Your rules / Your words / Local model / **Seen before**），`maskrec.Value.By`（JSON 的 `b`）逐值记录，`known` 就是 `known` 传播。`/api/masking/records?by=known` 也能按来源过滤记录。所以“看不出是规则命中还是 known 命中”这一半，升级到最新版就有了（悬停才显示，这一点确实不明显——你的建议里“标出来”如果指的是不悬停就可见，那是个 UI 改动，属于 yetone 的取舍）。

**“它最初是在哪条请求学到的”这条，目前确实没有**：`known` 的存储只有 值→占位符 的映射（内存里，重启即清），不记来源。最近的近似是记录列表——同一占位符第一条 `By != known` 的记录基本就是它被学进去的那次，但没有任何链接或跳转把它连起来。

**known 列表和一键加白**也都没有：allowlist（从不脱敏）是 `RedactAllow`，独立于 known，两者之间没有通道。要把“点一下不再脱敏 = 写进从不脱敏”做起来，得先有 known 列表，这在产品上是 yetone 的决定——默认保护强度不动、误伤可解释可解决，这个方向我同意。

**`nolearn` 那半仍然只在 fork 上**，你说得对：`upstream/main`（`3a89dd53`）的 redact 没有“URL 密码等于其用户”的豁免，`r3-batch`（`7f927133`）里的那段还没进主干。也就是说在 main 上，`postgresql://verifier:verifier@…` 这类值第一次经过时仍会被学习并替换后文——和 `paperclip` 同一个坑。你确认的这个样本正好把它钉住了；要不要把这一小块单独重放（不带 r3-batch 其余的返工）到当前 main，我可以在你点头后做。
