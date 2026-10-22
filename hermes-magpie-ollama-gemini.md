# Hermes Agent + Magpie + Google Gemini HTTP 400：Ollama 自动探测误判

调查日期：2026-10-22（Australia/Brisbane）  
状态：**Hermes 配置绕过已应用；Hermes/Magpie 上游源码修复及真实 Gemini 端到端回归待办。**

## 1. 现象和影响

- 客户端：Hermes Agent v0.22.4，本机源码 checkout `~/.hermes/hermes-agent`，调查时 HEAD `4d04303818b`。
- 客户端配置：`provider: custom`，`model: group/smart-split`，OpenAI-compatible `base_url: http://10.0.0.31:3425/v1`。
- 路由网关：Magpie，运行于 `docker1`，推理 API 端口 `3425`，Web UI 端口 `3430`；本地 checkout 为 `~/Projects/magpie`。部署镜像先前记录为 `ghcr.io/yetone/magpie:latest`，具体部署版本与本地 checkout 是否一致，未来改源码前须重新核对。
- 主要复现：Hermes → Magpie `group/smart-split` → Google Gemini，返回 HTTP 400。非 Google 上游可能容忍或忽略额外参数，因此经常正常。
- 两次用户日志时间：`2026-10-22T03:31:57Z` 和 `2026-10-22T04:08:32Z`（UTC）。

错误摘要：

```text
provider: custom
model: group/smart-split
code: format_error, retryable: false
HTTP 400: Google Gemini: 400 Bad Request
Invalid JSON payload received. Unknown name "options": Cannot find field.
```

Hermes 的后续 `custom rejected this request as malformed` 是上游 400 的传播结果；提示使用 `/new`、`/model` 或 `hermes doctor` 无法修正这个确定的请求字段兼容问题。

## 2. 证据链：根因已经定位

**A. 失败请求确实由 Hermes 携带 Ollama 参数。**

与 `2026-10-22T04:08:32Z` 对应的请求 dump：

```text
~/.hermes/sessions/request_dump_20261022_140831_93e7b1_20261022_140832_619970.json
```

从其中 `request.body` 提取的安全精简片段（原始 dump 还含 47 条 messages、38 个 tools，请勿向公开 issue/PR 原样上传）：

```json
{
  "model": "group/smart-split",
  "reasoning_effort": "high",
  "extra_body": {
    "options": {"num_ctx": 131072}
  }
}
```

在 OpenAI Python SDK 发往兼容 API 的请求中，`extra_body` 会并入实际 JSON 请求体；因此 `options` 会成为上游看见的未知字段，而非 Gemini 所支持的请求配置。

**B. Hermes 自动将 Magpie 当作 Ollama。**

`agent/agent_init_ollama.py::_configure_ollama_num_ctx` 在模型配置未提供 `ollama_num_ctx` 且 `base_url` 被归类为本地服务时，调用 `query_ollama_num_ctx`；`10.0.0.31` 属于私网地址。`agent/model_metadata.py::query_ollama_num_ctx` 通过 `detect_local_server_type` 探测类型，再请求服务器 `/api/show` 获取上下文窗口。

`agent/model_metadata.py::detect_local_server_type` 的 Ollama 检测条件是服务器 `GET /api/tags` 返回 HTTP 200 且 JSON 含有 `models`。Magpie **故意**兼容这套 API：

- 用配置中的 Magpie API Key 调用 `GET http://10.0.0.31:3425/api/tags`，实测 **HTTP 200**，返回 `models` 列表，调查时包含约 100 个条目和 `group/smart-split`。
- 带相同认证调用 `POST http://10.0.0.31:3425/api/show`，请求 `{"name":"group/smart-split"}`，实测 **HTTP 200**，返回 `model_info`，其中含 `magpie.context_length`。
- 不带网关 Key 请求 `/api/tags` 得到 HTTP 401，说明排查探测必须使用正确认证。
- Hermes 的探测缓存 `~/.hermes/cache/local_endpoint_probes.json` 曾存有键 `ollama_num_ctx:http://10.0.0.31:3425|group/smart-split`，其值为 **131072**。缓存只反映之前成功获取的值，并非主因；删缓存不能防止下一次再次误判。

Magpie 本地源码证据：

- `internal/gateway/ollama.go` 的 `ollamaTags` / `ollamaShow` 提供兼容端点。
- `internal/gateway/ollama_test.go::TestOllamaModelRoutes` 明确测试 `/api/version`、`/api/tags`、`/api/show` 以及模型上下文和能力信息。这是为要求 Ollama 风格端点的客户端提供兼容能力；**不应简单删除这些端点以迎合 Hermes 的探测**。

