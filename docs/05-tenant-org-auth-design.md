# OA 智审平台 — 租户·组织·认证·权限 详细设计文档

> 文档版本：v1.0 | 更新日期：2026-03-02  
> 本文档是 Phase 1 开发的核心指南，详细解析多租户架构、组织人员管理、登录认证、权限控制的完整逻辑。

---

## 一、概念模型：五种角色身份

本系统中存在 **五种关键身份**，它们的定义、权限边界和交互逻辑各不相同：

### 1.1 身份定义

```
┌─────────────────────────────────────────────────────┐
│                    系统管理员                         │
│  (system_admin)                                     │
│  · 全局角色，不绑定任何租户（可有多个用户）              │
│  · 管理所有租户、系统设置、全局监控                      │
│  · 可以查看所有租户数据，但不直接操作业务                 │
│  · 当前模拟数据中有3个: admin(陈刚),                  │
│    sysadmin2(周敏), sysadmin3(吴强)                  │
├─────────────────────────────────────────────────────┤
│              租户管理员 (tenant_admin)                │
│  · 绑定到特定租户                                    │
│  · 管理该租户的规则、组织、数据、用户偏好                 │
│  · 一个用户可以是多个租户的管理员                       │
├─────────────────────────────────────────────────────┤
│              业务用户 (business)                     │
│  · 绑定到特定租户                                    │
│  · 使用审核工作台、定时任务、归档复盘                    │
│  · 权限受「组织角色」控制                              │
├─────────────────────────────────────────────────────┤
│              审计管理员 (ROLE-002)                    │
│  · 实际上是组织角色(OrgRole)的一种                     │
│  · 在 business 权限组内，拥有更多页面权限               │
│  · 可使用归档复盘和定时任务                            │
├─────────────────────────────────────────────────────┤
│              租户管理员角色 (ROLE-003)                 │
│  · 组织角色(OrgRole)的一种                           │
│  · 拥有前台+后台管理的全部页面权限                      │
│  · 与系统级 tenant_admin 角色的关系见下文               │
└─────────────────────────────────────────────────────┘
```

### 1.2 两层角色体系的关系

系统存在 **两层角色体系**，嵌套使用：

#### 第一层：系统角色 (UserRole / UserRoleAssignment)

| 类型 | 存储位置 | 说明 |
|------|----------|------|
| `system_admin` | `user_role_assignments` 表 | 全局角色，`tenant_id = NULL` |
| `tenant_admin` | `user_role_assignments` 表 | 租户管理角色，绑定 `tenant_id` |
| `business` | `user_role_assignments` 表 | 业务用户角色，绑定 `tenant_id` |

- 决定用户可以访问哪些 **功能区域**（业务前台/租户后台/系统后台）
- 一个用户可以拥有多个系统角色分配
- 登录时选择激活哪个角色

#### 第二层：组织角色 (OrgRole)

| 类型 | 存储位置 | 说明 |
|------|----------|------|
| ROLE-001 业务用户 | `org_roles` 表 | 基本前台权限 |
| ROLE-002 审计管理员 | `org_roles` 表 | 扩展前台权限 |
| ROLE-003 租户管理员 | `org_roles` 表 | 完整前后台权限 |
| 自定义角色... | `org_roles` 表 | 租户自定义的角色 |

- 决定 `business` 类型用户可以访问哪些 **具体页面**
- 通过 `org_member_roles` 多对多关联到成员
- 每个 OrgRole 有 `page_permissions` 数组

### 1.3 两层角色协作流程

```
用户登录选择「业务用户」入口
    ↓
系统设置 activeRole.role = 'business'
    ↓
调用 GET /api/auth/menu 获取当前角色可访问的菜单列表
    ↓
侧边栏生成：根据 menus 中的 path 过滤菜单项
    ↓
例如：张明有 ROLE-001 + ROLE-002
      → GetMenu 返回 [/overview, /dashboard, /cron, /archive, /settings]
      → 侧边栏显示全部业务功能

例如：李芳只有 ROLE-001
      → GetMenu 返回 [/overview, /dashboard, /settings]
      → 侧边栏只显示仪表盘和工作台
```

> 注：侧边栏菜单过滤统一通过 GetMenu API 驱动，不再在前端直接查询 org_members/org_roles。
> 后端 GetMenu 根据用户的系统角色和组织角色综合计算可访问页面后返回。

---

## 二、多租户架构详解

### 2.1 租户隔离模型

本系统采用 **共享数据库 + 行级隔离** 的多租户模型：

