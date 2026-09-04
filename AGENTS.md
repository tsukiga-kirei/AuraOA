# AuraOA Agent 开发指引

本文档是 [`docs/development-guide.md`](docs/development-guide.md) 的**简版**，供 AI Agent 与快速开发参考。完整规范以开发指南为准。

---

## 1. 项目范围

| 范围 | 路径 |
|------|------|
| 后端 | `go-service/`（Gin + GORM + 分层：handler → service → repository） |
| 前端 | `frontend/`（Nuxt 3 + Ant Design Vue） |
| 接口文档 | `docs/api/`（Markdown，按模块拆分） |
| 智能体需求/设计 | `docs/agents/`（对话、两级分配、系统工具/MCP/Skills、OA 适配器） |
| 数据库迁移 | `db/migrations/` |

**核心原则**：最小改动、沿用既有约定、租户隔离、前后端 `snake_case` 字段一致。

---

## 2. 时区（必须遵守）

应用统一时区由配置 `app.timezone`（默认 `Asia/Shanghai`）在启动时注入，**禁止**在业务代码中硬编码时区或假设 UTC。

| 场景 | 正确做法 |
|------|----------|
| 获取「当前业务时间」 | `apptime.Now()` |
| 格式化展示时间 | `apptime.FormatRFC3339(t)` |
| 提示词中的日期变量 | `apptime.Now()` / `apptime.Location()`（见 `prompt_system_variables.go`） |
| SQL 按自然日/周聚合 | 传入 `apptime.Name()` 给 `AT TIME ZONE ?`（见 `llm_message_log_repo.go`、`audit_log_repo.go`） |
| 文件名中的日期戳 | `apptime.Now().Format("20060102")` |

**不要**：`time.Now().In(time.FixedZone(...))`、写死 `"Asia/Shanghai"`、在统计 SQL 里省略时区参数。

包路径：`go-service/internal/pkg/apptime`。

---

## 3. 注释与日志（使用中文）

### 3.1 注释

- 导出函数、Handler、Service 公共方法、非显而易见的业务规则：**简体中文**。
- Handler 建议注明 HTTP 方法、路径、主要参数、返回结构。
- Composable 顶部 JSDoc 列出对接的路由前缀。
- 自解释的 CRUD 透传不必废话注释。

### 3.2 日志

先判断「给谁用」，再选记录方式：

| 内容 | 用什么 |
|------|--------|
| 租户审核/归档/OA/AI 长链路排错 | `pkglogger.GetTenantLogger(tenant.Code)` |
| 系统级事件（启动、Cron、备份） | `pkglogger.Global()` |
| 用户界面要看的审核/归档结果 | `audit_logs` / `archive_logs` + 快照表 |
| 每次 AI 调用的 Token 与耗时 | `tenant_llm_message_logs`（见下文） |
| HTTP 访问 | 中间件已处理，**Handler 不要重复打** |

- 日志消息可用中文；字段用 `zap.String` / `zap.Error` 等结构化键。
- **禁止** `fmt.Println`、记录密码/完整 Token/未脱敏提示词正文。

---

## 4. 系统耦合（新增功能必读）

AuraOA 各模块相互联动。新增或修改功能时，须检查是否触达下列耦合点。

### 4.1 AI 调用 → 必须走统一入口并落库

**所有**会消耗 LLM Token 的调用，必须通过 `AIModelCallerService.Chat` / `ChatWithFallback`，**禁止**在业务 Service 里直接 `ai.NewAIModelCaller` 或裸调 HTTP。

`AIModelCallerService` 会自动完成：

1. Token 配额预扣与结算  
2. 异步写入 `tenant_llm_message_logs` + `tenant_llm_message_payloads`  
3. 失败重试与降级日志  

调用时在 `ai.ChatRequest` 中填写元数据，供数据管理页与仪表盘统计：

```go
&ai.ChatRequest{
    RequestType:  "audit",      // 场景：audit | archive | summary | chat | 新业务须扩展并同步统计 SQL
    CallType:     "reasoning",  // 类型：reasoning | structured
    ProcessID:    processID,
    ProcessTitle: title,
    BusinessLogID: &logID,      // 关联 audit_logs / archive_logs 等
    // ...
}
```

新增 AI 场景时同步检查：

- [ ] 是否经 `AIModelCallerService` 调用  
- [ ] `request_type` / `call_type` 是否有意义且与现有枚举一致  
- [ ] `llm_message_log_repo.go` 统计（如 `CountStats` 按场景分布）是否需扩展  
- [ ] 前端数据管理页 `admin/data` 筛选与 i18n 是否需补充  
- [ ] 仪表盘 `DashboardOverviewService` 是否聚合新指标  

