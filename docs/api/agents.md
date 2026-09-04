# 智能体与分配接口（拟定）

> 需求契约，代码未落地。两级分配语义以 [`docs/agents/allocation.md`](../agents/allocation.md) 为准。  
> 实现后同步 OpenAPI 与错误码。

## 系统管理员：租户配额

前缀 `/api/admin`，角色 `system_admin`。

### 平台目录（只读）

```
GET /api/admin/agent-catalog
```

返回平台系统工具、内置智能体、内置 Skills、MCP 模板摘要（无密钥）。

### 读/写某租户配额

```
GET /api/admin/tenants/:id/chat-allocation
PUT /api/admin/tenants/:id/chat-allocation
```

拟定 `data`：

```json
{
  "chat_enabled": true,
  "chat_retention_days": 90,
  "chat_primary_model_id": null,
  "chat_fallback_model_id": null,
  "agent_codes": ["oa_query", "oa_assist"],
  "tool_codes": ["list_my_todos", "get_process"],
  "skill_codes": [],
  "allow_custom_skills": false,
  "allow_tenant_mcp": false,
  "max_mcp_servers": 0,
  "mcp_template_ids": []
}
```

`PUT` 缩小配额时服务端摘除越界绑定（或拒绝保存并返回冲突列表，实现时二选一，**推荐拒绝并返回冲突**，避免静默改智能体）。

租户开关也可并入现有 `PUT /api/admin/tenants/:id`；若合并，须在 [`system-admin.md`](./system-admin.md) 同一 PR 增补字段，本文改为索引。

## 租户管理员：智能体

前缀 `/api/tenant/agents`，角色 `tenant_admin`，且租户 `chat_enabled`。

列表分页：`page` / `page_size`。

```
GET    /api/tenant/agents
POST   /api/tenant/agents
GET    /api/tenant/agents/:id
PUT    /api/tenant/agents/:id
DELETE /api/tenant/agents/:id
```

创建/更新 body 含：名称、说明、提示词、`tool_codes`、`skill_codes`、`mcp_bindings`、`enabled`。所有 code 必须 ⊆ 租户配额。种子智能体可禁用、不可删除（或删除后下次种子修复，实现时选「不可删只可停用」）。

## 租户管理员：MCP / Skills

```
GET/POST /api/tenant/mcp-servers
POST     /api/tenant/mcp-servers/:id/test
POST     /api/tenant/mcp-servers/:id/refresh-tools
DELETE   /api/tenant/mcp-servers/:id

GET/POST /api/tenant/skills
PUT/DELETE /api/tenant/skills/:id
```

无 `allow_tenant_mcp` / `allow_custom_skills` 时写接口 403。密钥只写不读。

## 组织角色再分配

扩展现有组织角色 API（[`org.md`](./org.md)）：

- 角色详情增加 `agent_codes[]`、`tool_codes[]`（含 `mcp:...`、`skill:...`）。
- 写入时校验 ⊆ 租户配额。
- `page_permissions` 仍单独表示页面，`/chat` 与智能体授权同时需要。

实现时优先扩展 `GET/PUT /api/tenant/org/roles/:id`，避免平行一套授权接口导致不一致。

## 配额查询（租户管理端）

```
GET /api/tenant/chat-allocation
```

只读：当前租户被系统管理员授予的 code 列表，供勾选框禁用未分配项。