```
┌────────────────────┐
│   PostgreSQL 实例   │
│                    │
│  ┌──────────────┐  │
│  │   tenants     │  │
│  │   T-001       │  │  ← 示例集团总部
│  │   T-002       │  │  ← 华东分公司
│  │   T-003       │  │  ← 测试租户
│  └──────────────┘  │
│                    │
│  所有业务表都有     │
│  tenant_id 字段     │
│  实现行级数据隔离   │
└────────────────────┘
```

### 2.2 租户上下文注入

#### Go 中间件实现

```go
// middleware/tenant.go
func TenantMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从 JWT Claims 中获取 tenant_id
        claims := c.MustGet("jwt_claims").(*JWTClaims)
        
        if claims.ActiveRole.Role == "system_admin" {
            // 系统管理员：从查询参数获取目标租户
            tenantID := c.Query("tenant_id")
            if tenantID != "" {
                c.Set("tenant_id", tenantID)
            }
            c.Set("is_system_admin", true)
        } else {
            // 非系统管理员：从角色分配获取租户
            c.Set("tenant_id", claims.ActiveRole.TenantID)
            c.Set("is_system_admin", false)
        }
        
        c.Next()
    }
}
```

#### Repository 层自动过滤

```go
// repository/base_repo.go
func (r *BaseRepo) WithTenant(ctx *gin.Context) *gorm.DB {
    tenantID := ctx.GetString("tenant_id")
    if tenantID == "" {
        return r.db
    }
    return r.db.Where("tenant_id = ?", tenantID)
}
```

### 2.3 租户数据模型

```typescript
// 前端 TenantInfo 结构（后端需要完全匹配）
interface TenantInfo {
  id: string                    // UUID
  name: string                  // "示例集团总部"
  code: string                  // "DEMO_HQ" — 登录时使用
  oa_type: string               // "weaver_e9"
  oa_db_connection_id: string   // 关联系统级OA数据库连接
  token_quota: number           // Token额度上限
  token_used: number            // 已消耗Token
  max_concurrency: number       // 最大并发数
  status: 'active' | 'inactive'
  created_at: string
  contact_name: string
  contact_email: string
  contact_phone: string
  description: string
  ai_config: {
    default_provider: string    // "本地部署"
    default_model: string       // "Qwen2.5-72B"
    fallback_provider: string   // 备用
    fallback_model: string
    max_tokens_per_request: number
    temperature: number
    timeout_seconds: number
    retry_count: number
  }
  log_retention_days: number
  data_retention_days: number
  allow_custom_model: boolean
  sso_enabled: boolean
  sso_endpoint: string
  tenant_admin_id?: string      // 租户管理员用户名（用于反向关联）
}
```

### 2.4 租户与 OA 系统的关系

```
OADatabaseConnection (系统级配置)
   │
   ├── OADB-001 "总部泛微E9数据库"
   │       ↑
   │   Tenant T-001 "示例集团总部"
   │       oa_db_connection_id = "OADB-001"
   │
   ├── OADB-002 "华东分公司E9数据库"
   │       ↑
   │   Tenant T-002 "华东分公司"
   │       oa_db_connection_id = "OADB-002"
   │
   └── OADB-003 "测试环境数据库"
           ↑
       Tenant T-003 "测试租户"
           oa_db_connection_id = "OADB-003"
```

**设计要点**：
- OA 数据库连接在 **系统级** 统一管理
- 租户通过 `oa_db_connection_id` **引用** 连接配置
- 同一个 OA 连接可被多个租户共享（但当前实现为一对一）
- Go Service 根据租户的 OA 连接配置动态创建数据库连接池

---

## 三、组织人员管理

### 3.1 数据模型关系图

