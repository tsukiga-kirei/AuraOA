# 组织架构接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/org`

## 部门管理

### 获取部门列表

```
GET /api/tenant/org/departments
```

---

### 创建部门

```
POST /api/tenant/org/departments
```

---

### 更新部门

```
PUT /api/tenant/org/departments/:id
```

---

### 删除部门

```
DELETE /api/tenant/org/departments/:id
```

---

## 角色管理

角色仅通过 `page_permissions` 控制可访问页面，包括 `/chat`。智能体的使用范围统一在智能体管理的 `access_control` 配置，见 [`docs/api/agents.md`](./agents.md)。

### 获取角色列表

```
GET /api/tenant/org/roles
```

---

### 创建角色

```
POST /api/tenant/org/roles
```

请求体支持：`name`, `description`, `page_permissions[]`。

---

### 更新角色

```
PUT /api/tenant/org/roles/:id
```

请求体支持：`name`, `description`, `page_permissions[]`。

---

### 删除角色

```
DELETE /api/tenant/org/roles/:id
```

---

## 成员管理

### 获取成员列表

```
GET /api/tenant/org/members
```

---

### 创建成员

```
POST /api/tenant/org/members
```

`username`：1–100 位英文字母、数字或下划线，允许纯数字（可与 OA `loginid` 一致）。未提供密码时使用系统配置 `auth.default_password`。

---

### 更新成员

```
PUT /api/tenant/org/members/:id
```

---

### 删除成员

```
DELETE /api/tenant/org/members/:id
```

---

### 批量导入成员

```
POST /api/tenant/org/members/import
```

上传 Excel 文件批量导入成员。

---

### 下载导入模板

```
GET /api/tenant/org/members/import-template
```

下载成员导入 Excel 模板文件。

## 智能体使用权限

角色创建、更新与查询不再包含 `agent_codes`、`tool_codes`。业务用户需 `page_permissions` 包含 `/chat`；可用智能体由启用状态及 `access_control` 决定。历史角色授权表保留但不再读写，无须清理旧授权记录。工具、MCP、Skill 由智能体绑定和租户开关决定。参见 [智能体接口](./agents.md)。
