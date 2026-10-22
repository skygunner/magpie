<!-- 2026-10-22：从 smart_data C5 Round 2 的一次 8h46m 任务复盘得出 -->
# Claude Code 会话效率复盘（2026-10-22）

一次 8 小时 46 分的 C5 Round 2 收口任务里，真正做事约 4.5 小时。其余时间是可以避免的。
这份记录给 magpie 的 Claude Code 接入方看：哪些是客户端问题、哪些是网关问题、哪些是操作习惯。

## 结论速览

| 耗时 | 原因 | 归属 |
|---|---|---|
| 3h35m | 一个 Bash 调用**秒失败**，但结果 3.5 小时后才被投递 | Claude Code 客户端 |
| 1h07m | 用户不在，任务已完成等待验收 | 正常 |
| 24m | 一次读 60 行的 Read 调用 24 分钟才返回 | 客户端/网关卡死 |
| ~4.3h | 每回合 250–297 秒，8 次上下文压缩，1M 上下文 | 固有开销 |
| ~1h | 容器 4 小时到期、宿主机争用、工具误用的返工 | 操作习惯 |

## 一、客户端把失败结果攥了 3.5 小时

```
use@ 22:49:57  →  result@ 02:25:01
result: Exit code 1  /bin/bash: line 1: /tmp/pool.patch: No such file or directory
```

命令**在 22:49:57 立刻就失败了**（`git diff > /tmp/pool.patch && ssh ...` 里 ssh 段被拒，
`&&` 短路，补丁文件压根没生成）。但 Claude Code 进程把这个失败结果攥在手里攥了 3.5 小时
才吐出来，期间 transcript 零记录。

排除了机器休眠：journal 无 suspend 记录，`last` 显示 22:22 开机后连续运行到 18:48 未重启。
所以是**客户端事件循环卡死**。

**同一条路径上有两个可疑因素**：

- `PreToolUse` 的 `rtk hook claude` **没有设 timeout**（`~/.claude/settings.json`）。
  其余 hook 都有（5s / 30s / 60s），只有它没有。一个卡住的 hook 就能拖住整个事件循环。
  **已修：给 `rtk hook claude` 加 `timeout: 10`。**
- magpie 网关的长轮询。客户端持有一次未完成的请求时不会继续消费。

**给接入方的通用建议**：PreToolUse/PostToolUse hook 一律设 timeout，别留无上限的。
Claude Code 的 hook 超时后会跳过该 hook，而不是卡住工具调用。

## 二、auto-mode 分类器：36 次后端超时 + 29 次策略拒绝

本次会话 1433 次工具调用 / 140 次错误，其中 **65 次（46%）与模型无关**，全部是 auto-mode
（去重后按 `tool_use_id` 统计，`is_error` 与文案双重确认）：

| 拒绝原因 | 次数 | 性质 |
|---|---|---|
| 分类器后端超时（`group/general is temporarily unavailable`） | 36 | 基础设施 |
| 策略拒绝 Data Exfiltration | 18 | **全部是误判**（见下） |
| 策略拒绝 Credential Materialization | 5 | 多数正确 |
| Permission Grant / Self-Modification | 4 | 正确 |
| Credential Leakage | 1 | 正确 |
| 无 Reason 标签 | 1 | — |

### 后端超时的机制

magpie 的 `fitAutoModeClassifier`（`internal/gateway/automode.go:128`）把分类器请求
路由到**非 Claude 模型**上推理。`ANTHROPIC_DEFAULT_SONNET_MODEL=group/general`，
`group/general` 背后是 `deepseek-v4.1-flash-sg`。**这个 provider 在 02:58–03:03 集中不可用**，
24 次超时报错全落在那一段。

**这是 magpie 侧的可观测性缺口**：分类器后端挂了，用户看到的只是 Claude Code 状态栏一行
"bash denied by auto mode"，看不出是后端挂了还是策略拒绝。

**可改进**：magpie 在 `fitAutoModeClassifier` 选定的 provider 不可用时，应降级到网关内
另一个可用成员，或让客户端能区分"后端不可用"和"策略拒绝"两种文案。

### 策略拒绝的真实原因

**18 次 Data Exfiltration 全部是误判**，且 18 条命令**全部**指向 `docker1`：

```
tar -cf - scripts/... | ssh docker1 "docker exec -i ... tar -xf - -C /app"
ssh docker1 'cat > /home/hermes/reads.go' < controller-go/internal/controller/reads.go
scp -q /tmp/m1-controller docker1:/home/hermes/m1-controller
for f in ...; do ssh docker1 "cat > ..." < $f; done
```

分类器把"仓库源码 → 外部 SSH 主机"判成外泄。但 `docker1` 是**项目 AGENTS.md 指定的
开发/测试宿主机**，是这台机器自己的另一台主机，不是第三方外传目标。分类器不知道这一点。

**关键事实：`~/.claude/settings.json` / `.claude/settings.local.json` 的 `permissions.allow`
一旦命中，auto-mode 根本不参与判定。** 本次会话的证据：