```
Tenant (T-001 示例集团总部)
  │
  ├── Departments (部门)
  │   ├── D-001 研发部 (manager: 张明, 12人)
  │   ├── D-002 销售部 (manager: 周磊, 8人)
  │   ├── D-003 市场部 (manager: 陈伟, 6人)
  │   ├── D-004 人力资源部 (manager: 赵丽, 5人)
  │   ├── D-005 IT部 (manager: 王强, 7人)
  │   ├── D-006 财务部 (manager: 张华, 4人)
  │   ├── D-007 行政部 (manager: 刘洋, 3人)
  │   └── D-008 法务部 (manager: 孙律, 2人)
  │
  ├── OrgRoles (组织角色)
  │   ├── ROLE-001 业务用户 [/overview, /dashboard, /settings] (系统角色)
  │   ├── ROLE-002 审计管理员 [/overview, /dashboard, /cron, /archive, /settings] (系统角色)
  │   └── ROLE-003 租户管理员 [全部页面] (系统角色)
  │
  └── OrgMembers (组织成员) — 关联 User + Department + OrgRole[]
      ├── M-001 张明(zhangming) → D-001研发部 → [ROLE-001, ROLE-002]
      ├── M-002 李芳(lifang) → D-002销售部 → [ROLE-001]
      ├── M-003 王强(wangqiang) → D-005 IT部 → [ROLE-001, ROLE-002]
      ├── M-004 赵丽(zhaoli) → D-004人力资源部 → [ROLE-001]
      ├── M-005 陈伟(chenwei) → D-003市场部 → [ROLE-001]
      ├── M-006 刘洋(liuyang) → D-007行政部 → [ROLE-001]
      ├── M-007 张华(zhanghua) → D-006财务部 → [ROLE-001, ROLE-002]
      ├── M-008 孙律(sunlv) → D-008法务部 → [ROLE-001]
      ├── M-009 周磊(zhoulei) → D-002销售部 → [ROLE-001]
      ├── M-010 赵伟(tenantadmin) → D-005 IT部 → [ROLE-001, ROLE-003]
      ├── M-011 陈刚(admin) → D-005 IT部 → [ROLE-001, ROLE-002, ROLE-003]
      └── M-012 测试用户(user) → D-001研发部 → [ROLE-001] (disabled)
```

### 3.2 OrgMember 与 User 的关系

```
User (users 表)                    OrgMember (org_members 表)
┌──────────────┐                   ┌─────────────────────┐
│ id: UUID     │                   │ id: UUID            │
│ username     │  1:N (跨租户)     │ tenant_id: UUID     │
│ password_hash│ ◄─────────────── │ user_id: UUID       │
│ display_name │                   │ department_id: UUID │
│ email        │                   │ position            │
│ status       │                   │ status              │
└──────────────┘                   └─────────────────────┘
                                          │
                                   org_member_roles (多对多)
                                          │
                                   ┌──────▼──────┐
                                   │ OrgRole     │
                                   │ role_id     │
                                   │ page_perms  │
                                   └─────────────┘
```

**关键设计**：
- `users` 表是全局的，一个 User 可以在多个租户中有 OrgMember
- `org_members` 表是租户级的，每个租户内一个用户只有一条 OrgMember 记录
- `user_role_assignments` 连接 User 和系统角色
- `org_member_roles` 连接 OrgMember 和组织角色

### 3.3 创建成员的业务逻辑

```
POST /api/tenant/org/members
{
  "username": "newuser",
  "display_name": "新用户",
  "password": "123456",
  "department_id": "D-001",
  "role_ids": ["ROLE-001"],
  "email": "new@example.com",
  "phone": "13800138000",
  "position": "工程师"
}

流程：
1. 检查 username 在 users 表中是否已存在
   a. 已存在 → 使用现有 user_id
   b. 不存在 → 创建 users 记录（含密码哈希）
2. 检查该 user 在当前租户是否已有 org_member
   a. 已有 → 返回冲突错误
   b. 没有 → 继续
3. 创建 org_members 记录（关联 user_id + tenant_id + department_id）
4. 创建 org_member_roles 记录（关联 member_id + role_ids）
5. 根据 role_ids 判断是否需要创建 user_role_assignments：
   a. 如果包含 ROLE-003（租户管理员角色），自动创建 tenant_admin 的 UserRoleAssignment
   b. 默认创建 business 的 UserRoleAssignment
6. 返回完整的成员信息
```

### 3.4 角色页面权限配置

租户管理员可以创建和编辑 OrgRole，为每个角色分配可访问的页面：

```typescript
// 前端可分配的页面列表（来自 org.vue 中的 allPages 配置）
const allPages = [
  { path: '/overview', label: '仪表盘' },
  { path: '/dashboard', label: '审核工作台' },
  { path: '/cron', label: '定时任务' },
  { path: '/archive', label: '归档复盘' },
  { path: '/settings', label: '个人设置' },
  { path: '/admin/tenant/rules', label: '规则配置' },
  { path: '/admin/tenant/org', label: '组织人员' },
  { path: '/admin/tenant/data', label: '数据信息' },
  { path: '/admin/tenant/user-configs', label: '用户偏好' },
]
```

---

## 四、认证流程完整设计

### 4.1 JWT Token 结构

#### Access Token Claims

