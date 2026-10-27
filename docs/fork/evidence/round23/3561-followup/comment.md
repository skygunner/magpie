是上游自己修的，不在已发布的 release 里（截至现在）：上游 `3a89dd53`（`pii: the masking model downloads through magpie's proxy, and from hf-mirror when huggingface.co fails`，commit 里署名 "hh0592821 on Discord"，即你在 Discord 的那份报告）已落在 main 上并打了 `v0.1.1686` 的 tag，但 **`v0.1.1686` 还没有对应的 GitHub release**——目前最新的已发布 release 是 `v0.1.1685`，它不含这个修复。你用的 `v0.1.1672` 更早，所以还在旧路径上：下载客户端在包加载时就克隆了 `http.DefaultTransport`，那时 `netproxy.Install` 还没把代理装进去，克隆下来的只认 `ProxyFromEnvironment`，而从 Dock 启动的应用环境里没有 `HTTPS_PROXY`，于是走直连，撞上被污染的 huggingface.co 解析（`*.facebook.com` 的证书就是这个形状）。

几点对着你的现象说：

- 你观察到的“外部代理捕获了 magpie 的出站连接、到代理的 TCP 和 TLS 都成功，但证书是 `*.facebook.com`”，和根因一致：请求没走 magpie 自己的代理设置（或系统代理），直接出去了，握手对象是投毒后的解析。`v0.1.1686` 之后客户端 `t.Proxy = netproxy.Func`（magpie 里设置的代理，其次系统代理），和 magpie 其它请求一个路。
- 你的两个模型命运不同：`AstrLink Guard` 在修复后还多了一条 hf-mirror 的后备源（上游对真实镜像核过三个文件的 sha256）；`Privacy Filter` 的 GGUF 则 hf-mirror 会 308 重定向回 huggingface.co，所以它的出路仍然是你把代理填进 Settings → Proxy，让下载走 `CONNECT`。
- 等待版本：tag `v0.1.1686` 已含修复，release 一发布（或之后的任何版本）就带上它；`v0.1.1685` 及更早都没有。

顺带交代我们这条线：#3561 最初报的初始化顺序 bug，我们 fork 上的 `fix/pii-download-proxy`（`55dfe0a5`）和上游的修法等价（同样的 `t.Proxy = netproxy.Func` + `netproxy.Dispatch`），上游独立落地了它自己的版本（还多了镜像后备），所以这条分支就不再追了，以 main 的 `3a89dd53` 为准。
