根因在 `tlsListener` 与 `ListenAndServe` 的收尾顺序（internal/gateway/tls.go、internal/gateway/gateway.go）：TLS 开启时，来自本机的连接先在 sniff goroutine 里等第一个字节（最多 `sniffWait` 10s），读到才通过 `conns` channel 交给 `http.Server`。交接开始时 drain goroutine 调 `srv.Shutdown`，它先关 listener（`tlsListener.Close` → `close(l.done)`）——这时还没交给 server 的连接，其 sniff 读到第一个字节后调 `hand()`，select 走进 `<-l.done` 分支被 `c.Close()`，一个字都没回。#3568 的"Shutdown 前等每个已接下的连接读完第一个请求"管不到它，因为 `http.Server` 从没见过这个连接。

修复：交接收尾（`HandOff` 和 `make dev` 共用的 drain 路径，`Handover || handedOff`）先调 `tlsListener.handOff()`，它只停内层 Accept，并等到每个已接下的连接都被交给 server 且在那里被处理完（读到了请求并应答，或被 hijack，或已关闭；一直不说话的连接仍按原样在 `sniffWait` 到点被切）。之后才照常 `srv.Shutdown`——这样 Shutdown 开始时没有还没读的请求，server 会应答而不是按它的 Shutdown 语义把后到的请求直接丢弃；Shutdown 等待在途请求、忽略 hijacked 连接的行为不变。非交接的退出路径（`Close`）完全不变。

证据：

- RED：干净 base（19253f67）上只加测试 `TestHandedOffTLSGatewayAnswersAConnectionWaitingForItsFirstByte`（internal/gateway/tls_handover_test.go），最终事件同步版失败为 `no answer to a request sent once the handover began: ... connection reset by peer`。
- GREEN：最终事件同步版同一测试在 `-race -shuffle=on -count=1` 下通过（退出 0）。不再依赖固定等待连接被 accept。

分支：https://github.com/skygunner/magpie/tree/fix/r25-tls-handover-local （base 19253f67；包含原修复及两次正常追加的测试修订，共 3 个提交）

运行过的检查：`gofmt -l`（改动目录，无输出）、`go vet ./...`、`go build ./...`、`GOOS=linux GOARCH=amd64 go build -tags nogui ./...`、`GOOS=windows GOARCH=amd64 go build ./...`，以及 `go test ./internal/gateway/ -count=1 -timeout 45m` 完整包套件：除 `TestClaudeClientCacheAbsentUnchanged`（Claude 客户端缓存的 prompt 参数断言，与本改动无关；在干净的 19253f67 副本上同样失败，见 `claude_client_cache_test.go:291 ordinary prompt changed`）外全部 ok。

没有尝试 / 边界：

- 没在真实桌面端（GUI 进程 + 真实证书）里做端到端交接，测试走的是同一 `ListenAndServe`/drain 路径加自签证书。
- 一直不说话的本地连接（浏览器 preconnect 那类）现在会把交接的 drain 拖住最多 `sniffWait`（10s）——这是修复的代价，换来的是开口的连接能被应答。
- Windows 只做了构建，没在实际 Windows 环境跑过。


测试修订（2026-10-26）：最终 HEAD 为 [691aa25451562cc614c7fc90ec6d30a56f8ded2f](https://github.com/skygunner/magpie/commit/691aa25451562cc614c7fc90ec6d30a56f8ded2f)。测试仅在测试文件内包装真实 socket，沿已有 `Inherit` → `ListenAndServe` 路径，先等首次 Read 事件，再开始 handover，等 listener Close 事件后才发请求。删除原先 500ms 等待；生产实现未改。干净基线仅加入同一测试即因 connection reset by peer 失败，最终分支通过。格式、全仓库 vet、原生/Linux nogui/Windows 构建再次退出 0。扩大 TLS/handover 相关测试时，`TestHandedOverGatewayFinishesItsStream` 的 request 0 报 EOF；同一 shuffle seed `1793007267019698289` 在干净基线上也有同一断言失败，另有本修复预期的 RED，故如实列为既有失败，未宣称相关集合全绿。LSP 的 gopls 未安装。