```json
{
  "sub": "user-uuid-001",                  // 用户ID
  "username": "admin",
  "display_name": "陈刚",
  "active_role": {
    "id": "admin-r1",
    "role": "system_admin",
    "tenant_id": null,
    "tenant_name": null,
    "label": "系统管理员"
  },
  "permissions": ["system_admin"],
  "all_role_ids": ["admin-r1", "admin-r2", "admin-r3"],
  "jti": "unique-token-id",               // 用于黑名单
  "iat": 1709366096,
  "exp": 1709373296                        // 2小时后到期
}
```

#### Refresh Token Claims

```json
{
  "sub": "user-uuid-001",
  "jti": "refresh-token-id",
  "iat": 1709366096,
  "exp": 1709970896                        // 7天后到期
}
```

### 4.2 完整登录流程

```
客户端                          Go Service                      数据库
  │                               │                               │
  │ POST /api/auth/login          │                               │
  │ {username, password,          │                               │
  │  tenant_id, preferred_role}   │                               │
  │ ─────────────────────────────>│                               │
  │                               │                               │
  │                               │ 1. 查询 users 表              │
  │                               │ ──────────────────────────────>│
  │                               │ <──────────────────────────────│
  │                               │                               │
  │                               │ 2. bcrypt 验证密码             │
  │                               │                               │
  │                               │ 3. 检查账户锁定状态            │
  │                               │    (login_fail_count >= 5      │
  │                               │     && locked_until > NOW)     │
  │                               │                               │
  │                               │ 4. 查询 user_role_assignments  │
  │                               │ ──────────────────────────────>│
  │                               │ <──────────────────────────────│
  │                               │                               │
  │                               │ 5. 如果 tenant_id 非空:       │
  │                               │    验证 tenant.code 与请求匹配 │
  │                               │    验证用户在该租户有角色分配   │
  │                               │                               │
  │                               │ 6. 根据 preferred_role 选择    │
  │                               │    activeRole：               │
  │                               │    a. 优先匹配 preferred_role  │
  │                               │    b. 回退: sys > tenant > biz │
  │                               │                               │
  │                               │ 7. 生成 JWT Token 对           │
  │                               │    (access + refresh)          │
  │                               │                               │
  │                               │ 8. 写入 login_history          │
  │                               │ ──────────────────────────────>│
  │                               │                               │
  │                               │ 9. 缓存会话到 Redis            │
  │                               │    session:{user_id}           │
  │                               │                               │
  │ <─────────────────────────────│                               │
  │ {access_token, refresh_token, │                               │
  │  user, roles, active_role,    │                               │
  │  permissions}                 │                               │
  │                               │                               │
  │ 存储到 localStorage           │                               │
  │ 跳转到 /overview              │                               │
```

### 4.3 角色切换流程

```
客户端                          Go Service                      Redis
  │                               │                               │
  │ PUT /api/auth/switch-role     │                               │
  │ {role_id: "admin-r2"}        │                               │
  │ Header: Bearer <token>       │                               │
  │ ─────────────────────────────>│                               │
  │                               │                               │
  │                               │ 1. 验证 JWT                   │
  │                               │ 2. 查找 user_role_assignments │
  │                               │    中 id = "admin-r2"         │
  │                               │ 3. 验证该分配属于当前用户      │
  │                               │ 4. 生成新 JWT: activeRole 变更 │
  │                               │ 5. 旧 token JTI 加入黑名单    │
  │                               │ ────────────────────────────> │
  │                               │ 6. 更新 Redis 会话缓存        │
  │                               │ ────────────────────────────> │
  │                               │                               │
  │ <─────────────────────────────│                               │
  │ {access_token(新),            │                               │
  │  active_role, permissions}    │                               │
  │                               │                               │
  │ 更新 localStorage             │                               │
  │ 重新生成菜单                   │                               │
```

### 4.4 Token 刷新流程

```
客户端                          Go Service
  │                               │
  │ access_token 即将过期           │
  │ (前端检测 exp - now < 5min)    │
  │                               │
  │ POST /api/auth/refresh        │
  │ {refresh_token}               │
  │ ─────────────────────────────>│
  │                               │
  │                               │ 1. 验证 refresh_token
  │                               │ 2. 检查 JTI 是否在黑名单
  │                               │ 3. 查询用户最新的角色分配
  │                               │ 4. 生成新的 access_token
  │                               │ 5. 返回
  │                               │
  │ <─────────────────────────────│
  │ {access_token(新), expires_in}│
```

### 4.5 登出流程

