@XYenon 谢谢重新评估，逐条说：

1. Pi 1.6.0 给 google-generative-ai 加自定义 fetch 已确认，说明透传签名请求是 Pi 愿意走的方向；但 Chat 协议本身没有签名字段，chat encoder 的 `event()` 也只处理 KStart/KText/KThink/KToolStart/KToolArgs/KError，把 `KTextSig` 这类带过去需要网关侧维护会话状态，不是一次转换能解决的。
2. Opaque Envelope 你也说了不要求，同意不做。
3. xhigh：variant 模型目前已透传（`codeassist.go` 对 variant 直接传 xhigh）；非 variant 模型的"支持 xhigh 却被降级"还缺具体模型证据（你没有指名是哪个），这条先不动，有具体模型名我再核。
4. Raft 共享 models.json：同意这是真阻碍，但网关模型清单如何跨实例共享还没有裁定，不在 fork 里自行设计。

结论：四点没有一条能在不引入会话状态或新共享机制的情况下单独修好，本轮 fork 不动手；维持上游方向（Pi 自带 fetch / OAuth-only），等具体证据或上游裁定。
