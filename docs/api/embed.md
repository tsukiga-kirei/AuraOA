# OA 嵌入接口

## 概述

供固定地址嵌入页 `/embed/audit`、`/embed/summary` 与无界面的 `/embed/runner` 使用。**浏览器不直接调用 Go 接口**，而是请求 Nuxt 代理 `/api/embed/*`。

鉴权链路：

1. OA 父页面通过 `postMessage` 将 `embed_token` 传给 iframe
2. 嵌入页把令牌保存在当前 iframe 内存，并调用 `POST /api/embed/session` 尝试写入 httpOnly Cookie
3. 浏览器请求通过同源 `X-Embed-Token` 请求头传递令牌；Cookie 作为同站场景补充
4. Nuxt 代理携带 `X-Embed-Token` 访问 Go
5. Go `EmbedAccess` 中间件根据令牌哈希反查租户，并注入 `tenant_id`

Nuxt 服务端通过私有运行时配置 `NUXT_INTERNAL_API_BASE` 访问 Go。Docker Compose
默认使用 `http://go-service:8080`；该地址不暴露给浏览器，也不应使用
`localhost:8080`。浏览器公开地址仍由 `AURAOA_PUBLIC_API_BASE` 控制，同源部署时
保持为空。

Go 路由前缀：`/api/embed`  
鉴权：`EmbedAccess` 中间件（**非 JWT**）

---

## 租户级嵌入配置

在 **系统管理 → 租户管理 → OA 嵌入** 中：

| 配置项 | 说明 |
|--------|------|
| `embed_enabled` | 是否允许该租户使用嵌入能力 |
| `embed_access_token` | 每租户独立密钥，生成/重置后仅展示一次明文 |
| `tenant_code` / `tenant_id` | 展示给实施与排查使用 |
| 嵌入地址 | `/embed/audit`、`/embed/summary` |

管理端 API：

```
POST /api/admin/tenants/:id/embed-token
```

用于生成或重置租户嵌入密钥。

---

## Nuxt 代理（浏览器调用）

| 方法 | 路径 | 转发至 Go |
|------|------|-----------|
| POST | `/api/embed/session` | 写入 httpOnly Cookie（不转发 Go） |
| POST | `/api/embed/events` | `POST /api/embed/events` |
| GET | `/api/embed/context?process_id=` | `GET /api/embed/context` |
| POST | `/api/embed/execute` | `POST /api/embed/execute` |
| GET | `/api/embed/jobs/:id` | `GET /api/embed/jobs/:id` |
| GET | `/api/embed/stream/:id` | `GET /api/embed/stream/:id` |
| GET | `/api/embed/summary/context?process_id=` | `GET /api/embed/summary/context` |
| POST | `/api/embed/summary/execute` | `POST /api/embed/summary/execute` |
| GET | `/api/embed/summary/jobs/:id` | `GET /api/embed/summary/jobs/:id` |
| GET | `/api/embed/summary/stream/:id` | `GET /api/embed/summary/stream/:id` |

---

## Go 接口

### 安排后台刷新检查

```
POST /api/embed/events
```

```json
{
  "process_id": "598488",
  "action": "save_or_submit",
  "event_id": "oa-598488-1722150000000"
}
```

新脚本只发送 `save_or_submit`。后端仍兼容旧脚本的 `save`、`submit`，并统一归一化为
`save_or_submit`；旧版 `page_open` 会正常确认但不再创建后台任务。接口只把审核和总结检查写入
Redis 延迟队列并立即返回 `202`，不会等待 OA 查询或 AI 执行。

保存/提交事件首次在约 2 秒后读取 OA；未发现 OA 数据落库时会继续在约 5 秒、10 秒检查。
同一租户、流程和模块的连续事件自动合并。只有保存/提交事件会在任务执行期间保留一次后续检查；
定时扫描不会持续追踪进行中的任务。

### 获取嵌入上下文

```
GET /api/embed/context?process_id=598488
```

请求头：

```
X-Embed-Token: <tenant embed access token>
```

返回中的 `stale` / `should_auto_audit` 已按流程配置过滤。变化来源分为：

