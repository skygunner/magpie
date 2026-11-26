@comicchang 谢谢补充 RFC1918 私网经 VPN/系统代理才可达的场景。当前分支 [fix/1139-issue-1139](https://github.com/skygunner/magpie/tree/fix/1139-issue-1139)（d1ba11f3d）只处理"名称解析到 loopback 时不再被 `onLAN` 误判直连"，你说的 `10.x`/`172.16.x` 仅经代理可达的情况确实不在其覆盖内，而且 `NO_PROXY` 语义（"绕过代理"）也表达不了"强制走代理"，这点你说得对。

你要的自动 LAN 绕过 opt-out（或 per-provider/global force-proxy 开关）是行为裁定层面的新增配置，超出原修复范围，这里先记录给维护者裁定；在那之前 per-provider proxy override 的临时办法照旧可用。
