# 实施计划：租户·组织·认证·权限子系统 (tenant-org-auth)

## 概述

基于设计文档，按增量方式构建 Go 后端服务、数据库脚本和前端接口迁移。从数据库脚本开始，逐步搭建 Go 服务骨架、实现核心模块，最后迁移前端组织相关接口。每个任务构建在前一个任务之上，确保无孤立代码。

## Tasks

- [x] 1. 创建数据库迁移脚本和种子数据
  - [x] 1.1 创建 db/migrations/ 目录下的 DDL 迁移脚本
    - 创建 `000001_init_extensions.up.sql` 和 `000001_init_extensions.down.sql`（启用 uuid-ossp / pgcrypto 扩展）
    - 创建 `000002_tenants_users.up.sql` 和 `000002_tenants_users.down.sql`（tenants、users、user_role_assignments、login_history 表）
    - 创建 `000003_org_structure.up.sql` 和 `000003_org_structure.down.sql`（departments、org_roles、org_members、org_member_roles 表）
    - 创建 `000004_system_configs.up.sql` 和 `000004_system_configs.down.sql`（system_configs 表）
    - 所有包含 tenant_id 的外键字段建立索引
    - _Requirements: 12.1, 12.3, 15.5_

  - [x] 1.2 创建 db/seeds/ 目录下的种子数据脚本
    - 创建 `001_tenants.sql`（演示租户数据）
    - 创建 `002_users.sql`（用户 + user_role_assignments 数据，密码使用 bcrypt 哈希）
    - 创建 `003_departments.sql`（部门数据）
    - 创建 `004_org_roles.sql`（3 个系统角色：审核员、审核主管、租户管理员）
    - 创建 `005_org_members.sql`（组织成员 + org_member_roles 关联）
    - 创建 `006_system_config.sql`（系统通用配置）
    - _Requirements: 12.2, 12.4_

- [x] 2. 搭建 Go 服务项目骨架
  - [x] 2.1 初始化 Go 模块和项目结构
    - 在 `go-service/` 下创建 `go.mod`，初始化模块
    - 创建目录结构：cmd/server/、internal/config/、internal/middleware/、internal/model/、internal/repository/、internal/service/、internal/handler/、internal/router/、internal/dto/、internal/pkg/
    - 创建 `cmd/server/main.go` 应用入口（初始化配置、数据库、Redis、路由、启动 HTTP 服务）
    - 创建 `config.yaml` 默认配置文件（数据库、Redis、JWT、服务端口等配置项）
    - _Requirements: 11.1, 15.4_

  - [x] 2.2 实现基础工具包
    - 创建 `internal/pkg/response/response.go`（统一响应格式：Success、Error、PageData）
    - 创建 `internal/pkg/errcode/errcode.go`（错误码常量定义：40001-50002）
    - 创建 `internal/pkg/jwt/jwt.go`（JWT 生成/解析，含 JWTClaims、ActiveRoleClaim 结构体）
    - 创建 `internal/pkg/hash/bcrypt.go`（bcrypt 密码哈希/验证，cost=12）
    - _Requirements: 11.1, 11.2, 11.3, 14.1, 14.2_

  - [x] 2.3 实现配置加载
    - 创建 `internal/config/config.go`（使用 Viper 加载 config.yaml，定义 Config 结构体）
    - 包含数据库连接池配置（最大 50 连接、空闲 10 连接）
    - 包含 Redis 连接配置
    - 包含 JWT 密钥和有效期配置
    - 包含 CORS 允许域名配置
    - _Requirements: 15.4_

- [x] 3. 实现数据模型和中间件
  - [x] 3.1 创建 GORM 数据模型
    - 创建 `internal/model/user.go`（User 结构体，含所有字段和 GORM 标签）
    - 创建 `internal/model/tenant.go`（Tenant 结构体，含 AIConfig jsonb 字段）
    - 创建 `internal/model/user_role_assignment.go`（UserRoleAssignment 结构体）
    - 创建 `internal/model/department.go`（Department 结构体，含 ParentID 自引用）
    - 创建 `internal/model/org_role.go`（OrgRole 结构体，含 PagePermissions jsonb）
    - 创建 `internal/model/org_member.go`（OrgMember 结构体，含 User/Department/Roles 关联）
    - 创建 `internal/model/login_history.go`（LoginHistory 结构体）
    - _Requirements: 1.1, 7.1, 8.1, 9.1, 10.1_

  - [x] 3.2 实现中间件栈
    - 创建 `internal/middleware/auth.go`（JWT 认证中间件：提取 Bearer Token、解析验证、黑名单检查、注入 Claims）
    - 创建 `internal/middleware/tenant.go`（租户上下文中间件：system_admin 从 query 获取 tenant_id，其他从 Claims 获取）
    - 创建 `internal/middleware/role.go`（角色校验中间件：RequireRole 函数）
    - 创建 `internal/middleware/cors.go`（CORS 中间件）
    - 创建 `internal/middleware/logger.go`（请求日志中间件，使用 zap）
    - 创建 `internal/middleware/recovery.go`（异常恢复中间件，返回错误码 50000）
    - _Requirements: 4.1, 4.2, 4.3, 5.1, 5.2, 6.1, 11.4, 14.3_

  - [x] 3.3 实现 Repository 基础层
    - 创建 `internal/repository/base_repo.go`（WithTenant 方法：tenant_id 非空时自动附加 WHERE 条件，为空时返回无过滤实例）
    - _Requirements: 5.3, 5.4_

