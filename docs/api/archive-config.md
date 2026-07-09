# 归档复盘配置接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/archive`

结构与 [流程审核配置](./audit-config.md) 对称。

## 归档数据源配置

### 获取配置列表

```
GET /api/tenant/archive/configs
```

---

### 创建配置

```
POST /api/tenant/archive/configs
```

---

### 获取配置详情

```
GET /api/tenant/archive/configs/:id
```

---

### 更新配置

```
PUT /api/tenant/archive/configs/:id
```

---

### 删除配置

```
DELETE /api/tenant/archive/configs/:id
```

---

### 测试 OA 连接

```
POST /api/tenant/archive/configs/test-connection
```

---

### 拉取流程字段

```
POST /api/tenant/archive/configs/:id/fetch-fields
```

---

## 归档规则

### 获取规则列表

```
GET /api/tenant/archive/rules
```

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `config_id` | uuid | 必填，归档配置 ID |
| `rule_scope` | string | 可选 |
| `enabled` | boolean | 可选 |

---

### 创建规则

```
POST /api/tenant/archive/rules
```

---

### 更新规则

```
PUT /api/tenant/archive/rules/:id
```

---

### 删除规则

```
DELETE /api/tenant/archive/rules/:id
```

---

## 外部关联数据

### 测试关联数据配置

```
POST /api/tenant/archive/context/test
```

请求体与审核侧相同：`process_id`、`context_mounts`。返回注入 AI 的 `context_text`。

---

## 提示词模板

### 获取提示词模板列表

```
GET /api/tenant/archive/prompt-templates
```

返回系统预置的归档提示词模板（只读）。
