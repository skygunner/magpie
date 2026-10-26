根因：`dshParse` 把第一条目之前的行交给了 `head`（#2667）、flow list 的尾部交给了它的条目（#2674），但**位于两个条目之间、顶格的注释行**落进了 default 分支，被追加到它*前面*那个条目上。`dshPutRoute`/`dshPutSearch` 是整块重建条目的行（`yaml.Node` 不携带注释，`dshSearchLines` 是模板），所以下一轮把它丢掉了。你的 Vibe Island 区块里 `>>> vibe-island managed block >>>` 跟着被 magpie 重写的那个条目一起消失，而 `<<<` 标记还在，工具于是以“有 closing 标记但没有 begin 标记”拒绝这个文件。

修复在 [fix/dsh-comment-between-entries](https://github.com/skygunner/magpie/tree/fix/dsh-comment-between-entries)（`91a19f25`，基于今天的 `main` `01468003`）：条目之间顶格的注释行或空行现在作为独立的一项保留在两者之间，重建任何一侧邻居都拿不走它。缩进的行仍然属于它前面的条目，所以“什么算真正的 patch 条目”没有变化。故意不把它挂到后面的条目：那个条目（agent-default-model、llm-pi-ai、web-search-deepseek）本身也可能被 magpie 重写，那样标记就会在那一轮丢掉。

- 红 → 绿：`TestDshCommentsBetweenEntriesSurviveRewrite` 写入你的文件（magpie 的 route、托管区块的两个标记、`- insert:` 条目），用 catalog 的 route 经 `dshRouteAgain` 走一轮 `keepDshWired`。在 `01468003` 上只加测试：`the managed block's comment markers lost`；在 head 上文件完整回来。
- `internal/agent`（`-tags nogui`）通过；`gofmt -l` 为空；`go vet`/`go build` 在 linux、windows、`-tags nogui linux` 上干净。
- 未尝试：macOS 上的真实 DeepSeek Harness。*缩进在*条目内部、且该条目被 magpie 重写的注释仍属于那个条目、不会被保留——本次改动覆盖的是条目之间的顶格位置，与你的报告一致。
