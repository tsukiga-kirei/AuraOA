# 需求文档

## 简介

本需求文档基于已确认的设计文档，定义 OA 智审平台 Phase 1 核心子系统——租户·组织·认证·权限（tenant-org-auth）的功能需求。涵盖多租户架构、JWT 认证流程、RBAC 权限控制、组织人员管理、前端接口迁移及数据库脚本组织等方面。

## 术语表

- **Go_Service**: Go (Gin + GORM) 后端服务，处理所有 HTTP 请求、业务逻辑和数据持久化
- **Auth_Handler**: 认证处理器，负责登录、登出、Token 刷新、角色切换、菜单获取
- **Org_Handler**: 组织人员处理器，负责部门 CRUD、组织角色 CRUD、成员 CRUD
- **Tenant_Handler**: 租户管理处理器，负责系统管理员对租户的 CRUD 操作
- **JWT_Middleware**: JWT 认证中间件，从请求头提取并验证 Bearer Token
- **Tenant_Middleware**: 租户上下文中间件，根据 JWT Claims 注入 tenant_id 到请求上下文
- **Role_Middleware**: 角色校验中间件，验证当前用户的 active_role 是否匹配所需角色
- **Repository**: 数据访问层，提供 WithTenant 方法实现行级租户隔离
- **Frontend**: Nuxt 3 前端应用，通过 useOrgApi composable 调用后端 API
- **Redis**: 缓存服务，用于 Token 黑名单、会话缓存和租户配置缓存
- **UserRole**: 系统角色（system_admin / tenant_admin / business），决定用户可访问的功能区域
- **OrgRole**: 组织角色，决定 business 用户可访问的具体页面，通过 page_permissions 配置
- **access_token**: JWT 访问令牌，有效期 2 小时，包含用户身份和权限信息
- **refresh_token**: JWT 刷新令牌，有效期 7 天，用于刷新 access_token
- **JTI**: JWT Token ID，用于标识唯一 Token 并支持黑名单机制

## 需求

### 需求 1：用户登录认证

**用户故事:** 作为平台用户，我希望通过用户名和密码登录系统，以便获取认证令牌并访问对应权限的功能。

#### 验收标准

1. WHEN 用户提交包含 username、password、tenant_id 和 preferred_role 的登录请求, THE Auth_Handler SHALL 验证用户凭证并返回包含 access_token、refresh_token、user 信息、roles 列表、active_role 和 permissions 的响应
2. WHEN 用户提交的密码与数据库中的 bcrypt 哈希不匹配, THE Auth_Handler SHALL 返回错误码 40103（用户名或密码错误）并将该用户的 login_fail_count 递增 1
3. WHEN 用户登录成功, THE Auth_Handler SHALL 将该用户的 login_fail_count 重置为 0 并清除 locked_until
4. WHILE 用户的 login_fail_count 大于等于 5 且 locked_until 晚于当前时间, THE Auth_Handler SHALL 拒绝登录请求并返回错误码 40104（账户被锁定）
5. WHEN 用户的 login_fail_count 达到 5, THE Auth_Handler SHALL 设置 locked_until 为当前时间加 15 分钟
6. WHEN 用户的 status 为 disabled, THE Auth_Handler SHALL 拒绝登录请求并返回错误码 40105（账户已被禁用）
7. WHEN 登录请求中的 tenant_id 对应的租户不存在或 status 为 inactive, THE Auth_Handler SHALL 返回错误码 40106（租户不存在或已停用）
8. WHEN 用户在指定租户中没有 user_role_assignment 记录, THE Auth_Handler SHALL 返回错误码 40107（用户在该租户无角色分配）
9. WHEN 登录成功, THE Auth_Handler SHALL 生成有效期为 2 小时的 access_token 和有效期为 7 天的 refresh_token
10. WHEN 登录请求包含 preferred_role, THE Auth_Handler SHALL 按优先级选择 active_role：preferred_role 匹配 > system_admin > tenant_admin > business
11. WHEN 登录成功, THE Auth_Handler SHALL 向 login_history 表写入一条登录记录并在 Redis 中缓存 session:{user_id}

### 需求 2：Token 管理

**用户故事:** 作为已登录用户，我希望系统能刷新即将过期的令牌并在登出时使令牌失效，以便保持会话安全。

#### 验收标准

