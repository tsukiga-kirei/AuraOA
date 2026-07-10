# 流程审核配置接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/rules`

## 流程审核配置

### 获取配置列表

```
GET /api/tenant/rules/configs
```

返回当前租户的所有流程审核配置。

---

### 创建配置

```
POST /api/tenant/rules/configs
```

为指定流程类型创建审核配置（字段选择、AI 参数、权限控制等）。

---

### 获取配置详情

```
GET /api/tenant/rules/configs/:id
```

---

### 更新配置

```
PUT /api/tenant/rules/configs/:id
```

---

### 删除配置

```
DELETE /api/tenant/rules/configs/:id
```

---

### 测试 OA 连接

```
POST /api/tenant/rules/configs/test-connection
```

在配置流程时测试 OA 数据库连通性。

---

### 拉取流程字段

```
POST /api/tenant/rules/configs/:id/fetch-fields
```

从 OA 系统拉取指定流程的全部字段定义（主表 + 明细表），用于配置字段选择。

---

## 审核规则

### 获取规则列表

```
GET /api/tenant/rules/audit-rules
```

返回当前租户的所有审核规则。

---

### 创建规则

```
POST /api/tenant/rules/audit-rules
```

---

### 更新规则

```
PUT /api/tenant/rules/audit-rules/:id
```

---

### 删除规则

```
DELETE /api/tenant/rules/audit-rules/:id
```

---

## 提示词模板

### 获取提示词模板列表

```
GET /api/tenant/rules/prompt-templates
```

返回系统预置的提示词模板（只读），用于配置流程审核时选择提示词模板。

---

## 外部关联数据

### 测试关联数据配置

```
POST /api/tenant/rules/context/test
```

**请求体**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `process_id` | string | OA 流程 ID |
| `context_mounts` | object | 外部关联数据挂载配置 |

**响应**：`data.context_text` 为注入 AI 的文本预览。

建模表挂载会在 `context_text` 中显示“建模表：中文名（英文表名）”；`mode=rows` 时，`return_fields` 必须填写英文物理列名，服务端会基于泛微字段元数据将返回结果的列名转换为中文显示名。`max_rows=-1` 表示返回全部匹配行，正整数表示行数上限。
