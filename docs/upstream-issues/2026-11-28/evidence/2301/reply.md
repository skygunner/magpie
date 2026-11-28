@NkAntony777 感谢极其详尽的复现——Usage reader 读到 MiniMax Code（1.21B tokens、111577 calls，且 `magpie ls` 解析 `D:\minimax data\.minimax\config.yaml` 正确），但 `magpie sessions read` 仍零行；换回旧路径或合成最小样本都能复现，说明不是路径/空格/移动文件夹的问题。

这说明 `sessions reader`（`internal/sessions/sessions.go` 的聚合管道）没有把 minimax-code 的会话算进它的“找到的代理”列表，而 `usage reader` 的 `Record.Decode` 能读到。两个 reader 走的代码路径不同（一个聚合、一个逐文件解码），差异可能在 `internal/sessions/decoder.go` 的模型 ID 识别、或 `internal/sessions/sessions.go` 的 `DecodeAll` 过滤上。

这个 issue 在 357 清单里吗？搜了清单里没有 `2301` 的条目，说明不在跟踪范围内。如果想让我们跟进，请确认是否需要把它加入清单（需要 opener 语言、标题、分类等字段），或者由维护者在原 issue 明确让我们接手。目前先记录在此；如需跟进请在 issue 里说一声。
