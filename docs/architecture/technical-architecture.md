# 技术架构文档

> 文档版本：v1.0 | 创建日期：2026-03-19
> 从技术架构、前后端关联、API 接口层面全面介绍 OA 智审平台的技术设计。

---

## 一、系统架构总览

```
                          ┌──────────────────────────┐
                          │      Nginx（可选）        │
                          │    反向代理 / 负载均衡     │
                          └────────┬─────────────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              │                    │                    │
    ┌─────────▼────────┐  ┌───────▼────────┐  ┌───────▼────────┐
    │   Frontend       │  │   Go Service   │  │  AI Service    │
    │   (Nuxt 3 SPA)   │  │   (Gin REST)   │  │  (Python)      │
    │   Port: 3000     │  │   Port: 8080   │  │  Port: 8000    │
    └──────────────────┘  └───────┬────────┘  └────────────────┘
                                  │                     ▲
                          ┌───────┼──────────┐          │
                          │       │          │     HTTP 调用
                    ┌─────▼──┐ ┌──▼───┐ ┌───▼──┐       │
                    │ Postgres│ │Redis │ │ OA DB│  ┌────┴───────┐
                    │ (pg16)  │ │ (7)  │ │ MySQL│  │ LLM Models │
                    └─────────┘ └──────┘ │Oracle│  │ Xinference │
                                         │DM    │  │ Ollama/API │
                                         └──────┘  └────────────┘
```

---

## 二、后端架构（Go Service）

### 2.1 分层架构

采用经典的 **Controller → Service → Repository** 三层架构：

```
┌─ Handler（Controller 层）─ 处理 HTTP 请求/响应、参数校验、权限检查
│
├─ Service（业务逻辑层）── 编排业务流程、事务管理、跨 Repo 协调
│
├─ Repository（数据访问层）── GORM 数据库操作封装，每张表一个 Repo
│
├─ Model（数据模型层）── 对应数据库表结构的 Go struct
│
├─ DTO（数据传输对象）── 请求/响应 JSON 结构定义
│
├─ Middleware（中间件层）── JWT 认证、CORS、日志、异常恢复、角色鉴权
│
└─ Pkg（工具包层）── AI 调用、OA 适配、加密、JWT、错误码等
```

### 2.2 核心文件清单

| 层 | 文件 | 职责 |
|---|------|------|
| **入口** | `cmd/server/main.go` | 应用初始化：配置加载→DB连接→Redis→DI组装→Router→HTTP |
| **配置** | `internal/config/config.go` | 配置结构定义与加载 |
| **路由** | `internal/router/router.go` | 所有 API 路由注册 |
| **中间件** | `internal/middleware/auth.go` | JWT 认证 |
| | `internal/middleware/role.go` | 角色鉴权 |
| | `internal/middleware/tenant.go` | 租户上下文注入 |
| | `internal/middleware/cors.go` | 跨域处理 |
| | `internal/middleware/logger.go` | 请求日志 |
| | `internal/middleware/recovery.go` | 异常恢复 |

### 2.3 依赖注入

采用 **手动构造注入**（在 `main.go` 中显式创建），无框架依赖：

```
main.go:
  1. Repo = NewXxxRepo(db)
  2. Service = NewXxxService(repo1, repo2, ...)
  3. Handler = NewXxxHandler(service)
  4. SetupRouter(r, handlers...)
```

### 2.4 认证与授权体系

```
请求 → JWT Middleware → TenantContext Middleware → RequireRole Middleware → Handler
```

| 中间件 | 职责 |
|--------|------|
| `JWT` | 验证 Bearer Token，提取 UserID/RoleID 写入 Context |
| `TenantContext` | 从角色分配中解析 TenantID 写入 Context |
| `RequireRole` | 检查用户角色是否满足路由要求 |

**JWT 实现**：
- 签发/刷新：`internal/pkg/jwt/jwt.go`
- Access Token TTL：2小时（默认）
- Refresh Token TTL：7天（默认）
- 登出时 Access Token 存入 Redis 黑名单

**密码安全**：
- 存储：bcrypt 哈希（`internal/pkg/hash/bcrypt.go`）
- 登录失败锁定：由 `system_configs` 配置（默认5次锁定15分钟）

---

## 三、前端架构（Frontend）

### 3.1 技术选型