```
客户端                          Go Service                      Redis
  │                               │                               │
  │ POST /api/auth/logout         │                               │
  │ Header: Bearer <token>        │                               │
  │ ─────────────────────────────>│                               │
  │                               │ 1. 将 access_token JTI        │
  │                               │    加入黑名单                  │
  │                               │ ────────────────────────────> │
  │                               │ 2. 将 refresh_token JTI       │
  │                               │    加入黑名单                  │
  │                               │ ────────────────────────────> │
  │                               │ 3. 删除 Redis 会话             │
  │                               │ ────────────────────────────> │
  │                               │                               │
  │ <─────────────────────────────│                               │
  │ {success}                     │                               │
  │                               │                               │
  │ 清除 localStorage              │                               │
  │ 跳转 /login                    │                               │
```

---

## 五、权限校验中间件

### 5.1 中间件栈

```go
router.Use(
    middleware.Logger(),          // 请求日志
    middleware.Recovery(),        // 异常恢复
    middleware.CORS(),            // 跨域
    middleware.RateLimit(),       // 限流
    middleware.Tracing(),         // 链路追踪
)

// 需要认证的路由组
authed := router.Group("/api")
authed.Use(middleware.JWT())      // JWT 验证
authed.Use(middleware.Tenant())   // 租户上下文

// 需要特定角色的路由组
tenantAdmin := authed.Group("/tenant")
tenantAdmin.Use(middleware.RequireRole("tenant_admin"))

systemAdmin := authed.Group("/admin")
systemAdmin.Use(middleware.RequireRole("system_admin"))

systemConfig := authed.Group("/system")
systemConfig.Use(middleware.RequireRole("system_admin"))
```

### 5.2 JWT 认证中间件

```go
func JWT() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 从 Header 提取 token
        token := extractBearerToken(c)
        if token == "" {
            c.AbortWithStatusJSON(401, response.Error(40100, "未提供认证令牌"))
            return
        }

        // 2. 解析和验证 JWT
        claims, err := jwt.ParseToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, response.Error(40101, "认证令牌无效或已过期"))
            return
        }

        // 3. 检查 Token 黑名单（Redis）
        if isBlacklisted(claims.JTI) {
            c.AbortWithStatusJSON(401, response.Error(40102, "认证令牌已失效"))
            return
        }

        // 4. 注入 Claims 到上下文
        c.Set("jwt_claims", claims)
        c.Set("user_id", claims.Sub)
        c.Set("username", claims.Username)
        
        c.Next()
    }
}
```

### 5.3 角色校验中间件

```go
func RequireRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := c.MustGet("jwt_claims").(*JWTClaims)
        
        for _, r := range roles {
            if claims.ActiveRole.Role == r {
                c.Next()
                return
            }
        }

        c.AbortWithStatusJSON(403, response.Error(40300, "权限不足"))
    }
}
```

---

## 六、复杂场景处理

### 6.1 跨租户用户

**场景**：王刚(wanggang) 是华东分公司的租户管理员 + 总部的业务用户

```
User: wanggang
  └── UserRoleAssignments:
      ├── wg-r1: tenant_admin @ T-002 (华东分公司)
      └── wg-r2: business @ T-001 (示例集团总部)
```

**登录行为**：
- 王刚登录时选择"租户管理员"入口：
  - 在租户选择器中选择"华东分公司"
  - 系统激活 `wg-r1`，进入华东分公司的管理后台
  
- 王刚想切换到总部的业务用户身份：
  - 在 AppHeader 的角色切换下拉中选择"示例集团总部 · 业务用户"
  - 调用 `switchRole("wg-r2")`
  - 系统生成新 Token，tenant_id 变为 T-001
  - 菜单重新生成为业务用户菜单
  - 侧边栏显示业务功能

**后端实现要点**：
- 角色切换时必须重新生成 JWT（因为 tenant_id 变了）
- 新 Token 的权限必须反映目标角色
- 旧 Token 加入黑名单

### 6.2 超级管理员的多重身份

**场景**：陈刚(admin) 拥有系统管理员 + 总部租户管理员 + 总部业务用户

```
User: admin
  └── UserRoleAssignments:
      ├── admin-r1: system_admin (全局)
      ├── admin-r2: tenant_admin @ T-001
      └── admin-r3: business @ T-001
```

**登录行为**：
- 如果选择"系统管理员"入口登录：
  - 激活 `admin-r1`，不绑定任何租户
  - 可以管理所有租户和系统设置
  
