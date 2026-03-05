# OA 智审平台 — 前端代码架构与交互逻辑详解

> 文档版本：v1.2 | 更新日期：2026-03-04  
> 本文档详细解析前端代码的架构设计、核心交互逻辑、认证流程和技术实现细节。

---

## 一、Nuxt 3 应用配置

### 1.1 nuxt.config.ts

```typescript
export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  ssr: false,                       // SPA 模式（客户端渲染）
  modules: ['@ant-design-vue/nuxt'], // Ant Design Vue 自动导入
  css: ['~/assets/css/variables.css', '~/assets/css/global.css'],
  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8080',
      mockMode: process.env.NUXT_PUBLIC_MOCK_MODE || 'false',
    },
  },
  app: {
    pageTransition: { name: 'page', mode: 'out-in' },  // 页面切换动画
  },
})
```

**关键点**：
- `mockMode` 控制是否使用模拟数据（`'true'` 时不调用后端 API）
- `apiBase` 指向 Go 后端服务地址
- SSR 已关闭（SPA 模式），认证状态通过 `localStorage` 在客户端管理

### 1.2 app.vue 根组件

```vue
<a-config-provider :locale="zhCN">
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</a-config-provider>
```

- 使用 Ant Design 的 `ConfigProvider` 注入中文 locale
- 使用 dayjs 中文语言包

---

## 二、Composables 核心逻辑

### 2.1 useAuth — 认证与会话管理

**文件**: `composables/useAuth.ts` (472 行)

#### 状态管理

所有认证状态通过统一的 `PersistedAuthState` 结构持久化到 localStorage 单个 key（`auth_state`），token 对独立存储（高频读写）。支持从旧版分散 key 自动迁移。

| 状态变量 | 类型 | 说明 | 持久化 |
|----------|------|------|--------|
| `token` | `string \| null` | JWT 访问令牌 | localStorage `token` |
| `refreshToken` | `string \| null` | 刷新令牌 | localStorage `refresh_token` |
| `userRole` | `UserRole` | 当前角色类型 | `auth_state` |
| `allRoles` | `RoleInfo[]` | 用户所有角色分配 | `auth_state` |
| `activeRole` | `RoleInfo \| null` | 当前激活的角色 | `auth_state` |
| `userPermissions` | `PermissionGroup[]` | 当前权限组 | `auth_state` |
| `currentUser` | `object \| null` | 当前用户信息 | `auth_state` |
| `menus` | `MenuItem[]` | 当前菜单列表 | `auth_state` |
| `userLocale` | `string` | 用户语言偏好 | `auth_state` |

#### 错误代码映射

`authFetch` 内置错误代码到用户友好消息的映射（`ERROR_CODE_MAP`），覆盖 40103（密码错误）、40104（账户锁定）、40105（账户禁用）、40106（租户停用）、40300（权限不足）、40400（资源不存在）、50000（服务器错误）等场景。网络不可达时抛出"网络连接失败"提示。

#### 登录流程

```
1. login({ username, password, tenant_id })
2. → POST /api/auth/login → 获取 { access_token, refresh_token, user, roles, active_role, permissions }
3. → 存储 token 对到 state + localStorage
4. → 设置 allRoles、activeRole、currentUser、userPermissions
5. → 从 user.locale 同步语言偏好
6. → authFetch GET /api/auth/menu 拉取菜单（失败不影响登录）
7. → persistState() 统一写入 auth_state
8. → return { ok: true }
   失败时：解析后端 error.data.code，通过 ERROR_CODE_MAP 映射为用户友好消息
   → return { ok: false, errorMsg?: string }
```

#### 角色切换

```typescript
switchRole(roleId: string): Promise<boolean>
// 1. PUT /api/auth/switch-role { role_id } → 获取新 access_token + active_role + permissions + menus
// 2. 更新 token、activeRole、userPermissions、menus
// 3. persistState()
```

#### 菜单获取

```typescript
getMenu(): Promise<MenuItem[]>
// GET /api/auth/menu → 更新 menus 状态并持久化
```

#### authFetch — 带自动刷新的认证请求包装器

所有认证 API 调用统一通过 `authFetch<T>(path, options)` 发起，自动附加 Bearer Token。核心特性：
- 统一响应解析：期望 `{ code: 0, data: T }` 格式，非 0 code 映射为友好错误
- 401 自动刷新：收到 401 时调用 `doRefreshToken()` 换取新 token 后重试
- 并发刷新队列：多个请求同时 401 时，仅触发一次刷新，其余请求排队等待新 token
- 刷新失败自动登出：refresh 失败时清除状态并跳转 /login

#### 辅助方法

