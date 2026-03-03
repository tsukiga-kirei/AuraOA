# OA 智审平台 — 后端需求文档 (Go + Python)

> 文档版本：v1.0 | 更新日期：2026-03-02  
> 本文档定义后端服务的完整需求，基于前端功能反推后端 API、架构和中间件需求。

---

## 一、架构总览

### 1.1 微服务架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Nginx / Traefik                          │
│                        (反向代理 / 负载均衡)                         │
├───────────┬──────────────────┬──────────────────┬───────────────────┤
│  Frontend │   Go Service     │   AI Service     │   Monitoring      │
│  Nuxt 3   │   (业务中台)     │   (AI 引擎)      │   (Prometheus     │
│  :3000    │   :8080          │   :8000           │    + Grafana)     │
└───────────┴───────┬──────────┴────────┬─────────┴───────────────────┘
                    │                   │
           ┌───────┼───────┐    ┌──────┼──────┐
           │       │       │    │      │      │
        ┌──▼──┐ ┌──▼──┐ ┌─▼─┐ │  ┌───▼───┐  │
        │ PG  │ │Redis│ │MQ │ │  │ Xin-  │  │
        │     │ │     │ │   │ │  │ ference│  │
        └─────┘ └─────┘ └───┘ │  └───────┘  │
                               │  ┌───▼───┐  │
                               │  │pgvector│  │
                               │  └───────┘  │
                               └─────────────┘
```

### 1.2 服务职责划分

| 服务 | 语言 | 职责 |
|------|------|------|
| **Go Service** | Go | 业务中台：认证、权限、租户管理、规则管理、流程编排、定时任务、数据查询 |
| **AI Service** | Python | AI引擎：大模型调用、两阶段审核、RAG 检索、OCR（TODO） |
| **Frontend** | TypeScript | Nuxt 3 前端应用 |
| **PostgreSQL** | — | 主数据库（结构化数据 + pgvector） |
| **Redis** | — | 缓存、会话、分布式锁、限流 |
| **RabbitMQ/Kafka** | — | 异步任务队列（批量审核、报告生成） |

### 1.3 技术栈选型

| 组件 | 选型 | 说明 |
|------|------|------|
| Web 框架 | Gin | 高性能 HTTP 框架 |
| ORM | GORM | 数据库操作 |
| 认证 | JWT (golang-jwt) | 访问令牌 + 刷新令牌 |
| 配置管理 | Viper | 多环境配置 |
| 日志 | Zap | 结构化日志 |
| 依赖注入 | Wire | 编译期依赖注入 |
| 定时任务 | robfig/cron | Cron 表达式调度 |
| 缓存 | go-redis | Redis 客户端 |
| 消息队列 | amqp091-go | RabbitMQ 客户端 |
| API 文档 | Swagger (swag) | 自动生成 API 文档 |
| 数据校验 | go-playground/validator | 请求参数校验 |
| 限流 | uber-go/ratelimit | 接口限流 |
| 链路追踪 | OpenTelemetry | 分布式追踪 |
| 监控指标 | Prometheus client | 指标暴露 |
| 迁移工具 | golang-migrate | 数据库版本迁移 |

---

## 二、Go Service 项目结构

```
go-service/
├── cmd/
│   └── server/
│       └── main.go                 # 应用入口
├── internal/
│   ├── config/                     # 配置管理
│   │   ├── config.go               # 配置结构体
│   │   └── loader.go               # Viper 加载
│   ├── middleware/                  # HTTP 中间件
│   │   ├── auth.go                 # JWT 认证
│   │   ├── cors.go                 # 跨域
│   │   ├── ratelimit.go            # 限流
│   │   ├── logger.go               # 请求日志
│   │   ├── recovery.go             # 异常恢复
│   │   ├── tenant.go               # 租户上下文注入
│   │   └── tracing.go              # 链路追踪
│   ├── model/                      # 数据模型
│   │   ├── user.go
│   │   ├── tenant.go
│   │   ├── role.go
│   │   ├── department.go
│   │   ├── member.go
│   │   ├── audit_rule.go
│   │   ├── process_config.go
│   │   ├── cron_task.go
│   │   ├── audit_log.go
│   │   ├── archive.go
│   │   └── system_config.go
│   ├── repository/                 # 数据访问层
│   │   ├── user_repo.go
│   │   ├── tenant_repo.go
│   │   ├── rule_repo.go
│   │   ├── cron_repo.go
│   │   ├── audit_log_repo.go
│   │   ├── org_repo.go
│   │   └── system_config_repo.go
│   ├── service/                    # 业务逻辑层
│   │   ├── auth_service.go         # 认证 & 授权
│   │   ├── tenant_service.go       # 租户管理
│   │   ├── rule_service.go         # 规则管理
│   │   ├── audit_service.go        # 审核编排
│   │   ├── cron_service.go         # 定时任务
│   │   ├── archive_service.go      # 归档复盘
│   │   ├── org_service.go          # 组织人员
│   │   ├── dashboard_service.go    # 仪表盘数据聚合
│   │   ├── user_config_service.go  # 用户偏好
│   │   └── system_config_service.go # 系统设置
│   ├── handler/                    # HTTP 处理器
│   │   ├── auth_handler.go
│   │   ├── tenant_handler.go
│   │   ├── rule_handler.go
│   │   ├── audit_handler.go
│   │   ├── cron_handler.go
│   │   ├── archive_handler.go
│   │   ├── org_handler.go
│   │   ├── dashboard_handler.go
│   │   ├── user_config_handler.go
│   │   └── system_config_handler.go
│   ├── router/                     # 路由定义
│   │   └── router.go
│   ├── dto/                        # 数据传输对象
│   │   ├── request/
│   │   └── response/
│   ├── pkg/                        # 公共工具
│   │   ├── jwt/                    # JWT 工具
│   │   ├── hash/                   # 密码哈希
│   │   ├── response/               # 统一响应格式
│   │   ├── pagination/             # 分页工具
│   │   └── errors/                 # 错误码定义
│   └── scheduler/                  # 定时任务调度器
│       └── scheduler.go
├── migrations/                     # 数据库迁移文件
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── docs/                           # Swagger 文档（自动生成）
├── Dockerfile
├── go.mod
└── go.sum
```

---

## 三、API 接口设计

### 3.1 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": { ... },
  "trace_id": "abc-123"
}
```

