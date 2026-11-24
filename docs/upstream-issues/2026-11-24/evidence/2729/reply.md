@real-jacket 谢谢 [#7426](https://github.com/yetone/magpie/issues/7426) 的实测——flash 在 Responses 上 4 次里 0 次搜索、Anthropic 上 2 次里 2 次，正好落在 fork 分支 [fix/2729-gateway-deepseek-s-search-api](https://github.com/skygunner/magpie/tree/fix/2729-gateway-deepseek-s-search-api)（a8de723e，base 8bade6c5）的判断上：Responses 只把 `deepseek-v4-pro` 当作会自己搜索，其余模型由 magpie 代搜；Anthropic 端点保持不动。你的 Anthropic 实测也正好解释了为什么那里不改。

一个诚实的边界：这个分支的测试目前只覆盖 Responses 路径。上游 `searchHosts` 对 Anthropic 仍含 `api.deepseek.com`，翻译路径的循环会把 flash 的 `web_search` 挪到 Anthropic——也就是你说的选项 2 的实际行为（Anthropic 原生搜索），但这一跳还没有测试盖住。我会在分支上补一个双协议可用时的 translate 路径回归测试，证明 flash 的搜索请求确实落到 Anthropic 而不是静默丢弃，补好后在这里更新。

如果你更希望选项 1（Anthropic 也不透传、一律 magpie 代搜），那是行为裁定，留给维护者；分支现在的形状是"Responses 收窄 + Anthropic 不动"。
