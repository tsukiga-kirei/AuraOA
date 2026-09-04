# OA 嵌入接口

## 概述

供固定地址嵌入页 `/embed/audit`、`/embed/summary` 与 OA 保存/提交事件使用。**浏览器不直接调用 Go 接口**，而是请求 Nuxt 代理 `/api/embed/*`。

鉴权链路：

1. OA 父页面通过 `postMessage` 将 `embed_token` 传给 iframe
2. 嵌入页把令牌保存在当前 iframe 内存，并调用 `POST /api/embed/session` 尝试写入 httpOnly Cookie
3. 浏览器请求通过同源 `X-Embed-Token` 请求头传递令牌；Cookie 作为同站场景补充
4. Nuxt 代理携带 `X-Embed-Token` 访问 Go
5. Go `EmbedAccess` 中间件根据令牌哈希反查租户，并注入 `tenant_id`

OA 父页面保存/提交时不创建隐藏 iframe，而是直接向 Nuxt `POST /api/embed/events` 发送简单表单请求。
表单体中的 `embed_token` 由 Nuxt 移除并转换为内部 `X-Embed-Token`，不会继续进入业务请求体。

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
  "workflow_id": "127",
  "oa_belong_user_id": "1042",
  "oa_current_user_id": "1042",
  "occurred_at_ms": 1785748255617,
  "action": "submit_requested",
  "event_id": "oa-598488-1722150000000"
}
```

接口只接受 `save_requested` 和 `submit_requested`，分别对应 `WfForm.OPER_SAVE`、
`WfForm.OPER_SUBMIT`。旧动作会返回 `400`，运行时没有兼容转换。已有 `process_id` 时，接口把
审核和总结检查写入 Redis 延迟队列并返回 `202`，不会等待 AI 执行。

首次新建流程允许 `process_id` 为空，但必须传 `workflow_id`。服务会在
放行 OA 操作前读取 `workflow_requestbase.requestid` 高水位并将事件写入
`embed_refresh_events`；约 2 秒后按“高水位之后 + workflow_id”解析新 requestid，
未落库时继续在约 5 秒、10 秒检查。唯一候选直接采用；出现多个候选时，
`oa_belong_user_id` 和 `oa_current_user_id` 只用于辅助消歧，不作为创建人硬过滤条件。
仍不能唯一确认时标记 `ambiguous`，不会猜测错误流程，也不会影响 OA 流程自身的保存或提交。

`occurred_at_ms` 是 OA 点击保存/提交时冻结的客户端 Unix 毫秒时间。服务会把它写入
`embed_refresh_events`，并在日志输出 `clientDelayMs`，用于区分正常网络耗时与浏览器晚发事件。

同一租户、流程和模块的连续事件自动合并。只有保存/提交事件会在任务执行期间保留一次后续检查；
定时扫描不会持续追踪进行中的任务。

通知脚本只注册 `OPER_SAVE` 和 `OPER_SUBMIT`，不再注册 `OPER_SAVECOMPLETE`，也不再创建隐藏 iframe。
脚本在页面就绪后直接注册 OA 事件；点击时立即冻结 requestid、workflow_id、人员标识和发生时间，
并使用唯一嵌入密钥直接异步 POST Nuxt 代理。请求完成或最多等待 800ms 后
放行 OA 操作；超时或 AuraOA 不可用也必须放行。浏览器不会轮询 requestid，服务端接收事件后
自行持久化并解析。

Go 服务会以结构化日志记录事件接收和每个模块的检查结论，不记录表单正文、附件正文或提示词：

```text
OA 嵌入刷新事件已接收
processID=598488 workflowID=127 action=submit_requested eventID=oa-...
clientDelayMs=12 scheduledModules=[audit,summary]

OA 嵌入刷新检查完成
processID=598488 module=summary action=submit_requested eventID=oa-...
attempt=0 result=done reason=unchanged retryScheduled=false

