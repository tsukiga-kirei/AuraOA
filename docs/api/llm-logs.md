# AI 调用记录接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/llm-logs`

租户管理员在**数据管理 → AI 调用记录**页使用。每次经 `AIModelCallerService` 的 LLM 调用均异步写入 `tenant_llm_message_logs`。

## 分页查询调用流程列表

```
GET /api/tenant/llm-logs/processes
```

按流程维度聚合展示 AI 调用记录。

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `request_type` | string | 场景：`audit` / `archive` / `summary` |
| `call_type` | string | 类型：`reasoning` / `structured` |
| `keyword` | string | 流程标题关键词 |
| `operator` | string | 操作人 |
| `start_date` | string | 开始日期 YYYY-MM-DD |
| `end_date` | string | 结束日期 YYYY-MM-DD |
| `page` | int | 页码，从 1 开始 |
| `page_size` | int | 每页条数，默认 20 |

**响应**：`PagedResult`（`items` + `total` + `page` + `page_size`）

---

## 获取统计

```
GET /api/tenant/llm-logs/stats
```

按场景分布统计 AI 调用流程数量。

---

## 获取流程调用链

```
GET /api/tenant/llm-logs/:processId/chain
```

返回指定流程的全部 AI 调用记录（时间倒序，含 Token 与耗时摘要）。

---

## 获取单条调用详情

```
GET /api/tenant/llm-logs/calls/:id
```

返回单条调用详情，含系统提示词、用户提示词与模型响应正文。

---

## 相关接口

租户 Token 消耗统计见 [用户设置接口](./user-settings.md)：

```
GET /api/tenant/stats/token-usage
```

全平台统计见 [系统管理接口](./system-admin.md)：

```
GET /api/admin/stats/token-usage
```
