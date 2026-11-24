@pluo 谢谢你提供的前后对比——重启前 `windowAt` 是右上角的 `[2113,44]`、重启后被写成 `(3072-942)/2 = 1065` 的居中值，这就把"保存正常、恢复失败后又被回写"这条链坐实了。你指出的时序判断和补丁思路都是对的。

我按这个思路在当前 upstream/main（8d0226d9b）上重建了修复：Wails 屏幕缓存还没有可用工作区时，在主线程直接读 AppKit 的 `visibleFrame`，坐标按你评论里的换算（主屏左上角为原点、y 向下的逻辑点）交给现有 `openAt` 恢复；缓存里有可用数据时仍优先用缓存，原位置不在任何现存屏幕上时仍按原逻辑居中。提交里已按仓库约定标注 `Co-authored-by: pluo`。

- 分支：[fix/2785-window-place-before-screens](https://github.com/skygunner/magpie/tree/fix/2785-window-place-before-screens)（提交 3e93c30ea）
- 回归测试 `TestOpenAtBeforeScreenCache`：把兜底逻辑手动去掉后失败（`openAt = 0, 0, false; want 2113, 44, true`），恢复后通过；`go vet -tags gtk3`、`-tags nogui`、`GOOS=windows` 干净，`go build -tags nogui` 通过，internal/gui 与 internal/settings 全包测试除一个在干净 upstream/main 上同样红的 `TestCoreScriptBudget`（脚本预算 1278640，当前 1279029）外全绿。红/绿/基线日志在 fork 的 `docs/upstream-issue-followup-20261119` 分支 `docs/upstream-issues/2026-11-24/evidence/2785/`。

边界照旧说明：我这边没有 Mac，这条分支还没在实机上跑过完整的应用内升级重启；你已验证的是同一思路下的"空缓存 + 真实窗口"场景。等维护者收取后，实机单屏/双屏的升级重启验证仍是必要一步。