- 如果选择"租户管理员"入口登录：
  - 需要选择租户（总部）
  - 激活 `admin-r2`，绑定 T-001
  - 可以管理总部的规则、组织等

- 运行中随时切换角色

### 6.3 密码安全策略

```go
// 登录失败处理
func (s *AuthService) HandleLoginFailure(user *User) {
    user.LoginFailCount++
    
    if user.LoginFailCount >= 5 {
        user.LockedUntil = time.Now().Add(15 * time.Minute)
        user.Status = "locked"
    }
    
    s.userRepo.Update(user)
}

// 登录成功处理
func (s *AuthService) HandleLoginSuccess(user *User) {
    user.LoginFailCount = 0
    user.LockedUntil = nil
    user.Status = "active"
    
    s.userRepo.Update(user)
}
```

### 6.4 OrgMember 状态与 User 状态的关系

```
User.status:
  - active: 用户账号正常
  - disabled: 用户账号被禁用（全局禁用，所有租户无法登录）
  - locked: 登录失败过多（临时锁定）

OrgMember.status:
  - active: 该成员在此租户中正常
  - disabled: 该成员在此租户中被禁用（其他租户不受影响）
```

**判断逻辑**：
```
用户能否登录？
  → User.status == 'active' && User.locked_until < NOW()

用户能否访问某租户？
  → user_role_assignments 中有该租户的有效分配
  → 对应的 OrgMember.status == 'active'
```

---

## 七、Phase 1 开发清单

### 7.1 后端 API 开发顺序

```
Week 1: 基础框架
  ☐ Go 项目初始化 (Gin + GORM + Viper + Zap)
  ☐ 配置管理 (环境变量 + 配置文件)
  ☐ 数据库连接 + 迁移 (000001-000004)
  ☐ 统一响应格式和错误码
  ☐ 健康检查 /api/health

Week 2: 认证系统
  ☐ POST /api/auth/login (含密码验证+角色分配)
  ☐ POST /api/auth/refresh
  ☐ POST /api/auth/logout
  ☐ PUT /api/auth/switch-role
  ☐ GET /api/auth/menu
  ☐ JWT 中间件
  ☐ Redis 集成 (Token黑名单+会话缓存)

Week 3: 组织人员
  ☐ 部门 CRUD
  ☐ 角色 CRUD
  ☐ 成员 CRUD (含用户创建逻辑)
  ☐ 成员-角色关联管理
  ☐ 权限校验中间件

Week 4: 对接前端
  ☐ CORS 配置
  ☐ 种子数据初始化
  ☐ 前端 mockMode 切换测试
  ☐ 前端 API 调用对接
  ☐ 端到端测试
```

### 7.2 前端改造要点

在后端 Phase 1 完成后，前端需要进行以下改造：

1. **`.env` 文件**：将 `NUXT_PUBLIC_MOCK_MODE` 设为 `false`
2. **`useAuth.ts`**：API 模式分支已预置，无需大改
3. **`useSidebarMenu.ts`**：✅ 已改为使用 `useAuth()` 的 `menus`（GetMenu API）驱动菜单过滤，不再依赖 `useOrgApi`
4. **`middleware/auth.ts`**：业务角色权限检查需从 API 获取，而非 mock 数据
5. **`useMockData.ts`**：保留为开发模式后备，不删除

### 7.3 测试策略

```
单元测试:
  - Service 层: 100% 核心逻辑覆盖
  - Repository 层: 使用 testcontainers 集成测试

集成测试:
  - 认证流程端到端
  - 权限隔离测试（跨租户访问应被拒绝）
  - 角色切换 + 菜单生成

压力测试:
  - 并发登录
  - Token 刷新（高频场景）
```

---

## 八、错误码设计

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 40001 | 参数校验失败 |
| 40100 | 未提供认证令牌 |
| 40101 | 令牌无效或已过期 |
| 40102 | 令牌已被吊销（黑名单） |
| 40103 | 用户名或密码错误 |
| 40104 | 账户被锁定 |
| 40105 | 账户已被禁用 |
| 40106 | 租户不存在或已停用 |
| 40107 | 用户在该租户无角色分配 |
| 40108 | 角色切换失败（目标角色不存在） |
| 40300 | 权限不足 |
| 40301 | 不允许跨租户访问 |
| 40400 | 资源不存在 |
| 40900 | 资源冲突（如用户名已存在） |
| 50000 | 服务器内部错误 |
| 50001 | 数据库错误 |
| 50002 | Redis 连接错误 |
| 50003 | 外部服务调用失败 |
