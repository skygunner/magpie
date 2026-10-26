已在分支 [`fix/r25-write-stdin-wait`](https://github.com/skygunner/magpie/tree/fix/r25-write-stdin-wait)（基于上游 main 19253f67，单个提交 d9c57ab0）修复，未开 PR。

### 根因

`internal/gateway/tool_loop.go` 的 `toolLoopOf` 比较末尾连续 8 个工具交互（`toolLoopRepeats`）：`exchangeOf` 按「调用名 + 参数 + 结果」求和，`resultSum`/`steadyResult` 会把 Codex 的 `Wall time`、`Chunk ID` 归一化掉。于是 `write_stdin` 空输入轮询一个还在运行、暂时没有输出的进程时，8 次结果的差异恰好都被抹平，每个交互求和完全相同，guard 认为是同一个调用反复得到同一结果，`stoppedToolLoop`（`serveAgent` 与 Codex 自身登录的 `codex_backend.go` 两条入口都会经过）就在下一次请求处提前结束轮次——而进程其实还活着。

### 改动

`exchangeOf` 现在会识别「等待」型交互并在 `toolLoopOf` 里不计入循环：仅当该轮的每个调用都是 `write_stdin`、`chars` 为空、`session_id` 可读为数字，且结果里有一行 `Process running with session ID <同一 id>` 时，该轮记为 pending wait；对话以这样一轮结尾就不算 loop。非空 `chars`、结果没有声明该句柄仍在运行、句柄不匹配或 `session_id` 无法解析的轮次，仍按原来的重复工作判定，重复读同一文件等真实死循环不受影响。长期等待的独立超时/无进展策略没有在这次实现，属于另一个话题。

### 证据（RED → GREEN）

用的是 issue 里的确定性用例，原样保存为 `internal/gateway/pending_write_stdin_test.go`：

- 干净 BASE 19253f67 只加这个测试文件：`--- FAIL: TestPendingWriteStdinLoopGuard/pending_wait_must_continue ... stopped=true, want false for write_stdin`（`repeated_read_must_still_stop` 通过）。
- 修复后同一命令：两个子测试都 `PASS`，`ok github.com/yetone/magpie/internal/gateway`。

### 包内测试

`go test -tags nogui ./internal/gateway -timeout 45m`（HEAD d9c57ab0，290.7s）：除 `TestClaudeClientCacheAbsentUnchanged` 外全部通过。该失败在干净 BASE（同样加上新测试文件）上以相同方式复现，重跑 `-count=3` 仍失败，与本改动无关（Claude Code 的 cache-control 参数回归，`tool_loop.go` 不触及该路径）。

另外验证过：`gofmt -l internal/gateway` 无输出，`go vet ./...`、`go build ./...`、`GOOS=linux GOARCH=amd64 go build -tags nogui ./...`、`GOOS=windows GOARCH=amd64 go build ./...` 全部通过。

### 没有尝试的部分

真实 Windows 上的 Codex Desktop 端到端没有跑（本机是 Linux，只做了跨平台编译）；`session_id` 为非数字句柄、以及 "Process running with session ID <id>" 之外的 pending 文案没有覆盖——这些情况会保守地继续按重复工作计数。
