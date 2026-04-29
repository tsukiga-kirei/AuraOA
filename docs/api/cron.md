# 定时任务接口

## 任务类型配置 — 只读（JWT + TenantContext）

### 获取任务类型配置列表

```
GET /api/tenant/cron/configs
```

返回所有 6 个任务类型的当前配置（系统预设 + 租户覆盖合并）。`is_enabled=false` 表示该任务类型未启用，配置值为系统预设。

---

## 任务类型配置 — 写操作（JWT + TenantContext + `tenant_admin`）

### 保存任务类型配置

```
PUT /api/tenant/cron/configs/:taskType
```

启用或更新指定任务类型配置（Upsert）。

**路径参数**：

| 参数 | 说明 |
|------|------|
| `taskType` | 任务类型编码（如 `audit_batch`、`archive_daily`） |

---

### 重置任务类型配置

```
DELETE /api/tenant/cron/configs/:taskType
```

重置指定任务类型为系统预设（删除租户覆盖配置）。重置后 `is_enabled` 变为 `false`。

---

## 任务实例管理（JWT + TenantContext）

> 路由前缀：`/api/tenant/cron/tasks`

### 获取任务列表

```
GET /api/tenant/cron/tasks
```

返回当前租户所有任务实例。

---

### 创建任务

```
POST /api/tenant/cron/tasks
```

---

### 更新任务

```
PUT /api/tenant/cron/tasks/:id
```

更新任务实例（Cron 表达式、标签、推送邮箱等）。

---

### 删除任务

```
DELETE /api/tenant/cron/tasks/:id
```

内置任务不可删除。

---

### 切换任务启用状态

```
POST /api/tenant/cron/tasks/:id/toggle
```

---

### 立即执行任务

```
POST /api/tenant/cron/tasks/:id/execute
```

手动触发任务执行（异步，后端 goroutine 执行）。

---

### 获取任务执行日志

```
GET /api/tenant/cron/tasks/:id/logs
```

返回指定任务的最近执行日志。

---

## 全量日志管理（JWT + TenantContext + `tenant_admin`）

> 路由前缀：`/api/tenant/cron/logs`

### 获取全量日志列表

```
GET /api/tenant/cron/logs
```

跨任务查询所有定时任务执行日志。

---

### 获取全量日志统计

```
GET /api/tenant/cron/logs/stats
```

---

### 导出全量日志

```
GET /api/tenant/cron/logs/export
```