```typescript
changePassword(req: { current_password, new_password }): Promise<boolean>
// PUT /api/auth/change-password

getProfile(): Promise<MeResponse | null>
// GET /api/auth/me → 返回用户完整资料、组织角色、页面权限

updateLocale(locale: string): Promise<boolean>
// PUT /api/auth/locale → 同步语言偏好到后端

setUserLocale(locale: string): void
// 仅更新前端 locale 状态并持久化（不调用后端）
```

#### 会话恢复

```typescript
restore()
// 在页面刷新时从 localStorage 恢复所有认证状态（同步）
// 优先从 auth_state 合并 key 恢复，兼容旧版分散 key 并自动迁移
// 在 middleware/auth.ts 中被调用

tryRestoreAsync(): Promise<boolean>
// 异步恢复：当 access_token 丢失但 refresh_token 仍未过期时，
// 先通过 parseJwtExp() 检查 refresh_token 是否已过期，
// 若未过期则调用 doRefreshToken() 换取新 token。
// 返回 true 表示恢复成功（token 已可用），false 表示无法恢复。

isRefreshTokenValid(): boolean
// 判断 refresh_token 是否仍然有效（未过期）
```

### 2.2 useMockData — 模拟数据中心

**文件**: `composables/useMockData.ts` (2716 行)

这是系统最核心的数据文件，包含了所有前端模拟数据和类型定义。

#### 导出结构

**顶层导出**（模块级别，可被其他模块直接引用）：
- `MOCK_USERS` — 用户账户列表
- `PAGE_PERMISSIONS` — 页面权限矩阵
- `OVERVIEW_WIDGETS` — 仪表盘Widget定义
- `mockDepartments` — 部门列表
- `mockOrgRoles` — 组织角色列表
- `mockOrgMembers` — 组织成员列表
- `mockProcessAuditConfigs` — 审核工作台配置
- `mockArchiveReviewConfigs` — 归档复盘配置
- `mockStrictnessPresets` — 审核尺度预设
- `mockAuditLogs` / `mockCronLogs` / `mockArchiveLogs` — 各类日志
- `mockUserPersonalConfigs` — 用户偏好配置
- 辅助函数: `hasPagePermission`, `getDefaultPage`, `getMockMenusByActiveRole` 等

**composable 返回**（`useMockData()` 返回，需在 setup 中调用）：
- 各种流程数据（待办、已通过、已退回、归档）
- 审核结果数据
- 定时任务数据
- 租户信息
- 系统配置数据
- 仪表盘数据

#### 模拟 API 函数

```typescript
fetchStrictnessPresets(tenantId?): Promise<StrictnessPromptPreset[]>
// 模拟 300ms 延迟后返回审核尺度预设（深拷贝）

saveStrictnessPresets(tenantId, presets): Promise<boolean>
// 模拟 500ms 延迟后更新内存中的预设数据
```

### 2.3 useSidebarMenu — 侧边栏菜单

**文件**: `composables/useSidebarMenu.ts`

菜单完全由权限驱动，不依赖路由上下文切换。菜单项的过滤统一通过 `useAuth()` 返回的 `menus`（来自 GetMenu API）实现，不再依赖 `useOrgApi`：

```typescript
// 菜单段生成逻辑：
sections = computed(() => {
  // 1. overview 始终显示
  // 2. business 权限 → 根据 GetMenu API 返回的 menus 过滤业务菜单
  // 3. tenant_admin 权限 → 根据 menus 过滤租户管理菜单（menus 未加载时显示全部）
  // 4. system_admin 权限 → 显示系统管理菜单
})
```

**菜单权限过滤机制**：
```
所有角色（business、tenant_admin）统一使用 GetMenu API：
  1. 登录/角色切换时，后端根据当前 activeRole 返回 menus 列表
  2. useSidebarMenu 从 menus 中提取 path 构建 menuPagePerms 集合
  3. 过滤菜单项：只显示 path 在 menuPagePerms 中的项目
  4. menus 未加载时：business 不显示菜单，tenant_admin 显示全部（降级）
```

### 2.4 useI18n — 国际化

**文件**: `composables/useI18n.ts`

- 支持 `zh-CN` 和 `en-US` 两种语言
- 翻译文件约 81KB 每组，超过 1500 个翻译键
- 支持插值：`t('login.loginAs', '业务用户')` → `以 业务用户 身份登录`
- 语言偏好由 `useAuth` 的 `userLocale` 统一管理，持久化在 `auth_state` 中（不再使用独立的 `localStorage.app_locale`）

### 2.5 useTheme — 主题切换

**文件**: `composables/useTheme.ts` (63 行)

