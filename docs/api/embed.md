# OA 嵌入审核接口

## 概述

供固定地址嵌入页 `/embed/audit` 使用。**浏览器不直接调用 Go 接口**，而是请求 Nuxt 代理 `/api/embed/*`；代理携带 `X-Embed-Token` + `X-Tenant-Code` 访问 Go。

Go 路由前缀：`/api/embed`  
鉴权：`EmbedAccess` 中间件（**非 JWT**）

---

## Nuxt 代理（浏览器调用）

| 方法 | 路径 | 转发至 Go |
|------|------|-----------|
| GET | `/api/embed/context?process_id=` | `GET /api/embed/context` |
| POST | `/api/embed/execute` | `POST /api/embed/execute` |
| GET | `/api/embed/jobs/:id` | `GET /api/embed/jobs/:id` |

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
| `embed_enabled` | `process_audit_configs` / 租户规则 UI |

详细部署步骤见 [02-embed-audit-sidebar.md](../oa-configurations/02-embed-audit-sidebar.md)。
