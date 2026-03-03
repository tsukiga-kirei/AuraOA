# 需求文档：前端 API 对接与默认角色功能 (frontend-api-integration)

## 需求 1: 类型定义独立化

### 验收标准

1.1 Given `useMockData.ts` 中定义了 `Department`、`OrgRole`、`OrgMember` 类型，When 执行类型迁移，Then 这些类型定义在 `types/org.ts` 中独立存在，且 `useOrgApi.ts` 从 `~/types/org` 导入

1.2 Given 认证相关类型（`UserRole`、`PermissionGroup`、`RoleInfo`、`LoginRequest`、`LoginResponse`、`MenuItem`、`TenantOption`）分散在各文件中，When 执行类型迁移，Then 这些类型统一定义在 `types/auth.ts` 中，且 `useAuth.ts` 和 `login.vue` 从 `~/types/auth` 导入

## 需求 2: useAuth.ts 全面改造

### 验收标准

2.1 Given `useAuth.ts` 中存在 `isMockMode` 判断和 `MOCK_USERS` 引用，When 改造完成，Then 文件中不包含任何 `isMockMode`、`MOCK_USERS`、`useMockData` 相关引用

2.2 Given 用户调用 `login(req)` 方法，When 请求发送到后端，Then 调用 `POST /api/auth/login`，并将响应中的 `access_token`、`refresh_token`、`user`、`roles`、`active_role`、`permissions` 正确映射到 useState 状态和 localStorage

2.3 Given 用户调用 `getMenu()` 方法，When 请求发送到后端，Then 调用 `GET /api/auth/menu`（携带 Bearer token），返回菜单数据并更新 `menus` 状态

2.4 Given 用户调用 `switchRole(roleId)` 方法，When 请求发送到后端，Then 调用 `PUT /api/auth/switch-role { role_id }`，同时更新 `token`、`activeRole`、`permissions`、`menus`；若失败则所有状态保持不变

2.5 Given 用户调用 `logout()` 方法，When 执行登出，Then 调用 `POST /api/auth/logout`（Bearer token），清除 localStorage 中所有认证相关键和所有 useState 认证状态，跳转到登录页

## 需求 3: authFetch 封装

### 验收标准

3.1 Given `authFetch` 方法被调用，When 发送 API 请求，Then 请求头自动包含 `Authorization: Bearer {access_token}`

3.2 Given API 请求返回 401（token 过期），When `authFetch` 拦截到 401，Then 自动调用 `POST /api/auth/refresh { refresh_token }` 获取新 token，更新 localStorage，并使用新 token 重试原始请求

3.3 Given refresh_token 也已过期（刷新请求失败），When `authFetch` 检测到刷新失败，Then 清除所有认证状态并跳转登录页

3.4 Given 多个并发 API 请求同时遇到 401，When 触发 token 刷新，Then 系统只执行一次刷新操作，所有等待中的请求在刷新完成后使用新 token 重试

3.5 Given `authFetch` 收到后端统一响应格式 `{ code, message, data, trace_id }`，When `code` 为 0，Then 返回 `data` 字段；When `code` 非 0，Then 抛出包含 `message` 的错误

## 需求 4: 登录页改造

### 验收标准

4.1 Given 登录页加载，When 页面 mounted，Then 从 `GET /api/tenants/list` 获取租户列表并填充到租户选择下拉框

4.2 Given 登录页改造完成，When 查看页面代码，Then 不包含 `quickAccounts`、`fillAccount`、`mock-accounts`、`isMockMode`、`MOCK_USERS`、`mockTenants` 相关代码

4.3 Given 用户选择租户并输入账号密码后点击登录，When 登录成功，Then 从 `LoginResponse` 获取完整用户信息和角色信息，调用 `getMenu()` 获取菜单后跳转到 `/overview`

## 需求 5: useOrgApi.ts 改造

### 验收标准

5.1 Given `useOrgApi.ts` 中的类型导入指向 `useMockData`，When 改造完成，Then 类型从 `~/types/org` 导入

5.2 Given `useOrgApi.ts` 中的 `apiFetch` 方法手动构建请求，When 改造完成，Then 所有 API 调用改为使用 `useAuth().authFetch`，自动携带 Bearer token 和统一响应解析

## 需求 6: 个人设置页改造

### 验收标准

6.1 Given 用户在个人设置页修改密码，When 提交密码修改表单，Then 调用 `PUT /api/auth/change-password { current_password, new_password }`，由后端验证当前密码

6.2 Given 个人设置页改造完成，When 查看页面代码，Then 不包含 `MOCK_USERS` 引用和本地密码验证逻辑（`mockUserSecurityInfo` 等）

## 需求 7: 系统配置页改造

### 验收标准

7.1 Given 系统管理员访问平台配置页，When 页面加载，Then 从 `GET /api/admin/system/configs` 获取系统配置数据

7.2 Given 系统管理员修改配置并保存，When 点击保存按钮，Then 调用 `PUT /api/admin/system/configs` 提交配置更新

7.3 Given 新增 `useSystemApi.ts` composable，When 被系统配置页使用，Then 提供 `getConfigs()` 和 `updateConfigs()` 方法，内部使用 `authFetch`

## 需求 8: 默认角色创建

### 验收标准

8.1 Given 系统管理员创建新租户，When `TenantService.CreateTenant` 执行，Then 在同一数据库事务中创建三个 `is_system=true` 的组织角色：业务用户、审计管理员、租户管理员

8.2 Given 默认角色创建过程中发生错误，When 角色插入失败，Then 整个租户创建事务回滚，租户和角色均不被创建

8.3 Given 默认角色创建成功，When 查询该租户的组织角色，Then 业务用户的 `page_permissions` 包含 `["/overview", "/dashboard", "/settings"]`，审计管理员包含 `["/overview", "/dashboard", "/cron", "/archive", "/settings"]`，租户管理员包含 `["/overview", "/dashboard", "/cron", "/archive", "/settings", "/admin/tenant/rules", "/admin/tenant/org", "/admin/tenant/data", "/admin/tenant/user-configs"]`

## 需求 9: 数据库种子脚本更新

### 验收标准

9.1 Given `db/seeds/004_org_roles.sql` 中的角色数据与前端路由不匹配，When 更新种子脚本，Then 三个角色的名称更新为"业务用户"、"审计管理员"、"租户管理员"，描述和 `page_permissions` 与 TenantService 中的默认角色定义一致

9.2 Given 种子脚本中的 `page_permissions` 使用了旧的 JSON 对象格式（含 key/label/path），When 更新种子脚本，Then `page_permissions` 改为路径字符串数组格式（如 `["/overview", "/dashboard"]`），与前端路由和后端代码保持一致

## 需求 10: 错误码处理

### 验收标准

10.1 Given 后端返回错误码 40103，When 前端处理响应，Then 显示"用户名或密码错误"提示

10.2 Given 后端返回错误码 40104，When 前端处理响应，Then 显示"账户已锁定，请稍后重试"提示

10.3 Given 后端返回错误码 40106，When 前端处理响应，Then 显示"租户不存在或已停用"提示

10.4 Given 后端返回错误码 40300，When 前端处理响应，Then 显示"权限不足"提示

10.5 Given 网络不可达或请求超时，When `authFetch` 捕获异常，Then 显示"网络连接失败，请检查网络"提示
