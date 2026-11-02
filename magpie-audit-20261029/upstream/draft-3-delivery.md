已更新 issue 正文中的 inline diff；先前候选补丁存在并发回归，请以正文修订后的完整 diff 为准。

分支：https://github.com/skygunner/magpie/tree/fix/4136-wsl-probe-lock
提交历史：https://github.com/skygunner/magpie/commits/fix/4136-wsl-probe-lock/
最终提交：https://github.com/skygunner/magpie/commit/788f18e38b92cdd1a3a1c1cff8cf846552850319

宿主与 Windows 交叉 build/vet 通过；Windows 测试及真实 WSL 尚未运行。分支在 skygunner/magpie，欢迎直接 cherry-pick 吸收。
