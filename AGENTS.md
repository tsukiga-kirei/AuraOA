# AuraOA Agent 开发指引

本文档是 [`docs/development-guide.md`](docs/development-guide.md) 的**简版**，供 AI Agent 与快速开发参考。完整规范以开发指南为准。

---

## 1. 项目范围

| 范围 | 路径 |
|------|------|
| 后端 | `go-service/`（Gin + GORM + 分层：handler → service → repository） |
| 前端 | `frontend/`（Nuxt 3 + Ant Design Vue） |
| 接口文档 | `docs/api/` |
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
    RequestType:  "audit",      // 场景：audit | archive | summary | 新业务须扩展并同步统计 SQL
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

参考：[`docs/ai-integration.md`](docs/ai-integration.md)

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

### 4.4 租户与数据

- Repository 查询带租户条件，使用 `WithTenant(c)`。  
- 结构变更只通过 `db/migrations/` 迁移，禁止手改生产库。  
- 接口变更同步更新 `docs/api/` 对应文档。

---

## 5. 前端要点

- 用户可见文案走 `useI18n()` + `locales/zh-CN.ts` / `en-US.ts`，**禁止**硬编码 UI 文案。  
- API 用 `authFetch<T>()`，类型放 `types/`，字段 `snake_case`。  
- 服务端分页：`page` / `page_size`，响应用 `PagedResult<T>`。  
- 审核/归档的轮询、SSE、取消任务逻辑保持对称。

---

## 6. 合并前自检（精简）

- [ ] 时间相关逻辑是否使用 `apptime`，SQL 聚合是否传入 `apptime.Name()`  
- [ ] 注释与 Zap 日志消息是否为中文（用户界面文案除外，走 i18n）  
- [ ] 新增 AI 调用是否经 `AIModelCallerService` 并写入 LLM 日志表  
- [ ] 审核/归档对称模块是否两侧一致  
- [ ] 租户隔离、分页字段、`docs/api` 是否已对齐  
- [ ] 无敏感信息写入日志  

---

## 7. 延伸阅读

| 文档 | 说明 |
|------|------|
| [开发规范（完整版）](docs/development-guide.md) | 分页、i18n、API、Git 等详细约定 |
| [AI 集成](docs/ai-integration.md) | 两阶段审核、模型调用架构 |
| [API 总览](docs/api/README.md) | 认证、响应格式、路由索引 |
