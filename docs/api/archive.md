# 归档复盘接口

## 归档复盘执行（JWT + TenantContext）

> 路由前缀：`/api/archive`

### 获取已归档流程列表

```
GET /api/archive/processes
```

分页查询当前 OA 用户发起或在审批历史中实际处理过的已归档流程列表，支持多维度筛选。

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `keyword` | string | 流程标题关键词 |
| `applicant` | string | 申请人姓名 |
| `process_type` | string | 流程类型 |
| `department` | string | 部门名称 |
| `audit_status` | string | 复盘状态筛选 |
| `page` | int | 页码（从 1 开始） |
| `page_size` | int | 每页条数 |
| `start_date` | string | 开始日期 |
| `end_date` | string | 结束日期 |

---

### 导出流程列表

```
GET /api/archive/processes/export
```

按当前筛选条件导出全量归档流程为 Excel 文件。

---

### 获取归档统计

```
GET /api/archive/stats
```

返回归档复盘统计数据。

---

### 提交归档复盘任务

```
POST /api/archive/execute
```

提交单条归档复盘任务，后端异步执行。

请求可选字段 `use_latest_config` 默认 `false`。首次复盘会将最终生效的字段、规则、尺度和提示词
保存为不可变执行配置版本并绑定流程；再次复盘默认沿用原版本。只有明确传 `true` 时才升级绑定。
租户流程配置在首次发布前仅为草稿，不会进入归档复盘工作台或执行入口；首次发布生成基础配置 v1。
既有归档结果不会因管理端配置变化自动重新生成。响应中的 `config_version_no` 为实际使用版本。

---

### 批量提交归档复盘

```
POST /api/archive/batch
```

---

### 取消归档复盘任务

```
POST /api/archive/cancel/:id
```

---

### 查询任务状态

```
GET /api/archive/jobs/:id
```

---

### 获取任务流式输出

```
GET /api/archive/stream/:id
```

---

### 获取复盘历史

```
GET /api/archive/history/:processId
```

获取指定流程的历史归档复盘记录列表（按时间倒序）。

---

### 获取复盘结果

```
GET /api/archive/result/:id
```

获取指定归档复盘任务的最终结果。

---

## 归档日志管理（JWT + TenantContext + `tenant_admin`）

> 路由前缀：`/api/archive/logs`

### 获取归档日志列表

```
GET /api/archive/logs
```

---

### 获取归档日志统计

```
GET /api/archive/logs/stats
```

---

### 导出归档日志

```
GET /api/archive/logs/export
```

---

## 归档快照管理（JWT + TenantContext + `tenant_admin`）

> 路由前缀：`/api/archive/snapshots`

### 获取快照列表

```
GET /api/archive/snapshots
```

---

### 获取快照统计

```
GET /api/archive/snapshots/stats
```

---

### 导出快照

```
GET /api/archive/snapshots/export
```

按筛选条件导出全部匹配数据为 Excel，不受当前页、`page` / `page_size` 或 5000 条上限限制。筛选条件与列表接口一致，支持：

- `columns`：自定义展示列（按参数顺序导出，前端按表格显示顺序传入；逗号分隔，如 `process_id,workflow_code,title,operator,department,process_type,compliance,audit_count,time`）
- `include_history`：布尔值（`true`/`false`），是否追加复核历史列展示当前租户内该流程的历次有效复核记录（按时间倒序），查询失败时返回错误，不生成空白历史文件

---

### 获取快照复盘链

```
GET /api/archive/snapshots/:processId/chain
```

复盘链中的每条执行记录均返回其实际使用的 `config_version_no`；迁移前历史记录可能为空，数据管理页
在“查看详情”抽屉中明确展示为未记录版本。

### 日期筛选边界

日志/快照列表及其导出接口的 `start_date`、`end_date` 按应用配置时区解释，结束日期包含当天全部记录。
实现使用 `时间 >= 开始日期零点 AND 时间 < 结束日期次日零点`，包含当天最后一秒内的微秒记录，次日边界按自然日计算以兼容夏令时。

## 查询证据

工作台结果详情增加 `query_evidence`，记录每条规则引用查询的版本、参数、数据与完整性。历史结果可能为空数组；按原结果权限访问。失败/截断会降级最终通过建议并要求人工复核，详见 [记忆库 / 数据记忆](./audit-data-queries.md#给审核-ai-的数据与证据)。

## 规则适用前提结果

规则结果继续使用 `passed` 表示通过/不通过。适用前提为空时沿用原校验；满足前提时校验正文；明确不满足前提时 `passed=true`，原因说明“适用前提不满足，本规则直接通过”及依据，不扣分。规则通过数量包含这些直接通过项，不新增状态。
