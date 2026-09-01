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

返回指定流程的全部 AI 调用记录（时间倒序，含 Token、耗时、提示词及本次调用关联业务执行记录的
`config_version_no`）。同一流程先后使用多个配置版本时，每条调用返回各自实际版本；迁移前无法还原
配置版本的历史调用该字段为空，管理页展示为“历史记录（未记录配置版本）”。

---

## 获取单条调用详情

```
GET /api/tenant/llm-logs/calls/:id
```

返回单条调用详情，含系统提示词、用户提示词、深度思考过程（`reasoning_content`）、模型响应正文与 `config_version_no`。版本号通过
`business_log_id` 关联审核、归档复盘或流程总结执行日志获取，不使用流程当前绑定版本覆盖历史调用。

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