- 明/暗主题通过 `document.documentElement.setAttribute('data-theme', 'dark')` 切换
- CSS Variables 在 `variables.css` 中定义，使用 `[data-theme='dark']` 覆盖
- 切换时使用全屏遮罩实现平滑过渡动画（overlay fade out）

### 2.6 useAudit — 审核业务逻辑

**文件**: `composables/useAudit.ts` (74 行)

封装与后端 API 交互的审核操作：

```typescript
getTodoList()     // GET /api/audit/todo → 获取待审流程列表
executeAudit(id)  // POST /api/audit/execute → 执行AI审核
submitFeedback()  // POST /api/audit/feedback → 提交审核采纳反馈
```

### 2.7 useLayoutPrefs — 布局偏好

**文件**: `composables/useLayoutPrefs.ts` (49 行)

- 持久化侧边栏折叠状态到 localStorage
- 页面刷新后恢复

### 2.8 usePagination — 通用分页

**文件**: `composables/usePagination.ts` (35 行)

- 通用客户端分页逻辑
- 数据源变化时自动重置页码

---

## 三、路由守卫与认证流程

### 3.1 middleware/auth.ts

```typescript
export default defineNuxtRouteMiddleware(async (to) => {
  // 0. SSR 端没有 localStorage / token，认证检查仅在客户端执行
  if (import.meta.server) return

  // 1. 恢复认证状态（同步，从 localStorage）
  restore()

  // 2. token 不存在时，先尝试用 refresh_token 异步恢复
  if (!isAuthenticated) {
    await tryRestoreAsync()
  }

  // 3. /login 页面：已认证则跳转 /overview，否则放行
  if (to.path === '/login') return isAuthenticated ? navigateTo('/overview') : undefined

  // 4. 仍未认证 → 重定向 /login
  if (!isAuthenticated) return navigateTo('/login')

  // 5. 第一层：系统角色级别权限检查（hasRoleAccess）
  //    /overview、/settings 始终放行
  //    /admin/system → 需要 system_admin
  //    /admin/tenant → 需要 tenant_admin
  //    /dashboard、/cron、/archive → 需要 business

  // 6. 第二层：基于后端 menus（org_roles.page_permissions）的细粒度检查
  //    /overview 和 /settings 始终放行，不依赖 menus
  //    menus 未加载时放行（降级）
})
```

### 3.2 完整的认证流程图

```
用户访问任意页面
       ↓
  middleware/auth.ts
       ↓
  SSR 端？ ─── 是 → 直接放行（服务端无 localStorage）
       ↓ 否（客户端）
  restore() 同步恢复 token
       ↓
  有 token？ ─── 否 → tryRestoreAsync()（用 refresh_token 换新 token）
       ↓ 是            ↓
       ↓          （无论成功与否继续）
       ↓←──────────────┘
  是否 /login？ ─── 是 → 已认证跳 /overview，否则放行
       ↓ 否
  有 token？ ─── 否 → 重定向 /login
       ↓ 是
  第一层：系统角色级权限检查 ─── 无权 → 重定向 /overview
       ↓ 有权
  第二层：menus 细粒度检查
       ↓
  路径在 menus 中？ ─── 否 → 重定向 /overview
       ↓ 是
  放行，渲染页面
```

---

## 四、布局与组件体系

### 4.1 default.vue 布局

- **顶栏 (AppHeader)**：Logo、面包屑、角色切换器、用户菜单、语言切换、主题切换
- **侧边栏 (AppSidebar)**：权限驱动的菜单，支持折叠
- **内容区**：嵌套的 NuxtPage
- **底栏**：用户信息卡片（头像、姓名、角色标签）

### 4.2 核心组件

| 组件 | 文件 | 功能 |
|------|------|------|
| AppHeader | `AppHeader.vue` | 顶部导航栏 |
| AppSidebar | `AppSidebar.vue` | 侧边栏菜单 |
| AuditPanel | `AuditPanel.vue` | AI审核结果展示面板 |
| RuleEditor | `RuleEditor.vue` | 规则编辑器 |
| RuleList | `RuleList.vue` | 规则列表展示 |
| CronHistory | `CronHistory.vue` | 定时任务执行历史 |
| SnapshotDetail | `SnapshotDetail.vue` | 审核快照详情 |

---

## 五、前端 API 接口预期

前端代码中已预置了以下 API 调用点（Mock 模式下被跳过）：

### 5.1 认证接口

