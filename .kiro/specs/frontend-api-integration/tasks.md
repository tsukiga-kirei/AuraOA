# 实施任务：前端 API 对接与默认角色功能 (frontend-api-integration)

## 任务 1: 创建独立类型定义文件
- [x] 1.1 创建 `frontend/types/org.ts`，定义 `Department`、`OrgRole`、`OrgMember` 接口（从 `useMockData.ts` 中提取），并在 `useMockData.ts` 中改为从 `~/types/org` 重新导入这些类型 [需求 1.1]
- [x] 1.2 创建 `frontend/types/auth.ts`，定义 `UserRole`、`PermissionGroup`、`RoleInfo`、`LoginRequest`、`LoginResponse`、`MenuItem`、`TenantOption` 接口 [需求 1.2]

## 任务 2: useAuth.ts 全面改造
- [x] 2.1 新增 `authFetch<T>` 方法：自动注入 Bearer token，解析统一响应格式 `{ code, message, data }`，401 时自动调用 `refreshToken()` 并重试，刷新期间并发请求排队（防抖），刷新失败则清除状态跳转登录页 [需求 3.1, 3.2, 3.3, 3.4, 3.5]
- [x] 2.2 改造 `login()` 方法：移除 `isMockMode` 分支和 `MOCK_USERS` 引用，调用 `POST /api/auth/login`，将 `LoginResponse` 中的 `access_token`、`refresh_token`、`user`、`roles`、`active_role`、`permissions` 映射到 useState 和 localStorage [需求 2.1, 2.2]
- [x] 2.3 改造 `getMenu()` 方法：移除 Mock 分支，调用 `GET /api/auth/menu`（通过 `authFetch`），返回菜单数据更新 `menus` 状态 [需求 2.3]
- [x] 2.4 改造 `switchRole()` 方法：移除 Mock 分支，调用 `PUT /api/auth/switch-role { role_id }`（通过 `authFetch`），更新 `token`、`activeRole`、`permissions`、`menus`；失败时状态不变 [需求 2.4]
- [x] 2.5 改造 `logout()` 方法：调用 `POST /api/auth/logout`（通过 `authFetch`），然后清除所有 localStorage 认证键和 useState 状态，跳转登录页 [需求 2.5]
- [x] 2.6 新增 `refreshToken()` 方法：调用 `POST /api/auth/refresh { refresh_token }`，更新 `access_token` [需求 3.2]
- [x] 2.7 新增 `changePassword()` 方法：调用 `PUT /api/auth/change-password { current_password, new_password }` [需求 6.1]
- [ ] 2.8 清理：移除所有 `isMockMode` 计算属性、`MOCK_USERS` 导入/导出、`getMockMenusByActiveRole` 引用，类型从 `~/types/auth` 导入，移除从 `useMockData` 的类型导入 [需求 2.1]

## 任务 3: login.vue 登录页改造
- [x] 3.1 租户列表改为从 `GET /api/tenants/list` 获取（公开接口，无需 token，直接用 `$fetch`），移除 `mockTenants` 引用 [需求 4.1]
- [x] 3.2 移除测试账号快速填充区域（`quickAccounts`、`fillAccount`、`mock-accounts` 区块）和所有 `isMockMode`、`MOCK_USERS` 引用 [需求 4.2]
- [x] 3.3 登录成功后从 `LoginResponse` 获取用户信息和角色信息，调用 `getMenu()` 获取菜单后跳转 `/overview` [需求 4.3]

## 任务 4: useOrgApi.ts 改造
- [x] 4.1 类型导入从 `~/composables/useMockData` 改为 `~/types/org` [需求 5.1]
- [x] 4.2 `apiFetch` 替换为 `useAuth().authFetch`，移除手动 `$fetch` + `ApiResponse` 解包逻辑，所有 CRUD 方法改用 `authFetch` [需求 5.2]

## 任务 5: settings.vue 个人设置页改造
- [x] 5.1 密码修改改为调用 `useAuth().changePassword()`，移除 `MOCK_USERS` 引用和本地密码验证逻辑（`mockUserSecurityInfo` 等） [需求 6.1, 6.2]

## 任务 6: admin/system/settings.vue 平台配置页改造
- [x] 6.1 新增 `frontend/composables/useSystemApi.ts`，提供 `getConfigs()` 和 `updateConfigs()` 方法，内部使用 `authFetch` [需求 7.3]
- [x] 6.2 平台配置页改为 `onMounted` 时从 `GET /api/admin/system/configs` 加载数据，保存时调用 `PUT /api/admin/system/configs`，移除 `mockOASystemConfigs`、`mockAIModelConfigs`、`mockSystemGeneralConfig` 引用 [需求 7.1, 7.2]

## 任务 7: 后端默认角色功能
- [x] 7.1 改造 `go-service/internal/service/tenant_service.go` 的 `CreateTenant` 方法：在同一事务中创建租户后自动创建三个默认角色（业务用户、审计管理员、租户管理员），角色创建失败则整个事务回滚 [需求 8.1, 8.2, 8.3]

## 任务 8: 数据库种子脚本更新
- [x] 8.1 更新 `db/seeds/004_org_roles.sql`：角色名称改为"业务用户"、"审计管理员"、"租户管理员"，描述与 TenantService 一致，`page_permissions` 改为路径字符串数组格式且与前端路由匹配 [需求 9.1, 9.2]

## 任务 9: 错误码处理
- [x] 9.1 在 `authFetch` 中实现错误码映射：40103→"用户名或密码错误"、40104→"账户已锁定"、40105→"账户已被禁用"、40106→"租户不存在或已停用"、40300→"权限不足"、50000→"服务器错误"，网络异常→"网络连接失败，请检查网络" [需求 10.1, 10.2, 10.3, 10.4, 10.5]
