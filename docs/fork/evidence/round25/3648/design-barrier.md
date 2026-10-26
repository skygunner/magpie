# #3648 确定性事件屏障设计（源码依据，供 lead 审阅）

## 问题

083079e4 的测试用 `<-backendCtx.Done()` 作为"交接已开始"的屏障。基线 devBackend 的顺序是
`go srv.Shutdown(ctx)` → `stopServing()`：`stopServing` 取消 backendCtx 与
Shutdown goroutine 置位 `inShutdown` **并发无序**，因此放行连接时标记可能尚未置位，
基线可能假绿。且基线 RED 证据出自旧 sleep 版测试字节，与 HEAD 测试不同字节。

## 源码事实（go1.26.8 net/http/server.go）

1. `Shutdown`（:3150）：`s.inShutdown.Store(true)`（:3151）在 `s.closeListenersLocked()`
   （:3154，持 `s.mu`）**之前**。⇒ 基线中"listener 的 Close 被调用"蕴含标记已置位。
2. `Serve`（:3410-3421）把传入 listener 包成 `onceCloseListener` 并 `trackListener(&l, true)`，
   Shutdown 关闭的正是这个对象 ⇒ 测试包装的 heldListener 会被 Shutdown 调到 Close。
3. Shutdown 在 `s.mu` 临界区内调用 `Close` ⇒ wrapper 的 Close **不得阻塞**：
   只 close 事件 channel 再转发 `l.Listener.Close()` 并返回。阻塞会把 Shutdown goroutine
   停在 net/http 的 listener 锁内（无谓的挂起面）。
4. Shutdown 随后 `s.listenerGroup.Wait()`（:3160）等 Serve 返回；Serve 卡在测试持有的
   Accept 里 ⇒ 基线的 `done`（devBackend 返回）依赖测试放行；测试在事件后立即放行，无死锁。
5. 方向性：
   - 基线：Close 事件 ⇒ `inShutdown` 已置位（channel close 的 happens-before）。
   - HEAD：`finish` 先 `p.ln.Close()` 再等 Serve、unread、Shutdown ⇒ Close 事件 ⇒
     Shutdown 尚未调用、drain 已在等连接读。
   测试在事件后才放行 Accept 并写请求 ⇒ 基线必然切（RED），HEAD 必然答（GREEN）。

## Notify 就绪

基线在 `go srv.Serve(ln)` **之后**才 `signal.Notify`，首连 barrier 不能证明已注册。
处理：测试在启动 devBackend **前**自行 `signal.Notify(sigTest, SIGUSR2)`——进程对早期
Kill 免疫（不会默认终止）；若 devBackend 错过某次 Kill（未注册时收到），它停在 `<-sig`，
测试在事件上等不到 close 就再次发信号（有界事件驱动重试，无 sleep；多余的 Kill 只是缓冲
在各 channel 里）。HEAD 的注册在 `go Serve` 之前，首次 Kill 必达。5 次后仍无事件则
t.Fatal（响亮失败，无假绿）。

## 诚实上限

HEAD 侧 `unread` 轮询（10ms）观察到 StateActive 与 Shutdown 置位之间仍存在与上游
#3568 修复相同的纳秒级窗口；以 -count=20 双侧证据覆盖，不做形式化声明。

## 不改生产代码

heldListener、closed 事件、重试循环全部在测试文件内；`Inherit` 与自定义 `net.Listener`
是既有公开面。