- 2026-10-21T19:03:08Z 读到的 allowlist **没有** `Bash(ssh docker1 *)`；
- 2026-10-22T09:00:15Z 再读时它已经在了；
- **18 次 Data Exfiltration 全部落在 2026-10-21T19:02:34Z ~ 2026-10-22T04:19:50Z 这个窗口内**，
  规则出现之后再没有传输类拒绝；
- 规则出现后（09:19–09:20Z）实测各种写法都直接放行：复合命令、`for` 循环、
  `tar | ssh` 管道、`git archive | ssh`、直接 `ssh docker1 'cat > f' < f`，全部 OK。

**所以 `permissions.allow` 是最有效的解法，不要去放宽分类器。** 但注意它只解决
Data Exfiltration 这一类：凭据类和 self-modification 类拒绝照旧会触发，
因为它们是按命令内容判的，不是按目标主机判的。

### 凭据类拒绝：5 次里多数是正确且可避免的

分类器抓的是"把密码/控制 token 打印进 transcript"。规避方式不是绕，是**别打印**：

- fixture 密码由脚本从源码读取后直接使用（`scripts/migration/c5r2_run.sh` 已如此），
  不 `grep`/`cat`/`base64` 出来；
- 控制 token 校验存在性用 `test -s`，不 `cat ... | head -c 6`。

**这两条都写进了 smart_data 的 AGENTS.md。**

## 三、工具误用导致的返工

### `docker exec -i ... cat > file` 会留下僵死进程

```
docker exec -i <container> sh -c "cat > /path" < localfile
```

在这个环境下会留下烧 CPU 的僵死 `cat`（实测 3 个进程共 110% CPU），并曾把
`worker.go` 截断成 0 字节。**tar 一跳是唯一安全写法**：

```
tar -cf - ... | ssh docker1 "docker exec -i <container> tar -xf - -C /app"
```

tar 自结束、不留残留。**smart_data 的 `scripts/migration/c5r2_sync.sh` 本来就是这么写的**，
本次会话中途漂移开、自己手拼 `cat >` 管道，才反复踩坑。**仓库里已有的一跳封装必须用。**

### 容器 PID-1 自带 TTL = 定时炸弹

`sleep 14400`（4 小时）作为 PID-1，容器到期退出，tmpfs `/tmp` 一起清空：已构建的二进制、
matrix 输出、浏览器 world 全丢，40 分钟重跑。

**已修**：`sleep infinity` + `--restart unless-stopped`；持久产物移出 tmpfs 到
`/app/c5r2-work/`。回收时必须 `docker rm -f`——`docker stop` 会被自动拉起。

### 共享宿主机负载污染性能读数

docker1 上其他租户的容器把 loadavg 顶到 3–5，此时 C1 延迟读数 220–240ms（超 200ms 预算）
是噪声。真实值 162ms。**读数异常时先做多二进制交错 A/B 证明不是自己改坏的**，
并先查宿主机 loadavg。**已修**：`c5r2_run.sh` 内置 load 等待与告警（load > 2 等待或警告）。

## 四、固有开销（无法消除，只能规划）

- 1M 上下文 + 8 次 compaction，单回合 250–297 秒。**长任务应该分段**，
  每完成一个 P1 就落盘证据并提交，避免一个会话扛完所有 P1。
- 25MB transcript 的会话，任何"回头查历史"都要靠脚本解析 jsonl，而不是靠记忆。

## 五、`API Error: 400 OpenRouter: Invalid Anthropic Messages API request` 溯源

2026-10-22 09:17:22Z 与 09:26:03Z 两次，回合被这条 400 直接打断。查 magpie 的
`/config/magpie/usage.jsonl`（docker1 上的 magpie 容器）得到确证：

```
provider=openrouter  model=minimax/minimax-m3.1-flash:free  req=group/smart-split
ep=/v1/messages  ← 没有 "→ /v1/chat/completions"，即原生 Anthropic 协议直发
status=400  err="OpenRouter: Invalid Anthropic Messages API request"  err_type=invalid_request_error
tries=2
```

四条结论：

1. **报错来自 OpenRouter，不是 magpie**。magpie 源码里没有这句话；`provider.APIError`
   把上游 body 原样带出，`p.Explain` 只加了 `OpenRouter: ` 前缀。`ep` 字段证明这次
   请求是**按原生 `/v1/messages` 发出去的**，不是翻译成 chat。
2. **没有回退是 magpie 的判定**。`retryable(400, body)` 对这句文案全不匹配
   （`shapeWords` / `quotaWords` / `unservedWords` / `regionWords` / `wrongEndpoint` 都不命中），
   于是 group 里另一个成员（kilo）没被问，400 直接回给客户端，Claude Code 显示
   `API Error: 400 ...` 并结束回合。
3. **不是工具列表、不是上下文、不是鉴权**。用同一份 56 个工具的完整 shape、156KB body
   对两个成员各重放一次，全部 200。把 tool schema 里 `allOf`/`const`/`propertyNames`
   单独抽出来测，也都 200。所以不是 mid-conversation tool change 引起的。
