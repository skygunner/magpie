@yetone 三条评审意见都按你说的方向重做了，新分支：[fix/4144-scanframes-resync](https://github.com/skygunner/magpie/tree/fix/4144-scanframes-resync)，head [`7a1054161`](https://github.com/skygunner/magpie/commit/7a1054161c64)，基于当前 `main`（`289184a40`）。旧支 `fix/4144-sessions-frame-tail` 保留作历史。

**方向：跳过损坏帧、重同步、让尾部继续走（fail-open），不再 fail-closed。**

对应你的三条：

1. **不再只是停在损坏处**：`scanFrames` 读失败后，从损坏点向后按帧头扫描（`frameSize` 的纯头遍历，不解码）找下一个合法帧起点；找到就从那里继续读，尾部照常前进。帧级损坏的三种形态都覆盖：magic 坏（errFrame）、块长度字段坏到跑出文件（ErrUnexpectedEOF，但**先探测后方是否真有后续帧**——没有就按「正在写入的尾部」等待下一次读取，健康文件的追加尾部不会被误判成损坏）、块解码失败/checksum 不符（DecodeAll 错误，跳过该帧继续）。找不到后续帧就不重同步，保持原等待语义。
2. **不再回退到空**：全程不返回错误。`parseDsh`/`readDshCalls` 保留已读内容并从损坏帧之后继续，`markTail` 落在损坏帧之后，后续轮询从那里增量读——不会每次全量重读也不会给出空文件。跨帧的半行随损坏帧丢弃（它的剩余部分已丢失）。
3. **测试断言用户可见结果，不用真实文件也先不硬凑**：测试用 `testdata/dsh/.../dsh-a/session.v3.jsonl` 的**真实会话行**、按 dsh 实际写盘方式打包成帧（每帧一批整行、追加），对中间帧做磁盘上三种真实损坏形态（checksum 翻转、magic 覆写、块长度改成 2MB），断言的全是可见行为：损坏后的帧计入（fixture 里第二条 assistant 消息的 10 input tokens 在、第一条的 100 不在）、`markTail` 落在损坏帧之后、文件追加后从那里增量读、文件以损坏帧结尾时仍按「写入中」等待。三个场景在未修的 `upstream/main` 上全部红（tail 卡在 234 不动），修复后绿。如果你手上仍有真实环境的损坏文件，发我（脱敏即可）我就把它加成第二个 fixture——这条邀约继续有效。

其他：`parseVersions` dsh 提到 4、`callParseVersions` 加 `dsh: 1`（损坏文件的旧缓存/分片只含读到损坏处的内容，需重读；**只动 dsh 自己的两级版本**，`callCacheVersion` 仍是 `calls-v15` 没碰——上一轮的 v16 注释失误不会再犯）。`go test ./internal/sessions/` 全绿，gofmt/vet/`go build -tags nogui ./...` 干净。

未验证：没有真实环境的损坏文件（如上，fixture 行是真的、打包方式是 dsh 的、损坏形态是模拟的）；未在真实 dsh 客户端下跑长会话复现。