1. WHEN 用户提交有效的 refresh_token 到刷新接口, THE Auth_Handler SHALL 验证 refresh_token 并返回新的 access_token
2. WHEN 用户提交的 refresh_token 的 JTI 存在于 Redis 黑名单中, THE Auth_Handler SHALL 拒绝刷新请求并返回错误码 40102（令牌已被吊销）
3. WHEN 用户调用登出接口, THE Auth_Handler SHALL 将 access_token 和 refresh_token 的 JTI 加入 Redis 黑名单并删除 Redis 中的会话缓存
4. WHEN 请求携带的 Token 的 JTI 存在于 Redis 黑名单中, THE JWT_Middleware SHALL 拒绝该请求并返回错误码 40102（认证令牌已失效）

### 需求 3：角色切换

**用户故事:** 作为拥有多个角色的用户，我希望在不重新登录的情况下切换角色，以便快速访问不同功能区域。

#### 验收标准

1. WHEN 用户提交角色切换请求并指定目标 role_id, THE Auth_Handler SHALL 验证该 role_id 存在于当前用户的 user_role_assignments 中
2. WHEN 角色切换验证通过, THE Auth_Handler SHALL 生成包含新 active_role 的 access_token、将旧 Token 的 JTI 加入 Redis 黑名单并更新 Redis 会话缓存
3. WHEN 目标角色绑定的 tenant_id 与当前角色不同, THE Auth_Handler SHALL 在新 Token 中反映新的租户上下文
4. IF 目标 role_id 不存在于当前用户的 user_role_assignments 中, THEN THE Auth_Handler SHALL 返回错误码 40108（角色切换失败）

### 需求 4：JWT 认证中间件

**用户故事:** 作为系统开发者，我希望所有受保护的 API 请求都经过 JWT 认证验证，以便确保只有合法用户能访问系统资源。

#### 验收标准

1. WHEN 请求未携带 Authorization Header 或 Bearer Token 为空, THE JWT_Middleware SHALL 返回 HTTP 401 和错误码 40100（未提供认证令牌）
2. WHEN 请求携带的 Token 解析失败或已过期, THE JWT_Middleware SHALL 返回 HTTP 401 和错误码 40101（认证令牌无效或已过期）
3. WHEN Token 验证通过, THE JWT_Middleware SHALL 将 jwt_claims、user_id 和 username 注入到 gin.Context 中供后续处理器使用

### 需求 5：租户上下文隔离

**用户故事:** 作为租户用户，我希望系统自动隔离不同租户的数据，以便确保我只能访问本租户的数据。

#### 验收标准

1. WHILE 当前用户的 active_role 为 system_admin, THE Tenant_Middleware SHALL 从请求查询参数 tenant_id 获取目标租户并设置 is_system_admin 为 true
2. WHILE 当前用户的 active_role 不是 system_admin, THE Tenant_Middleware SHALL 从 JWT Claims 的 ActiveRole.TenantID 获取 tenant_id
3. WHEN Repository 的 WithTenant 方法被调用且 tenant_id 非空, THE Repository SHALL 返回自动附加 WHERE tenant_id = ? 条件的数据库查询实例
4. WHEN Repository 的 WithTenant 方法被调用且 tenant_id 为空, THE Repository SHALL 返回无租户过滤的数据库查询实例

### 需求 6：角色权限校验

**用户故事:** 作为系统开发者，我希望不同角色的用户只能访问其权限范围内的 API，以便实现细粒度的访问控制。

#### 验收标准

