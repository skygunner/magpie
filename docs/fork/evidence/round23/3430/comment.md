根因确认在 `internal/gateway/anthropic.go`：`aUsage` 只有 input/output/cache 三个字段，没有 `output_tokens_details`，所以 `usage()` 永远填不出 `Usage.Reasoning`；反方向（magpie 把用量写回给 Claude 客户端的 `message_delta`）也没有 `thinking_tokens`。你自己贴的那行 Copilot 用量（output 1277、thinking 710）就是最干净的复现输入。

修复在 [fix/anthropic-thinking-tokens](https://github.com/skygunner/magpie/tree/fix/anthropic-thinking-tokens)（`fb6763d9`，基于今天的 `main` `01468003`）：`aUsage` 读 `output_tokens_details.thinking_tokens` 进 `Usage.Reasoning`，转述路径和翻译路径共用同一个解码，所以 Anthropic / Copilot / Bedrock / 中转一起生效；回写方向在有思考时补上 `output_tokens_details.thinking_tokens`，Chat/Responses 上游的思考量也能被 Claude 客户端读到。

- 红 → 绿：`TestMessagesReplyCountsItsThinkingTokens` 用你的 `message_delta` 原文，两个子用例分别走 relayed（`usageSniffer`）与 translated（Anthropic 解码器）。在干净 base 上只放测试：`ledger says output 1277, reasoning 0; want the reply's 1277 and 710`，且 translated 的 `message_delta` 里没有思考 token；改动后两个子用例都过。
- `internal/gateway`（`-tags nogui`）在本机有一项与本题无关的红：`TestClaudeClientCacheAbsentUnchanged`，在干净 `01468003` 上同样红（#3280），没有掩盖也没改弱。`go vet`/`go build`（linux、windows、`nogui linux`）通过，`gofmt -l` 为空。
- 如实说明：没有调用真实 Copilot/Anthropic 账号，也没有跑 Claude Desktop；证据是网关解码层与你贴的用量 JSON 逐字对齐。