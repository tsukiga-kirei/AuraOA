# OA 智审 — 流程智能审核平台

> **OA Smart Audit** — 基于大语言模型的 OA 流程智能审核与归档复盘系统

---

## 项目简介

OA 智审是一套面向企业内部 OA 流程的 AI 辅助审核平台。通过连接企业 OA 系统的数据库，提取流程表单数据与审批流信息，结合自定义审核规则与大语言模型（LLM），实现对 OA 流程的智能合规性审核与归档复盘。

### 核心能力

| 能力 | 说明 |
|------|------|
| 🔍 **智能审核** | 两阶段 AI 审核（推理→结构化提取），支持严格/标准/宽松三种审核尺度 |
| 📦 **归档复盘** | 对已归档流程进行全流程合规复核，含审批流节点完整性分析 |
| ⏰ **定时任务** | 批量审核、日报/周报自动推送，支持自定义 Cron 表达式 |
| 🏢 **多租户** | 租户隔离的数据与配置，支持独立 AI 模型分配与 Token 配额管理 |
| 🔗 **OA 适配** | 可扩展的 OA 适配器架构，当前支持泛微 Ecology E9（MySQL/Oracle/达梦） |
| 🤖 **多模型** | 支持本地部署（Xinference、Ollama、vLLM）与云端 API（阿里云百炼、DeepSeek、OpenAI 等） |
| 👤 **个性化配置** | 用户可自定义审核字段、规则、AI 尺度偏好，支持租户管理员集中查看与管理 |
| 🌐 **国际化** | 支持中文/英文双语界面 |

---

## 技术栈

### 后端（Go Service）
- **语言**：Go 1.25+
- **Web 框架**：Gin
- **ORM**：GORM
- **数据库**：PostgreSQL 16（pgvector 镜像）
- **缓存**：Redis 7
- **认证**：JWT（Access Token 2h + Refresh Token 7d）
- **配置**：Viper（YAML + 环境变量）
- **日志**：Zap（支持租户级日志隔离）
- **加密**：AES-256（数据库密码等敏感字段）

### 前端（Frontend）
- **框架**：Nuxt 3（SSR 关闭，SPA 模式）
- **UI 库**：Ant Design Vue 4
- **语言**：TypeScript / Vue 3 Composition API
- **国际化**：自研 i18n（基于 `zh-CN.ts` / `en-US.ts`）
- **数据可视化**：内置图表组件

### 基础设施
- **容器化**：Docker Compose（开发环境 + 生产环境）
- **数据库迁移**：golang-migrate（`db/migrations/`）

---

## 系统架构

### 认证架构

```
┌─────────────────────────────────────────────────────────────┐
│                      JWT 双令牌架构                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Access Token (2h)          Refresh Token (7d)              │
│  ├── 用户信息                ├── 用户 ID                    │
│  ├── 当前角色                └── JTI (用于黑名单)           │
│  ├── 权限列表                                               │
│  └── JTI (用于黑名单)                                       │
│                                                             │
│  Redis 存储:                                                │
│  ├── session:{user_id} → 用户会话缓存 (2h TTL)             │
│  └── blacklist:{jti} → 已吊销令牌 (与 Token TTL 一致)      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 角色体系

| 层级 | 角色 | 说明 |
|-----|------|------|
| 系统级 | `system_admin` | 管理租户、OA 连接、AI 模型、系统配置 |
| 系统级 | `tenant_admin` | 管理组织架构、流程配置、审核规则、用户配置 |
| 系统级 | `business` | 使用审核工作台、归档复盘、个人设置 |
| 组织级 | 自定义角色 | 通过 `page_permissions` 控制页面访问权限 |

### 配置层级

```
系统配置 (system_configs)
    │
    ├── auth.* — 认证相关配置
    ├── tenant.* — 租户默认配置
    └── system.* — 系统全局配置
    │
    ▼
租户配置 (tenants + process_audit_configs)
    │
    ├── 流程审核配置 — 字段/规则/AI 配置
    ├── 用户权限控制 — 允许自定义字段/规则/尺度
    └── 访问控制 — 角色/成员/部门白名单
    │
    ▼
用户个人配置 (user_personal_configs)
    │
    ├── 字段覆盖 — 在租户基础上新增字段
    ├── 规则覆盖 — 开关租户规则 + 自定义规则
    └── AI 尺度覆盖 — 个人审核严格度偏好
```

### 审核流程

```
用户选择流程 → 获取配置 → 从 OA 提取数据 → 合并规则 → 构建提示词 → AI 审核 → 返回结果
                  │              │              │              │
                  ▼              ▼              ▼              ▼
            租户配置 +      OA 适配器      MergeRules()   两阶段审核
            用户配置       (Weaver E9)    (优先级排序)   (推理→提取)