**C. Hermes 将探测结果无条件作用于任意 `custom` 请求。**

代码调用链（以上述 Hermes commit 为准）：

```text
agent/agent_init_ollama.py::_configure_ollama_num_ctx
  -> agent/model_metadata.py::query_ollama_num_ctx
  -> agent/model_metadata.py::detect_local_server_type
  -> Magpie GET /api/tags（被判定为 ollama）
  -> Magpie POST /api/show（取 context length = 131072）
  -> agent._ollama_num_ctx = 131072
  -> agent/chat_completion_helpers.py（传递 ollama_num_ctx）
  -> agent/transports/chat_completions.py::build_kwargs
  -> plugins/model-providers/custom/__init__.py::CustomProfile.build_api_kwargs_extras
  -> extra_body["options"] = {"num_ctx": 131072}
  -> Magpie group/smart-split
  -> Gemini 上游拒绝 options，HTTP 400
```

`CustomProfile.build_api_kwargs_extras` 目前仅判断 `if ollama_num_ctx:`，未再验证目的端是否是纯 Ollama。这让一个支持 Ollama 兼容探测的**多模型路由网关**产生全局性的 Ollama 专属注入。

**D. Gemini 严格拒绝未知字段是合理行为。**

`options.num_ctx` 是 Ollama 请求配置，不是 Google Gemini 的请求字段。Magpie 在 Google 路由上未有效清理这一字段，最终 Gemini 报 `Unknown name "options"`。其他模型工作正常不代表注入行为正确。

注意：调查时另有 JEV 分类器限流/启停情况，但 **JEV 并不是本次 HTTP 400 的根因**。路由决策质量和上游请求 JSON 兼容性应分别排查。

## 3. 已应用的即时绕过（2026-10-22）

只修改 `~/.hermes/config.yaml` 的顶层 `model` 块，增加：

```yaml
model:
  default: group/smart-split
  provider: custom
  base_url: http://10.0.0.31:3425/v1
  api_key: ${HERMES_CUSTOM_10_0_0_31_3425_API_KEY}
  ollama_num_ctx: 0
```

关键是 `ollama_num_ctx: 0`。现有 Hermes 的 `_configure_ollama_num_ctx` 把明确给出的 `0` 当作覆盖值，不再发起 Ollama 上下文探测；`CustomProfile.build_api_kwargs_extras` 中 `if ollama_num_ctx:` 为 false，于是不再添加 `options.num_ctx`。

**这个 0 表示禁用 Hermes 的 Ollama 专用请求参数，不是将模型上下文窗口设置为 0。** Magpie 的模型上下文信息和自身路由规则仍照常工作。

修复前后的代码级自检此前已做过：

```text
ollama_num_ctx = 131072 => extra_body.options = {num_ctx: 131072}
ollama_num_ctx = 0      => extra_body.options 不存在
```

需用**新 Hermes 进程/会话**加载更新后的配置。若存在当前还在运行的旧 Agent，旧进程的内存参数不会因磁盘配置变动自动刷新。是否完整端到端验证、验证时间及证据，见本文件末尾状态记录。

## 4. 等有时间再做的两个上游 PR

### 4.1 Hermes PR：修正 Ollama 特征探测与参数注入的范围

目标：对多模型网关或标准 OpenAI-compatible 服务，不应仅因为存在 `/api/tags`、`/api/show` 就给所有模型请求强加 Ollama 的 `options.num_ctx`。保留真实 Ollama 服务可自动设置上下文窗口的能力。

建议从上述 `_configure_ollama_num_ctx`、`query_ollama_num_ctx`、`detect_local_server_type` 与 `CustomProfile.build_api_kwargs_extras` 一起追踪设计，再选**一个明确的最小修复点**。检查有没有可用的 endpoint/provider 能力信息来区分“实际 Ollama 服务”和“Ollama 协议兼容网关”。不要简单以 RFC1918、`custom` 或服务提供 `/api/tags` 作为充分条件，也不要强制只允许端口 11434，因为远程 Ollama/反向代理可能使用其他端口。

建议的回归用例：

1. OpenAI-compatible Magpie（同时提供 `/api/tags` 和 `/api/show`）+ 路由模型 `group/smart-split`：无 `options.num_ctx`。
2. 真实 Ollama（本地或远程非默认端口）+ 真正的 Ollama 模型：仍按需发送正确的 `options.num_ctx`。
3. 用户显式指定 `model.ollama_num_ctx: 0`：不探测、不注入参数。
4. Hermes 配置的 `custom` 端点未标明 Ollama 时：不得因可选兼容 API 造成不必要的下游字段泄漏。

