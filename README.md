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
- **认证**：JWT（Access Token + Refresh Token）
- **配置**：Viper（YAML + 环境变量）
- **日志**：Zap
- **加密**：AES-256（数据库密码等敏感字段）

### 前端（Frontend）
- **框架**：Nuxt 3（SSR 关闭，SPA 模式）
- **UI 库**：Ant Design Vue 4
- **语言**：TypeScript / Vue 3 Composition API
- **国际化**：自研 i18n（基于 `zh-CN.ts` / `en-US.ts`）
- **数据可视化**：内置图表组件（基于 mock 数据）

### 基础设施
- **容器化**：Docker Compose（开发环境 + 生产环境）
- **数据库迁移**：PostgreSQL 原生 SQL 迁移脚本（`db/migrations/`）
- **种子数据**：SQL 种子脚本（`db/seeds/`）

---

## 项目结构

```
oa-smart-audit/
├── README.md                     # 本文件
├── docker-compose.yml            # 生产环境编排（含 AI 服务）
├── docker-compose.dev.yml        # 开发环境编排（仅基础设施）
├── .env.example                  # 环境变量模板
│
├── go-service/                   # Go 后端服务
│   ├── cmd/server/main.go        # 应用入口
│   ├── config.yaml               # 默认配置
│   └── internal/
│       ├── config/               # 配置加载
│       ├── dto/                  # 请求/响应数据传输对象
│       ├── handler/              # HTTP 处理器（Controller 层）
│       ├── middleware/           # 中间件（JWT/CORS/日志/权限）
│       ├── model/                # 数据模型（对应数据库表）
│       ├── pkg/                  # 工具包
│       │   ├── ai/               # AI 模型调用（OpenAI 兼容协议）
│       │   ├── crypto/           # AES 加解密
│       │   ├── errcode/          # 统一错误码
│       │   ├── hash/             # bcrypt 密码哈希
│       │   ├── jwt/              # JWT 签发与验证
│       │   ├── oa/               # OA 系统适配器
│       │   ├── response/         # 统一响应格式
│       │   └── sanitize/         # 数据脱敏
│       ├── repository/           # 数据访问层（Repository 模式）
│       ├── router/               # 路由注册
│       └── service/              # 业务逻辑层
│
├── frontend/                     # Nuxt 3 前端
│   ├── pages/                    # 页面路由
│   │   ├── login.vue             # 登录页
│   │   ├── dashboard.vue         # 仪表盘
│   │   ├── overview.vue          # 审核工作台
│   │   ├── cron.vue              # 定时任务
│   │   ├── archive.vue           # 归档复盘
│   │   ├── settings.vue          # 个人设置
│   │   └── admin/                # 管理后台
│   ├── components/               # 公共组件
│   ├── composables/              # 组合式 API（业务逻辑封装）
│   ├── layouts/                  # 布局模板
│   ├── locales/                  # 国际化语言包
│   ├── types/                    # TypeScript 类型定义
│   └── middleware/               # 路由守卫
│
├── db/                           # 数据库
│   ├── migrations/               # 迁移脚本（000001 ~ 000011）
│   └── seeds/                    # 种子数据（001 ~ 012）
│
└── docs/                         # 项目文档
    ├── features/                 # 功能说明文档
    ├── database/                 # 数据库设计文档
    ├── architecture/             # 技术架构文档
    └── todo/                     # 待办事项与改进计划
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

Go 后端日志写入 Docker named volume `go_logs_dev`（挂载至容器 `/app/logs`），可通过以下命令查看：

```bash
docker volume inspect go_logs_dev
```

### 2. 启动前端

```bash
cd frontend
pnpm install
pnpm dev
```

访问 `http://localhost:3000` 进入系统。

### 3. 全栈启动（生产模式）

```bash
docker-compose up -d
```

---

## 系统角色

| 角色 | 标识 | 说明 |
|------|------|------|
| 系统管理员 | `system_admin` | 管理租户、OA 连接、AI 模型、系统配置 |
| 租户管理员 | `tenant_admin` | 管理组织架构、流程配置、审核规则、用户配置 |
| 业务用户 | `business` | 使用审核工作台、归档复盘、个人设置 |

---

## 文档目录

| 文档 | 说明 |
|------|------|
| [功能说明 — OA 适配](docs/features/oa-integration.md) | OA 系统连接与数据适配能力说明 |
| [功能说明 — AI 智能审核](docs/features/ai-audit.md) | AI 审核引擎架构与审核流程说明 |
| [功能说明 — 归档复盘后端化方案](docs/features/archive-review-implementation-plan.md) | `archive.vue` 运行时后端化、队列、接口、数据结构与实施建议 |
| [数据库设计](docs/database/database-schema.md) | 全部数据表结构与关系说明 |
| [技术架构](docs/architecture/technical-architecture.md) | 系统架构、前后端关联、API 接口说明 |
| [TODO — 业务待办](docs/todo/business-todo.md) | 审核工作台、定时任务、归档复盘、仪表盘、消息等业务待办 |
| [TODO — 细节改进](docs/todo/detail-todo.md) | 默认账号、提示词模板、前端分页等细节改进 |
| [TODO — 技术改造](docs/todo/technical-todo.md) | Redis 扩展、消息队列、OCR、后端分页等技术改造计划 |
| [TODO — 脏数据清理](docs/todo/personal-config-dirty-data-cleanup.md) | 个人配置脏数据清理方案（已有） |

---

## 许可证

内部项目，仅限授权使用。
