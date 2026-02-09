# 实现计划：OA智审（流程智能审核平台）

## 概述

基于设计文档，按增量方式实现 OA 智审平台第一阶段（规则库模式）。每个任务构建在前一个任务之上，确保无孤立代码。聚焦规则库（Rules_Only）模式，RAG 和 OCR 接口预留但不实现。

## 任务

- [x] 1. 项目脚手架与基础设施搭建
  - [x] 1.1 创建项目根目录结构和 Docker Compose 编排文件
    - 创建 `frontend/`、`go-service/`、`ai-service/`、`db/init/` 目录
    - 编写 `docker-compose.yml`（PostgreSQL pgvector、MongoDB、Go 服务、Python 服务、Nuxt 前端）
    - 创建 `.env.example` 环境变量模板
    - 创建 `.gitignore` 文件
    - _Requirements: 全局基础设施_

  - [x] 1.2 初始化 Go 业务服务项目
    - 初始化 Go module（`go mod init`），创建 `cmd/server/main.go` 入口
    - 搭建目录结构：`internal/auth/`、`internal/rule/`、`internal/oa/`、`internal/security/`、`internal/cron/`、`internal/history/`、`internal/tenant/`
    - 引入核心依赖：`gin`（HTTP）、`pgx`（PostgreSQL）、`golang-jwt/jwt`、`testify`、`pgregory.net/rapid`
    - 编写 `Dockerfile`
    - _Requirements: 全局基础设施_

  - [x] 1.3 初始化 Python AI 服务项目
    - 创建 `ai-service/` 目录结构：`chains/`、`rag/`、`ocr/`、`models/`
    - 编写 `requirements.txt`（FastAPI、LangChain、hypothesis、pytest）
    - 创建 FastAPI 入口 `main.py` 和健康检查端点
    - 编写 `Dockerfile`
    - _Requirements: 全局基础设施_

  - [x] 1.4 初始化 Nuxt 3 前端项目
    - 使用 `nuxi init` 创建 Nuxt 3 项目
    - 安装 Ant Design Vue、`@ant-design-vue/nuxt`
    - 配置 `nuxt.config.ts`（SSR 模式、API 代理）
    - 创建基础布局 `layouts/default.vue`
    - 编写 `Dockerfile`
    - _Requirements: 13.1_

  - [x] 1.5 创建数据库初始化脚本
    - 编写 `db/init/001_schema.sql`，包含所有 PostgreSQL 业务表（tenants、users、audit_rules、user_private_rules、user_preferences、user_sensitivity、kb_mode_config、cron_tasks、log_retention_policies、ai_configs、masking_rules）
    - pgvector 扩展启用（`CREATE EXTENSION IF NOT EXISTS vector`），向量表留空注释标记第二阶段
    - _Requirements: 数据模型_

- [ ] 2. 检查点 — 确保 Docker Compose 可正常启动所有服务
  - 确保所有服务可通过 `docker-compose up` 启动，数据库初始化脚本执行成功，如有问题请询问用户。

- [x] 3. 用户认证与 RBAC 权限
  - [x] 3.1 实现 Go 认证鉴权模块
    - 实现 `AuthService` 接口：`Login`（密码验证 + JWT 签发）、`ValidateToken`、`RefreshToken`
    - JWT payload 包含 user_id、tenant_id、role
    - 实现 `RBACService` 接口：`GetUserMenus`（根据角色返回菜单）、`CheckPermission`
    - 编写 Gin 中间件：JWT 验证中间件、权限检查中间件
    - 注册 `/api/auth/login`、`/api/auth/refresh`、`/api/auth/menu` 路由
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [ ]* 3.2 编写认证模块属性测试
    - **Property 1: JWT Token 包含角色信息**
    - **Property 2: RBAC 菜单权限过滤**
    - **Property 3: 无效 Token 拒绝**
    - **Property 4: 未授权资源访问拒绝**
    - **Validates: Requirements 1.1, 1.2, 1.3, 1.4**

  - [x] 3.3 实现 Nuxt 前端登录页和动态路由
    - 创建 `pages/login.vue` 登录页面
    - 实现 `composables/useAuth.ts`（login、getMenu、token 管理）
    - 实现 `middleware/auth.ts` 路由守卫（未登录跳转登录页）
    - 基于菜单数据动态注册路由（`/dashboard`、`/cron`、`/archive`、`/admin/*`）
    - _Requirements: 1.5, 13.2_

