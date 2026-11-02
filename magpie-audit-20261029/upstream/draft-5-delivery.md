补丁已推送：[fix/4144-sessions-frame-tail](https://github.com/skygunner/magpie/tree/fix/4144-sessions-frame-tail)。完整 inline diff 在 issue 正文。

提交历史：https://github.com/skygunner/magpie/commits/fix/4144-sessions-frame-tail/
最终提交：https://github.com/skygunner/magpie/commit/ba4f3ef69a0fbfd85c0250b9e8d8dcef0ba3866f

此补丁安全停止并保留上次成功结果，不自动恢复损坏帧之后的记录；需修复源文件并触发缓存重读。分支在 skygunner/magpie，欢迎直接 cherry-pick 吸收。