- 审核实际使用的主表与明细字段；
- 主表附件字段的 `docId` 及 `DocImageFile` 最新版本；
- 退回或退回后的重新提交；
- 普通审批日志、批注、转发、抄送与当前节点变化。

普通审批流变化是否触发由 `auto_audit_on_flow_change` 单独控制，默认关闭。
`trigger_source=embed_auto` 时后端会再次校验 `should_auto_audit`，无需刷新时不会创建审核任务。
若最近一次相同依赖指纹（流程数据、附件版本、流程信息、审核规则和模型配置）的审核已经失败，
响应中 `auto_retry_blocked=true`，保存/提交和定时来源不会重复执行；失败记录直接作为结果展示，
只有用户点击“重新审核”才会再次尝试。

### 触发嵌入审核

```
POST /api/embed/execute
```

```json
{
  "process_id": "598488",
  "trigger_source": "embed_auto",
  "trigger_detail": "visible_open"
}
```

`trigger_detail` 的可见页取值为 `visible_open`，手动按钮为 `manual`。后台内部使用
`save_or_submit`、`scheduled_scan` 区分保存提交与定时扫描。

### 查询任务状态

```
GET /api/embed/jobs/:id
```

与审核工作台 `GET /api/audit/jobs/:id` 响应结构相同。

### SSE 审核推理流式输出

```
GET /api/embed/stream/:id
```

与 `GET /api/audit/stream/:id` 相同，返回 `text/event-stream`。

---

## 流程总结接口

### 获取总结嵌入上下文

```
GET /api/embed/summary/context?process_id=598488&prefer_cached=true
```

除通用字段外，响应可包含 `stale_block_ids`，列出需要重新生成的总结块 ID。判断以每个块的
`field_mode`、`selected_fields` 和数据变量为准：

- `{{main_table}}` / `{{detail_tables}}`：只比较该块使用的业务字段；
- `{{attachments}}`：只比较该块选中的主表附件字段版本；
- `{{flow_history}}` / `{{flow_graph}}`：审批日志或流程图变化时由流程刷新策略控制；
- `{{process_meta}}`：当前节点或流程基础信息变化时由相应策略控制。

自动总结只重新调用 `stale_block_ids` 中的块并合并旧结果；手动总结执行全部启用块。
`trigger_source=summary_embed_auto` 时后端会再次校验 `should_auto_summary`，无需刷新时不会创建总结任务。
可见页传 `prefer_cached=true` 时，已有成功结果会直接返回，不等待 OA 变化扫描；保存/提交和定时
协调器仍使用完整上下文检查。
若最近一次相同依赖指纹（流程数据、附件版本、流程信息和总结块提示词配置）的任务已经失败，
响应中 `auto_retry_blocked=true`，自动来源不会重复执行，失败记录直接作为可展示结果返回。

可见总结页已有成功结果时直接展示，不因打开页面重新执行；没有结果时以 `visible_open` 进入交互队列。
手动“重新总结”使用最高优先级。保存/提交进入后台队列，流程级定时扫描优先级最低。

### 触发总结

```
POST /api/embed/summary/execute
```

```json
{
  "process_id": "598488",
  "trigger_source": "summary_embed_auto",
  "trigger_detail": "visible_open"
}
```

`trigger_detail` 的可见页取值为 `visible_open`，手动按钮为 `manual`。后台内部还会记录
`save_or_submit`、`scheduled_scan`，用于查询任务的真实来源。

### 查询任务状态

```
GET /api/embed/summary/jobs/:id
```

### SSE 原始模型输出

```
GET /api/embed/summary/stream/:id
```

---

## 流程如何命中规则配置

嵌入请求不会直接用 `process_id` 查规则，而是：

1. `X-Embed-Token` → 反查 `tenant_id`
2. `process_id` → 查询 OA 流程摘要，得到 `process_type`
3. `tenant_id + process_type` → 查询 `process_audit_configs` 或 `process_summary_configs`
4. 校验配置 `status=active` 且 `embed_enabled=true`

因此：

- **租户归属** 由嵌入密钥决定
- **规则归属** 由流程类型决定
- **是否允许嵌入** 由流程级 `embed_enabled` 决定

---

## postMessage 协议（OA 自定义 JS ↔ 嵌入页）