参考：[`docs/ai-integration.md`](docs/ai-integration.md)。对话/智能体另见 [`docs/agents/README.md`](docs/agents/README.md)。

### 4.2 审核工作台 ↔ 归档复盘（对称模块）

两侧能力成对出现，改一侧须评估另一侧：

| 维度 | 审核 | 归档 |
|------|------|------|
| 页面 | `pages/dashboard.vue` | `pages/archive.vue` |
| API | `/api/audit` | `/api/archive` |
| Service | `AuditExecuteService` | `ArchiveReviewService` |
| 日志表 | `audit_logs` + `audit_process_snapshots` | `archive_logs` + `archive_process_snapshots` |
| LLM `request_type` | `audit` | `archive` |
| 提示词前缀 | `audit_*` | `archive_*` |

### 4.3 其他共享能力

修改以下模块时扩大回归范围：

- `AttachmentRecognitionService`（附件识别）  
- OA 适配器（`ecology9` 等）  
- `UserPersonalConfigService`（同时服务审核与归档）  
- `rule_merge.go`（`AuditRule` / `ArchiveRule`）  
- Redis 缓存与 `invalidationManager`  
- `CronTaskService`、仪表盘 `DashboardOverviewService`  
- 智能体运行时（系统工具必须走 `OAAdapter`；配额由系统管理员分配、租户管理员再分配，见 `docs/agents/allocation.md`）

### 4.4 租户与数据

- Repository 查询带租户条件，使用 `WithTenant(c)`。  
- 结构变更只通过 `db/migrations/` 迁移，禁止手改生产库。  
- 接口变更同步更新 `docs/api/` 对应文档（及 OpenAPI，若已引入，见 §7）。

---

## 5. 分页规范

项目存在**服务端分页**与**客户端分页**两种模式，按数据量选择，**不可混用语义**。

### 5.1 服务端分页（列表接口默认）

**查询参数**（前后端统一）：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `page` | int | `1` | 从 1 开始 |
| `page_size` | int | `20` | 后端 Repository 通常限制 `1–100`，非法回落 20 |