| 方法 | 路径 | 请求体 | 响应 |
|------|------|--------|------|
| POST | `/api/auth/login` | `{ username, password, tenant_id }` | `{ access_token, refresh_token, user, roles, active_role, permissions }` |
| POST | `/api/auth/refresh` | `{ refresh_token }` | `{ access_token }` |
| POST | `/api/auth/logout` | Header: Bearer Token | — |
| GET | `/api/auth/menu` | Header: Bearer Token | `{ menus: MenuItem[] }` |
| PUT | `/api/auth/switch-role` | `{ role_id }` | `{ access_token, active_role, permissions, menus }` |
| GET | `/api/auth/me` | Header: Bearer Token | `MeResponse`（用户资料、组织角色、页面权限） |
| PUT | `/api/auth/change-password` | `{ current_password, new_password }` | — |
| PUT | `/api/auth/locale` | `{ locale }` | — |

### 5.2 审核业务接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/audit/todo` | 获取待审核流程列表 |
| POST | `/api/audit/execute` | 执行AI审核，body: `{ process_id }` |
| POST | `/api/audit/feedback` | 提交审核反馈，body: `{ process_id, adopted, action_taken }` |

### 5.3 组织人员接口（已实现）

`useOrgApi` composable 已完成从 Mock 到真实 API 的切换，不再依赖 `useMockData`。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tenant/org/departments` | 获取部门列表 |
| POST | `/api/tenant/org/departments` | 创建部门 |
| PUT | `/api/tenant/org/departments/:id` | 更新部门 |
| DELETE | `/api/tenant/org/departments/:id` | 删除部门 |
| GET | `/api/tenant/org/roles` | 获取角色列表 |
| POST | `/api/tenant/org/roles` | 创建角色 |
| PUT | `/api/tenant/org/roles/:id` | 更新角色 |
| DELETE | `/api/tenant/org/roles/:id` | 删除角色 |
| GET | `/api/tenant/org/members` | 获取成员列表 |
| POST | `/api/tenant/org/members` | 创建成员 |
| PUT | `/api/tenant/org/members/:id` | 更新成员 |
| DELETE | `/api/tenant/org/members/:id` | 删除成员 |

统一响应格式：`{ code: number, message: string, data: T }`，`code === 0` 表示成功。

### 5.4 其他接口（页面中暗含，需后端实现）

| 领域 | 预期接口 |
|------|----------|
| 定时任务 | CRUD 定时任务、执行历史查询 |
| 归档复盘 | 归档流程查询、触发合规复核 |
| 租户管理 | 租户 CRUD、配置管理 |
| 规则管理 | 审核规则 CRUD、审核尺度预设 |
| 数据信息 | 审核/定时/归档日志查询 |
| 用户偏好 | 用户个性化配置查询 |
| 系统设置 | OA连接/AI模型/平台配置 CRUD |
| 仪表盘 | 各类统计数据聚合查询 |

---

## 六、关键设计模式

### 6.1 Mock 模式隔离

所有与后端交互的逻辑都通过 `isMockMode` 判断：

```typescript
if (isMockMode.value) {
  // 使用 useMockData() 返回的模拟数据
} else {
  // 调用后端 API
}
```

**后端开发策略**：只需实现 API 端点，返回与 Mock 数据相同结构的 JSON，前端无需修改即可切换。

### 6.2 权限驱动UI

UI 渲染完全由权限状态驱动：
- 侧边栏菜单：`useSidebarMenu` 根据 `userPermissions` 动态生成
- 路由守卫：`middleware/auth.ts` 阻止越权访问
- 页面内组件：通过 `v-if` 检查权限决定显示/隐藏

### 6.3 角色上下文切换

用户可能有多个角色（如同时是系统管理员+租户管理员+业务用户）：
- **AppHeader** 中提供角色切换下拉
- 切换后重新生成菜单，权限组变更为当前角色的组
- 不刷新页面，通过 Vue 响应式自动更新 UI

### 6.4 数据层级关系

```
系统级
  └── OADatabaseConnection[] (全局OA数据库连接)
  └── AIModelConfig[] (全局AI模型)
  └── SystemGeneralConfig (平台配置)

租户级
  └── TenantInfo (租户信息+AI配置)
  └── ProcessAuditConfig[] (审核规则配置)
  └── ArchiveReviewConfig[] (归档复盘配置)
  └── Department[] (部门)
  └── OrgRole[] (业务角色)
  └── OrgMember[] (组织成员)
  └── CronTaskTypeConfig[] (定时任务类型配置)
  └── StrictnessPromptPreset[] (审核尺度预设)

用户级
  └── UserPersonalConfig (用户偏好覆盖)
  └── UserDashboardPrefs (仪表盘配置)
  └── SecurityInfo (密码/登录历史)
  └── LocalePrefs (语言偏好)
```
