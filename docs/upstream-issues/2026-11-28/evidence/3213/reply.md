@xiao-hans 谢谢真实 upstream 复现与拆解：15 个模型同秒 503、拒绝文案直指单模型队列（`for dfmodel (this plan's queue: p4, 1 ahead)`），随后整账号被 rest —— 正好是 A 方向（仅 rest 该模型）要解决的现象。你的复现还补充了 `lmodel` delisted 这一细节，确认其 400 属预期内。

我们的 [fix/queue-model-rest](https://github.com/skygunner/magpie/tree/fix/queue-model-rest)（`50eac57f6`，`TestQueuedModelRestsAlone` 红→绿）方向与你的结论一致；你提到的 `restAfterMarked` 迁移后需 rebase，上游 main 已推进到 `0d310af00`，rebase 后会由维护者在下轮集成里挑取。目前证据已足够支撑 A 方案，不需要再补什么。
