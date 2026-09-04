# 对话接口（拟定）

> 需求契约。路由未在 `router.go` 落地前，本文不作为已实现行为。实现后补示例响应并同步 [`auraoa.openapi.yaml`](./auraoa.openapi.yaml)。  
> 产品与权限：[`docs/agents/`](../agents/README.md)。

**前缀**：`/api/chat`  
**鉴权**：JWT → TenantContext → 业务角色；另校验 `/chat` 与智能体授权。

嵌入对话见 [`embed.md`](./embed.md) 增补（实现时增加 `/api/embed/chat/*`，鉴权为 EmbedAccess）。

## 可用智能体

```
GET /api/chat/agents
```

返回当前用户**有效智能体**列表（配额 ∩ 角色），含 `agent_code`、名称、说明。不分页（数量应很小）；若超过 100 再改分页。

## 会话列表

```
GET /api/chat/sessions?page=1&page_size=20
```

服务端分页，字段 `items` / `total` / `page` / `page_size`。仅本人会话。可按 `keyword` 筛标题。

## 会话详情（消息）

```
GET /api/chat/sessions/:id
```

含会话元数据与消息列表。消息量大时后续可对消息再分页；第一期可一次返回并设上限。

## 新建会话

```
POST /api/chat/sessions
```

```json
{
  "agent_code": "oa_query",
  "process_id": null
}
```

`agent_code` 必须在有效集内。

## 重命名 / 删除

```
PATCH /api/chat/sessions/:id
DELETE /api/chat/sessions/:id
```

`PATCH` body：`title`。删除级联消息。

## 发送消息（SSE）

```
POST /api/chat/sessions/:id/messages/stream
Content-Type: application/json
Accept: text/event-stream
```

```json
{
  "content": "我有哪些待办"
}
```

事件类型见 [architecture.md](../agents/architecture.md)。取消：关闭客户端连接，服务端将助手消息标为 `interrupted`。

## 错误

沿用统一 `{ code, message }`。建议新增 errcode（实现时登记）：无对话权限、智能体不在有效集、工具不在有效集、未配置 OA、流程不可见。