- [x] 4. 租户管理与系统配置
  - [x] 4.1 实现 Go 租户管理模块
    - 实现 `TenantService` 接口：`CreateTenant`（初始化配置空间）、`UpdateTenantQuota`、`GetTenantConfig`、`SetKBMode`
    - 注册 `/api/admin/tenants` CRUD 路由（系统管理员权限）
    - 实现 Token 配额检查中间件（请求前检查 token_used < token_quota）
    - 实现并发数控制（基于 semaphore 限制同时审核请求数）
    - _Requirements: 7.1, 7.2, 7.5, 6.1_

  - [ ]* 4.2 编写租户管理属性测试
    - **Property 17: 租户数据隔离**
    - **Property 18: Token 配额限制**
    - **Property 19: 并发数控制**
    - **Property 16: KB_Mode 配置往返一致性**
    - **Validates: Requirements 7.1, 7.2, 7.5, 6.1**

- [x] 5. 规则引擎与个性化偏好
  - [x] 5.1 实现 Go 规则引擎核心
    - 实现 `RuleEngine` 接口：`MergeRules`（按优先级合并：租户强制 > 用户私有 > 租户默认）、`GetConfigurableRules`
    - 实现规则 CRUD API（租户管理员）：`/api/admin/rules`
    - 实现用户偏好 API：`/api/preferences`（规则开关、私有规则、敏感度设置）
    - 偏好持久化到 user_preferences、user_private_rules、user_sensitivity 表
    - _Requirements: 3.1, 3.2, 3.4, 3.5, 6.5, 6.6_

  - [ ]* 5.2 编写规则引擎属性测试
    - **Property 9: 规则优先级合并顺序**
    - **Property 8: 私有规则数据隔离**
    - **Property 7: 规则 UI 可编辑性由作用域决定**
    - **Property 10: 用户偏好持久化往返一致性**
    - **Validates: Requirements 3.1, 3.2, 3.4, 3.5, 6.5, 6.6**

- [ ] 6. 检查点 — 确保认证、租户、规则引擎测试通过
  - 确保所有测试通过，如有问题请询问用户。

- [x] 7. 数据脱敏与 OA 适配器
  - [x] 7.1 实现 Go 脱敏处理模块
    - 实现 `DataMasker` 接口：`MaskFormData`（正则匹配 + 替换）、`LoadMaskingRules`
    - 支持可配置脱敏规则（从 masking_rules 表加载）
    - 无法识别格式时使用默认全遮蔽策略（`***`）
    - 注册脱敏规则管理 API：`/api/admin/masking-rules`
    - _Requirements: 10.1, 10.2, 10.3_

  - [ ]* 7.2 编写脱敏模块属性测试
    - **Property 26: 敏感字段脱敏正确性**
    - **Validates: Requirements 10.1, 10.2**

  - [x] 7.3 实现 OA 适配器（泛微 E9）
    - 实现 `OAAdapter` 接口：`FetchTodoProcesses`、`FetchProcessDetail`、`HealthCheck`
    - 实现 `AdapterRegistry`：按 OA 类型和版本加载适配器
    - 实现泛微 E9 表结构映射（将 OA 原始数据映射为统一 `ProcessFormData`）
    - 实现连接断线重试逻辑（可配置重试间隔和最大重试次数）
    - 实现流程选择器（按目录、路径或 ID 过滤）
    - _Requirements: 8.1, 8.3, 8.4_

  - [ ]* 7.4 编写 OA 适配器属性测试
    - **Property 20: OA 数据映射一致性**
    - **Property 21: OA 连接断线重试**
    - **Property 22: 流程选择器过滤**
    - **Validates: Requirements 8.1, 8.3, 8.4**

