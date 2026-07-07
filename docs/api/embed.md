# OA 嵌入接口

## 概述

供固定地址嵌入页 `/embed/audit` 与 `/embed/summary` 使用。**浏览器不直接调用 Go 接口**，而是请求 Nuxt 代理 `/api/embed/*`；代理携带 `X-Embed-Token` + `X-Tenant-Code` 访问 Go。

Go 路由前缀：`/api/embed`  
鉴权：`EmbedAccess` 中间件（**非 JWT**）

---

## Nuxt 代理（浏览器调用）

| 方法 | 路径 | 转发至 Go |
|------|------|-----------|
| GET | `/api/embed/context?process_id=` | `GET /api/embed/context` |
| POST | `/api/embed/execute` | `POST /api/embed/execute` |
| GET | `/api/embed/jobs/:id` | `GET /api/embed/jobs/:id` |
| GET | `/api/embed/summary/context?process_id=` | `GET /api/embed/summary/context` |
| POST | `/api/embed/summary/execute` | `POST /api/embed/summary/execute` |
| GET | `/api/embed/summary/jobs/:id` | `GET /api/embed/summary/jobs/:id` |
| GET | `/api/embed/summary/stream/:id` | `GET /api/embed/summary/stream/:id` |

环境变量：

```bash
EMBED_ACCESS_TOKEN=...
EMBED_TENANT_CODE=...
NUXT_PUBLIC_API_BASE=http://localhost:8080
```

---

## Go 接口

### 获取嵌入上下文

```
GET /api/embed/context?process_id=598488
```

请求头：

```
X-Embed-Token: <embed.access_token>
X-Tenant-Code: <embed.tenant_code>
```

响应字段见 [OA 嵌入配置文档](../oa-configurations/02-embed-audit-sidebar.md)。

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

---

## 流程总结接口

### 获取总结嵌入上下文

```
GET /api/embed/summary/context?process_id=598488
```

响应要点：

| 字段 | 说明 |
|------|------|
| `supported` | 当前流程是否已配置且允许嵌入总结 |
| `process` | OA 流程标题、申请人、部门、流程类型、当前节点 |
| `has_summary` | 是否已有有效总结快照 |
| `stale` | OA 上下文相对上次总结是否变化 |
| `should_auto_summary` | 前端是否应自动发起总结 |
| `running_job_id` | 已有进行中的总结任务 |
| `summary_result.blocks` | 结构化总结块数组 |

### 触发总结

```
POST /api/embed/summary/execute
```

```json
{
  "process_id": "598488",
  "trigger_source": "summary_embed_manual"
}
```

`trigger_source` 可选：

| 值 | 说明 |
|----|------|
| `summary_embed_auto` | 嵌入页自动发起 |
| `summary_embed_manual` | 用户点击重新总结 |

### 查询任务状态

```
GET /api/embed/summary/jobs/:id
```

响应结构：

```json
{
  "status": "completed",
  "id": "uuid",
  "process_id": "598488",
  "blocks": [
    {
      "block_id": "basic",
      "title": "流程摘要",
      "content": "Markdown 正文",
      "points": ["要点"]
    }
  ],
  "parse_error": ""
}
```

### SSE 原始模型输出

```
GET /api/embed/summary/stream/:id
```

用于嵌入页在任务执行中展示模型原始输出；结构化结果仍以 `jobs/:id` 为准。

---

## postMessage 协议（OA 自定义 JS ↔ 嵌入页）

| 方向 | `type` | 载荷 |
|------|--------|------|
| iframe → OA | `aura-oa-request-requestid` | 无 |
| OA → iframe | `aura-oa-requestid` | `{ requestid: string }` |

OA 示例脚本：[../oa-configurations/assets/aura-embed-notify.js](../oa-configurations/assets/aura-embed-notify.js)

---

## 配置说明

| 配置项 | 位置 |
|--------|------|
| `embed.access_token` | `go-service/config.yaml` |
| `embed.tenant_code` | `go-service/config.yaml` |
| `EMBED_ACCESS_TOKEN` | 前端部署环境变量 |
| `EMBED_TENANT_CODE` | 前端部署环境变量 |
| 审核 `embed_enabled` | `process_audit_configs` / 租户规则 UI |
| 总结 `embed_enabled` | `process_summary_configs` / 租户规则 UI |

详细部署步骤见 [02-embed-audit-sidebar.md](../oa-configurations/02-embed-audit-sidebar.md) 与 [03-embed-process-summary.md](../oa-configurations/03-embed-process-summary.md)。
