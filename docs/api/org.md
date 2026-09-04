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

角色的 `page_permissions` 控制可访问页面。规划中的 AI 对话另有智能体/工具再分配字段，见 [`docs/api/agents.md`](./agents.md) 与 [`docs/agents/allocation.md`](../agents/allocation.md)，实现时扩展本模块角色读写接口。

### 获取角色列表

```
GET /api/tenant/org/roles
```

---

### 创建角色

```
POST /api/tenant/org/roles
```

---

### 更新角色

```
PUT /api/tenant/org/roles/:id
```

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