| 方向 | `type` | 载荷 |
|------|--------|------|
| iframe → OA | `aura-oa-request-requestid` | 无 |
| OA → iframe | `aura-oa-requestid` | `{ requestid: string, embed_token: string }` |
| runner → OA | `aura-runner-ready` | 无 |
| OA → runner | `aura-oa-refresh-event` | `{ requestid: string, action: string, event_id: string }` |
| runner → OA | `aura-runner-event-ack` | `{ event_id: string }` |

OA 保存/提交检查最多等待 150ms 的事件接收确认；收到确认会立即放行，超时或 AuraOA
不可用也会放行。runner 使用 `keepalive` 提交事件，确认或超时都不代表等待 AI 执行完成。

OA 示例脚本：[../oa-configurations/assets/aura-embed-notify.js](../oa-configurations/assets/aura-embed-notify.js)

---

## 配置说明

| 配置项 | 位置 |
|--------|------|
| 租户 `embed_enabled` | 系统管理 → 租户管理 → OA 嵌入 |
| 租户 `embed_access_token` | 系统管理 → 租户管理 → OA 嵌入 |
| 审核 `embed_enabled` | `process_audit_configs` / 租户规则 UI |
| 总结 `embed_enabled` | `process_summary_configs` / 租户规则 UI |

流程级自动刷新策略位于对应配置的 `embed_config`。默认行为是：业务数据变化、附件换版、
退回/重提会自动刷新；普通审批推进不自动刷新。

流程配置还可开启定时兜底检查：

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `scheduled_refresh_enabled` | `false` | 是否定时检查该流程类型 |
| `scheduled_refresh_lookback_days` | `3` | 拉取近几天创建的流程，可选 1–30 天 |
| `scheduled_refresh_interval_minutes` | `5` | 检查频率，支持 5、10、15、30、60 分钟 |

定时检查单次最多拉取 500 个候选流程，只负责发现候选；最终仍由审核上下文指纹和总结块依赖
指纹决定是否调用 AI。

流程配置保存时会同步写入独立的 `embed_refresh_schedules` 调度记录，并立即注册或移除内存
Cron；服务启动时从该表恢复全部活跃任务。因此配置开启、关闭和频率修改立即生效，不再每分钟
轮询流程配置。5/10/15/30/60 分钟频率分别转换为对应的六段式 Cron 表达式并按
`app.timezone` 到点执行。多实例通过 Redis 变更通知同步内存调度，并使用分布式执行锁防止
重复拉取。

关闭定时检查时会同时清除该配置尚未触发的 Redis 候选，并把已经入库但仍为 `pending` 的审核/
总结任务标记为 `cancelled`；已经开始执行的任务不强制中断。服务升级时还会清理旧版缺少
`config_id` 的定时候选，启用中的 Cron 会按新格式重新生成。

调度表保存 `last_run_at`、`next_run_at`、`last_status` 和 `last_error`，便于排查最近执行结果。

刷新依据和执行结果可从以下记录追溯：

- `audit_logs.oa_context_anchor`：审核实际使用字段、附件版本、流程日志/节点及执行配置指纹；
- `audit_logs.attempt_fingerprint`：最近一次审核尝试的完整依赖指纹，用于阻止相同失败被自动重试；
- `process_summary_logs.oa_context_anchor`：总结时的 OA 变化锚点；
- `process_summary_logs.process_snapshot.block_dependencies`：各总结块的数据、附件、流程依赖指纹；
- `process_summary_logs.process_snapshot.regenerated_block_ids`：本次实际重新生成的总结块；
- `audit_logs.trigger_detail` / `process_summary_logs.trigger_detail`：区分可见页、手动、保存提交和定时扫描；
- `schedule_config_id`：定时扫描任务归属的流程配置；
- `tenant_llm_message_logs` / `tenant_llm_message_payloads`：实际发生的审核、总结模型调用及 Token、耗时。

如果普通审批被策略过滤且未调用 AI，不会新增审核/总结执行日志或 LLM 日志。

详细部署步骤见 [02-embed-audit-sidebar.md](../oa-configurations/02-embed-audit-sidebar.md) 与 [03-embed-process-summary.md](../oa-configurations/03-embed-process-summary.md)。