- [x] 4. 检查点 - 确保项目骨架可编译
  - 确保 `go-service/` 下所有代码可编译通过，如有问题请向用户确认。

- [x] 5. 实现认证模块（Auth）
  - [x] 5.1 创建认证相关 DTO
    - 创建 `internal/dto/auth_dto.go`（LoginRequest、LoginResponse、RefreshRequest、SwitchRoleRequest、SwitchRoleResponse、MenuResponse 等结构体）
    - _Requirements: 1.1, 2.1, 3.1_

  - [x] 5.2 实现 User Repository
    - 创建 `internal/repository/user_repo.go`（FindByUsername、FindByID、UpdateLoginFail、ResetLoginFail、CreateLoginHistory 等方法）
    - _Requirements: 1.2, 1.3, 1.4, 1.5, 1.11_

  - [x] 5.3 实现 Auth Service
    - 创建 `internal/service/auth_service.go`
    - 实现 Login 方法：验证用户名密码、检查账户状态/锁定、查询角色分配、按优先级选择 activeRole、生成双令牌、写入登录历史、缓存会话
    - 实现 Logout 方法：将 access_token 和 refresh_token 的 JTI 加入 Redis 黑名单、删除会话缓存
    - 实现 Refresh 方法：验证 refresh_token、检查黑名单、生成新 access_token
    - 实现 SwitchRole 方法：验证目标角色归属、生成新 Token、旧 JTI 加黑名单、更新会话
    - 实现 GetMenu 方法：system_admin/tenant_admin 返回固定菜单，business 用户合并 OrgRole 的 page_permissions
    - _Requirements: 1.1-1.11, 2.1-2.4, 3.1-3.4, 6.4, 14.2, 15.1, 15.2_

  - [x] 5.4 实现 Auth Handler
    - 创建 `internal/handler/auth_handler.go`（Login、Logout、Refresh、SwitchRole、GetMenu 路由处理函数）
    - 参数校验使用 validator 标签
    - 统一使用 response.Success / response.Error 返回
    - _Requirements: 1.1, 2.1, 3.1, 6.4, 11.1_

- [x] 6. 实现组织人员管理模块（Org）
  - [x] 6.1 创建组织相关 DTO
    - 创建 `internal/dto/org_dto.go`（部门/角色/成员的 Create/Update/Response 结构体）
    - _Requirements: 7.1, 8.1, 9.1_

  - [x] 6.2 实现 Org Repository
    - 创建 `internal/repository/org_repo.go`
    - 部门：ListByTenant、Create、Update、Delete、CountMembersByDept
    - 角色：ListByTenant、Create、Update、Delete、FindByIDs
    - 成员：ListByTenant（含 Preload User/Department/Roles）、Create、Update、Delete、FindByUserAndTenant
    - _Requirements: 7.1-7.5, 8.1-8.5, 9.1-9.10_

  - [x] 6.3 实现 Org Service
    - 创建 `internal/service/org_service.go`
    - 部门 CRUD：含删除前检查成员数、租户隔离
    - 角色 CRUD：含系统角色不可删除检查、租户隔离
    - 成员 CRUD：含自动创建用户（bcrypt 密码哈希）、自动创建 UserRoleAssignment（business 默认 + ROLE-003 自动 tenant_admin）、租户内唯一性检查、删除级联清理
    - _Requirements: 7.1-7.5, 8.1-8.5, 9.1-9.10_

  - [x] 6.4 实现 Org Handler
    - 创建 `internal/handler/org_handler.go`（部门/角色/成员的 CRUD 路由处理函数）
    - _Requirements: 7.1-7.5, 8.1-8.5, 9.1-9.10_