| 项 | 选型 |
|----|------|
| 框架 | Nuxt 3（SSR 关闭，SPA 模式） |
| UI 库 | Ant Design Vue 4 |
| 语言 | TypeScript + Vue 3 Composition API |
| 状态管理 | `useState`（Nuxt 3 内置） |
| API 调用 | `authFetch`（基于 `$fetch`，自动刷新 Token） |
| 路由守卫 | `middleware/auth.ts` |
| 国际化 | 自研 `useI18n` composable |
| 图标 | Ant Design Icons Vue |
| 数据导出 | xlsx |

### 3.2 页面结构

| 页面文件 | 路由 | 说明 | 角色 |
|---------|------|------|------|
| `login.vue` | `/login` | 登录页（含租户选择） | 公开 |
| `dashboard.vue` | `/dashboard` | 仪表盘 | all |
| `overview.vue` | `/overview` | 审核工作台 | business |
| `cron.vue` | `/cron` | 定时任务管理 | business |
| `archive.vue` | `/archive` | 归档复盘 | business |
| `settings.vue` | `/settings` | 个人设置 | business |
| `admin/system/` | `/admin/system` | 系统管理 | system_admin |
| `admin/tenant/` | `/admin/tenant` | 租户管理 | tenant_admin |

### 3.3 Composables（组合式 API）

| 文件 | 职责 |
|------|------|
| `useAuth.ts` | 认证核心：登录/登出/角色切换/Token刷新/authFetch |
| `useMockData.ts` | Mock 数据（开发用，含全部业务模拟数据） |
| `useOrgApi.ts` | 组织架构 API（部门/角色/成员，**已对接后端**） |
| `useSettingsApi.ts` | 个人设置 API（**已对接后端**） |
| `useRulesApi.ts` | 规则配置 API（**已对接后端**） |
| `useSystemApi.ts` | 系统设置 API（**已对接后端**） |
| `useArchiveApi.ts` | 归档配置 API（**已对接后端**） |
| `useCronApi.ts` | 定时任务 API（**已对接后端**） |
| `useAdminUserConfigApi.ts` | 管理员用户配置 API（**已对接后端**） |
| `usePagination.ts` | 前端分页工具 |
| `useTheme.ts` | 主题/暗色模式 |
| `useI18n.ts` | 国际化 |
| `useSidebarMenu.ts` | 侧边栏菜单 |
| `useLayoutPrefs.ts` | 布局偏好 |

### 3.4 Mock 模式

前端支持 **Mock 模式**（环境变量 `NUXT_PUBLIC_MOCK_MODE=true`），在无后端时使用 `useMockData.ts` 中的模拟数据运行。

**当前 Mock 状态**：
- ✅ 配置管理类页面（设置、规则、系统管理）已对接真实后端
- ❌ 业务运行类页面（审核工作台 `overview.vue`、仪表盘 `dashboard.vue`）仍使用 Mock 数据

---

## 四、前后端对接状态

### 4.1 已完成对接的模块

| 模块 | 前端 | 后端 Handler | 状态 |
|------|------|-------------|------|
| 登录/登出/Token 刷新 | `useAuth.ts` | `auth_handler.go` | ✅ 完成 |
| 角色切换 | `useAuth.ts` | `auth_handler.go` | ✅ 完成 |
| 菜单权限 | `useAuth.ts` | `auth_handler.go` | ✅ 完成 |
| 个人资料 | `useAuth.ts` | `auth_handler.go` | ✅ 完成 |
| 组织架构（部门/角色/成员） | `useOrgApi.ts` | `org_handler.go` | ✅ 完成 |
| 租户管理 | `useSystemApi.ts` | `tenant_handler.go` | ✅ 完成 |
| 系统配置 | `useSystemApi.ts` | `system_handler.go` | ✅ 完成 |
| OA 连接管理 | `useSystemApi.ts` | `system_handler.go` | ✅ 完成 |
| AI 模型管理 | `useSystemApi.ts` | `system_handler.go` | ✅ 完成 |
| 选项数据 | `useSystemApi.ts` | `system_handler.go` | ✅ 完成 |
| 流程审核配置 | `useRulesApi.ts` | `process_audit_config_handler.go` | ✅ 完成 |
| 审核规则 | `useRulesApi.ts` | `audit_rule_handler.go` | ✅ 完成 |
| 归档复盘配置 | `useArchiveApi.ts` | `archive_config_handler.go` | ✅ 完成 |
| 归档规则 | `useArchiveApi.ts` | `archive_config_handler.go` | ✅ 完成 |
| 定时任务配置 | `useCronApi.ts` | `cron_config_handler.go` | ✅ 完成 |
| 个人设置（审核/归档/定时/仪表板） | `useSettingsApi.ts` | `user_personal_config_handler.go` | ✅ 完成 |
| 管理员用户配置查看 | `useAdminUserConfigApi.ts` | `user_config_management_handler.go` | ✅ 完成 |
| Token 消耗统计 | `useSystemApi.ts` | `llm_message_log_handler.go` | ✅ 完成 |

