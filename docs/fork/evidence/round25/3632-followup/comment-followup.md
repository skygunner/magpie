已修，分支 [`fix/r25-filememo-forget-bubble`](https://github.com/skygunner/magpie/tree/fix/r25-filememo-forget-bubble)（基于 19253f67bca2310c32739a6eaac88546b3574dbc，单提交 3a0b4479）。

**根因**：#3185 之后 `internal/filememo` 的 `Forget()`（filememo.go:91）在锁内 `close(written)` 再 `written = make(chan struct{})`。`written` 是包级变量，第一个 synctest bubble 里的 `Forget` 把新 channel 造在了那个 bubble 里；`internal/provider` 的 `TestUnreadLoginsWriteErrorsKeepCredentials` 每个子测试各跑一个 bubble，第二个 bubble 里 `provider.Changed()`（hold.go:146）→ `writeLogins`（logins.go:272）→ `Forget` 去 close 它，运行时直接 `fatal error: close of synctest channel from outside bubble`，整个包的测试进程崩掉。生产代码不在 bubble 里，用户侧无影响，纯属测试崩溃。

**改法**：与楼主的思路一致——channel 只在有人等的时候才存在。`Written()` 首次被调用才 `make`；`Forget` 里有就 `close` 并置回 nil，没有就完全不碰。没有等待者的写（bubble 里跑的测试正是这种）既不造也不关 channel；等着的循环拿到的 channel 仍由下一次写关掉叫醒，"a wait begun after it waits for the one after" 的语义不变（原有 `TestWritten` 原样通过）。改动只有 `internal/filememo/filememo.go` 一处生产代码。

**回归测试**：`TestForgetsAcrossSynctestBubbles`（internal/filememo/forget_bubble_test.go），两个 `synctest.Test` bubble 各调一次 `Forget`：

- 红证：干净 BASE + 仅加该测试文件 → `fatal error: close of synctest channel from outside bubble`，栈顶正是 `filememo.Forget()` filememo.go:91；
- 绿证：修复后同一测试通过，`go test ./internal/filememo/` 全绿。

**目标包测试结果**（go1.26.8，linux/amd64）：

```
go test -count=1 ./internal/filememo/    ok
go test -count=1 ./internal/provider/    ok 43.0s   (原崩溃测试在内)
go vet ./...                             exit 0
go build ./...                           exit 0
GOOS=linux GOARCH=amd64 go build -tags nogui ./...   exit 0
GOOS=windows GOARCH=amd64 go build ./...             exit 0
gofmt -l internal/filememo internal/provider         无输出
```

**没试的**：没在 macOS 上跑（本地只有 linux/windows 构建，mac CI 的崩溃未本地复现）；分支基于 19253f67，upstream main 上更新的 7 个提交（de49433f 等）给 filememo 加了 `wake()`，用的是同一个 `close(written)+make` 模式，落到 main 时需要同样改成懒创建；GUI 端到端未跑（本修复不触 UI 路径）。


---

**2026-10-27 跟进：分支已并入最新 main，`wake()` 同步改为懒关闭**。上游 `de49433f`（#3185）给 filememo 加了 `wake()`（Watch 看到其他进程写文件时唤醒等待循环），用的是与 `Forget` 相同的 `close(written)` + 重造模式——正如 @JoeyWu-tech 在楼上指出、其补丁 `0576b589` 已一并覆盖的，只改 `Forget` 在当前 main 上不再完整。分支现已 merge upstream/main（`d9a8148d`）并把 `wake()` 改成同样的"有人等才造、关闭即置 nil"，新头 [`47eb5102`](https://github.com/skygunner/magpie/commit/47eb5102a58e7d931ab57bffa350124fadcbdc6e)（普通推送，未强推；净 diff 对 main 只剩 filememo 一处 + 回归测试）。

新红绿（go1.26.8，linux/amd64）：

- 红：干净 `d9a8148d` + 仅加 `TestWakeAcrossSynctestBubbles` → `fatal error: close of synctest channel from outside bubble`，栈顶 `wake()` filememo.go:101（新增的第二个关闭点）；
- 绿：分支头 `filememo` 包全绿；原崩溃测试 `TestUnreadLoginsWriteErrorsKeepCredentials` 通过；`vet` / `build`（linux、windows、nogui）/ `gofmt` 全过。

**一个如实的边界**：分支头满包跑 `internal/provider` 时，`TestOpenCodeSessionOnEveryCall` 出现顺序/负载相关红——`openCodeNewID` 的每毫秒计数器越过 15 后，自己刚生成的 id 过不了 `openCodeRealSession` 的低位校验（失败样本计数器已到 ~1006）。该测试单独跑、`TestOpenCode*` 家族 `-count=3` 全绿；干净 main 跳过崩溃测试的满包复跑同样出现过这个红（两次一绿一红），且本分支净 diff 不触任何 provider 文件，判定为 main 既有负载 flake，与本改动无关，但满包并非全绿，如实记下。