OA 嵌入刷新检查完成
processID= module=resolve action=submit_requested eventID=oa-...
result=done reason=requestid_resolved resolvedProcessID=617100 clientDelayMs=12
```

常见 `reason` 包括 `triggered`、`unchanged`、`unchanged_waiting_commit`、`job_running`、`not_found_in_oa`、
`requestid_not_ready`、`requestid_resolved`、`requestid_expired`、`requestid_ambiguous`、`auto_retry_blocked`、
`unsupported_no_config`、`schedule_config_missing` 和 `execute_failed`。保存/提交检查使用 INFO，
定时候选的逐流程检查使用 DEBUG，避免生产 INFO 日志被无变化候选淹没。

服务升级启动时会删除 Redis 延迟集合中尚未执行的旧动作。由于上一版迁移已经把历史保存、提交
合并为同一个来源，本次数据库迁移统一标记为 `legacy_operation`，不伪造无法还原的动作类型；
该值仅用于历史展示，不能作为接口或队列动作。

### 获取嵌入上下文

```
GET /api/embed/context?process_id=598488&oa_user_id=1042
```

请求头：

```
X-Embed-Token: <tenant embed access token>
X-Embed-OA-User-ID: 1042 (可选，当前泛微 OA 用户 ID)
```

支持泛微 OA 身份反查与双模审查视角：
- 若携带 `oa_user_id` / `X-Embed-OA-User-ID`，后端通过泛微 `hrmresource` 反查对应系统用户。
- 响应中返回 `personal_view`（含该用户的定制规则能力、个人审核结论）及 `default_perspective`（"standard" | "personal"）。
- 若用户拥有个人定制配置且已有专属审核记录，`default_perspective` 自动设定为 `"personal"`；否则默认 `"standard"`。

返回中的 `stale` / `should_auto_audit` 已按流程配置过滤。变化来源分为：

- 审核实际使用的主表与明细字段；
- 主表附件字段的 `docId` 及 `DocImageFile` 最新版本；
- 退回或退回后的重新提交；
- 普通审批日志、批注、转发、抄送与当前节点变化。

普通审批流变化是否触发由 `auto_audit_on_flow_change` 单独控制，默认关闭。
审核规则、尺度、提示词和字段配置变化不属于 OA 业务变化，不会设置 `should_auto_audit`。
响应中的 `config_version_no` 表示流程已绑定版本；当前管理配置与绑定版本不同时返回
`config_upgrade_available=true`。当前 OA 嵌入审核与总结页面暂不展示“使用最新配置重新执行”入口，
只保留普通重新执行；后端参数与业务前端入口继续保留，后续启用嵌入入口时无需迁移历史数据。
`trigger_source=embed_auto` 时后端会再次校验 `should_auto_audit`，无需刷新时不会创建审核任务。
可见审核页首次加载会先读取审核实际依赖的业务字段、附件版本、流程锚点和规则配置并比较指纹，
不下载或识别附件正文，也不调用 AI。无变化时直接展示已有结果；有变化时不先展示旧结果，
直接以 `visible_open` 进入交互队列。显式传 `prefer_cached=true` 时跳过变化检查，主要用于
任务完成后的结果刷新。
若最近一次相同依赖指纹（流程数据、附件版本、流程信息、审核规则和模型配置）的审核已经失败，
响应中 `auto_retry_blocked=true`，保存、提交和定时来源不会重复执行；失败记录直接作为结果展示，
只有用户点击“重新审核”才会再次尝试。

### 触发嵌入审核

```
POST /api/embed/execute
```

```json
{
  "process_id": "598488",
  "trigger_source": "embed_auto",
  "trigger_detail": "visible_open",
  "use_latest_config": false,
  "perspective": "standard",
  "oa_user_id": "1042"
}
```

- `perspective`：审查视角，可选 `"standard"`（官方标准基准，默认）或 `"personal"`（当前人员的定制规则视角）。
  - 当为 `"personal"` 时，任务使用系统内 `workbench` 队列并绑定用户 ID，生成用户专属审核快照，不污染官方公共快照。


`use_latest_config` 默认 `false`，自动来源必须保持为 `false`。手动传 `true` 会把流程绑定升级到
当前最终生效配置后执行；普通“重新审核”继续沿用原版本。

`trigger_detail` 的可见页取值为 `visible_open`，手动按钮为 `manual`。后台内部使用
`save_requested`、`submit_requested`、`scheduled_scan` 区分保存、提交与定时扫描。
嵌入审核与总结使用相同的队列路由语义：手动重新执行和可见页进入交互队列，保存/提交进入
普通后台队列，流程定时扫描进入独立定时队列。尚未领取的同流程任务可从后台队列提升到
交互队列；已经执行中的任务不会被中断。系统内审核工作台使用独立 `workbench` 队列，
不参与嵌入来源比较，也不会共享嵌入结果快照。

各任务类型使用独立 Redis Stream 和 worker，并由模块总并发统一限流。总并发大于 `1` 时，
非交互任务最多占用 `total-1` 个名额，为可见页和手动操作预留一个执行名额：

```env
WORKERS_AUDIT_WORKBENCH_CONCURRENCY=2
WORKERS_AUDIT_INTERACTIVE_CONCURRENCY=1
WORKERS_AUDIT_BACKGROUND_CONCURRENCY=1
WORKERS_AUDIT_SCHEDULED_CONCURRENCY=1
WORKERS_AUDIT_TOTAL_CONCURRENCY=3

