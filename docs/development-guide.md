# AuraOA 开发规范

本文档基于当前代码库与 `docs/api/` 等既有文档整理，供前后端协作与日常开发参考。新增或修改功能时，应优先遵循本规范，并保持与现有实现一致。

---

## 1. 适用范围与目标

| 范围 | 路径 |
|------|------|
| 后端 | `go-service/` |
| 前端 | `frontend/` |
| 接口文档 | `docs/api/` |
| 数据库迁移 | `db/migrations/` |

**目标**：保证多租户 OA 审核场景下，功能可维护、接口可预期、中英文界面一致，并在「审核工作台」与「归档复盘」等对称模块间避免遗漏联动修改。

---

## 2. 项目结构速览

### 2.1 后端分层

```
cmd/server/main.go          # 入口、依赖注入
internal/router/            # 路由与中间件
internal/handler/           # HTTP 绑定、校验、响应
internal/service/           # 业务逻辑
internal/repository/        # GORM 数据访问（含租户过滤）
internal/model/             # 实体
internal/dto/               # 请求/响应结构体
internal/pkg/               # response、errcode、oa、ai、excel 等
```

### 2.2 前端目录

```
pages/           # 文件路由页面
components/      # 可复用组件（PascalCase）
composables/     # useXxx 组合式 API
types/           # 与后端对齐的 TS 类型
locales/         # zh-CN.ts / en-US.ts 文案
middleware/      # 如 auth 路由守卫
```

### 2.3 文档与变更

- 接口行为变更须同步更新 `docs/api/` 对应文档。
- 已知缺陷登记在 `docs/known-issues/`。
- 面向用户的版本变更写入根目录 `CHANGELOG.md`。

---

## 3. 通用原则

