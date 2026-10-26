根因已定位并修复，分支 [`fix/r25-mask-alias-plan`](https://github.com/skygunner/magpie/tree/fix/r25-mask-alias-plan)（基于上游 main `19253f67`，单个提交 `95c0511`）。

## 根因

两个来源，都在 `internal/gui/assets/app.js` 的账号打码逻辑里：

1. **自定义别名被当作账号收集。** `noteAccounts()` 遇到 `k === "alias"` 就把值收进 `accountNames`（第 24697 行附近），`names()` 又把 `p.accountNames`（用户自定账号名，#1515）和 `q.alias` 加进待打码集合。随后的打码 walker 用 `TreeWalker` 扫描整个 body 的文本节点并按字符串匹配替换，于是用量页里用户自己起的别名（`.quota-alias`）被打码。
2. **套餐文字误遮蔽是同名碰撞。** 报告者的 Antigravity 账号原始值就是 `Google`（非邮箱账号会进入 `names()` 集合），打码是全页字符串匹配，套餐 `Google AI Pro` 和模型名里的 `Google` 因此被一起模糊——用量页和供应商页都命中。这解释了为什么只有"账号值恰好是品牌词"时才出现套餐误遮蔽。

## 修复方式

- `noteAccounts()` 不再收集 `alias`；`names()` 不再把 `p.accountNames` 和 `q.alias` 加进打码集合——展示别名不是身份信息。
- 打码 walker 对展示类元素（别名 `.quota-alias`、可重命名的账号名 `.acct-who .rename`、套餐 `.plan`、卡片标题 `.subscription-head b`、模型值 `.model-value`）只按邮箱匹配，不再按账号名匹配；这些元素上的 `title` 提示同样跳过。
- 原始账号身份（邮箱、手机号、非邮箱的账号值）在行内、句子、tooltip 里的打码行为不变；地址即使在上述展示元素内也仍然整体隐藏。

## RED→GREEN 证据

- 新回归测试 `internal/gui/tests/account-mask-alias-plan.test.cjs`（Chromium，en/zh/zh-TW/ja/de × 1100/420 宽）：在干净 base `19253f67` 上只加测试文件运行，20 个用例全部失败——别名被渲染成打码占位 `cencrrhllct`/`tldhsr` 而不是 `Work laptop`/`Google`；修复后 20 个全部通过。
- 打码相关 GUI 套件（window-mask、account-mask、account-mask-sessions、mask-emails、mask-actions、masking-records、provider-unmasked + 新测试）在本分支 Chromium 下 **90/90 通过**；同一 window-mask 套件在干净 base（stash 修复后）也通过，确认没有预存红。
- 本分支 head `95c0511` 上：`gofmt -l`（无输出）、`go vet ./...`、`go build ./...`、`GOOS=linux go build -tags nogui ./...`、`GOOS=windows go build ./...` 全部通过；受影响的 `internal/gui` 包 `go test -tags nogui` 全量通过（47s）。

## 未尝试 / 未覆盖

- Playwright 套件只跑了 Chromium（本地未跑 WebKit 一轮）。
- 没有用报告者的真实运行数据在真实应用里验证（截图场景按测试夹具重建：Antigravity 账号值 `Google` + 套餐 `Google AI Pro` + 自定义别名）。
- 托盘面板里同名碰撞的展示路径未单独加用例，依赖同一 walker 逻辑。