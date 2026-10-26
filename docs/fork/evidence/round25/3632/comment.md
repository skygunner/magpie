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