```

---

## 项目结构

```
oa-smart-audit/
├── README.md                     # 本文件
├── docker-compose.yml            # 生产环境编排
├── docker-compose.dev.yml        # 开发环境编排
├── .env.example                  # 环境变量模板
│
├── go-service/                   # Go 后端服务
│   ├── cmd/server/main.go        # 应用入口
│   ├── config.yaml               # 默认配置
│   └── internal/
│       ├── config/               # 配置加载
│       ├── dto/                  # 请求/响应 DTO
│       ├── handler/              # HTTP 处理器
│       ├── middleware/           # 中间件（JWT/CORS/日志/权限）
│       ├── model/                # 数据模型
│       ├── pkg/                  # 工具包
│       │   ├── ai/               # AI 模型调用
│       │   ├── crypto/           # AES 加解密
│       │   ├── jwt/              # JWT 签发与验证
│       │   └── oa/               # OA 系统适配器
│       ├── repository/           # 数据访问层
│       └── service/              # 业务逻辑层
│
├── frontend/                     # Nuxt 3 前端
│   ├── pages/                    # 页面路由
│   ├── components/               # 公共组件
│   ├── composables/              # 组合式 API
│   ├── middleware/               # 路由守卫
│   └── locales/                  # 国际化语言包
│
├── db/                           # 数据库
│   └── migrations/               # 迁移脚本（30+ 个）
│
└── docs/                         # 项目文档
    ├── code-review/              # 代码审查报告 ⭐ 新增
    ├── features/                 # 功能说明文档
    ├── database/                 # 数据库设计文档
    └── architecture/             # 技术架构文档
```

---

## 快速开始

### 环境要求

- Docker & Docker Compose
- Node.js 18+（前端本地开发）
- Go 1.25+（后端本地开发，可选）

### 1. 启动基础服务（开发模式）

```bash
# 复制环境变量
cp .env.example .env

# 启动 PostgreSQL + Redis + Go 后端
docker-compose -f docker-compose.dev.yml up -d
```

### 2. 启动前端

```bash
cd frontend
pnpm install
pnpm dev
```

访问 `http://localhost:3000` 进入系统。

### 3. 首次初始化

系统首次启动时会自动检测是否需要初始化：
1. 访问 `/setup` 页面创建系统管理员账号
2. 登录后进入系统管理后台
3. 创建租户并配置 OA 数据库连接
4. 配置 AI 模型
5. 创建流程审核配置

---

## 核心配置说明

### JWT 配置 (`config.yaml`)

```yaml
jwt:
  secret: "change-me-in-production"  # 生产环境必须修改
  access_token_ttl: 2h               # Access Token 有效期
  refresh_token_ttl: 168h            # Refresh Token 有效期（7天）
```

### 数据库配置

```yaml
database:
  host: localhost
  port: 5432
  user: oa_admin
  password: changeme_pg_password
  dbname: oa_smart_audit
  sslmode: disable
```

### 加密配置

```yaml
encryption:
  key: "4f9e2b8c5a1d7f0e3a6c9b2d5e8f1a4c"  # 32 字节 AES-256 密钥
```

---

## 文档目录

### 代码审查报告 ⭐

| 文档 | 说明 |
|------|------|
| [认证系统分析](docs/code-review/01-authentication-analysis.md) | Token 机制、刷新逻辑、过期处理分析 |
| [核心业务逻辑分析](docs/code-review/02-core-business-logic-analysis.md) | OA 数据提取、规则组装、提示词构建分析 |
| [人员组织与配置分析](docs/code-review/03-organization-and-config-analysis.md) | 角色体系、配置层级、数据关联分析 |
| [Bug 清单与优化建议](docs/code-review/04-bug-list-and-optimization.md) | 问题汇总、修复方案、优先级排序 |

### 功能文档

| 文档 | 说明 |
|------|------|
| [OA 适配](docs/features/oa-integration.md) | OA 系统连接与数据适配能力说明 |
| [AI 智能审核](docs/features/ai-audit.md) | AI 审核引擎架构与审核流程说明 |
| [归档复盘方案](docs/features/archive-review-implementation-plan.md) | 归档复盘后端化实施方案 |

### 技术文档

| 文档 | 说明 |
|------|------|
| [数据库设计](docs/database/database-schema.md) | 全部数据表结构与关系说明 |
| [技术架构](docs/architecture/technical-architecture.md) | 系统架构、API 接口说明 |

---

## 已知问题

> 详见 [Bug 清单与优化建议](docs/code-review/04-bug-list-and-optimization.md)

### 高优先级

1. **Token TTL 配置不同步**: 数据库配置与 config.yaml 配置未关联
2. **Token 过期未自动清理**: 前端在 Token 过期后不会自动清除本地状态

### 中优先级

3. **默认密码硬编码**: 创建成员时使用固定默认密码 "123456"
4. **审批流信息未接入**: 提示词中的审批流占位符使用固定文本

---

## 开发指南

### 添加新的 OA 适配器

1. 在 `go-service/internal/pkg/oa/` 下创建新适配器
2. 实现 `OAAdapter` 接口
3. 在 `NewOAAdapter` 工厂函数中注册

### 添加新的 AI 模型

1. 在系统管理后台添加 AI 模型配置
2. 支持 OpenAI 兼容协议的模型可直接使用
3. 非兼容协议需在 `go-service/internal/pkg/ai/` 中添加适配

### 添加新的系统配置

1. 在 `db/migrations/` 中添加迁移脚本
2. 在 `system_config_service.go` 中添加读取逻辑
3. 在前端系统设置页面添加配置项

---

## 许可证

内部项目，仅限授权使用。
