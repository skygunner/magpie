我把这个弄清楚了，也把之前那条评论里站不住的验证说法一并更正。分支 [fix/r25-dev-backend-handover](https://github.com/skygunner/magpie/commits/fix/r25-dev-backend-handover)，现在 head 是 `6d3f202fa62935bc480db953efcfb15256641c08`（在 `90095766` 的修复之上，只改测试），基于 `19253f67`。

先更正：我上一版说"事件版测试在基线上 20/20 红"，那是我搞错了。上一版把放行时机挂在 `backendCtx.Done`（`stopServing` 取消 backendCtx）上，而基线 `devBackend` 里 `go srv.Shutdown(ctx)` 和 `stopServing()` 是并发的，`backendCtx.Done` **不能**保证 `inShutdown` 已经置位；放行太早的话基线可能反而答了请求，那条红证据其实是更早的 sleep 版测试跑出来的，和分支上的测试不是同一份字节。这条已经改掉。

现在的屏障是测试自己包的 listener 的 `Close`。依据 go1.26.8 的源码：`Shutdown` 先 `s.inShutdown.Store(true)`（`net/http/server.go:3151`）再 `closeListenersLocked()`（`:3154`），而 `Serve` 会把传进去的 listener 包一层并 track，Shutdown 关的就是测试包的那个对象；修复侧的 `finish` 是先关 listener 再 Shutdown。所以"listener 的 Close 开始"这个事件，在基线上一定是"标记已置位之后"，在修复上一定是"Shutdown 之前"——测试只在看到它之后才放行被扣住的连接、写请求。wrapper 的 `Close` 只 close 一个 channel 再转发、不阻塞（Shutdown 是拿着 net/http 自己的 listener 锁调它的，阻塞会把 Shutdown 挂在锁里）。

信号那边：基线的 `signal.Notify` 在 `go srv.Serve` **之后**，首连 barrier 证明不了它已注册，所以测试先自己注册 SIGUSR2（早期信号不会把进程打死），先发一次信号，此后每次收到测试自己的信号通知才重发，直到 listener Close 事件出现。`f5fd1da3` 中按 2 秒超时重发的循环已在 `6d3f202f` 改为这个事件驱动循环；一个总计 10 秒的超时只用于失败退出，不用于触发重试。

最终版本的 base/head 是**同一份测试字节**：`git archive 19253f67` 出来，只把这个测试文件拷进去（cmp 和 SHA-256 确认一致），`-count=3` 是 3/3 红，报的就是客户端那侧 `unexpected EOF`（go test 退出码 1）。分支最终版本普通运行通过，`-race -shuffle=on -count=3` 也通过（退出码 0）。gofmt、`go vet -tags dev ./internal/gui`、`go build -tags dev ./...` 均退出码 0。

诚实上限：修复侧 `unread` 轮询（10ms）看到 StateActive 到 Shutdown 置位之间，仍有和上游 #3568 修复同一个纳秒级窗口；上述重复执行只是回归覆盖，不是形式化证明。信号握手也依赖被测代码最终能推进，不能推进时会明确超时失败。没试的照旧：真机 `make dev` 重启（测试在一个进程里交 socket）、Windows 运行时（不交接，只跑 vet/build）。这个测试文件改动之外的结论不变，gateway 没碰。
