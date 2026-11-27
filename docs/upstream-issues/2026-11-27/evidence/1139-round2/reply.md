@comicchang 谢谢——分支拉下来验证过了，背书如下。

[comicchang 的分支](https://github.com/comicchang/magpie/tree/fix/proxy-lan-bypass)（`1849425db`，基于 `bcb136c9e`）在我这边：harm check 干净（无新依赖、无新网络地址或凭据外传，改动集中在 netproxy / settings / settingsapi / plugin host / GUI，与声明一致）；`go build ./...`、`go vet ./...` 通过；定向测试全绿——`internal/netproxy`、`internal/settings`、`internal/settingsapi` 全包，及 `internal/plugin` 的 LAN/NoProxy 用例。行为见证：把 `Func` 里 `none` 模式破坏（落回 `onLAN` 直连）后 `TestNoProxyModesAndCIDR` 立即红（`nas.tail1234.ts.net:3425 with NoProxy=none: got (<nil>, <nil>), want proxy 127.0.0.1:7890`），恢复后绿——Off 模式"除 loopback 外全走代理"的语义是被测试钉住的。

设计上我也认同三段式比单纯 opt-out 好：`Auto` 保持 #982 零回归，`Off` 覆盖你报告的 RFC1918-over-VPN 场景，`Custom` 的 CIDR 预填对混合网络用户友好，且 Go 侧与 `host.js` 规则同步、`KeepOwn` 不跨机同步这些细节都处理了。我原来的 [fix/1139-issue-1139](https://github.com/skygunner/magpie/tree/fix/1139-issue-1139)（loopback-only）被这个分支完整吸收并扩展，可以以后者为准（归属按仓库惯例由维护者裁定，提交里已含你分支的作者信息即可）。

一个提醒：分支基于 `bcb136c9e`，合入前需要 rebase 到当前 main——`netproxy` 附近上游有变动。等维护者挑取。