WORKERS_SUMMARY_INTERACTIVE_CONCURRENCY=1
WORKERS_SUMMARY_BACKGROUND_CONCURRENCY=1
WORKERS_SUMMARY_SCHEDULED_CONCURRENCY=1
WORKERS_SUMMARY_TOTAL_CONCURRENCY=2
```

以上总并发限制按单个 `go-service` 实例生效；多实例部署时实际集群并发为各实例之和。
提高任一队列并发前应同时评估模块总并发，以及 MinerU 和模型服务容量。

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
GET /api/embed/summary/context?process_id=598488
```

除通用字段外，响应可包含 `stale_block_ids`，列出需要重新生成的总结块 ID。判断以每个块的
`field_mode`、`selected_fields` 和数据变量为准：

- `{{main_table}}` / `{{detail_tables}}`：只比较该块使用的业务字段；
- `{{attachments}}`：只比较该块选中的主表附件字段版本；
- `{{flow_history}}` / `{{flow_graph}}`：审批日志或流程图变化时由流程刷新策略控制；
- `{{process_meta}}`：当前节点或流程基础信息变化时由相应策略控制。

自动总结只重新调用 `stale_block_ids` 中的块并合并旧结果；手动总结执行全部启用块。
`trigger_source=summary_embed_auto` 时后端会再次校验 `should_auto_summary`，无需刷新时不会创建总结任务。
可见总结页首次加载不传 `prefer_cached`：页面先读取总结块实际依赖的业务字段、附件版本和流程
锚点并比较指纹，不下载或识别附件正文，也不调用 AI。无变化时直接展示已有结果；有变化时不先
展示旧结果，直接以 `visible_open` 进入交互队列。调用方显式传 `prefer_cached=true` 时，已有
成功结果会直接返回，不执行 OA 变化检查，主要用于任务完成后的结果刷新。
若最近一次相同依赖指纹（流程数据、附件版本、流程信息和总结块提示词配置）的任务已经失败，
响应中 `auto_retry_blocked=true`，自动来源不会重复执行，失败记录直接作为可展示结果返回。

可见总结页会先轻量比较指纹；已有结果且未变化时直接展示，退回重提、启用策略范围内的数据或
提示词变化则以 `visible_open` 进入交互队列。没有结果时按首次打开策略执行。手动“重新总结”
与可见页都进入交互队列，保存/提交进入普通后台队列，流程级定时扫描进入定时队列。

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
`save_requested`、`submit_requested`、`scheduled_scan`，用于查询任务的真实来源。

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

OA 保存/提交事件不使用 postMessage，而是由父页 JS 直接异步 POST Nuxt 代理。请求完成会立即放行，
最多等待 800ms；超时或 AuraOA 不可用也会放行，不代表等待 AI 执行完成。

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
重复拉取。调度执行前会重新核验对应的审核/总结源配置；调度表状态与源配置不一致时自动停用，
未明确开启 `scheduled_refresh_enabled` 的流程不会访问 OA 数据库。

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
- `audit_logs.trigger_detail` / `process_summary_logs.trigger_detail`：区分可见页、手动、保存、提交和定时扫描；
- `audit_logs.queue_kind`：明确记录 `workbench`、`interactive`、`background` 或 `scheduled`；
- `process_summary_logs.queue_kind`：明确记录 `interactive`、`background` 或 `scheduled`；
- `schedule_config_id`：定时扫描任务归属的流程配置；
- `tenant_llm_message_logs` / `tenant_llm_message_payloads`：实际发生的审核、总结模型调用及 Token、耗时。

如果普通审批被策略过滤且未调用 AI，不会新增审核/总结执行日志或 LLM 日志。

详细部署步骤见 [02-embed-audit-sidebar.md](../oa-configurations/02-embed-audit-sidebar.md) 与 [03-embed-process-summary.md](../oa-configurations/03-embed-process-summary.md)。
