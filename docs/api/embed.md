# OA 嵌入接口

## 概述

供固定地址嵌入页 `/embed/audit` 与 `/embed/summary` 使用。**浏览器不直接调用 Go 接口**，而是请求 Nuxt 代理 `/api/embed/*`。

鉴权链路：

1. OA 父页面通过 `postMessage` 将 `embed_token` 传给 iframe
2. 嵌入页调用 `POST /api/embed/session` 写入 httpOnly Cookie
3. Nuxt 代理从 Cookie 读取令牌，携带 `X-Embed-Token` 访问 Go
4. Go `EmbedAccess` 中间件根据令牌哈希反查租户，并注入 `tenant_id`

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

### 触发嵌入审核

```
POST /api/embed/execute
```

```json
{
  "process_id": "598488",
  "trigger_source": "embed_auto"
}
```

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

### 触发总结

```
POST /api/embed/summary/execute
```

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

刷新依据和执行结果可从以下记录追溯：

- `audit_logs.oa_context_anchor`：审核实际使用字段、附件版本、流程日志/节点及执行配置指纹；
- `process_summary_logs.oa_context_anchor`：总结时的 OA 变化锚点；
- `process_summary_logs.process_snapshot.block_dependencies`：各总结块的数据、附件、流程依赖指纹；
- `process_summary_logs.process_snapshot.regenerated_block_ids`：本次实际重新生成的总结块；
- `tenant_llm_message_logs` / `tenant_llm_message_payloads`：实际发生的审核、总结模型调用及 Token、耗时。

如果普通审批被策略过滤且未调用 AI，不会新增审核/总结执行日志或 LLM 日志。

详细部署步骤见 [02-embed-audit-sidebar.md](../oa-configurations/02-embed-audit-sidebar.md) 与 [03-embed-process-summary.md](../oa-configurations/03-embed-process-summary.md)。
