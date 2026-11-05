# Magpie fork 维护入口

此页只说明 fork 的组织方式。**当前生产版本、补丁去留和发布门以
[PATCHES.md](../../PATCHES.md) 为准**，不要从旧目录名、旧 `prod/*` 标签
或某个修复分支推断生产已经部署。Docker 镜像入口是
[bigshort/magpie-patched](https://hub.docker.com/r/bigshort/magpie-patched)，
已验收的源码用 `production/stable` 和 `deployed/*` 核对。

## 本机只保留两个固定入口

| 目录 | 用途 |
| --- | --- |
| `~/Projects/magpie` | 主 Git 仓库和正在使用的开发工作台；其他 worktree 共用这里的 `.git`。主目录可能在 `dev`，不是当前生产源码的证明。 |
| `~/Projects/magpie-integration` | `integration/nightly` 的固定工作树；维护当前 fork 文档、审查上游变更和准备下一轮集成。 |
| `~/.cache/magpie-worktrees/<name>` | 临时修复、干净基线、候选验证工作树。任务完成后删除，不在 `~/Projects` 继续堆放版本/issue 目录。 |
| `~/.local/state/magpie-rounds/<version>` | 私有测试、构建和部署原始记录，不是源码 checkout。 |
| `~/.local/share/magpie/archive/2026-11-05-cleanup` | 本轮清理前的 Git bundle、未提交补丁和散落材料；恢复步骤见其中的 `README.md`。 |

主目录的 `.codebase-memory/` 和 `.omo/` 属于本机工具状态，不是待集成补丁。
它们保留在本机，并由本地 Git exclude 排除，不提交到 fork。
插件主仓库仍在 `~/workspace/projects/magpie-community-plugins`；Cloudflare
Workers AI 的修复和发布在那里维护，不需要在 Projects 留一个永久插件副本。

## 分支不是目录

| 引用 | 含义 |
| --- | --- |
| `main` / `upstream/*` | 跟踪上游；不在这里叠加 fork 的运行规则。 |
| `dev` | 旧 issue 研究、修复批次和开发工作台；不能用它判断当前生产状态。 |
| `integration/nightly` | 长期集成线：合并一个固定的上游版本，逐项审查仍需保留的差异。 |
| `fix/*`、`feature/*` | 独立补丁来源。分支可以保留，完成任务后无需继续占用一个工作目录。 |
| `production/stable`、`deployed/<version>` | 只指向已通过验收和观察的生产源码。文档整理可以让集成线前进，不能据此移动生产引用。 |
| `candidate/<version>` | 经过源码门的候选；不是生产发布证明。 |
| 旧 `prod` / `prod/*` | 历史集成引用，其中包含失败候选。只供追溯。 |

清理本地分支时，只删除已完整发布到同名远端、或已被保留引用完整包含的
闲置本地分支。保留当前补丁来源和未发布的独有提交。远端 `fix/*` 仍可能被
上游 issue 回帖引用，不能因为删除工作目录就顺手删除远端分支。

## 临时 worktree 的最小流程

```sh
# 从固定 release 或已审查的集成提交开始；不要把版本号目录留在 Projects。
mkdir -p ~/.cache/magpie-worktrees
git -C ~/Projects/magpie worktree add \
  -b fix/example ~/.cache/magpie-worktrees/example <reviewed-base>

# 在临时工作树里完成修改、验证和提交，然后保存来源分支。
git -C ~/.cache/magpie-worktrees/example push -u origin fix/example

# 先确认没有未保存文件，再删除工作树；来源分支和提交仍在 Git 中。
git -C ~/.cache/magpie-worktrees/example status --short
git -C ~/Projects/magpie worktree remove ~/.cache/magpie-worktrees/example
```

临时目录有未提交内容时，先提交有意义的工作，或保存基线、暂存/未暂存补丁
和新增文件并验证能恢复。不要用 `git clean -fdx` 清理共享主目录。
不要移动或删除其他会话正在使用的工作树。

## 文档放在哪里

| 内容 | 唯一入口 |
| --- | --- |
| 当前生产事实、保留/暂停/已吸收补丁、部署门 | [PATCHES.md](../../PATCHES.md) |
| 历次部署、失败回滚、旧补丁测试和历史操作路径 | [history.md](history.md) |
| 已提交的可审阅验证结果 | [evidence/](evidence/)；目录按日期/版本分组，失败候选也保留身份 |
| 已暂停冷却方案的历史设计 | [archive/cooldown-settings.md](archive/cooldown-settings.md) |
| 当前运行子系统的责任、接口和失败行为 | [../subsystems/](../subsystems/README.md) |
| 早期 issue campaign / round 研究 | `dev` 历史中的 `docs/fork`；例如 [2026-10 工作台快照](https://github.com/skygunner/magpie/tree/46eeee5d7/docs/fork)，不是当前生产说明 |

不要把新一轮长日志继续追加到根清单；将证据放在对应日期/版本目录，并从
历史记录链接。私有请求、凭据、生产配置和本地归档不能推送到公开仓库。

## 每轮集成与发布

1. 在固定入口读取最新清单和子系统说明，固定一个上游 release/tag。
2. 按行为审查每项补丁：上游已吸收就移除；语义重复或冲突就暂停，不能只靠
   cherry-pick 是否冲突判断。必要的新修复单独保存在 `fix/*`。
3. 验证最终源码；有已知基线失败时保留原始结果、对照、归属和发布判断，
   不把扩展测试写成“全绿”。源码集成和候选镜像都不代表已部署。
4. 按清单中的部署门保存实际快照，运行 detached 部署事务、真实推理和
   46 分钟观察。只发布验收过的同一个镜像，不在晋级时重新构建。
5. 验收成功后推进生产引用，记录镜像摘要与证据；历史记录移入 history，
   根清单只保留当前事实。清理已完成的临时工作树。

本轮目录/文档整理不重建镜像、不重启服务、不改变现有自动任务的启停状态。
