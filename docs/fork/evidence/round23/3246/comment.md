你的推测方向是对的：`wired()` 看两个文件，而 `Sync` 在 `!wired()` 时直接 `return nil`，所以只要 ZCode 自己在重启/更新时把 `provider_config.json` 里的 magpie 规则去掉，magpie 既不会写回，也不会提示——Agents 页就显示未接入，只能手动重连。网关一直活着这一点也吻合：掉的是接入状态，不是进程。

修复在 [fix/zcode-reconnect-after-rewrite](https://github.com/skygunner/magpie/tree/fix/zcode-reconnect-after-rewrite)（`c4495b30`，基于 `01468003` 一个提交）：`Sync` 发现 magpie 的 provider 在两个文件里都没了时，先看 `applied.json`——只有「magpie 确实写入过、且用户没在 magpie 里断开」才把 provider 规则重新写进 `provider_config.json` 与 `config.json`；`applied.json` 说不在、或用户断开过、或旧版 ZCode 根本没有规则文件（provider 只存在 `config.json`），都照旧不动。也就是说：外部重写会被修回来，用户在 ZCode 里主动删掉的仍然算用户的决定。

红 → 绿（独立复跑）：
- 干净 base `01468003` 只放新测试：`after a rewrite: "", want magpie back` —— 重写后同步什么都不做。
- 分支 head：`TestZCodeSyncRestoresARewrite` 通过，同时验证了「magpie 断开后同样的重写不会被写回」。
- `go test -tags nogui ./internal/agent` 整包通过（含 13 个 ZCode 测试）；`gofmt -l` 为空；`go vet` / `go build` 的默认、`nogui` linux、windows 三种组合都通过。

需要说明的边界：没有真的跑 ZCode 3.16.2，修的是两个配置文件与 `applied.json` 这一层；写回发生在 magpie 本来就有的同步时机（启动、模型目录变化等），不是秒级自愈，中间那段时间行上仍会显示 drift。如果你在 3.16.2 上还看到「显示已接入但实际掉线」，那是另一条路径，麻烦带 `applied.json` 与两个 provider 文件的内容再开一贴。