错误响应：
```json
{
  "code": 40001,
  "message": "用户名或密码错误",
  "data": null,
  "trace_id": "abc-123"
}
```

### 3.2 认证接口

#### POST `/api/auth/login`

```json
// Request
{
  "username": "admin",
  "password": "123456",
  "tenant_id": "DEMO_HQ",      // 租户编码（system_admin 可不传）
  "preferred_role": "system_admin"  // 可选，偏好角色
}

// Response
{
  "code": 0,
  "data": {
    "access_token": "eyJhbG...",
    "refresh_token": "eyJhbG...",
    "expires_in": 7200,
    "user": {
      "username": "admin",
      "display_name": "陈刚",
      "tenant_id": "T-001",
      "role_label": "系统管理员"
    },
    "roles": [
      { "id": "admin-r1", "role": "system_admin", "tenant_id": null, "tenant_name": null, "label": "系统管理员" },
      { "id": "admin-r2", "role": "tenant_admin", "tenant_id": "T-001", "tenant_name": "示例集团总部", "label": "示例集团总部 · 租户管理员" }
    ],
    "active_role": { "id": "admin-r1", "role": "system_admin", ... },
    "permissions": ["system_admin"]
  }
}
```

#### POST `/api/auth/refresh`

```json
// Request
{ "refresh_token": "eyJhbG..." }

// Response
{ "access_token": "eyJhbG...", "expires_in": 7200 }
```

#### POST `/api/auth/logout`

```json
// 清除服务端 session / 加入 token 黑名单
```

#### GET `/api/auth/menu`

```json
// Response: 根据当前激活角色返回菜单
{
  "menus": [
    { "key": "overview", "label": "仪表盘", "icon": "PieChartOutlined", "path": "/overview" },
    { "key": "tenants", "label": "租户管理", "icon": "TeamOutlined", "path": "/admin/system/tenants" }
  ]
}
```

#### PUT `/api/auth/switch-role`

```json
// Request
{ "role_id": "admin-r2" }

// Response: 返回新的 token（包含新角色权限）
{ "access_token": "...", "active_role": { ... }, "permissions": ["tenant_admin"] }
```

### 3.3 审核业务接口

#### GET `/api/audit/todo`
```json
// Query: ?page=1&page_size=10&process_type=采购审批
// Response
{
  "processes": [
    {
      "process_id": "WF-2025-001",
      "title": "办公设备采购申请",
      "applicant": "张明",
      "department": "研发部",
      "submit_time": "2025-06-10 09:30",
      "process_type": "采购审批",
      "status": "pending",
      "current_node": "财务总监审批",
      "amount": 156000
    }
  ],
  "total": 12
}
```

