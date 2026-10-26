根因与修复：https://github.com/skygunner/magpie/commit/82d23059d13bf87fc1ef426bb785e4daf68aa77d（分支 [`fix/r25-library-uncheckbox`](https://github.com/skygunner/magpie/tree/fix/r25-library-uncheckbox)，基于 upstream main `19253f67`）

**根因**（`internal/library/skills.go` 的 `importSkill`，行为自 #227 的 ac1f5221 起）：从 Agent 发现列表把 `~/.agents/skills/<name>` 里的技能“导入”资源库时，`importSkill` 对 `shared`（实体在共享目录内）的技能刻意不搬家，只在库目录放一条指向它的软链接（`dirLink(via, skillDir(name))`）；随后 `syncSkills` 里的 `sharedHas()` 发现共享目录里的实体与资源库技能是同一路径，就把所有直接读取 `~/.agents/skills` 的 Agent（`targets.go` 的 `readsShared`，正是本报告所列 14 个：codex、gemini、opencode、deveco、crush、dsh、commandcode、devin、droid、cline、grok、grok-app、fx、alma）标为 `Always`。于是取消勾选（`SkillAgents` 清空 `agents`）只会移除资源库写入的链接，共享目录里的实体纹丝不动，“移除”也只拿掉库目录那条链接——勾选框成了假开关。

**修改**：`importSkill` 现在把共享目录里用户自己的技能文件夹像其他文件夹一样**移入资源库**（`move(f.real, skillDir(name))`），资源库持有实体；读共享目录的 Agent（如 Goose/Kimi，它们的技能目录就是 `~/.agents/skills`）由同步在其位置放/收资源库的链接，其余 `readsShared` Agent 勾选时在自己目录获得链接。之后取消全部勾选或“移除”，共享目录里的入口也会真正消失。两类保持原状：共享目录里**指向别处的链接**（如指向 `D:\aimer-skills\...`，只链接过去），以及 **skills CLI 锁文件（`~/.agents/.skill-lock.json`）登记的文件夹**（CLI 自管、自行更新，`TestPastedRepoBindsSameCopy` 固定此行为）。

**红→绿证据**：新增回归测试 `TestImportSharedSkillUnticksForItsReaders`（`internal/library/shared_skills_test.go`）——在干净基线 `19253f67` 上仅加测试文件运行即失败：库目录是指向共享目录的链接、codex 勾选后没有链接、`Always:[codex gemini opencode deveco goose crush dsh]`、取消勾选后共享目录入口仍在；修复后同一测试通过。两个固定旧行为的既有测试（`TestSharedAgentsSkills`、`TestImportSharedSkillForKimi`）按新预期更新。

**验证**（GOTOOLCHAIN=go1.26.8）：`go vet ./...`、`go build ./...`、`GOOS=linux GOARCH=amd64 go build -tags nogui ./...`、`GOOS=windows GOARCH=amd64 go build ./...` 全部通过；`gofmt -l internal/library/` 为空（基线上 `skillgit_test.go` 本就未格式化，本次仅 gofmt 顺手修正）；`go test ./internal/library/ -count=1` 全量 414 通过。

**没试/没做的**：没有跑真实 GUI 的资源库页面（修复在 `internal/library`，通过 GUI 调用的同一层包测试验证）；已按旧方式导入过的技能不会自动迁移——在资源库里“移除”后重新“导入”一次即可获得真开关；magpie 不认识的、直接扫 `~/.agents/skills` 的第三方程序不在 `readsShared` 名单里，导入搬走后它们不会再看到该技能。
