# 流程总结接口

## 流程总结配置（JWT + TenantContext + `tenant_admin`）

> 路由前缀：`/api/tenant/summary`

### 获取配置列表

```
GET /api/tenant/summary/configs
```

---

### 创建配置

```
POST /api/tenant/summary/configs
```

---

### 获取配置详情

```
GET /api/tenant/summary/configs/:id
```

---

### 更新配置

```
PUT /api/tenant/summary/configs/:id
```

---

### 删除配置

```
DELETE /api/tenant/summary/configs/:id
```

---

### 测试 OA 连接

```
POST /api/tenant/summary/configs/test-connection
```

---

### 拉取流程字段

```
POST /api/tenant/summary/configs/:id/fetch-fields
```

---

### 测试外部关联数据

```
POST /api/tenant/summary/context/test
```

---

## 流程总结快照（JWT + TenantContext + `tenant_admin`）

> 路由前缀：`/api/summary/snapshots`

数据管理页使用，与审核快照结构类似。

### 获取快照列表

```
GET /api/summary/snapshots
```

**查询参数**：`channel`、`keyword`、`process_type`、`operator`、`department`、`start_date`、`end_date`、`page`、`page_size`

---

### 获取快照统计

```
GET /api/summary/snapshots/stats
```

---

### 导出快照

```
GET /api/summary/snapshots/export
```

按筛选条件导出 Excel。

---

### 获取总结链

```
GET /api/summary/snapshots/:processId/chain
```

返回指定流程的历次总结记录链。
