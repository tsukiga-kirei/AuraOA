# OA 嵌入接口

## 概述

供固定地址嵌入页 `/embed/audit` 与 `/embed/summary` 使用。**浏览器不直接调用 Go 接口**，而是请求 Nuxt 代理 `/api/embed/*`。

鉴权链路：

1. OA 父页面通过 `postMessage` 将 `embed_token` 传给 iframe
2. 嵌入页调用 `POST /api/embed/session` 写入 httpOnly Cookie
3. Nuxt 代理从 Cookie 读取令牌，携带 `X-Embed-Token` 访问 Go
4. Go `EmbedAccess` 中间件根据令牌哈希反查租户，并注入 `tenant_id`

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

详细部署步骤见 [02-embed-audit-sidebar.md](../oa-configurations/02-embed-audit-sidebar.md) 与 [03-embed-process-summary.md](../oa-configurations/03-embed-process-summary.md)。