### 4.2 未完成对接的模块

| 模块 | 状态 | 说明 |
|------|------|------|
| 审核工作台（overview.vue） | ❌ 使用 Mock | 前端 UI 已完成，但审核执行链路未后端化 |
| 仪表盘（dashboard.vue） | ❌ 使用 Mock | 所有统计数据为 Mock 生成 |
| 定时任务执行与日志 | ❌ 未实现 | 后端无 Cron 调度引擎 |
| 归档复盘执行 | ❌ 使用 Mock | 页面 UI 完成，归档审核执行未后端化 |
| 消息/通知 | ⚠️ 部分实现 | 站内通知已实现（审核/归档/定时任务完成），邮件推送待完善 |

---

## 五、API 接口总览

### 5.1 公开接口（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| POST | `/api/auth/login` | 用户登录 |
| POST | `/api/auth/refresh` | 刷新 Token |
| GET | `/api/tenants/list` | 获取公开租户列表（登录页用） |

### 5.2 认证接口（JWT 必须）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/logout` | 登出 |
| PUT | `/api/auth/switch-role` | 切换角色 |
| GET | `/api/auth/menu` | 获取菜单 |
| PUT | `/api/auth/change-password` | 修改密码 |
| GET | `/api/auth/me` | 获取当前用户信息 |
| PUT | `/api/auth/locale` | 更新语言偏好 |
| PUT | `/api/auth/profile` | 更新个人资料 |

### 5.3 租户组织管理（JWT + TenantContext + tenant_admin）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST/PUT/DELETE | `/api/tenant/org/departments[/:id]` | 部门 CRUD |
| GET/POST/PUT/DELETE | `/api/tenant/org/roles[/:id]` | 角色 CRUD |
| GET/POST/PUT/DELETE | `/api/tenant/org/members[/:id]` | 成员 CRUD |

### 5.4 系统管理（JWT + TenantContext + system_admin）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST/PUT/DELETE | `/api/admin/tenants[/:id]` | 租户 CRUD |
| GET | `/api/admin/tenants/:id/stats` | 租户统计 |
| GET | `/api/admin/tenants/:id/members` | 租户成员 |
| GET | `/api/admin/system/options/*` | 选项数据 |
| GET/POST/PUT/DELETE | `/api/admin/system/oa-connections[/:id]` | OA 连接 CRUD |
| POST | `/api/admin/system/oa-connections[/:id]/test` | 连接测试 |
| GET/POST/PUT/DELETE | `/api/admin/system/ai-models[/:id]` | AI 模型 CRUD |
| POST | `/api/admin/system/ai-models[/:id]/test` | 模型测试 |
| GET/PUT | `/api/admin/system/configs` | 系统配置 |
| GET | `/api/admin/stats/token-usage` | 全局 Token 统计 |

### 5.5 租户配置管理（JWT + TenantContext + tenant_admin）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST/PUT/DELETE | `/api/tenant/rules/configs[/:id]` | 审核流程配置 CRUD |
| POST | `/api/tenant/rules/configs/test-connection` | 测试 OA 连接 |
| POST | `/api/tenant/rules/configs/:id/fetch-fields` | 拉取字段 |
| GET/POST/PUT/DELETE | `/api/tenant/rules/audit-rules[/:id]` | 审核规则 CRUD |
| GET | `/api/tenant/rules/prompt-templates` | 查看提示词模板 |
| GET/PUT/DELETE | `/api/tenant/cron/configs[/:taskType]` | 定时任务配置 |
| GET/POST/PUT/DELETE | `/api/tenant/archive/configs[/:id]` | 归档配置 CRUD |
| POST | `/api/tenant/archive/configs/test-connection` | 测试连接 |
| POST | `/api/tenant/archive/configs/:id/fetch-fields` | 拉取字段 |
| GET/POST/PUT/DELETE | `/api/tenant/archive/audit-rules[/:id]` | 归档规则 CRUD |
| GET | `/api/tenant/archive/prompt-templates` | 查看提示词模板 |
| GET | `/api/tenant/user-configs` | 用户配置列表 |
| GET | `/api/tenant/user-configs/:userId` | 查看用户配置 |
| GET | `/api/tenant/stats/token-usage` | 租户 Token 统计 |