1. WHEN 用户访问需要特定角色的 API 且其 active_role 不在允许的角色列表中, THE Role_Middleware SHALL 返回 HTTP 403 和错误码 40300（权限不足）
2. THE Go_Service SHALL 将租户管理 API（/api/tenant/*）限制为 tenant_admin 角色访问
3. THE Go_Service SHALL 将系统管理 API（/api/admin/*）限制为 system_admin 角色访问
4. WHEN business 用户请求菜单, THE Auth_Handler SHALL 查询该用户的 OrgRole 的 page_permissions 并返回合并后的可访问菜单列表


### 需求 7：部门管理

**用户故事:** 作为租户管理员，我希望管理本租户的部门结构，以便组织人员归属清晰。

#### 验收标准

1. WHEN 租户管理员请求部门列表, THE Org_Handler SHALL 返回当前租户下所有部门的列表（包含 id、name、parent_id、manager、sort_order）
2. WHEN 租户管理员提交包含 name 的创建部门请求, THE Org_Handler SHALL 在当前租户下创建部门记录并返回完整部门信息
3. WHEN 租户管理员提交部门更新请求, THE Org_Handler SHALL 更新指定部门的 name、parent_id、manager、sort_order 字段
4. WHEN 租户管理员删除部门且该部门下存在成员, THE Org_Handler SHALL 拒绝删除并返回错误提示
5. IF 请求操作的部门不属于当前租户, THEN THE Org_Handler SHALL 返回错误码 40400（资源不存在）

### 需求 8：组织角色管理

**用户故事:** 作为租户管理员，我希望管理组织角色并配置每个角色的页面权限，以便控制业务用户可访问的功能页面。

#### 验收标准

1. WHEN 租户管理员请求角色列表, THE Org_Handler SHALL 返回当前租户下所有组织角色（包含 id、name、description、page_permissions、is_system）
2. WHEN 租户管理员提交包含 name 和 page_permissions 的创建角色请求, THE Org_Handler SHALL 在当前租户下创建组织角色记录
3. WHEN 租户管理员更新角色的 page_permissions, THE Org_Handler SHALL 更新该角色的页面权限配置
4. WHILE 组织角色的 is_system 为 true, THE Org_Handler SHALL 拒绝删除该角色
5. IF 请求操作的角色不属于当前租户, THEN THE Org_Handler SHALL 返回错误码 40400（资源不存在）

### 需求 9：组织成员管理

**用户故事:** 作为租户管理员，我希望管理组织成员（含自动创建用户账号），以便将人员纳入租户的组织体系。

#### 验收标准

1. WHEN 租户管理员请求成员列表, THE Org_Handler SHALL 返回当前租户下所有成员信息（包含 user 详情、department、roles 列表、position、status）
2. WHEN 租户管理员提交创建成员请求且 username 在 users 表中不存在, THE Org_Handler SHALL 先创建 users 记录（含 bcrypt 密码哈希）再创建 org_members 和 org_member_roles 记录
3. WHEN 租户管理员提交创建成员请求且 username 在 users 表中已存在, THE Org_Handler SHALL 使用现有 user_id 创建 org_members 和 org_member_roles 记录
4. WHEN 创建成员时 role_ids 包含 ROLE-003（租户管理员角色）, THE Org_Handler SHALL 自动创建 tenant_admin 类型的 UserRoleAssignment
5. WHEN 创建成员, THE Org_Handler SHALL 默认创建 business 类型的 UserRoleAssignment
6. IF 该用户在当前租户中已存在 org_member 记录, THEN THE Org_Handler SHALL 返回错误码 40900（资源冲突）
7. IF 创建成员请求中的 department_id 不存在于当前租户, THEN THE Org_Handler SHALL 返回错误码 40001（参数校验失败）
8. IF 创建成员请求中的 role_ids 包含不属于当前租户的角色 ID, THEN THE Org_Handler SHALL 返回错误码 40001（参数校验失败）
9. WHEN 租户管理员更新成员信息, THE Org_Handler SHALL 更新 department_id、position、status 和 role_ids 关联
10. WHEN 租户管理员删除成员, THE Org_Handler SHALL 删除 org_members 记录、org_member_roles 关联和对应的 user_role_assignments 记录

### 需求 10：租户管理

**用户故事:** 作为系统管理员，我希望管理所有租户的基本信息和配置，以便控制平台的多租户运营。

#### 验收标准

1. WHEN 系统管理员请求租户列表, THE Tenant_Handler SHALL 返回所有租户的列表（包含 id、name、code、status、oa_type、token_quota、token_used 等字段）
2. WHEN 系统管理员提交包含 name、code 的创建租户请求, THE Tenant_Handler SHALL 创建租户记录并返回完整租户信息
3. IF 创建租户时 code 已存在, THEN THE Tenant_Handler SHALL 返回错误码 40900（资源冲突）
4. WHEN 系统管理员更新租户信息, THE Tenant_Handler SHALL 更新租户的 name、status、ai_config、token_quota 等可修改字段
5. WHEN 系统管理员请求租户统计信息, THE Tenant_Handler SHALL 返回该租户的成员数、部门数、角色数等统计数据

### 需求 11：统一响应格式与错误处理

**用户故事:** 作为前端开发者，我希望后端 API 返回统一格式的响应，以便前端能一致地处理成功和错误情况。

#### 验收标准

1. THE Go_Service SHALL 对所有 API 响应使用统一的 JSON 格式：{code, message, data, trace_id}
2. WHEN 请求处理成功, THE Go_Service SHALL 返回 code 为 0、message 为 "success" 的响应
3. WHEN 请求处理失败, THE Go_Service SHALL 返回对应的错误码（40001-50002 范围）和描述性错误消息
4. IF 服务器发生未捕获的异常, THEN THE Go_Service SHALL 通过 Recovery 中间件捕获异常并返回错误码 50000（服务器内部错误）

### 需求 12：数据库脚本组织

**用户故事:** 作为开发者，我希望数据库的表结构迁移和种子数据分离管理，以便独立维护 DDL 和测试数据。

#### 验收标准

1. THE Go_Service SHALL 将数据库 DDL 迁移脚本存放在 db/migrations/ 目录下，每个迁移包含 up 和 down 两个文件
2. THE Go_Service SHALL 将种子数据脚本存放在 db/seeds/ 目录下，按编号顺序组织
3. THE Go_Service SHALL 在 migrations 中包含以下迁移：extensions 初始化、tenants 和 users 表、org 结构表（departments、org_roles、org_members、org_member_roles）、system_configs 表
4. THE Go_Service SHALL 在 seeds 中包含以下种子数据：租户数据、用户和角色分配数据、部门数据、组织角色数据、组织成员和成员角色关联数据、系统配置数据

### 需求 13：前端组织接口迁移

**用户故事:** 作为前端开发者，我希望将组织人员相关的 Mock 数据调用迁移为真实 API 调用，以便前端能与后端实际交互。

#### 验收标准

1. THE Frontend SHALL 新增 useOrgApi.ts composable，封装部门 CRUD、角色 CRUD、成员 CRUD 的 API 调用
2. WHEN useOrgApi 调用 API 失败, THE Frontend SHALL 回退到 Mock 数据作为后备
3. THE Frontend SHALL 保持 NUXT_PUBLIC_MOCK_MODE 环境变量为 true 不变
4. THE Frontend SHALL 在 useSidebarMenu.ts 中将 mockOrgMembers 和 mockOrgRoles 的直接引用替换为 useOrgApi 提供的响应式数据
5. THE Frontend SHALL 在 middleware/auth.ts 中将 Mock 数据引用替换为 useOrgApi 提供的权限数据
6. THE Frontend SHALL 在 admin/tenant/org.vue 页面中对接真实 API 实现部门、角色、成员的 CRUD 操作
7. THE Frontend SHALL 保留 useMockData.ts 中的所有 Mock 数据不删除，作为开发模式后备和其他模块的数据源

### 需求 14：安全策略

**用户故事:** 作为安全负责人，我希望系统实施密码安全、令牌安全和跨域安全策略，以便保护用户数据和系统安全。

#### 验收标准

1. THE Go_Service SHALL 使用 bcrypt（cost=12）对所有用户密码进行哈希存储
2. THE Go_Service SHALL 实施 JWT 双令牌机制：access_token 有效期 2 小时、refresh_token 有效期 7 天
3. THE Go_Service SHALL 通过 CORS 中间件限制允许的来源域名
4. THE Go_Service SHALL 通过请求限流中间件防止暴力破解攻击
5. WHEN 系统管理员查看其他租户数据, THE Go_Service SHALL 允许查看但拒绝修改业务数据

### 需求 15：性能与缓存

**用户故事:** 作为系统运维人员，我希望系统合理使用缓存和连接池，以便保证系统在正常负载下的响应性能。

#### 验收标准

1. THE Go_Service SHALL 使用 Redis 缓存 Token 黑名单，避免每次请求查询数据库
2. THE Go_Service SHALL 将用户会话缓存的 TTL 设置为 2 小时
3. THE Go_Service SHALL 将租户配置缓存的 TTL 设置为 5 分钟
4. THE Go_Service SHALL 配置数据库连接池最大连接数为 50、空闲连接数为 10
5. THE Go_Service SHALL 对所有包含 tenant_id 的外键字段建立数据库索引
