<!-- 草稿：未发送。发送需用户单独授权。目标：yetone/magpie#3447（issue 已由维护者关闭；本帖为更正，不争议关闭决定）-->

更正我此前（06:17）交付帖里的两处说法：

1. **「两条路由都走 magpie 代搜管线」不准确。** `deepseek/deepseek-flash`（api.deepseek.com）在 `searchHosts[Anthropic]` 名单里，模型自己会搜，请求是 passthrough 逐字节转发——它结果条目的五字段形态（`type/title/url/encrypted_content/page_age`）是 DeepSeek 自己写的，magpie 既没碰也无法补 citations。只有 `auto/deepseek-v4.1-flash` → raccoon 那条才走 magpie 代搜管线。@ggbdpq 在其交付帖（06:36）里已指出这一点，是对的。

2. **上游已用另一套语义修复并经报告者实测。** `1d6b72ad`（v0.1.1369 起）采用「回答文字里实际写出完整 URL 的行，就在该行附 citation、摘录取自搜索模型回复的 text.citations」的合同，与 @ggbdpq 的 69068ead（搜索器摘录直引，无论回答是否点名 URL）不是同一份合同。报告者 @HuHong-Tao 已在真实 0.1.1371 上端到端验证（两条路由、流式 + 整包、DSH 的 mapAnthropicResponse 拿到 snippet），维护者据此关闭 issue，这个结果我认可——真实流量里模型确实会点名 URL。

边界说明（不要求动作，仅供后续参考）：在「模型回答没有点名任何 URL、但搜索器给了摘录」的形状下，1d6b72ad 不产生 citations——ggbdpq 公开的两个端到端 fixture 在当前 main 上仍是红的（我方在 9d643631 上复跑同为 2 红 1 绿）。这是两套合同的真实分歧，是否值得覆盖由维护者判断；我方 fork 分支（fix/issue-batch-1024pm 的 e5a05c44 + 14c1551）按「搜索器摘录直引」语义保留为备选，其中 searchWith 对搜索器自带 cited_text 的保留缺口已被 r11 独立审查指出、记录在案，不再单独修补。无意与 @ggbdpq 的交付竞争或重复投递。