**响应结构**（`snake_case`，字段名固定）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [],
    "total": 0,
    "page": 1,
    "page_size": 20
  }
}
```

前端类型：`PagedResult<T>`（`frontend/types/admin-data.ts` 等），与后端 DTO 字段一致。

**后端**：列表分页方法内校正 `page`、`page_size` 范围；Handler 用 `parseIntQuery` 等统一辅助函数解析。

**前端**：

```ts
const query = computed(() => ({
  keyword: keyword.value || undefined,
  page: currentPage.value,
  page_size: pageSize.value,
}))
```

- 翻页容器用 `.pagination-wrapper` + `a-pagination`。  
- `page-size-options` 与项目一致：`['10', '20', '50']`。  
- 切换 `page_size` 时通常重置 `page` 为 1（与现有页面行为保持一致）。

**例外**：部分接口（如通知列表）使用 `limit` / `offset`，须在 composable 与 `docs/api` 中单独说明，**不得**与 `page` / `page_size` 混用于同一列表。

### 5.2 客户端分页

适用于**已一次性加载到内存**的数组（如租户内配置列表过滤后展示）：

- 使用 `usePagination(sourceRef, defaultPageSize)`（默认 20）。  
- 模板绑定 `current`、`pageSize`、`total`、`onChange`。

### 5.3 导出与全量拉取

- 导出接口按筛选条件导出全量，**不**依赖前端当前页数据。  
- 后端内部全量拉取常用 `page=1, page_size=5000` 等固定上限，新增导出须评估数据量与超时。

### 5.4 分页检查清单

- [ ] 列表接口在 `docs/api/<模块>.md` 写明分页参数与响应字段  
- [ ] 前端 `types` 与后端 DTO 字段名一致  
- [ ] 空列表、加载中、错误状态均有 i18n 文案  
- [ ] 「筛选 + 翻页」组合行为已手动验证  

---

## 6. API 与文档体系

### 6.1 统一响应与认证

完整约定见 [`docs/api/README.md`](docs/api/README.md)。要点：

- 前缀：`/api`；成功 `code: 0`，`data` 为业务载荷。  
- 认证：`Authorization: Bearer <access_token>`；中间件链 `JWT` → `TenantContext` → `RequireRole`。  
- JSON 字段 **snake_case**；新增错误码在 `go-service/internal/pkg/errcode/` 登记。  
- 修改路由、参数、响应字段时，**同一 PR** 更新对应 `docs/api/<模块>.md`。

### 6.2 模块文档索引（按任务查阅）

| 文档 | 何时读 |
|------|--------|
| [`docs/api/README.md`](docs/api/README.md) | 认证、响应格式、路由总索引 |
| [`docs/api/auth.md`](docs/api/auth.md) | 登录、Token、角色切换 |
| [`docs/api/audit.md`](docs/api/audit.md) | 审核工作台执行、列表、SSE |
| [`docs/api/audit-config.md`](docs/api/audit-config.md) | 流程审核配置、规则、提示词 |
| [`docs/api/archive.md`](docs/api/archive.md) | 归档复盘（与 audit 对称） |
| [`docs/api/archive-config.md`](docs/api/archive-config.md) | 归档复盘配置 |
| [`docs/api/summary.md`](docs/api/summary.md) | 流程总结配置与快照 |
| [`docs/api/llm-logs.md`](docs/api/llm-logs.md) | AI 调用记录（数据管理页） |
| [`docs/api/user-settings.md`](docs/api/user-settings.md) | 个人配置、仪表盘偏好 |
| [`docs/api/cron.md`](docs/api/cron.md) | 定时任务 |
| [`docs/api/org.md`](docs/api/org.md) | 组织架构 |
| [`docs/api/system-admin.md`](docs/api/system-admin.md) | 系统/租户管理、AI 模型 |
| [`docs/api/embed.md`](docs/api/embed.md) | OA 嵌入页 |
| [`docs/api/cache.md`](docs/api/cache.md) | 缓存管理 |
| [`docs/ai-integration.md`](docs/ai-integration.md) | AI 调用架构、两阶段审核 |
| [`docs/oa-integration.md`](docs/oa-integration.md) | OA 适配器与取数 |
| [`docs/agents/README.md`](docs/agents/README.md) | 智能体需求：两级分配、系统工具/MCP/Skills、对话 |
| [`docs/api/chat.md`](docs/api/chat.md) | 对话 HTTP/SSE（拟定） |
| [`docs/api/agents.md`](docs/api/agents.md) | 智能体配额与租户管理 API（拟定） |
| [`docs/development-guide.md`](docs/development-guide.md) | i18n、Git、完整日志规范等 |

**Agent 工作流建议**：改接口前先读 `docs/api/README.md` + 对应模块 md；改 AI 链路再读 `ai-integration.md`；改审核/归档时成对阅读 `audit.md` 与 `archive.md`；改对话/智能体先读 `docs/agents/` 再动代码。

### 6.3 OpenAPI（`auraoa.openapi.yaml`）

机器可读契约：[`docs/api/auraoa.openapi.yaml`](docs/api/auraoa.openapi.yaml)（OpenAPI `3.0.3`，覆盖 `router.go` 全部 **175** 个端点）。

| 项 | 说明 |
|----|------|
| 与 Markdown 关系 | **互补**：`docs/api/*.md` 保留叙述与示例；OpenAPI 承载路径、参数、响应 schema，供 Agent / IDE 预览 |
| 维护原则 | 新增/变更接口时，**同一 PR** 更新对应 `.md` **且** 手工同步 OpenAPI 中对应 `paths` / `components` |

二者与代码冲突时，以**代码实际行为**为准，并回头修订 Markdown 与 YAML。

---

## 7. 前端要点

- 用户可见文案走 `useI18n()` + `locales/zh-CN.ts` / `en-US.ts`，**禁止**硬编码 UI 文案。  
- API 用 `authFetch<T>()`，类型放 `types/`，字段 `snake_case`。  
- 列表分页遵循 §5；审核/归档的轮询、SSE、取消任务逻辑保持对称。

---

## 8. 合并前自检（精简）

- [ ] 时间相关逻辑是否使用 `apptime`，SQL 聚合是否传入 `apptime.Name()`  
- [ ] 注释与 Zap 日志消息是否为中文（用户界面文案除外，走 i18n）  
- [ ] 新增 AI 调用是否经 `AIModelCallerService` 并写入 LLM 日志表  
- [ ] 审核/归档对称模块是否两侧一致  
- [ ] 分页参数与 `PagedResult` 结构符合 §5；前后端字段一致  
- [ ] `docs/api` 与 `auraoa.openapi.yaml` 已同步更新  
- [ ] 租户隔离、无敏感信息写入日志  