可参考的既有测试：`tests/agent/test_ollama_num_ctx.py`、`tests/agent/test_local_probe_disk_cache.py`、`tests/agent/transports/test_chat_completions.py`、`tests/providers/test_transport_parity.py`。

### 4.2 Magpie PR：路由选定后净化目标协议不支持的字段

目标：OpenAI-compatible 客户端经 Magpie 转给 Gemini 时，过滤掉 `options` 这类 Ollama 专属**顶层生成参数**，避免 Google REST / Gemini 翻译路径接收未知字段。过滤必须在确认目标 provider/protocol 后执行，并且不得破坏真正 Ollama 上游收到其 `options` 的能力。

未来先沿 `internal/gateway` 的 Chat Completions 入站 → 路由解析 → provider/协议适配 → Google `generateContent` 请求构造链，定位字段第一次错误留存的位置；在共享协议转换边界做最小修复，而不是针对 `group/smart-split` 硬编码条件。

建议的回归用例：

1. `POST /v1/chat/completions` 带顶层 `"options":{"num_ctx":131072}`，路由到 Gemini 的模拟上游：该参数不进入 Google JSON；请求正常完成。
2. 同一入口路由到 Ollama 风格上游：真实合法的 `options` 按现有设计保留。
3. 普通不带 `options` 请求：输出不变。
4. 如果智能分组发生模型回退/切换，使用最终选中上游的协议清理规则，而不是入口 model/group 名称猜测。

本 PR 仅解决真实出现的字段兼容性问题，避免顺手开发新的转换框架、配置开关或全局特判。

## 5. 复现与验收指引

1. **修复前证据**：使用已有 request dump 证明 Hermes `extra_body.options.num_ctx` 存在；不用公开原始请求 dump，其中可能包含消息、工具数据和认证头。
2. **探测复现**：带网关 Key 分别访问 Magpie `/api/tags` 和 `/api/show`，确认可被误判为 Ollama；不要在日志、PR 或命令输出中泄露 Key。
3. **配置修复验证**：新 Hermes 进程读取 YAML，`model.ollama_num_ctx == 0`；检查新请求 dump 中没有 `extra_body.options`。
4. **路由回归**：让 `group/smart-split` 真正落到 Gemini，期望不再返回 `Unknown name "options"`；保留成功/失败状态与上游模型 ID 的脱敏证据。
5. **其他上游回归**：测试一个此前可用的非 Gemini 模型，确保 Hermes 的基本消息/工具调用能力没受影响。
6. 如果仍有 HTTP 400，保留新请求时间、脱敏 JSON 与路由结果，按新错误内容单独判断，不预设为同一个原因。

## 6. 状态与范围

- 已确认：**Hermes 自动误识别 Magpie 的 Ollama API，注入 `options.num_ctx=131072`；该参数与 Gemini HTTP 400 错误精确吻合**。
- 当前只做 Hermes YAML 配置绕过和本调查文档；**未修改 Hermes 或 Magpie 源码，未提交 PR，未向上游推送**。
- 注意：本地 Magpie checkout 与 `docker1` 运行镜像是否为同一版本，尚未核实。将来做 Magpie PR 前必须在当前源代码重新走一遍执行路径。
- 注意：历史日志证明了错误请求，但单靠配置/代码级检查不能代表真实 Gemini 全链路成功；端到端状态应以新的路由日志和实际回复为准。

### 本轮配置验证结果（2026-10-22）

实际从 `~/.hermes/config.yaml` 读取并用 PyYAML 解析后，进行 Hermes 源码级自检，全部通过：

```text
PASS: YAML parsed; model settings intact; autodetection skipped;
      no extra_body.options; reasoning_effort maintained
```

验证断言：`model.provider == custom`、`model.default == group/smart-split`、`model.base_url` 不变、`model.ollama_num_ctx == 0`；将配置传给 `_configure_ollama_num_ctx`，用会在探测时抛异常的 stub 确保**没有发出自动探测**，验证 `agent._ollama_num_ctx == 0`；再调用真实 `CustomProfile.build_api_kwargs_extras`，验证不含 `options`，而 `reasoning_effort` 仍是 `high`。`config.yaml` 权限保持 `0600`。

**未重启用户正在使用的 Hermes 会话，未执行真实 Gemini API 回归**；当前结论是已落地配置修复并验证实际请求构造路径。重新启动 Hermes Agent / 新进程后生效。未来验收请记录至少一次 Gemini 路由成功的脱敏日志。