4. **规避已经在生效**。`/config/magpie/settings.json` 里有
   `modelAPIs: {"openrouter/minimax/minimax-m3.1-flash:free": "chat"}`，把这个模型钉在
   chat 协议上。usage 日志里 09:47 之后该 provider 的每一条都是
   `ep=/v1/messages → /v1/chat/completions` 且 status 200。**只要这个 pin 在，就不会再走
   原生 /v1/messages**。

**给 magpie 的建议**：`retryable()` 应把"上游 400 且文案在说 API 协议不对"
（`Invalid .* API request`）算作 shape 类拒绝——它和 `shapeWords` 是同一类事，
都是"这个成员这条路上读不了"，换成员可能就好了。现在这种情况会让整个 group 失败。

## 六、本轮已落地的修复与交付现状（2026-10-22）

复盘不是纸面文章。8h46m 那个会话结束前，五项效率修复已经全部落地并复验：

| # | 修复 | 落点 | 效果 |
|---|---|---|---|
| 1 | 沙箱容器 PID-1 改 `sleep infinity` + `--restart unless-stopped`，持久产物移出 tmpfs | docker1 `smart-data-c5r2-sandbox` | 容器不再 4h 到期自毁，产物在 `/app/c5r2-work/` |
| 2 | 测量前读 docker1 loadavg，> 2.0 等待或告警 | `scripts/migration/c5r2_run.sh` | 共享宿主机争用不再污染延迟读数 |
| 3 | 构建产物不再覆盖发布二进制 | `scripts/migration/c5r2_sync.sh` → `/app/c5r2-work/m1-controller.dev` | 覆盖发布二进制会让已有测量的证据悄悄失效 |
| 4 | `rtk hook claude` 加 `timeout: 10` | `~/.claude/settings.json` | 它是唯一没设 timeout 的 hook，卡住能拖住事件循环 |
| 5 | 「docker1 开发效率纪律」章节 | smart_data `AGENTS.md` | 一跳封装、查负载、容器无 TTL、凭据不落 transcript |

同时加进 `AGENTS.md` 的四条纪律，是本次误用工具链的直接教训：
只用仓库现成的一跳封装（`c5r2_sync.sh` / `c5r2_run.sh`），不手拼
`ssh docker1 'cat > f' < f`（`docker exec -i` 下留僵死 `cat`，实测 3 进程吃 110% CPU，
还把 `worker.go` 截断成 0 字节）；测量前先查 loadavg；沙箱 PID-1 不得自带 TTL；
凭据用 `test -s` 查存在性而不是打印。

**交付现状**：smart_data 分支 `dev-go`，提交 `a7e7527`（移除 C5R2 沙箱里的返工源码）、
`9cc973a`（保持被接受的 manifest 仍可验证并补齐 receipt 哈希）。
交付代码 `c83b7639`、发布二进制 sha256 `8d887f98…` 均未改动。
**没有 push、没有 merge、没有部署生产、没有进入 C6。**

### 遗留与未知

- 上游具体嫌 `/v1/messages` body 里的哪一处，**没能复现**。所有忠实重放都返回 200。
  所以本文只写已证实的事实（原生 `/v1/messages`、`invalid_request_error`、`tries=2`、
  无回退、chat pin 规避生效），不猜根因。
- `modelAPIs` 的 pin 是**网关侧配置**，在 `settings.json` 里，不在 git 里。
  它会因配置漂移或重装而丢失。丢了就会重新走原生 `/v1/messages`，400 可能复现。
  建议 magpie 侧加个「400 + `Invalid .* API request` → 按 shape 类拒绝回退」的兜底，
  而不是只靠一个 pin。
- 分类器后端超时（36 次）与策略拒绝（29 次）在客户端是同一行文案，
  要区分只能看 magpie 侧日志。这属于网关可观测性缺口，客户端无法自行修复。

## 给 magpie / Claude Code 接入方的具体建议

1. **PreToolUse hook 必须设 timeout**。本次 `rtk hook claude` 无 timeout，
   疑似 3.5 小时事件循环卡死的共犯。（已在 `~/.claude/settings.json` 加 `timeout: 10`。）
2. **分类器后端不可用要可观测**。magpie 把分类器路由到非 Claude provider；
   该 provider 挂掉时用户只看到 "denied by auto mode"，分不清后端故障和策略拒绝。
   建议 magpie 侧加 fallback provider，并在响应里区分两类失败。
3. **`permissions.allow` 命中即绕过 auto-mode**——这是最有效的解法。
   项目自有开发/测试主机（如 docker1）应在会话开始前就进 allowlist，
   而不是每次靠分类器"判断这不是外泄"。规则在场之后，复合命令、`for` 循环、
   `tar | ssh`、`git archive | ssh` 都直接放行（2026-10-22 09:19Z 实测）。
   它只管传输类：凭据类和 self-modification 类仍按命令内容拒绝。
   传输本身仍应走仓库里已有的一跳封装（`c5r2_sync.sh`），那是为了不留僵死进程，
   不是为了躲分类器。
4. **长任务分段**：一个 P1 一个会话或一个提交点。
