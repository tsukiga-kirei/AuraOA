# 审核工作台接口

## 审核执行（JWT + TenantContext）

> 路由前缀：`/api/audit`

### 获取待审核流程列表

```
GET /api/audit/processes
```

分页查询 OA 待审批流程列表，支持多维度筛选。

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

---

### 获取快照统计

```
GET /api/audit/snapshots/stats
```

---

### 获取快照审核链

```
GET /api/audit/snapshots/:processId/chain
```