- [x] 7. 实现租户管理模块（Tenant）
  - [x] 7.1 创建租户相关 DTO
    - 创建 `internal/dto/tenant_dto.go`（CreateTenantRequest、UpdateTenantRequest、TenantResponse、TenantStatsResponse 等）
    - _Requirements: 10.1_

  - [x] 7.2 实现 Tenant Repository
    - 创建 `internal/repository/tenant_repo.go`（List、Create、Update、Delete、FindByCode、GetStats）
    - _Requirements: 10.1-10.5_

  - [x] 7.3 实现 Tenant Service 和 Handler
    - 创建 `internal/service/tenant_service.go`（含 code 唯一性检查、统计查询）
    - 创建 `internal/handler/tenant_handler.go`（ListTenants、CreateTenant、UpdateTenant、DeleteTenant、GetTenantStats）
    - 创建 `internal/handler/health_handler.go`（健康检查接口）
    - _Requirements: 10.1-10.5_

- [x] 8. 注册路由并整合所有模块
  - [x] 8.1 实现路由注册
    - 创建 `internal/router/router.go`
    - 公开路由：POST /api/auth/login、GET /api/health
    - 认证路由（需 JWT）：POST /api/auth/logout、POST /api/auth/refresh、PUT /api/auth/switch-role、GET /api/auth/menu
    - 租户管理路由（需 tenant_admin）：/api/tenant/org/departments/*、/api/tenant/org/roles/*、/api/tenant/org/members/*
    - 系统管理路由（需 system_admin）：/api/admin/tenants/*
    - 中间件栈按顺序挂载：Logger → Recovery → CORS → RateLimit → JWT → TenantContext → RequireRole
    - _Requirements: 6.2, 6.3, 11.1_

  - [x] 8.2 完善 main.go 启动流程
    - 初始化配置 → 连接 PostgreSQL（GORM）→ 连接 Redis → 初始化 Repository → 初始化 Service → 初始化 Handler → 注册路由 → 启动 HTTP 服务
    - _Requirements: 15.4_

- [x] 9. 检查点 - 确保 Go 服务可编译运行
  - 确保所有代码可编译通过，路由注册正确，如有问题请向用户确认。

- [ ] 10. 更新 Docker Compose 和环境配置
  - [x] 10.1 更新 docker-compose.yml 和 .env.example
    - 在 docker-compose.yml 中添加 go-service 服务定义（依赖 PostgreSQL 和 Redis）
    - 确保 PostgreSQL 和 Redis 服务配置正确
    - 更新 .env.example 添加 Go 服务相关环境变量（DB 连接、Redis 连接、JWT 密钥等）
    - 创建 go-service/Dockerfile
    - _Requirements: 15.4_

- [x] 11. 前端组织接口迁移
  - [x] 11.1 新增 useOrgApi.ts composable
    - 创建 `frontend/composables/useOrgApi.ts`
    - 封装部门 CRUD API（listDepartments、createDepartment、updateDepartment、deleteDepartment）
    - 封装角色 CRUD API（listRoles、createRole、updateRole、deleteRole）
    - 封装成员 CRUD API（listMembers、createMember、updateMember、deleteMember）
    - API 调用失败时回退到 Mock 数据
    - 提供与 Mock 数据相同的类型接口
    - _Requirements: 13.1, 13.2, 13.7_

  - [x] 11.2 更新 useSidebarMenu.ts
    - 将 mockOrgMembers / mockOrgRoles 的直接引用替换为 useOrgApi 提供的响应式数据
    - _Requirements: 13.4_

  - [x] 11.3 更新 middleware/auth.ts
    - 将 Mock 数据引用替换为 useOrgApi 提供的权限数据
    - _Requirements: 13.5_

  - [x] 11.4 更新 admin/tenant/org.vue 页面
    - 对接真实 API 实现部门、角色、成员的 CRUD 操作
    - 保留 Mock 数据作为后备
    - _Requirements: 13.6_

  - [x] 11.5 确认 .env 和 useMockData.ts 保持不变
    - 确认 `NUXT_PUBLIC_MOCK_MODE=true` 不变
    - 确认 useMockData.ts 中所有 Mock 数据保留
    - _Requirements: 13.3, 13.7_

- [x] 12. 最终检查点 - 确保所有代码可编译
  - 确保 Go 服务和前端代码均可编译通过，所有模块正确集成，如有问题请向用户确认。
63
## 备注

- 本项目以手动测试为主，不包含自动化测试任务
- 每个任务引用了具体的需求编号以确保可追溯性
- 检查点确保增量验证，及时发现问题
- 数据库脚本 DDL 和种子数据分离管理
- 前端仅迁移组织相关接口，其他模块继续使用 Mock 数据
