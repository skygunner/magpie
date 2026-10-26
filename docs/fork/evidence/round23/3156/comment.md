根因：那条说明里的成员是**请求发出时**规划下来的那份（trace 里 route 的成员），不是组现在的列表。所以编辑组并保存之后，这句文字仍然念旧成员，切页面也不会重新取——只有重启 magpie 让它重新规划才对得上。

修复在 [fix/routing-group-members-live](https://github.com/skygunner/magpie/tree/fix/routing-group-members-live)（`f816c358`，基于 `01468003` 一个提交）：实时路由的说明改为念这个组**当前**的成员——和下面「路由组」用的是同一份 `/api/groups`，保存后立即重画，不必再发一次请求、也不必重启；组本身不存在时（已删除）仍按 trace 里当时规划的那份显示。

红 → 绿（我在独立环境复跑）：
- 干净 base `01468003` 只放新测试（页面还是旧代码）：四组用例（Chromium/WebKit × 英/中）全部失败，实际文案仍是 `Claude Code 请求了路由组 Hand：a/m, b/m`，而组已经改成 `a/m, c/luna`。
- 分支 head：`routing-group-members-live.test.cjs` 4/4 通过，说明里读到 `A · MA · C · Luna`。
- 同一套 routing 页面测试里，`routing-pause.test.cjs` 与 `routing-rule-models.test.cjs`（WebKit，hover 时元素离开视口）在**干净 base 上同样失败**，与本次改动无关；这条我在 base 和 head 各跑了一次对照。

需要说明的边界：没有在真实 magpie 上点一遍——页面与接口用的是仓库既有 GUI 测试的 fixture 风格（假 /api/groups、假 trace），保存动作走的是页面真实的保存按钮；浏览器用的是本机 npx 缓存里的 Playwright 1.64.1（仓库默认模块路径那版没装对应浏览器）。另外这条只改「说明里念的成员」；如果你看到的是**路由实际上还在往已删除的成员发请求**，那是另一条路径（组的生效配置），请再开一贴带 trace。