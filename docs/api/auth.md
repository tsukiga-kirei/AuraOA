# 认证接口

## 公开接口（无需认证）

### 获取初始化状态

```
GET /api/auth/bootstrap-status
```

返回系统是否需要初始化（是否已创建超级管理员）。

**响应字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `needs_setup` | boolean | `true` 表示需要初始化 |

---

### 初始化超级管理员

```
POST /api/auth/bootstrap
```

仅在系统无任何用户时允许调用，创建第一个系统管理员账号。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | ✅ | 用户名 |
| `password` | string | ✅ | 密码（最少 8 位） |
| `display_name` | string | ✅ | 显示名称 |

---

### 用户登录

```
POST /api/auth/login
```

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | ✅ | 用户名 |
| `password` | string | ✅ | 密码 |
| `tenant_id` | string | — | 指定登录租户（可选） |
| `preferred_role` | string | — | 首选角色（可选） |

**响应字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `access_token` | string | JWT Access Token（有效期 2h） |
| `refresh_token` | string | JWT Refresh Token（有效期 7d） |
| `user` | object | 用户基本信息 |
| `roles` | array | 所有角色分配列表 |
| `active_role` | object | 当前激活的角色 |
| `permissions` | array | 当前角色的权限列表 |

---

### 刷新 Token

```
POST /api/auth/refresh
```

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `refresh_token` | string | ✅ | Refresh Token |

**响应字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `access_token` | string | 新的 Access Token |

---

### OA Basic 单点登录换址

```
GET /api/auth/sso/basic-redirection?portal=business
Authorization: Basic base64(tenantCode/username:sharedPassword)
```

该接口由已完成用户认证的 OA **服务端**调用。成功时返回 `302`，`Location` 为 60 秒内有效且仅可消费一次的 AuraOA 浏览器登录地址。OA HTTP 客户端必须禁止自动跟随重定向，再将 `Location` 原样交给用户浏览器。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `portal` | string | — | `business`（默认）或 `tenant_admin`；不支持 `system_admin` |

Basic username 固定为 `tenantCode/username`；密码是系统管理 → 租户管理 → 单点登录中配置的租户共享密码。AuraOA 还会校验租户状态、用户状态、本地角色，以及租户配置的来源 IP/CIDR 和 Origin/Referer 域名白名单。

### 消费 Basic 单点登录地址

```
GET /api/auth/sso/basic-consume?code=<一次性交接码>
```

该地址只能由用户浏览器访问。成功后会先加载当前入口角色对应的菜单权限，再写入 AuraOA 登录态并跳转业务审核工作台 `/dashboard`。如果账号没有审核工作台菜单权限，页面会保留在 `/dashboard` 并明确显示无权限，不会退回概览页。交接码过期或重复消费返回 `40110`。

完整 OA Java 接入示例见 [`../oa-basic-sso-integration.md`](../oa-basic-sso-integration.md)。

---

### 获取公开租户列表

```
GET /api/tenants/list
```

返回所有可选租户（用于登录页租户选择下拉框）。

---

## 认证接口（需要 JWT）

### 登出

```
POST /api/auth/logout
```

将当前 Token 加入黑名单。

---

### 切换角色

```
PUT /api/auth/switch-role
```

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `role_id` | string | ✅ | 目标角色分配 ID |

**响应字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `access_token` | string | 新的 Access Token（含新角色信息） |
| `active_role` | object | 新激活的角色 |
| `permissions` | array | 新角色的权限列表 |
| `menus` | array | 新角色的菜单列表 |

---

### 获取菜单

```
GET /api/auth/menu
```

返回当前角色可访问的菜单列表。

---

### 修改密码

```
PUT /api/auth/change-password
```

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `current_password` | string | ✅ | 当前密码 |
| `new_password` | string | ✅ | 新密码 |

---

### 获取当前用户信息

```
GET /api/auth/me
```

返回当前登录用户的完整信息，包括基本信息、角色、租户信息、组织角色、登录历史等。

---

### 更新界面语言

```
PUT /api/auth/locale
```

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `locale` | string | ✅ | 语言编码（`zh-CN` / `en-US`） |

---

### 更新个人资料

```
PUT /api/auth/profile
```

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `display_name` | string | — | 显示名称 |
| `email` | string | — | 邮箱 |
| `phone` | string | — | 手机号 |

---

## 站内通知（需要 JWT）

### 获取未读通知数

```
GET /api/auth/notifications/unread-count
```

---

### 获取通知列表

```
GET /api/auth/notifications
```

---

### 标记全部已读

```
PUT /api/auth/notifications/read-all
```

---

### 标记单条已读

```
PUT /api/auth/notifications/:id/read
```
