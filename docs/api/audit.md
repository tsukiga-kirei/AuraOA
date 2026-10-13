# 审核工作台接口

## 审核执行（JWT + TenantContext）

> 路由前缀：`/api/audit`

### 获取待审核流程列表

```
GET /api/audit/processes
```

分页查询 OA 待审批流程列表，支持多维度筛选。当前待办会合并个人工作台审核与 OA 嵌入审核结果；同一流程优先显示当前用户的个人审核结果，没有个人结果时显示 OA 嵌入结果。所有前台角色均按当前用户判断；自己的待办即使未手动审核，也可展示通用嵌入结果，不能回退到其他人的系统内个人审核。`audit_result.result_source` 为 `personal` 或 `embed`。

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `tab` | string | 页签（`pending_ai` 待审核 / `completed` 已完成） |
| `keyword` | string | 流程标题关键词 |
| `applicant` | string | 申请人姓名 |
| `process_type` | string | 流程类型 |
| `department` | string | 部门名称 |
| `audit_status` | string | 审核状态筛选 |
| `page` | int | 页码（从 1 开始） |
| `page_size` | int | 每页条数 |
| `start_date` | string | 开始日期 |
| `end_date` | string | 结束日期 |

---

### 导出流程列表

```
GET /api/audit/processes/export
```

按当前筛选条件导出全量审核流程为 Excel 文件。查询参数与列表接口一致。

---

### 获取审核统计

```
GET /api/audit/stats
```

返回审核统计数据（待审核数、已完成数、各状态分布等）。

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `start_date` | string | 开始日期 |
| `end_date` | string | 结束日期 |

---

### 提交审核任务

```
POST /api/audit/execute
```

提交单条审核任务，后端异步执行两阶段 AI 审核。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `process_id` | string | ✅ | OA 流程实例 ID |
| `process_type` | string | ✅ | 流程类型名称 |
| `title` | string | — | 流程标题 |
| `use_latest_config` | boolean | — | 默认 `false`，沿用该流程已绑定的执行配置版本；仅明确传 `true` 时升级到当前最新配置后重新审核 |

首次执行会把最终生效的字段范围、规则、审核尺度和提示词保存为不可变执行配置版本，并与流程实例绑定。
租户流程配置在首次发布前仅为草稿，不会进入审核工作台或执行入口；首次发布生成基础配置 v1。
同一流程后续重新审核默认沿用该版本；管理端修改规则、尺度或提示词不会自动使历史结果失效。
任务与审核链响应中的 `config_version_no` 表示本次实际使用的版本，迁移前历史记录可能为空。

**响应**：返回 `pending` 状态的任务 ID，前端通过轮询获取最终结果。

---

### 取消审核任务

```
POST /api/audit/cancel/:id
```

取消正在执行的审核任务。

---

### 查询任务状态

```
GET /api/audit/jobs/:id
```

轮询异步审核任务的当前状态和进度步骤。

**任务状态流转**：

```
pending → assembling → reasoning → extracting → completed
                                              → failed
```

---

### 获取任务流式输出

```
GET /api/audit/stream/:id
```

通过 SSE（Server-Sent Events）实时获取 AI 推理过程的流式输出。

---

### 批量提交审核

```
POST /api/audit/batch
```

批量提交多条审核任务，后端异步处理。

---

### 获取审核链

```
GET /api/audit/chain/:processId
```

获取指定流程的完整审核链（历次审核记录按时间排列）。

---

## 审核日志管理（JWT + TenantContext + `tenant_admin`）

> 路由前缀：`/api/audit/logs`

### 获取审核日志列表

```
GET /api/audit/logs
```

---

### 获取审核日志统计

```
GET /api/audit/logs/stats
```

---

### 导出审核日志

```
GET /api/audit/logs/export
```

---

## 审核快照管理（JWT + TenantContext + `tenant_admin`）

> 路由前缀：`/api/audit/snapshots`

### 获取快照列表

```
GET /api/audit/snapshots
```

支持按渠道筛选：`channel` 可选值：`workbench`（系统内）、`embed_standard`（嵌入通用）、`embed_personal`（嵌入个性化）、`embed`（历史兼容）。
嵌入通用按流程唯一汇总，系统内与嵌入个性化按（流程 + 操作人）唯一汇总。顶部统计卡片与列表总数严格保持一致。
嵌入通用行返回最新有效记录的 `trigger_detail`。若 OA 当前操作人员可解析，`operator` 展示为
“OA 嵌入审核（姓名）”，`department` 返回其 OA 部门；无法解析时展示“OA 嵌入审核”且部门为空。

---

### 获取快照统计

```
GET /api/audit/snapshots/stats
```

---

### 导出快照

```
GET /api/audit/snapshots/export
```

按筛选条件导出全部匹配数据为 Excel，不受当前页、`page` / `page_size` 或 5000 条上限限制。筛选条件与列表接口一致，支持：

- `columns`：自定义展示列（按参数顺序导出，前端按表格显示顺序传入；逗号分隔，如 `process_id,workflow_code,title,operator,department,process_type,channel,ai_result,audit_count,time`）
- `include_history`：布尔值（`true`/`false`），是否追加审核历史列展示当前租户内该流程的历次有效审核记录（按时间倒序），查询失败时返回错误，不生成空白历史文件

---

### 获取快照审核链

```
GET /api/audit/snapshots/:processId/chain
```

审核链中的每条嵌入执行记录分别返回当次 `user_name` 与 `trigger_detail`，用于展示当次操作人及
保存、提交、打开、重新审核或定时检查动作。每条记录均返回其实际使用的 `config_version_no`；迁移前历史记录可能为空，数据管理页
在“查看详情”抽屉中明确展示为未记录版本。

### 前台审核数据可见范围

`GET /api/audit/chain/:processId` 要求本人参与流程或曾主动触发该流程的个人审核，返回本人系统内/嵌入个性化记录及通用嵌入记录，按时间排列。租户管理员无前台额外权限。
任务结果与流式读取同样禁止读取其他人的个人审核；通用嵌入结果需校验当前用户参与权限。管理端全租户审核链接口保持独立。

### 日期筛选边界

日志/快照列表及其导出接口的 `start_date`、`end_date` 按应用配置时区解释，结束日期包含当天全部记录。
实现使用 `时间 >= 开始日期零点 AND 时间 < 结束日期次日零点`，包含当天最后一秒内的微秒记录，次日边界按自然日计算以兼容夏令时。

## 查询证据

工作台结果详情增加 `query_evidence`，记录每条规则引用查询的版本、参数、数据与完整性。历史结果可能为空数组；按原结果权限访问。失败/截断会降级最终通过建议并要求人工复核，详见 [记忆库 / 数据记忆](./audit-data-queries.md#给审核-ai-的数据与证据)。

## 规则适用前提结果

规则结果继续使用 `passed` 表示通过/不通过。适用前提为空时沿用原校验；满足前提时校验正文；明确不满足前提时 `passed=true`，原因说明“适用前提不满足，本规则直接通过”及依据，不扣分。规则通过数量包含这些直接通过项，不新增状态。


## 反馈记忆提示词变量

审核与归档两阶段提示词支持 `{{feedback_memory}}`。仅使用当前租户、当前业务模块和流程类型的已发布反馈记忆，未放置变量的旧模板自动追加；同一次两阶段共享记忆版本。反馈作为运行时参考资料，与流程执行配置版本独立，不能覆盖正式强制规则。实际记忆版本保留在 AI 调用提示词中，详见 [评论反馈记忆库](./feedback-memory.md)。
