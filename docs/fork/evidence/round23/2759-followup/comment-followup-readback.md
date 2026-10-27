根因在导入面板那排来源标签的样式：`.apptabs` 是一条不换行的 flex（`flex: 1 1 0; min-width: max-content`），标签缩不到文字宽度以下，五个来源一多就溢出；而外层 `overflow-x: auto` 配着 `scrollbar-width: none`，横向滚动条被藏掉，末尾的 OpenCode 既看不见也拖不到，只能先选中一个再用方向键移过去。量到的数字：390px 窗口下滚动区 350px、内容 646px；900px 时 OpenCode 仍被切掉。

修复在 [fix/import-source-row](https://github.com/skygunner/magpie/tree/fix/import-source-row)（`0483a01f`，基于当时的 `upstream/main` `c0985122`）：这排标签改为换行排布（`flex-wrap: wrap`、`flex: 0 1 auto`、居中），同时去掉那个藏起来的横向滚动区——两行 CSS，别处没动。390px 下这排会占三行，对话框整体纵向滚动。

红 → 绿（我在独立环境复跑）：
- 干净 base `c0985122` 只放新测试：8 个组合全红（Chromium/WebKit × 英/中 × 390px/900px），例如 `chromium en 390px ... Claude Code– is cut off: 243.6..390.5 of 350.0`。
- 分支 head：8/8 通过，每个来源块都完整落在它那一行里。
- 相关导入/发现类 GUI 用例 66/66 通过；`go test -tags nogui ./internal/gui` 通过；`go vet`、`gofmt -l`、三个平台的 `go build`（默认 / `-tags nogui` linux / windows）全部干净。

没验证的部分：只量了 390px 和 900px、英中两种语言（来源名不翻译，其它语言宽度一致）；短视口下这排换成三行后对话框是否仍能完整滚动没有断言。

---

**2026-10-27 跟进：分支已重接到当前 main，并把"没验证的那部分"量出来了。**

原分支基于 `c0985122`，离当前 `upstream/main`（`11dcc743`）已 172 个提交。已重接：现在是对 main 的单提交 [`f45e9fa3`](https://github.com/skygunner/magpie/commit/f45e9fa3c352b23cbc241eaa5a2f31e7276b1bcf)（`0483a01f` 的 cherry-pick），净 diff 仍是 `app.css` 两行 + 一个新测试文件（+83/−2），别处没动。分支被重写（force-push），旧头 `0483a01f` 不再是分支头。

新 base 上重新做过红绿：

- 红：干净 `11dcc743` + 仅加该测试文件 → 8/8 全红（Chromium/WebKit × 英/中 × 390px/900px），例如 `chromium en 900px … OpenCode1 is cut off: 507.6..642.7 of 564.0`（旧 base 上 900px 的样本是 `Claude Code`，现在换成了 `OpenCode`——同一处溢出，只是换行位置不同）；
- 绿：分支头 8/8 通过。

**上次说"短视口下对话框是否仍能完整滚动没有断言"的那部分，现在有数字了**（Chromium，390px 宽，5 个来源）：

| 窗口高度 | 标签排 | 面板内容 | 对话框自身滚动 | Import 按钮 |
| --- | --- | --- | --- | --- |
| 900 | 104px（三行） | 353px | 不需要 | 窗口内 |
| 600 | 104px | 353px | 不需要 | 窗口内 |
| 400 | 104px | 353px | 不需要 | 窗口内 |
| 300 | 104px | 284px | 不需要 | 窗口内 |
| 250 | 104px | 244px | 不需要 | 窗口内 |

每个来源有 8 个供应商时，是 `.applist` 在面板内部滚动（600px 高时溢出 118px，400px 时 230px），按钮栏仍在窗口内；`.dialog` 始终不需要自己滚。代价是标签排从 36px 变成 104px（多 68px），250px 以上都还装得下。

这一条我试着写成断言，然后**删掉了**：它在旧 base 上也通过，把 `.modal .dialog` 的 `overflow-y` 硬改成 `hidden` 之后仍然通过——它不可能因为它声称的原因失败，留着只是假的绿。所以上面是实测数字，不是回归测试覆盖；"短视口"这个边界仍然没有自动断言。

**新 base 上的其余结果**：导入/发现类 GUI 用例 66/66；`go test -tags nogui ./internal/gui` 通过；`go vet ./internal/gui`、`gofmt -l internal/gui` 干净；`go build`（默认 / linux `-tags nogui` / windows）三个都 exit 0。