- [x] 8. Python AI 审核引擎（Checklist Chain）
  - [x] 8.1 实现 Checklist Chain 审核逻辑
    - 实现 `ChainOrchestrator.execute_audit`（根据 kb_mode 分发，第一阶段仅 rules_only）
    - 实现 `_run_checklist_chain`：逐条执行结构化规则，调用 LLM 判断每条规则的通过/不通过
    - RAG 和 Hybrid 方法抛出 `NotImplementedError`
    - 定义请求/响应模型（Pydantic）：`AuditRequest`、`AuditResponse`、`ChecklistResult`
    - 注册 FastAPI 路由：`POST /api/audit`
    - 确保每次审核记录完整推理过程（ai_reasoning 字段）
    - _Requirements: 9.1, 9.2, 9.5, 6.2_

  - [ ]* 8.2 编写 AI 审核引擎属性测试
    - **Property 15: KB_Mode 链选择正确性**（第一阶段仅验证 Rules_Only 分支）
    - **Property 23: Checklist Chain 结果完整性**
    - **Property 6: AI 审核建议有效性**
    - **Property 25: AI 推理过程记录**
    - **Validates: Requirements 9.1, 9.2, 9.5, 2.5**

- [x] 9. 审核工作台端到端串联
  - [x] 9.1 实现 Go 审核编排流程
    - 编写审核编排 handler：接收前端请求 → 加载规则（MergeRules）→ 脱敏（MaskFormData）→ 调用 Python AI 服务 → 保存审核快照
    - 注册 `/api/audit/execute` 路由
    - 在请求中注入 Trace ID（UUID），传递给 Python 服务
    - Token 配额扣减（审核完成后 token_used++）
    - _Requirements: 2.1, 2.5, 10.4, 11.1, 12.1_

  - [ ]* 9.2 编写审核快照属性测试
    - **Property 14: 审核快照完整性**
    - **Property 27: 快照中无原始敏感数据**
    - **Property 28: 审核快照不可篡改**
    - **Property 30: 跨服务 Trace ID 传播**
    - **Validates: Requirements 5.3, 10.4, 11.2, 11.4, 12.1**

  - [x] 9.3 实现 Nuxt 审核工作台页面
    - 创建 `pages/dashboard.vue`：待办流程列表（分页、搜索）
    - 创建 `components/AuditPanel.vue`：规则侧边栏 + AI 推理展示面板
    - 创建 `components/RuleList.vue`：规则清单（通过/不通过状态、Toggle 开关、锁定标识）
    - 实现 `composables/useAudit.ts`：获取待办列表、触发审核、提交反馈
    - 实现骨架屏加载状态
    - _Requirements: 2.2, 2.3, 2.4, 2.6, 13.3, 13.4_

- [ ] 10. 检查点 — 确保审核工作台端到端流程可运行
  - 确保所有测试通过，如有问题请询问用户。

- [x] 11. 历史记录与归档复盘
  - [x] 11.1 实现 Go 历史记录服务
    - 实现 `HistoryService` 接口：`SaveAuditSnapshot`（追加写入 MongoDB）、`SearchSnapshots`（按条件检索）、`ExportSnapshots`（导出为 JSON/CSV）
    - 审核快照写入采用 append-only 模式，禁止 update/delete 操作
    - 注册 `/api/history/search`、`/api/history/export` 路由
    - _Requirements: 5.2, 5.3, 11.1, 11.2, 11.3_

  - [ ]* 11.2 编写历史记录属性测试
    - **Property 13: 历史检索过滤正确性**
    - **Property 29: 审计导出过滤正确性**
    - **Validates: Requirements 5.2, 11.3**

  - [x] 11.3 实现 Nuxt 归档复盘页面
    - 创建 `pages/archive.vue`：历史检索界面（时间、部门、流程类型过滤）
    - 创建 `components/SnapshotDetail.vue`：审核快照详情展示（Prompt、AI 回复、采纳状态）
    - 实现导出功能按钮
    - _Requirements: 5.2, 5.4, 11.3_

