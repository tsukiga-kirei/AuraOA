# OA 智审平台 — 前端代码架构与交互逻辑详解

> 文档版本：v1.1 | 更新日期：2026-03-03  
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

**文件**: `composables/useAuth.ts` (236 行)

#### 状态管理

| 状态变量 | 类型 | 说明 | 持久化 |
|----------|------|------|--------|
| `token` | `string | null` | JWT 访问令牌 | localStorage `token` |
| `refreshToken` | `string | null` | 刷新令牌 | localStorage `refresh_token` |
| `userRole` | `UserRole` | 当前角色类型 | localStorage `user_role` |
| `allRoles` | `UserRoleAssignment[]` | 用户所有角色分配 | localStorage `all_roles` |
| `activeRole` | `UserRoleAssignment | null` | 当前激活的角色 | localStorage `active_role` |
| `userPermissions` | `PermissionGroup[]` | 当前权限组 | localStorage `user_permissions` |
| `currentUser` | `object | null` | 当前用户信息 | localStorage `current_user` |

#### 登录流程 (Mock 模式)

```
1. login({ username, password, tenant_id, preferred_role })
2. → 在 MOCK_USERS 中查找匹配的用户
3. → 生成 mock_token（前端模拟）
4. → setAllRoles(matched.roles) — 存储所有角色分配
5. → 根据 preferred_role 选择默认激活角色：
     a. 优先匹配用户选择的入口类型
     b. 回退：system_admin > tenant_admin > 第一个角色
6. → setActiveRole(defaultRole) — 设置权限为该角色的权限组
7. → 设置 currentUser 信息
8. → 所有状态写入 localStorage
9. → return true
```

#### 登录流程 (API 模式)

```
1. login({ username, password, tenant_id })
2. → POST /api/auth/login → 获取 { access_token, refresh_token, expires_in }
3. → 存储 token 到 state + localStorage
4. → return true
```

#### 角色切换

```typescript
switchRole(roleId: string): Promise<boolean>
// 1. 在 allRoles 中查找目标角色
// 2. setActiveRole(target) — 更新权限为目标角色的权限组
// 3. 重新生成菜单 getMockMenusByActiveRole(target)
```

#### 菜单生成

```typescript
getMockMenusByActiveRole(role: UserRoleAssignment): MockMenuItem[]
// 根据角色类型生成对应菜单：
// - business: 仪表盘、审核工作台、定时任务、归档复盘
// - tenant_admin: 规则配置、组织人员、数据信息、用户偏好
// - system_admin: 租户管理、系统设置
// - overview 始终显示
```

#### 会话恢复

```typescript
restore()
// 在页面刷新时从 localStorage 恢复所有认证状态（同步）
// 在 middleware/auth.ts 中被调用

tryRestoreAsync(): Promise<boolean>
// 异步恢复：当 access_token 丢失但 refresh_token 仍在时，
// 尝试调用 doRefreshToken() 换取新 token。
// 返回 true 表示恢复成功（token 已可用），false 表示无法恢复。
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

  // 2. /login 页面：已认证则跳转 /overview，否则放行
  if (to.path === '/login') return isAuthenticated ? navigateTo('/overview') : undefined

  // 3. 未认证 → 尝试用 refresh_token 异步恢复
  if (!isAuthenticated) {
    const restored = await tryRestoreAsync()
    if (!restored) return navigateTo('/login')
  }

  // 4. 第一层：系统角色级别权限检查（hasRoleAccess）
  //    /overview、/settings 始终放行
  //    /admin/system → 需要 system_admin
  //    /admin/tenant → 需要 tenant_admin
  //    /dashboard、/cron、/archive → 需要 business

  // 5. 第二层：基于后端 menus（org_roles.page_permissions）的细粒度检查
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
  是否 /login？ ─── 是 → 已认证跳 /overview，否则放行
       ↓ 否
  有 token？ ─── 否 → tryRestoreAsync()（用 refresh_token 换新 token）
       │                    ↓
       │              恢复成功？ ─── 否 → 重定向 /login
       ↓ 是                ↓ 是
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
| POST | `/api/auth/login` | `{ username, password, tenant_id, preferred_role? }` | `{ access_token, refresh_token, expires_in }` |
| GET | `/api/auth/menu` | Header: Bearer Token | `{ menus: MockMenuItem[] }` |

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