#### POST `/api/audit/execute`
```json
// Request
{ "process_id": "WF-2025-001" }

// Response（需调用 AI Service 后返回）
{
  "trace_id": "TR-20250610-A3F8",
  "process_id": "WF-2025-001",
  "recommendation": "return",
  "score": 72,
  "duration_ms": 3850,
  "details": [...],
  "ai_reasoning": "...",
  "action_label": "建议退回",
  "confidence": 0.85,
  "risk_points": [...],
  "suggestions": [...],
  "ai_summary": "...",
  "model_used": "Qwen2.5-72B",
  "interaction_mode": "two_phase",
  "phase1_duration_ms": 2200,
  "phase2_duration_ms": 1650
}
```

#### POST `/api/audit/batch`
```json
// Request
{ "process_ids": ["WF-2025-001", "WF-2025-002", "WF-2025-003"] }

// Response（异步任务）
{
  "batch_id": "BATCH-20250610-001",
  "total": 3,
  "status": "processing"
}
```

#### GET `/api/audit/batch/:batch_id`
```json
// 轮询批量审核进度
{
  "batch_id": "BATCH-20250610-001",
  "total": 3, "completed": 2, "failed": 0,
  "status": "processing",
  "progress_percent": 66,
  "results": [...]
}
```

#### POST `/api/audit/feedback`
```json
{ "process_id": "WF-2025-001", "adopted": true, "action_taken": "approve" }
```

### 3.4 定时任务接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/cron/tasks` | 获取当前用户的定时任务列表 |
| POST | `/api/cron/tasks` | 创建定时任务 |
| PUT | `/api/cron/tasks/:id` | 更新定时任务 |
| DELETE | `/api/cron/tasks/:id` | 删除定时任务（非内建） |
| GET | `/api/cron/history` | 获取执行历史 |

### 3.5 归档复盘接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/archive/processes` | 获取归档流程列表 |
| GET | `/api/archive/processes/:id` | 获取归档流程详情（含审批链） |
| POST | `/api/archive/review` | 触发合规复核（调用AI） |
| GET | `/api/archive/review/:trace_id` | 获取复核结果 |
| GET | `/api/archive/audit-chains/:process_id` | 获取审核链历史 |

### 3.6 公共租户接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tenants/list` | 获取活跃租户列表（公共接口，无需鉴权，用于登录页租户选择器，仅返回 id 和 name） |

### 3.7 租户管理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/tenants` | 获取所有租户 |
| POST | `/api/admin/tenants` | 创建租户 |
| PUT | `/api/admin/tenants/:id` | 更新租户 |
| DELETE | `/api/admin/tenants/:id` | 删除租户 |
| GET | `/api/admin/tenants/:id/stats` | 租户统计数据 |

### 3.8 规则管理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tenant/rules/configs` | 获取审核规则配置列表 |
| PUT | `/api/tenant/rules/configs/:id` | 更新审核规则配置 |
| GET | `/api/tenant/rules/archive-configs` | 获取归档复盘配置列表 |
| PUT | `/api/tenant/rules/archive-configs/:id` | 更新归档复盘配置 |
| GET | `/api/tenant/rules/strictness-presets` | 获取审核尺度预设 |
| PUT | `/api/tenant/rules/strictness-presets` | 更新审核尺度预设 |
| GET | `/api/tenant/rules/cron-configs` | 获取定时任务类型配置 |
| PUT | `/api/tenant/rules/cron-configs` | 更新定时任务类型配置 |

### 3.9 组织人员接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tenant/org/departments` | 获取部门列表 |
| POST | `/api/tenant/org/departments` | 创建部门 |
| PUT | `/api/tenant/org/departments/:id` | 更新部门 |
| DELETE | `/api/tenant/org/departments/:id` | 删除部门 |
| GET | `/api/tenant/org/roles` | 获取角色列表 |
| POST | `/api/tenant/org/roles` | 创建角色 |
| PUT | `/api/tenant/org/roles/:id` | 更新角色 |
| DELETE | `/api/tenant/org/roles/:id` | 删除角色（非系统角色） |
| GET | `/api/tenant/org/members` | 获取成员列表 |
| POST | `/api/tenant/org/members` | 创建成员 |
| PUT | `/api/tenant/org/members/:id` | 更新成员 |
| DELETE | `/api/tenant/org/members/:id` | 删除成员 |

