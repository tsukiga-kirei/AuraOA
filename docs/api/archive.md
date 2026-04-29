# 归档复盘接口

## 归档复盘执行（JWT + TenantContext）

> 路由前缀：`/api/archive`

### 获取已归档流程列表

```
GET /api/archive/processes
```

分页查询 OA 已归档流程列表，支持多维度筛选。

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

### 获取快照复盘链

```
GET /api/archive/snapshots/:processId/chain
```