### 5.6 业务用户接口（JWT + TenantContext）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tenant/settings/processes` | 个人流程列表 |
| GET | `/api/tenant/settings/processes/:processType` | 查看流程设置 |
| PUT | `/api/tenant/settings/processes/:processType` | 更新流程设置 |
| GET | `/api/tenant/settings/processes/:processType/full` | 完整流程配置 |
| GET/PUT | `/api/tenant/settings/cron-prefs` | 定时任务偏好 |
| GET | `/api/tenant/settings/archive-configs` | 归档流程列表 |
| GET | `/api/tenant/settings/archive-configs/:processType/full` | 完整归档配置 |
| PUT | `/api/tenant/settings/archive-configs/:processType` | 更新归档设置 |
| GET/PUT | `/api/tenant/settings/dashboard-prefs` | 仪表板偏好 |

---

## 六、统一响应格式

所有 API 返回统一的 JSON 格式：

```json
{
  "code": 0,
  "message": "success",
  "data": { ... },
  "trace_id": "uuid"
}
```

| 字段 | 说明 |
|------|------|
| `code` | 业务状态码（0=成功，非0=错误） |
| `message` | 状态描述 |
| `data` | 业务数据（可选） |
| `trace_id` | 请求追踪ID |

### 错误码分组

| 错误码范围 | 分类 |
|-----------|------|
| 40100 ~ 40199 | 认证相关 |
| 40300 ~ 40399 | 权限相关 |
| 40400 ~ 40499 | 资源不存在 |
| 42200 ~ 42299 | 参数校验 |
| 50000 ~ 50099 | 服务器内部 |

---

## 七、中间件管线

请求处理顺序：

```
→ Logger      记录请求方法/路径/耗时
→ Recovery    捕获 panic，返回 500
→ CORS        跨域配置（允许的源来自 config.yaml）
→ JWT         验证 Bearer Token，解析 claims
→ TenantContext  注入租户ID到 context
→ RequireRole    检查角色权限
→ Handler        业务处理
```

---

## 八、配置体系

### 8.1 配置优先级

```
环境变量 > config.yaml 文件
```

Viper 同时读取 YAML 配置文件和环境变量，环境变量使用 `_` 替代 `.`（如 `database.host` → `DATABASE_HOST`）。

### 8.2 配置项列表

| 模块 | 配置项 | 说明 |
|------|--------|------|
| server | `port` | HTTP 端口（默认 8080） |
| database | `host/port/user/password/dbname/sslmode` | PostgreSQL |
| database | `max_open_conns/max_idle_conns` | 连接池（默认 50/10） |
| redis | `host/port/password/db` | Redis |
| jwt | `secret/access_token_ttl/refresh_token_ttl` | JWT 配置 |
| cors | `allowed_origins` | CORS 允许的源 |
| encryption | `key` | AES-256 密钥（必须 32 字节） |

---

## 九、Docker 编排

### 9.1 开发环境（docker-compose.dev.yml）

| 服务 | 说明 |
|------|------|
| `go-service` | Go 后端（Dockerfile 构建） |
| `postgres` | PostgreSQL 16（pgvector 镜像） |
| `redis` | Redis 7 Alpine |

前端在本地通过 `pnpm dev` 运行。

### 9.2 生产环境（docker-compose.yml）

| 服务 | 说明 |
|------|------|
| `frontend` | Nuxt 3 前端（Dockerfile 构建） |
| `go-service` | Go 后端 |
| `ai-service` | Python AI 服务（规划中） |
| `postgres` | PostgreSQL 16 |
| `redis` | Redis 7 |

### 9.3 数据持久化

| Volume | 用途 |
|--------|------|
| `pg_data` | PostgreSQL 数据目录 |
| `redis_data` | Redis AOF 持久化 |

---

## 十、数据库初始化流程

```
PostgreSQL 容器启动
  → 执行 /docker-entrypoint-initdb.d/migrations/*.sql（按文件名排序）
  → 执行 /docker-entrypoint-initdb.d/seeds/*.sql（按文件名排序）
  → 数据库初始化完成
```

种子数据包含：
- OA/AI 选项数据
- 示例用户（含管理员）
- 示例租户（含角色分配）
- 部门与组织结构
- 流程审核/归档配置
- 用户个人配置
