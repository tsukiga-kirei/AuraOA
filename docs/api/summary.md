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

`embed_config` 控制 OA 嵌入总结的自动刷新策略：

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `auto_summary_on_open` | `true` | 没有历史总结时自动生成 |
| `auto_summary_on_data_change` | `true` | 按总结块实际使用的字段和附件版本判断并增量刷新 |
| `auto_summary_on_return_resubmit` | `true` | 退回或重新提交后刷新依赖流程信息的总结块 |
| `auto_summary_on_flow_change` | `false` | 普通审批推进后刷新使用流程基础信息/审批历史的总结块 |

自动刷新只调用发生变化的启用总结块；未变化的块沿用最近一次有效结果。手动“重新总结”
仍会执行全部启用块。

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

建模表挂载会在 `context_text` 中显示“建模表：中文名（英文表名）”；`mode=rows` 时，`return_fields` 必须填写英文物理列名，服务端会基于泛微字段元数据将返回结果的列名转换为中文显示名。`max_rows=-1` 表示返回全部匹配行，正整数表示行数上限。

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