1. **最小改动**：只改与需求直接相关的代码，不顺手重构无关模块。
2. **沿用既有约定**：命名、分层、响应格式以仓库现有代码为准，不引入与项目风格冲突的新框架。
3. **租户隔离**：业务数据读写必须经过带 `TenantContext` 的路径，Repository 使用 `WithTenant(c)`，禁止绕过租户过滤。
4. **可观测性**：关键业务路径保留结构化日志（详见 [§7 日志规范](#7-日志规范)）；对外错误使用统一错误码，避免泄露内部堆栈。
5. **安全**：密钥、数据库密码等不得提交到仓库；敏感配置走环境变量或加密存储。

---

## 4. 国际化（i18n）

### 4.1 职责划分

| 层级 | 职责 |
|------|------|
| **前端** | 界面文案、菜单、表单标签、空状态、大部分错误提示 |
| **后端** | 用户 `locale` 持久化；Excel/CSV 导出列头与枚举值；监控类告警可返回 **i18n 键** 供前端翻译 |

前端**不依赖** `vue-i18n`，使用自研 `useI18n()` + `locales/zh-CN.ts`、`locales/en-US.ts` 扁平键字典。

### 4.2 前端文案规范

1. **禁止**在模板或脚本中硬编码面向用户的可见中文/英文（调试日志除外）。
2. 翻译键使用 **点分小写命名空间**，与菜单/页面域一致，例如：
   - `menu.dashboard`、`menu.archive`
   - `admin.userConfigs.noData`
   - `dashboard.filter.processType`
3. **新增键必须同时写入** `zh-CN.ts` 与 `en-US.ts`，保持键名一致。
4. 占位符使用 `{0}`、`{1}`（非 ICU），调用：`t('messages.subtitle', count)` 或 `t('key', [a, b])`。
5. 菜单、侧栏等配置项存 **键名**（如 `labelKey: 'menu.dashboard'`），渲染时 `t(item.labelKey)`。
6. 用户语言偏好由 `useAuth` 的 `userLocale`（`zh-CN` | `en-US`）管理，与后端 `PUT /api/auth/locale` 同步。

### 4.3 后端与导出

- Excel 导出通过 `internal/pkg/excel/i18n.go` 的 `ResolveLocale` 解析语言（JWT → `Accept-Language` → `zh-CN`）。
- 新增导出列或枚举展示时，须在 excel i18n 映射中补充中英文。
- API 错误 `message` 当前多为固定中文；若需前端按错误码展示，应在前端 `useAuth` 的错误码映射或 `locales` 中维护，而非散落硬编码。

### 4.4 检查清单

- [ ] 新页面/组件所有可见文案均走 `t()`
- [ ] 中英文 locale 文件键齐全
- [ ] 动态文案（统计数、分页总数等）使用占位符而非字符串拼接

---

## 5. 分页规范

项目存在 **客户端分页** 与 **服务端分页** 两种模式，按数据量选择，不可混用语义。

### 5.1 服务端分页（默认用于列表接口）

**查询参数**（前后端统一）：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `page` | int | `1` | 从 1 开始 |
| `page_size` | int | `20` | 后端 Repository 通常限制 `1–100`，非法回落 20 |

**响应结构**（统一字段名，snake_case）：

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

前端类型定义为 `PagedResult<T>`（`types/admin-data.ts` 等），字段与后端一致。

**例外**：通知列表等接口使用 `limit` / `offset`，须在 composable 与文档中单独说明，不得与 `page` / `page_size` 混用同一列表。

### 5.2 客户端分页

适用于**已一次性加载到内存**的数组（如租户内配置列表过滤后展示）：

- 使用 `usePagination(sourceRef, defaultPageSize)`（默认 20）。
- 模板使用 `a-pagination`，绑定 `current`、`pageSize`、`total`、`onChange`。
- `page-size-options` 与项目保持一致：`['10', '20', '50']`。

### 5.3 导出与全量拉取

- 导出接口（如 `GET /api/audit/processes/export`）按筛选条件导出，**不**依赖前端当前页数据。
- 后端内部全量拉取常用 `page=1, page_size=5000` 等固定上限，新增导出须评估数据量与超时。

### 5.4 前端实现要点

```ts
// 服务端分页：查询对象含 page / page_size
const query = computed(() => ({
  keyword: keyword.value || undefined,
  page: currentPage.value,
  page_size: pageSize.value,
}))

// 翻页时重置或保留筛选条件需与产品一致，改 bug 时检查「筛选 + 翻页」组合
```

### 5.5 检查清单

- [ ] 列表接口文档写明分页参数与响应字段
- [ ] 前端 `types` 与后端 DTO 字段名一致（`snake_case`）
- [ ] 切换 `page_size` 时是否重置 `page` 为 1（与现有页面行为一致）
- [ ] 空列表、加载中、错误状态均有 i18n 文案

---

## 6. 注释规范

### 6.1 语言与格式

- **业务说明、包注释、导出函数、Handler 方法**：使用 **简体中文**。
- **格式**：
  - 包：`// Package xxx 提供……`
  - 导出符号：`// FuncName 说明……`
  - Handler 建议注明 **HTTP 方法、路径、主要查询参数、返回结构**。
  - 文件内逻辑块可用分隔线：`// ===== 数据加载 =====` 或 `// ─── 审核链 ───`。

### 6.2 必须注释的场景

| 场景 | 说明 |
|------|------|
| 非显而易见的业务规则 | 如强制规则不可删、批量上限 10 条 |
| 与 OA/AI 的集成边界 | 超时、重试、快照与实时状态关系 |
| 前后端契约 | composable 顶部 JSDoc 列出对接的路由前缀 |
| 临时兼容或技术债 | 注明原因与后续处理方向 |
| 对称模块的成对逻辑 | 注明「与 archive/audit 对称」便于检索 |

### 6.3 不必过度注释的场景

- 自解释的 getter、简单 CRUD 透传。
- 与命名重复的废话注释（如 `// 返回列表` 紧跟 `List()`）。

### 6.4 示例（后端 Handler）

```go
// ListProcesses 分页查询审核工作台流程列表，支持多维度过滤。
// GET /api/audit/processes
// 查询参数：tab、keyword、page、page_size 等
// 返回：items + total + page + page_size
func (h *AuditHandler) ListProcesses(c *gin.Context) { ... }
```

### 6.5 示例（前端 composable）

```ts
/**
 * useAuditApi — 审核工作台 API 封装
 * 路由前缀：/api/audit
 *   GET  /processes     分页列表
 *   POST /execute       提交审核任务
 */
```

---

## 7. 日志规范

开发时先判断：**这条信息是给谁用的**（运维排错、用户回看、合规审计、产品统计），再选对应的记录方式。**不要**用文件日志替代库表业务数据，也**不要**在 Handler 里重复记 HTTP 访问日志（中间件已处理）。

### 7.1 选型：什么时候用什么

| 你要记录的内容 | 用什么 | 谁写入 | 典型场景 |
|----------------|--------|--------|----------|
| 单次 HTTP 请求的方法、路径、状态码、耗时 | **中间件** `Logger` → Zap 文件/控制台 | 框架自动 | 一般 **不必** 在 Handler 再打一遍 |
| 用户访问了哪个 API（合规审计） | **`operation_audit_logs`** | `AuditTrail` 中间件 | 已登录、`/api/*`；轮询接口已排除 |
| panic、未恢复的严重错误 | **Zap `Global()`** + `Recovery` 中间件 | 中间件 / 业务 catch | 必须带 `zap.Error`、必要时带堆栈 |
| 服务启动、迁移、连接池、缓存、定时任务等 **与租户无关** 的系统事件 | **`pkglogger.Global()`** | 业务代码 | 组织/租户 CRUD、Cron 注册、备份任务 |
| **某租户** 下审核/归档/OA/AI 长链路的步骤与失败原因 | **`pkglogger.GetTenantLogger(tenant.Code)`** | `AuditExecuteService` / `ArchiveReviewService` 等 | 任务执行、附件识别、推理阶段；便于按租户查 `tenant.log` |
| 用户要在界面看的 **审核/归档结果** | **`audit_logs` / `archive_logs`** + 快照表 | 对应 Service + Repo | 工作台、历史、数据管理；**不是**写进 app.log 代替 |
| 定时任务跑批结果摘要 | **`cron_logs`** | `CronTaskService` | 成功/失败、消息、耗时 |
| 每次 AI 调用的 Token 与耗时 | **`tenant_llm_message_logs`** | `AIModelCallerService`（异步） | 配额、仪表盘；**不要**只在文件里记 Token |
| 登录 IP、设备 | **`login_history`** | `AuthService` 登录成功时 | 个人信息页最近登录 |
| 前端接口失败（开发排查） | **浏览器 `console.error`** | 前端 composable | 带模块前缀；**不**上报后端 |

**前端**：用户可见提示用 `message` + i18n；不把 `console.log` 当产品能力。

### 7.2 运行时文件日志（Zap）

**包路径**：`go-service/internal/pkg/logger`。

| API | 何时使用 |
|-----|----------|
| `Global()` | 无租户上下文，或租户无关逻辑；绝大多数 Service/中间件补录 |
| `GetTenantLogger(tenantCode)` | 已解析出租户，且日志仅与该租户业务排错相关（审核/归档执行链路） |

**日志等级（生产默认 `info`）**：

| 等级 | 何时使用 |
|------|----------|
| `Debug` | 细节多、量大的诊断（GORM 普通 SQL、HTTP 轮询）；生产默认不输出 |
| `Info` | 正常业务里程碑（任务完成、配置保存、启动成功） |
| `Warn` | 可恢复异常、降级、重试耗尽、外部依赖失败但请求未崩 |
| `Error` | 必须介入的错误、数据不一致、panic 恢复 |

**写法约定**：

```go
pkglogger.Global().Info("定时任务类型配置保存成功",
    zap.String("taskType", taskType),
    zap.String("tenantID", tenantID.String()),
)
tlog := pkglogger.GetTenantLogger(tenant.Code)
tlog.Warn("审核任务执行失败",
    zap.String("processID", log.ProcessID),
    zap.Error(err),
)
```

- 消息可用中文；字段用 `zap.String` / `zap.Int` / `zap.Error` 等结构化键。
- **禁止** `fmt.Println`、标准库 `log` 写业务日志（`logger.Init` 之前的启动失败除外）。
- **禁止**记录密码、完整 Token、未脱敏提示词正文；只记 ID、类型、长度、错误信息。

**输出说明**（便于排错，非配置手册）：`Global()` 与租户 logger 均会写入 `logs/app.log`（JSON）并输出到 stdout（格式不同，内容同源）；租户另有一份 `logs/tenants/{code}/tenant.log`。轮转体积由 `config.yaml` 的 `log.max_size_mb`（或 `LOG_MAX_SIZE_MB`）控制。

### 7.3 业务库表：何时写入

| 表 | 何时必须写 / 更新 | 不要用来做什么 |
|----|-------------------|----------------|
| `audit_logs` | 提交审核任务、状态变更、最终结果 | 代替 Zap 打调试栈 |
| `archive_logs` | 归档复盘任务（与审核对称） | 同上 |
| `audit_process_snapshots` / `archive_process_snapshots` | 工作台列表需要展示的流程状态摘要 | 存完整 AI 原文（大字段放 log 表） |
| `cron_logs` | 每次 Cron 实例开始/结束 | 记单次 HTTP |
| `tenant_llm_message_logs` | 每次 `AIModelCallerService` 调用成功返回后 | 手工在业务里拼 Token 统计 |
| `login_history` | 登录成功 | 失败尝试（若未单独设计） |

修改 **审核工作台** 时，同步检查 **归档复盘** 是否也要写 `archive_logs` / 快照 / `GetTenantLogger` / LLM 的 `request_type`（`audit` vs `archive`）。

任务进度里的 `trace_id`（如 `TR-xxxxxxxx`）是给 **前端轮询/SSE** 关联 `audit_logs.id` 的，与 API 统一响应里的 `trace_id` 字段不是同一套机制。

### 7.4 中间件与基础设施（一般无需业务重复实现）

- **HTTP 访问**：`middleware.Logger` 已记录；新增 **高频轮询** 接口（任务状态、未读数、统计刷新）须加入 `middleware/path_class.go` 的 `IsLowValuePollingPath`，避免 INFO 刷屏。
- **操作审计**：`middleware.AuditTrail` 写 `operation_audit_logs`，受 `system.enable_audit_trail` 控制；业务代码无需再插一条「用户调用了某 API」。
- **panic**：`middleware.Recovery` 已记 ERROR + 堆栈。
- **GORM**：`NewGormLogger` 已接入 Zap；慢 SQL / 错误会自动打，业务层不必每条 SQL 再打。

### 7.5 常见错误

| 错误做法 | 应改为 |
|----------|--------|
| Handler 里 `Info` 打印每个请求的 path | 依赖 `Logger` 中间件 |
| 把审核结论只写在 `app.log` | 写入 `audit_logs` + 更新快照 |
| 用文件日志统计 Token 消耗 | 依赖 `tenant_llm_message_logs` |
| 租户审核链路用 `Global()` 且不带 `tenantCode` 字段 | `GetTenantLogger(tenant.Code)` |
| 前端 `console.log` 给用户看错误 | `message.error` + `t('...')` |

### 7.6 检查清单

- [ ] 是否选对了类型（Zap / 库表 / 中间件），没有重复记 HTTP？
- [ ] 租户业务排错是否用了 `GetTenantLogger`？
- [ ] 审核/归档对称能力是否两侧库表与文件日志一致？
- [ ] 是否避免敏感信息？高频接口是否加入轮询降级列表？

---

## 8. API 接口规范

完整约定见 [`docs/api/README.md`](./api/README.md)。开发时须遵守以下要点。

### 8.1 统一响应

- 前缀：`/api`
- 成功：`HTTP 200`，`code: 0`，`data` 为业务载荷
- 失败：`HTTP` 状态与业务 `code` 分离，`message` 为人类可读描述

```json
{ "code": 0, "message": "success", "data": { } }
{ "code": 40001, "message": "参数校验失败", "trace_id": "..." }
```

### 8.2 错误码

- 定义在 `go-service/internal/pkg/errcode/`，按段划分（400xx 参数、401xx 认证、403xx 权限、502xx OA、503xx AI 等）。
- **新增业务错误**须分配未占用码值，并在前端错误映射（如有）中补充。
- 业务层优先返回 `service.ServiceError`，Handler 使用 `handleServiceError` 统一写出。

### 8.3 认证与权限

| 中间件 | 用途 |
|--------|------|
| `JWT` | 解析用户与角色 |
| `TenantContext` | 注入租户 ID |
| `RequireRole("tenant_admin" \| "system_admin")` | 管理端接口 |

`system_admin` 可通过 query `tenant_id` 跨租户操作，须在 Handler/Service 内显式处理，普通用户禁止跨租户。

### 8.4 请求与 DTO

- JSON 字段使用 **snake_case**，与前端 `types` 一致。
- 入参校验：`binding` 标签 + `ShouldBindJSON`；复杂校验在 Service 返回 `ServiceError`。
- 新增接口：补充 `internal/dto/` 结构体，并在 `docs/api/` 增加参数表与示例。

### 8.5 非 JSON 响应

- Excel：`Content-Disposition` + xlsx（注意 locale 列头）。
- SSE：`GET /api/audit/stream/:id`、`GET /api/archive/stream/:id`，`text/event-stream`。

### 8.6 接口文档维护

修改路由、参数、响应字段时，**同一 PR** 内更新：

- `docs/api/<模块>.md`
- 若影响集成方，视情况更新 `docs/oa-integration.md` / `docs/ai-integration.md`

---

## 9. 后端开发规范

### 9.1 命名与文件

| 类型 | 约定 | 示例 |
|------|------|------|
| Handler | `XxxHandler`，方法接收 `*gin.Context` | `AuditHandler` |
| Service | `XxxService`，构造函数 `NewXxxService` | `AuditExecuteService` |
| Repository | `XxxRepo`，嵌入 `BaseRepo` | `AuditLogRepo` |
| DTO | `internal/dto/audit_list_dto.go` | `AuditProcessListResponse` |

注意：审核执行服务类型为 `AuditExecuteService`，实现文件可能为 `audit_review_service.go`，以代码为准，新增文件避免再引入易混淆命名。

### 9.2 Handler 标准流程

1. 认证信息检查（如 `getUsername(c)`）
2. 绑定/解析参数（query 用 `parseIntQuery` 等统一辅助函数）
3. 调用 Service
4. `response.Success` 或 `handleServiceError`
5. 不在 Handler 写复杂业务逻辑

### 9.3 Repository

- 列表分页方法内校正 `page`、`page_size` 范围。
- 所有业务查询带租户条件。
- 新增表或字段须添加 `db/migrations/` 迁移文件，可回滚。

### 9.4 Service

- 跨 Repo、OA、AI、缓存的编排放在 Service。
- 批量操作遵守上限（如审核/归档批量 `10` 条，常量定义处修改需两端与文档一致）。
- 写操作考虑缓存失效（`invalidationManager`）与通知（`UserNotificationService`）。

---

## 10. 前端开发规范

### 10.1 技术栈约定

- Nuxt 3、Vue 3 `<script setup lang="ts">`、Ant Design Vue 4。
- 页面默认：`definePageMeta({ middleware: 'auth', layout: 'default' })`（公开页除外）。
- 路径别名：`~/` 指向 `frontend/` 根目录。

### 10.2 命名

| 类型 | 约定 |
|------|------|
| 页面文件 | kebab-case，`dashboard.vue` |
| 组件 | PascalCase，`AppSidebar.vue` |
| Composable | `useXxx.ts` |
| API 封装 | `useDomainApi.ts`，如 `useAuditApi`、`useArchiveApi` |

### 10.3 API 调用

- 登录后统一使用 `authFetch<T>(path)`，解包 `code === 0` 的 `data`。
- 每个领域一个 composable，顶部 JSDoc 列出路由表；类型放在 `types/`，字段 **snake_case** 与后端一致。
- 401 由 `useAuth` 处理 refresh，页面内避免重复实现。

### 10.4 UI 与样式

- 优先使用 Ant Design Vue 组件；分页容器使用 `.pagination-wrapper`。
- 颜色与间距使用 `assets/css/variables.css` 中的 CSS 变量，避免随意硬编码色值。
- 权限：依赖后端 `page_permissions`，`middleware/auth.ts` 与菜单配置保持一致。

### 10.5 状态与异步任务

审核工作台、归档复盘均包含：**列表轮询 / SSE / 取消任务 / 历史记录**。修改一端时对照另一端：

- 任务 ID、状态枚举、`audit_status` 与归档侧等价字段
- 取消接口：`POST /api/audit/cancel/:id` ↔ `POST /api/archive/cancel/:id`
- 流式：`/api/audit/stream/:id` ↔ `/api/archive/stream/:id`

---

## 11. 模块关联与对称开发

AuraOA 中多处功能成对出现，修 bug 或加特性时必须做 **关联影响分析**。

### 11.1 审核工作台 ↔ 归档复盘

| 维度 | 审核 | 归档 |
|------|------|------|
| 前端页面 | `pages/dashboard.vue` | `pages/archive.vue` |
| API 前缀 | `/api/audit` | `/api/archive` |
| 租户配置 | `/api/tenant/rules` | `/api/tenant/archive` |
| Service | `AuditExecuteService` | `ArchiveReviewService` |
| 数据表 | `audit_logs`、`audit_process_snapshots` | `archive_logs`、`archive_process_snapshots` |
| 异步 Worker | `audit_stream_worker` | `archive_stream_worker` |
| 个人设置 | `settings` 下审核流程配置 | `settings` 下归档流程配置 |

**变更时建议同步检查**：

- [ ] Handler / Service / Repo 三层是否只需改一侧，还是两侧都要改
- [ ] 前端 `useAuditApi` / `useArchiveApi` 与 `types/audit*.ts`、`types/archive*.ts`
- [ ] `docs/api/audit.md` 与 `docs/api/archive.md`
- [ ] 定时任务 `CronTaskService` 是否引用该能力
- [ ] 仪表盘 `DashboardOverviewService` 是否聚合该指标
- [ ] Excel 导出列与 i18n 映射
- [ ] 规则合并 `service/rule_merge.go`（`AuditRule` 与 `ArchiveRule` 共用）

### 11.2 租户配置 ↔ 运行时

```
process_audit_configs + audit_rules + user_personal_configs
    → MergeRules → OA 取数 → AI → audit_logs / snapshots

process_archive_configs + archive_rules + user_personal_configs
    → 同上链路 → archive_logs / snapshots
```

修改配置模型、合并逻辑、提示词键（`audit_*` / `archive_*`）时，须确认 **配置页**（`admin/tenant/rules` 等）与 **工作台** 读取路径一致。

### 11.3 用户个人配置

`UserPersonalConfigService` 同时服务审核与归档。改动字段结构或 API 时，检查：

- `docs/api/user-settings.md`
- `pages/settings.vue`、`pages/admin/tenant/user-configs.vue`

### 11.4 共享能力

以下模块被多业务复用，修改接口或行为时扩大回归范围：

- `AIModelCallerService`、`AttachmentRecognitionService`
- OA 适配器（`ecology9` 等）
- Redis 缓存与 `invalidationManager`
- `systemflags.Resolver`、租户 Token 配额

### 11.5 提示词与模板键

系统提示词按尺度与场景区分，`audit_` 与 `archive_` 前缀在库内已分离（参见 `GetByStrictnessAuditWorkbench` 等注释）。新增模板键时勿混用前缀。

---

## 12. 数据库与迁移

1. 结构变更只通过 `db/migrations/` 下的 SQL 迁移完成，禁止手工改生产库不留迁移。
2. 迁移文件命名遵循现有序号与时间戳规则。
3. 新增索引或外键时考虑多租户查询模式（常带 `tenant_id`）。
4. 回滚脚本须可执行，或在 PR 说明中注明不可逆变更。

---

## 13. 测试与质量

当前仓库未强制 ESLint/Prettier 配置，依赖 TypeScript 与 Code Review。

**最低要求**：

- 本地可编译：后端 `go build ./...`，前端 `npm run build`（或项目脚本）。
- 涉及权限、租户、分页、异步任务的改动，在 PR 描述中写明 **手动测试步骤**。
- 修复缺陷时，若属于已知问题，更新 `docs/known-issues/` 状态。

---

## 14. Git 与 PR 建议

1. 提交信息使用中文或英文均可，但须 **一句话说清「为什么」**。
2. 一个 PR 聚焦一个主题；跨审核/归档的大改动建议在描述中列出对称文件清单。
3. 不提交 `.env`、密钥、本地 IDE 配置。
4. 接口或行为变更附带 `docs/api` 与 `CHANGELOG.md`（用户可见时）。

---

## 15. 开发自检清单（合并前）

### 功能与关联

- [ ] 是否影响审核/归档对称模块？另一侧是否已处理？
- [ ] 是否影响 Cron、仪表盘、个人设置、租户配置页？
- [ ] 缓存与通知是否需要失效/触发？

### 国际化

- [ ] 无硬编码 UI 文案；`zh-CN` / `en-US` 键完整
- [ ] 导出/模板（如有）支持 locale

### 分页与接口

- [ ] 分页参数与 `PagedResult` 结构符合规范
- [ ] `docs/api` 已更新；错误码已登记

### 注释与可读性

- [ ] 非显而易见逻辑有中文注释
- [ ] composable / Handler 有路由说明

### 类型与命名

- [ ] 前后端 JSON 字段 snake_case 一致
- [ ] 新错误码、新权限点前后端对齐

### 日志

- [ ] 日志类型选型正确（Zap / 库表 / 中间件），无重复、无敏感信息
- [ ] 审核/归档对称模块日志与库表写入一致

---

## 16. 参考文档索引

| 文档 | 说明 |
|------|------|
| [API 总览](./api/README.md) | 认证、响应格式、路由索引 |
| [审核工作台接口](./api/audit.md) | `/api/audit` |
| [归档复盘接口](./api/archive.md) | `/api/archive` |
| [流程审核配置](./api/audit-config.md) | `/api/tenant/rules` |
| [用户设置](./api/user-settings.md) | 个人配置与仪表盘 |
| [OA 集成](./oa-integration.md) | OA 适配器与数据提取 |
| [AI 集成](./ai-integration.md) | 模型调用与两阶段审核 |
| [已知问题](./known-issues/README.md) | 缺陷跟踪 |

---

*文档版本随项目演进更新；若代码与本文冲突，以代码与 `docs/api` 为准，并应回头修订本文。*
