@yetone 这条栈就是这条分支修的那个 close：[`fix/r25-filememo-forget-bubble`](https://github.com/skygunner/magpie/tree/fix/r25-filememo-forget-bubble)，现已并入当前 main（`e056779a`，含你跑的 `2f00fa642`），头 [`c9e34bbd`](https://github.com/skygunner/magpie/commit/c9e34bbd)（快进推送）。

`provider.Changed ← writeLogins ← addSideLogin ← addGoogleLogin → Forget` 是 `Forget` 那个 close 位。分支的机制对两个 close 位（`Forget` 和 #3185 加的 `wake`）一致：`written` 有人等（`Written()`）才造、close 即置 nil，没有等待者就什么都不 close——被 close 的 channel 永远是在 close 它的同一侧造出来的，跨 bubble 的 fatal 不可能再发生。

在你看到的这个 main 上复跑（go1.26.8, linux/amd64）：

- 干净 `e056779a` 整包 `go test ./internal/provider/`：16.5s 处同一个 fatal（`filememo.Forget`，`logins_unread_test.go:63` 的 `TestUnreadLoginsWriteErrorsKeepCredentials`），和你 macOS 看到的一致；
- 分支头整包 45.2s 完成、无 fatal；仅有的 4 个红（`TestCodexAppServersOnHome`、`TestFetchAsksOneListOnceForTwoBasesOfIt`、`TestFetchUnionsProtocolListsWithoutChat`、`TestFetchAsksASharedBaseOnce`）在干净 `e056779a` 上单独跑同样红，判定为本机基线，与分支无关（净 diff 只剩 `internal/filememo` 两个文件）；skip 这四个后整包 ok，`internal/filememo` ok；
- 分支的两个回归测试（`TestForgetsAcrossSynctestBubbles` / `TestWakeAcrossSynctestBubbles`）只放在干净 `e056779a` 上就红（fatal 落在 `Forget`/`wake` 的 close），分支头上绿。

净 diff：internal/filememo/filememo.go（+23/−6）+ 新增 forget_bubble_test.go（23 行）。