### 3.10 数据信息接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tenant/data/audit-logs` | 审核日志查询（支持分页/筛选） |
| GET | `/api/tenant/data/cron-logs` | 定时任务日志查询 |
| GET | `/api/tenant/data/archive-logs` | 归档复盘日志查询 |

### 3.11 用户偏好接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tenant/user-configs` | 获取所有用户偏好配置 |
| GET | `/api/tenant/user-configs/:user_id` | 获取单个用户偏好详情 |

### 3.12 系统设置接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/system/oa-connections` | OA数据库连接列表 |
| POST | `/api/system/oa-connections` | 创建OA连接 |
| PUT | `/api/system/oa-connections/:id` | 更新OA连接 |
| DELETE | `/api/system/oa-connections/:id` | 删除OA连接 |
| POST | `/api/system/oa-connections/:id/test` | 测试OA连接 |
| GET | `/api/system/ai-models` | AI模型配置列表 |
| POST | `/api/system/ai-models` | 创建AI模型 |
| PUT | `/api/system/ai-models/:id` | 更新AI模型 |
| DELETE | `/api/system/ai-models/:id` | 删除AI模型 |
| GET | `/api/system/general` | 平台通用配置 |
| PUT | `/api/system/general` | 更新平台配置 |

### 3.13 仪表盘接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/dashboard/overview` | 仪表盘综合数据 |
| GET | `/api/dashboard/prefs` | 用户仪表盘偏好 |
| PUT | `/api/dashboard/prefs` | 更新仪表盘偏好 |

### 3.14 个人设置接口

| 方法 | 路径 | 说明 |
|------|------|------|
| PUT | `/api/user/password` | 修改密码 |
| GET | `/api/user/login-history` | 登录历史 |
| GET | `/api/user/locale` | 语言偏好 |
| PUT | `/api/user/locale` | 更新语言偏好 |
| GET | `/api/user/security` | 安全信息 |

### 3.15 系统监控接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/monitor/metrics` | 运行指标 |
| GET | `/api/monitor/alerts` | 告警列表 |
| GET | `/api/monitor/health` | 健康检查 |
| GET | `/metrics` | Prometheus 指标端点 |

---

## 四、中间件需求

### 4.1 Redis 缓存策略

| 用途 | Key 模式 | TTL | 说明 |
|------|----------|-----|------|
| Token 黑名单 | `token:blacklist:{jti}` | 与 token 剩余有效期一致 | Logout 时加入 |
| 用户会话 | `session:{user_id}` | 2h | 缓存用户角色权限 |
| 租户配置缓存 | `tenant:config:{tenant_id}` | 5min | 避免频繁查询 |
| 仪表盘数据 | `dashboard:{tenant_id}:{widget}` | 1min | 减少聚合查询负载 |
| 限流计数器 | `ratelimit:{ip}:{api}` | 1min | 滑动窗口限流 |
| 分布式锁 | `lock:batch_audit:{tenant_id}` | 10min | 防止重复执行批量审核 |
| OA 流程缓存 | `oa:processes:{tenant_id}` | 30s | 缓存待办流程列表 |

### 4.2 消息队列

| 队列名称 | 消费者 | 说明 |
|----------|--------|------|
| `audit.execute` | AI Service | 单条AI审核任务 |
| `audit.batch` | Go Service Worker | 批量审核编排 |
| `report.daily` | Go Service Worker | 日报生成推送 |
| `report.weekly` | Go Service Worker | 周报生成推送 |
| `notification.email` | Go Service Worker | 邮件通知 |

### 4.3 负载均衡

```nginx
upstream go_service {
    least_conn;
    server go-service-1:8080;
    server go-service-2:8080;
    server go-service-3:8080;
}

upstream ai_service {
    least_conn;
    server ai-service-1:8000;
    server ai-service-2:8000;
}
```

### 4.4 监控体系

```
┌──────────┐    ┌────────────┐    ┌──────────┐
│  Go/AI   │───▶│ Prometheus │───▶│ Grafana  │
│ Services │    │            │    │ Dashboard│
└──────────┘    └────────────┘    └──────────┘
      │
      ▼
┌──────────┐    ┌────────────┐
│  Jaeger  │◀───│   OTel     │
│  Tracing │    │  Collector │
└──────────┘    └────────────┘
```

**关键指标**：
- API 请求成功率、延迟分布（p50/p95/p99）
- AI 模型响应时间、Token 消耗
- 数据库连接池使用率
- Redis 命中率
- 队列积压数量
- 各租户 Token 用量

---

## 五、安全需求

### 5.1 认证安全