- [x] 12. 定时任务中心
  - [x] 12.1 实现 Go Cron 任务调度
    - 实现 `CronScheduler` 接口：`CreateTask`、`DeleteTask`、`ListTasks`、`ExecuteTask`
    - 集成 Go cron 库（如 `robfig/cron`），注册用户任务
    - 实现批量审核执行逻辑（遍历待办 → 逐条审核 → 汇总结果）
    - 实现失败重试（更新 next_run_at）
    - 注册 `/api/cron` CRUD 路由
    - _Requirements: 4.1, 4.2, 4.5_

  - [ ]* 12.2 编写 Cron 任务属性测试
    - **Property 11: Cron 任务注册往返一致性**
    - **Property 12: 失败任务重试**
    - **Validates: Requirements 4.1, 4.5**

  - [x] 12.3 实现 Nuxt 定时任务页面
    - 创建 `pages/cron.vue`：任务列表、创建任务表单（cron 表达式选择器）
    - 创建 `components/CronHistory.vue`：历史推送记录展示
    - _Requirements: 4.3, 4.4_

- [x] 13. 后台管理页面
  - [x] 13.1 实现 Nuxt 租户管理员配置页面
    - 创建 `pages/admin/tenant.vue`：知识库模式配置（第一阶段仅 Rules_Only 可选）、规则管理（CRUD + 分级设置）、日志留存策略配置
    - 创建 `components/RuleEditor.vue`：规则编辑器（内容、作用域、优先级）
    - _Requirements: 6.1, 6.5, 6.7_

  - [x] 13.2 实现 Nuxt 系统管理员配置页面
    - 创建 `pages/admin/system.vue`：租户管理（CRUD）、Token 配额设置、OA 集成配置、并发数控制
    - 创建 `pages/admin/monitor.vue`：全局监控面板（系统健康度、API 成功率、模型响应时间）
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 14. 前端主题与体验优化
  - [x] 14.1 实现暗色/亮色模式切换和响应式布局
    - 配置 Ant Design Vue 主题 token（亮色/暗色）
    - 实现 `composables/useTheme.ts` 主题切换逻辑
    - 优化所有页面的响应式布局（桌面端 + 平板端）
    - _Requirements: 13.3, 13.5_

- [x] 15. 可观测性集成
  - [x] 15.1 实现跨服务链路追踪和监控指标
    - Go 服务：请求入口生成 Trace ID，通过 HTTP Header 传递给 Python 服务
    - Python 服务：从 Header 提取 Trace ID，记录到日志
    - Go 服务：记录 API 调用成功率和模型响应时间指标
    - 实现阈值告警逻辑（指标超阈值时写入告警记录）
    - _Requirements: 12.1, 12.2, 12.3, 12.4_

  - [ ]* 15.2 编写可观测性属性测试
    - **Property 30: 跨服务 Trace ID 传播**
    - **Property 31: 阈值告警触发**
    - **Validates: Requirements 12.1, 12.4**

- [ ] 16. 最终检查点 — 全量测试通过
  - 确保所有测试通过，如有问题请询问用户。

## 备注

- 标记 `*` 的子任务为可选测试任务，可跳过以加速 MVP 交付
- 每个任务引用具体需求编号，确保可追溯
- 第二阶段（RAG 制度库、OCR、混合模式）的接口已预留，待第一阶段稳定后实现
- 属性测试验证通用正确性，单元测试覆盖边界和错误条件
