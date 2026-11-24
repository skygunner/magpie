@wuyak 这条路径应该由 #4953 的修复覆盖，也按你的窄修法做了——它不属于 #2906/#1378 那类"不改 Codex 提示词"的场景：客户端把基础提示词自己的副本放进 input（首行即官方首行），再叠加一份官方文本是重复投递，不是改写。

实现：请求 `instructions` 为空时（Lite 形状本来就没有官方文本，不受影响），若 input 第一条 developer 消息的首行等于官方首行，就认定客户端已自带基础提示词，不再追加官方副本；首行不同则仍是客户端自己的提示词，原样保留。判断与 5.6 现有的 `firstLine(own) == firstLine(codex)` 直通同一标准；#4953 原有形状（整文/模板匹配、`model.base_instructions` 标记）不受影响。

- 分支：[fix/4953-translated-first-line](https://github.com/skygunner/magpie/tree/fix/4953-translated-first-line)（提交 58f2c396d，基于当前 upstream/main 8d0226d9b）
- 回归测试 `TestCodexBodyTranslatedFirstLineGetsNoSecondCopy`：把该判断手动去掉后失败（`client's copy: 2 base prompts, want 1`），恢复后通过；`TestCodexBodyAsCodexSendsIt`、`TestCodexBodyLiteGetsNoInstructions` 不受影响。整包 `internal/provider` 里两个 Claude 登录测试在干净 upstream/main 上同样红（本机环境，日志已附基线对照），与本次无关。红/绿/基线日志在 fork 的 `docs/upstream-issue-followup-20261119` 分支 `docs/upstream-issues/2026-11-24/evidence/4953/`。

一个确认请求：5.6 上那份 62-token 的客户端 system，首行是官方首行原文，还是客户端自己的变体？如方便请只贴首行前 ~80 个字符即可，不用全文——这能验证"首行相等"这个判断对真实 Claude Code 够不够准。
