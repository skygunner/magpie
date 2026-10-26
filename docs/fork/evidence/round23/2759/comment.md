根因在导入面板那排来源标签的样式：`.apptabs` 是一条不换行的 flex（`flex: 1 1 0; min-width: max-content`），标签缩不到文字宽度以下，五个来源一多就溢出；而外层 `overflow-x: auto` 配着 `scrollbar-width: none`，横向滚动条被藏掉，末尾的 OpenCode 既看不见也拖不到，只能先选中一个再用方向键移过去。量到的数字：390px 窗口下滚动区 350px、内容 646px；900px 时 OpenCode 仍被切掉。

修复在 [fix/import-source-row](https://github.com/skygunner/magpie/tree/fix/import-source-row)（`0483a01f`，基于当时的 `upstream/main` `c0985122`）：这排标签改为换行排布（`flex-wrap: wrap`、`flex: 0 1 auto`、居中），同时去掉那个藏起来的横向滚动区——两行 CSS，别处没动。390px 下这排会占三行，对话框整体纵向滚动。

红 → 绿（我在独立环境复跑）：
- 干净 base `c0985122` 只放新测试：8 个组合全红（Chromium/WebKit × 英/中 × 390px/900px），例如 `chromium en 390px ... Claude Code– is cut off: 243.6..390.5 of 350.0`。
- 分支 head：8/8 通过，每个来源块都完整落在它那一行里。
- 相关导入/发现类 GUI 用例 66/66 通过；`go test -tags nogui ./internal/gui` 通过；`go vet`、`gofmt -l`、三个平台的 `go build`（默认 / `-tags nogui` linux / windows）全部干净。

没验证的部分：只量了 390px 和 900px、英中两种语言（来源名不翻译，其它语言宽度一致）；短视口下这排换成三行后对话框是否仍能完整滚动没有断言。