- JWT Token 双令牌机制（access_token: 2h, refresh_token: 7d）
- Token 中包含：user_id, tenant_id, active_role, permissions, jti
- 登录失败 5 次后锁定账户 15 分钟
- 密码存储使用 bcrypt（cost=12）
- 支持 SSO 对接（OAuth2/OIDC）

### 5.2 数据安全

- 敏感字段（密码、API Key）在日志中脱敏
- OA 数据库密码加密存储（AES-256-GCM）
- 审计日志不可篡改
- 支持数据保留策略（按租户配置保留天数）

### 5.3 多租户隔离

- 所有数据查询自动附加 `tenant_id` 条件
- 中间件层自动注入租户上下文
- 跨租户数据访问严格禁止
- 系统管理员可跨租户查看但不可修改业务数据

---

## 六、AI Service (Python) — TODO

AI Service 当前标记为 **TODO**，后续开发将包含：

### 6.1 核心功能

| 功能 | 说明 |
|------|------|
| 两阶段审核 | Phase 1: 推理分析（reasoning）→ Phase 2: 结构化提取（extraction） |
| RAG 检索 | 基于 pgvector 的文档向量检索，补充规则上下文 |
| OCR 识别 | 流程附件（发票、合同等）的文字识别 |
| 模型管理 | 支持多模型（Qwen、DeepSeek 等）的动态切换和负载均衡 |

### 6.2 API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/ai/audit` | 执行 AI 审核（Go Service 调用） |
| POST | `/api/ai/archive-review` | 执行归档合规复核 |
| POST | `/api/ai/ocr` | OCR 文字识别 |
| GET | `/api/ai/models` | 可用模型列表 |
| GET | `/api/ai/health` | 健康检查 |

### 6.3 技术栈

| 组件 | 选型 |
|------|------|
| Web 框架 | FastAPI |
| 大模型客户端 | OpenAI SDK (兼容接口) |
| 向量数据库 | pgvector (通过 psycopg) |
| OCR | PaddleOCR / Tesseract |
| 异步任务 | Celery + RabbitMQ |

---

## 七、部署架构

### 7.1 Docker Compose

**生产/完整编排** (`docker-compose.yml`)：

```yaml
services:
  frontend:     # Nuxt 3 前端
  go-service:   # Go 业务中台
  ai-service:   # Python AI 引擎
  postgres:     # PostgreSQL 16 + pgvector
  redis:        # Redis 7
```

**本地开发** (`docker-compose.dev.yml`)：仅启动基础设施，前后端在本地运行。

```bash
docker-compose -f docker-compose.dev.yml up -d
```

```yaml
services:
  postgres:     # PostgreSQL 16 + pgvector（自动加载 migrations 和 seeds）
  redis:        # Redis 7
```

### 7.2 Kubernetes (生产环境)

```
Namespace: oa-smart-audit
├── Deployment: frontend (replicas: 2)
├── Deployment: go-service (replicas: 3, HPA min:2/max:10)
├── Deployment: ai-service (replicas: 2, GPU 节点)
├── StatefulSet: postgres (replicas: 1, PVC)
├── Deployment: redis (replicas: 1, 后续 Sentinel)
├── Service: 各服务 ClusterIP
├── Ingress: 统一入口
├── ConfigMap: 配置
├── Secret: 敏感配置
└── CronJob: 数据库备份
```

### 7.3 多版本构建

```dockerfile
# 多阶段构建
FROM golang:1.25.6-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.19
COPY --from=builder /app/server /server
EXPOSE 8080
CMD ["/server"]
```

---

## 八、开发优先级

### Phase 1: 基础框架（本期重点）

1. ✅ Go 项目初始化（Gin + GORM + Viper）
2. ✅ 数据库迁移框架
3. ✅ JWT 认证（登录/登出/刷新/角色切换）
4. ✅ 多租户中间件
5. ✅ 组织人员 CRUD
6. ✅ 权限校验中间件
7. ✅ 前端对接（替换 Mock 数据）

### Phase 2: 核心业务

8. 规则配置管理
9. OA 数据对接（泛微 E9 JDBC 直连）
10. 审核编排（调用 AI Service）
11. 定时任务调度
12. 数据日志查询

### Phase 3: 高级功能

13. 归档复盘
14. 仪表盘数据聚合
15. 系统设置管理
16. 用户偏好管理
17. 邮件通知推送

### Phase 4: 运维能力

18. Prometheus 监控集成
19. 日志集中管理
20. 数据备份恢复
21. 性能优化（Redis 缓存）
