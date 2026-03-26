# 归档复盘后端化实施方案

> 文档版本：v1.0 | 创建日期：2026-03-26
> 用于评审 `archive.vue` 从 Mock 页面升级为真实“归档复盘运行时”能力的设计方案。本文遵循“新增优先、必要复用”的原则，优先新增独立模块，只有在基础设施层面才复用现有审核工作台能力。

---

## 一、背景与目标

当前仓库中，归档复盘已经具备以下基础：

- 租户侧归档配置、归档规则、提示词模板已经落库并已对接前后端
- 用户侧“可访问归档配置列表”和“归档个人配置”已经实现
- 数据库中已有 `archive_logs` 表
- `archive.vue` 页面 UI 已完成，但运行数据仍完全来自 `useMockData.ts`

当前缺口也很明确：

- 后端没有“已归档流程列表”运行时接口
- 后端没有“归档复盘执行链路”
- `archive_logs` 尚未承载异步任务状态、推理过程、错误信息
- OA 适配器尚未提供“已归档流程列表 / 审批流快照”能力
- 前端没有独立的归档复盘 API/composable/types

本次方案的目标是：

1. 让有权限的业务用户可以看到自己可操作的已归档流程
2. 允许用户对已归档流程发起 AI 复盘，并可查看进度、结果、历史
3. 让归档复盘与审核工作台在运行时上解耦，避免后续互相牵连
4. 在接口、国际化、刷新机制、消息队列、数据库设计上保持统一规范

本阶段非目标：

- 不优先处理复杂导出模板能力，导出先以读取归档结果 JSON 为主
- 不在本阶段实现完整定时调度，只预留与 `archive_batch` 的接入点
- 不把归档复盘继续堆到现有 `audit` 模块中混合实现

---

## 二、设计原则

### 2.1 新增优先

归档复盘运行时建议新增独立模块，不与审核工作台共用 Handler / Service / DTO / Redis Stream。

建议新增：

- `ArchiveReviewHandler`
- `ArchiveReviewService`
- `ArchiveLogRepo`
- `archive_review_dto.go`
- `archive_review.ts`
- `useArchiveReviewApi.ts`
- `archive_prompt_builder.go`
- `archive_result_parser.go`
- `archive_queue.go`

### 2.2 必要复用

以下能力建议复用，而不是重复造轮子：

- JWT、租户上下文、中间件体系
- `AIModelCallerService`
- `useAuth().authFetch`
- 用户可访问归档配置的判定逻辑
- OA 适配器中的 `FetchProcessData`
- 通用响应格式、错误码、分页参数风格

### 2.3 模块隔离

归档复盘与审核工作台建议做到以下隔离：

- 独立路由前缀：`/api/archive/*`
- 独立 Redis Stream：`archive:jobs`
- 独立 SSE 频道：`archive:stream:<job_id>`
- 独立任务超时与 stale-reconcile
- 独立结果解析结构

这样后续即使归档复盘增加“审批流一致性校验”“归档批量任务”“归档报表”也不会污染 `audit` 语义。

---

## 三、建议的整体架构

```
archive.vue
   ↓
useArchiveReviewApi.ts
   ↓
/api/archive/*
   ↓
ArchiveReviewHandler
   ↓
ArchiveReviewService
   ├─ ArchiveConfig / ArchiveRule / UserPersonalConfig
   ├─ ArchiveLogRepo
   ├─ OAAdapter（已归档流程列表、审批流、流程数据）
   ├─ Redis Stream（archive:jobs）
   └─ AIModelCallerService
   ↓
archive_logs
```

建议把“归档配置管理”和“归档运行时”分开看待：

- 现有 `/api/tenant/archive/*` 继续只负责管理员配置
- 新增 `/api/archive/*` 只负责业务用户运行时能力

---

## 四、后端方案

### 4.1 新增路由

建议新增业务用户路由组：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/archive/processes` | 获取可复盘的已归档流程列表 |
| GET | `/api/archive/stats` | 获取归档复盘统计 |
| POST | `/api/archive/execute` | 发起单条归档复盘 |
| POST | `/api/archive/batch` | 发起批量归档复盘 |
| GET | `/api/archive/jobs/:id` | 查询任务状态 |
| GET | `/api/archive/stream/:id` | 获取推理流式输出 |
| POST | `/api/archive/cancel/:id` | 中止任务 |
| GET | `/api/archive/history/:processId` | 获取某流程复盘历史 |
| GET | `/api/archive/result/:id` | 获取单条复盘结果详情 |

路由权限建议与 `dashboard` 一致：

- `JWT + TenantContext`
- 不强制 `tenant_admin`
- 实际可见数据由“归档访问控制 + 用户归档配置”决定

### 4.2 新增 Service

建议新增 `go-service/internal/service/archive_review_service.go`，职责如下：

- 获取当前用户可访问的归档流程类型
- 调 OA 适配器拉取已归档流程列表
- 合并最新归档复盘结果
- 创建 pending 任务并写入 `archive_logs`
- 入 Redis Stream，异步执行 AI 复盘
- 轮询任务状态、SSE 输出推理文本
- 查询历史与统计

不建议把这些逻辑继续写进 `archive_config_service.go`，因为配置管理和运行时职责不同。

### 4.3 新增 Repository

建议新增 `go-service/internal/repository/archive_log_repo.go`：

- `Create`
- `GetByID`
- `UpdateFields`
- `GetLatestByProcessID`
- `GetLatestResultMap`
- `ListCompletedByProcessIDWithUser`
- `ListByTenantAndFilters`
- `CountByTenantAndFilters`

这样可以避免把 `archive_logs` 操作继续塞在 `model/audit_log.go` 所在语义里。

### 4.4 新增运行时 DTO

建议新增 `go-service/internal/dto/archive_review_dto.go`，定义：

- `ArchiveProcessListQuery`
- `ArchiveProcessListItem`
- `ArchiveReviewExecuteRequest`
- `ArchiveReviewSubmitResponse`
- `ArchiveReviewResult`
- `ArchiveReviewHistoryItem`
- `ArchiveReviewStats`

字段风格尽量与 `audit` 对齐，但不要强行复用 `recommendation` 语义。

---

## 五、OA 适配器扩展方案

### 5.1 需要新增的 OA 能力

当前 `OAAdapter` 只有：

- `FetchProcessData`
- `FetchTodoList`
- `IsProcessInTodo`

对于归档复盘还不够，建议新增：

```go
FetchArchivedList(ctx context.Context, username string, filter ArchivedListFilter) ([]ArchivedItem, error)
FetchProcessFlow(ctx context.Context, processID string) (*ProcessFlowSnapshot, error)
```

建议新增的数据结构：

- `ArchivedItem`
  - `process_id`
  - `title`
  - `applicant`
  - `department`
  - `process_type`
  - `process_type_label`
  - `submit_time`
  - `archive_time`
  - `main_table_name`
  - `current_node`
- `ProcessFlowSnapshot`
  - `is_complete`
  - `missing_nodes`
  - `nodes`
  - `history`

### 5.2 复用边界

以下能力直接复用现有接口：

- 表单主表/明细数据：`FetchProcessData`
- 流程基础配置校验：现有归档配置能力

### 5.3 Ecology9 实现建议

`Ecology9Adapter` 中新增实现即可，不需要为了归档复盘单独做第二个 OA 适配器。

但要注意：

- “已归档”筛选条件必须在适配器内部封装，不要暴露数据库细节到 Service 层
- `FetchArchivedList` 返回统一 DTO，Service 层不感知 OA 数据库表结构
- 若 E9 对“已归档”与“已办结”是两种状态，优先以“已归档”语义为准

---

## 六、归档复盘执行链路

### 6.1 单条复盘

执行流程建议如下：

1. 用户在 `archive.vue` 选择已归档流程
2. 前端调用 `POST /api/archive/execute`
3. 后端校验：
   - 当前用户是否有该流程类型的归档访问权限
   - 租户是否配置归档规则与 AI 模型
4. 写入 `archive_logs` pending 记录
5. 写入 Redis Stream `archive:jobs`
6. Worker 消费任务：
   - 拉取流程快照
   - 拉取流程审批流
   - 合并归档规则与用户自定义规则
   - 构建归档推理 Prompt
   - AI 推理
   - AI 结构化提取
   - 写回 `archive_logs`
7. 前端通过 `/jobs/:id` + `/stream/:id` 展示进度和推理过程

### 6.2 批量复盘

批量复盘建议和 `dashboard.vue` 保持一致体验，但实现仍建议走归档独立接口：

- 单次最多 10 条
- 每条记录独立建任务
- 前端展示总体进度
- 支持中止当前执行项

### 6.3 任务状态

建议归档复盘状态与审核工作台统一：

- `pending`
- `assembling`
- `reasoning`
- `extracting`
- `completed`
- `failed`

统一的好处：

- 前端进度组件可复用视觉模式
- 队列监控与超时处理可统一
- 后续统计维度一致

---

## 七、归档结果结构建议

当前 `archive.vue` 期待的结构比 `archive_logs.archive_result` 现状更丰富，建议显式定义如下：

```json
{
  "overall_compliance": "compliant | partially_compliant | non_compliant",
  "overall_score": 82,
  "confidence": 88,
  "flow_audit": {
    "is_complete": true,
    "missing_nodes": [],
    "node_results": [
      {
        "node_id": "N1",
        "node_name": "财务审批",
        "compliant": true,
        "reasoning": "节点完整"
      }
    ]
  },
  "field_audit": [
    {
      "field_key": "contract_no",
      "field_name": "合同编号",
      "passed": true,
      "reasoning": "编号一致"
    }
  ],
  "rule_audit": [
    {
      "rule_id": "xxx",
      "rule_name": "归档附件完整性",
      "passed": false,
      "reasoning": "缺少签章扫描件"
    }
  ],
  "risk_points": ["缺少签章附件"],
  "suggestions": ["补齐合同扫描件"],
  "ai_summary": "……"
}
```

建议新增独立解析器 `archive_result_parser.go`，不要继续复用 `ParseAuditResult()`。

原因：

- 归档复盘的结论字段是 `overall_compliance`，不是 `recommendation`
- 归档结果天然包含 `flow_audit` / `field_audit`
- 若强行复用审核解析器，后续只会让两套结果结构越来越难维护

---

## 八、消息队列与刷新机制

### 8.1 消息队列

建议新增独立队列：

- Stream：`archive:jobs`
- Consumer Group：`archive-review-workers`
- PubSub：`archive:stream:<job_id>`
- 历史推理缓存：`archive:reasoning:<job_id>`

这样可以和当前 `audit:jobs` 完全隔离，便于独立扩容和问题定位。

### 8.2 刷新机制

建议采用“三层刷新”：

1. 页面列表刷新
   - 首次进入加载
   - 用户手动点刷新按钮
   - 单条/批量任务结束后自动刷新
   - 页面重新可见时轻量刷新一次
2. 任务状态刷新
   - 对已发起任务的当前选中项，用 `/api/archive/jobs/:id` 轮询
3. 推理内容刷新
   - 用 SSE 获取增量文本

不建议做全页面高频定时轮询，原因是归档列表通常数据量更大。

### 8.3 超时与对账

建议新增归档版 stale-reconciler：

- 任务超过 30 分钟未完成则标记 `failed`
- Worker 单条执行超时建议 25 分钟
- 与审核工作台相同，但独立实现为 `StartArchiveStaleReconciler`

---

## 九、数据库与脚本变更建议

### 9.1 是否需要改库

需要。

当前 `archive_logs` 只有：

- 基础流程信息
- `compliance`
- `compliance_score`
- `archive_result`
- `created_at`

这不足以支撑：

- 异步任务状态
- 错误回溯
- 推理过程展示
- 任务耗时统计
- 任务更新时刻判断

### 9.2 建议新增字段

建议新增 migration，例如：

- `status VARCHAR(20) NOT NULL DEFAULT 'completed'`
- `duration_ms INT NOT NULL DEFAULT 0`
- `ai_reasoning TEXT NOT NULL DEFAULT ''`
- `confidence INT NOT NULL DEFAULT 0`
- `raw_content TEXT NOT NULL DEFAULT ''`
- `parse_error TEXT NOT NULL DEFAULT ''`
- `error_message TEXT NOT NULL DEFAULT ''`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `process_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb`

其中 `process_snapshot` 建议保存本次复盘所使用的关键元信息：

- applicant
- department
- submit_time
- archive_time
- process_type_label
- current_node
- flow snapshot

这样可以避免后续列表、历史、导出完全依赖 OA 实时数据。

### 9.3 索引建议

建议新增：

- `(tenant_id, process_id, created_at DESC)`
- `(tenant_id, status, updated_at DESC)`
- `(tenant_id, process_type, created_at DESC)`

### 9.4 脚本是否需要更新

需要更新：

- `db/migrations/`：新增 archive runtime migration
- `docs/database/database-schema.md`：补充新字段说明

可选更新：

- `db/seeds/`：若需要演示环境可补少量 `archive_logs` 样例数据

当前不强制要求更新：

- 现有归档配置 seed，可继续复用

---

## 十、AI 调用与日志规范

### 10.1 AI 调用

建议复用 `AIModelCallerService`，但在归档调用时显式标记 `request_type=archive`。

当前 `tenant_llm_message_logs` 模型已经支持 `request_type` 字段，但代码层尚未看到归档场景的落库标记，因此实现时需要补齐。

### 10.2 Prompt 构建

建议新增：

- `BuildArchiveReasoningPrompt`
- `BuildArchiveExtractionPrompt`

建议输入包括：

- 归档流程类型
- 主表数据
- 明细表数据
- 审批流快照
- 归档规则
- 当前归档节点

### 10.3 结果解析

建议新增：

- `ParseArchiveReviewResult`

解析结果需容错：

- `overall_compliance`
- `overall_score` / `score`
- `confidence`
- `flow_audit`
- `field_audit`
- `rule_audit`
- `risk_points`
- `suggestions`
- `ai_summary`

---

## 十一、前端改造方案

### 11.1 新增前端文件

建议新增：

- `frontend/composables/useArchiveReviewApi.ts`
- `frontend/types/archive-review.ts`

不建议把运行时接口继续混进 `useArchiveApi.ts`，因为后者是管理员配置 API。

### 11.2 archive.vue 重构方向

`archive.vue` 需要做的核心变化：

- 去掉 `useMockData()` 依赖
- 去掉页面内部随机生成审核结果逻辑
- 改成通过后端拉取列表、统计、详情、任务状态
- 保留当前页面视觉布局，减少不必要 UI 返工

建议页面状态结构对齐 `dashboard.vue`，但不直接照搬：

- 列表：后端分页、后端过滤
- 统计：后端返回
- 单条执行：SSE + Poll
- 批量执行：独立 batch 流程
- 历史结果：从 `archive_logs` 拉取

### 11.3 国际化

建议同步补全：

- `frontend/locales/zh-CN.ts`
- `frontend/locales/en-US.ts`

新增文案类型主要包括：

- 加载失败
- 刷新成功/失败
- 批量上限
- 任务取消
- 历史记录
- 空状态
- 异步阶段说明

### 11.4 刷新交互

建议在 `archive.vue` 页头新增显式刷新入口，避免用户误以为数据是静态的。

---

## 十二、接口规范建议

### 12.1 列表接口

建议从一开始就做服务端筛选和分页：

`GET /api/archive/processes`

查询参数建议：

- `keyword`
- `applicant`
- `process_type`
- `department`
- `compliance`
- `page`
- `page_size`

返回建议：

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "page_size": 10
}
```

### 12.2 统计接口

`GET /api/archive/stats`

返回建议：

- `total_count`
- `compliant_count`
- `partial_count`
- `non_compliant_count`
- `unaudited_count`
- `running_count`

### 12.3 执行接口

`POST /api/archive/execute`

请求体建议：

```json
{
  "process_id": "WF-2025-001",
  "process_type": "采购审批",
  "title": "采购申请",
  "archive_time": "2026-03-20 10:10:00"
}
```

立即返回：

```json
{
  "status": "pending",
  "id": "job-id",
  "trace_id": "AR-job",
  "process_id": "WF-2025-001",
  "created_at": "2026-03-26T10:00:00Z"
}
```

### 12.4 历史接口

`GET /api/archive/history/:processId`

用于右侧详情面板展示“最近复盘记录”或后续历史抽屉。

---

## 十三、建议的新增文件清单

### 后端

- `go-service/internal/handler/archive_review_handler.go`
- `go-service/internal/service/archive_review_service.go`
- `go-service/internal/service/archive_prompt_builder.go`
- `go-service/internal/service/archive_result_parser.go`
- `go-service/internal/service/archive_queue.go`
- `go-service/internal/repository/archive_log_repo.go`
- `go-service/internal/dto/archive_review_dto.go`

### 前端

- `frontend/composables/useArchiveReviewApi.ts`
- `frontend/types/archive-review.ts`

### 数据库

- `db/migrations/000015_archive_review_runtime.up.sql`
- `db/migrations/000015_archive_review_runtime.down.sql`

### 文档

- 本文档
- `docs/database/database-schema.md`
- `docs/architecture/technical-architecture.md`

---

## 十四、分阶段实施建议

### Phase 1：评审与定稿

- 明确接口与结果结构
- 明确 `archive_logs` 是否扩表
- 明确 OA 适配器需要新增的查询能力

### Phase 2：后端底座

- migration
- repo/service/handler/router
- Redis worker
- OA adapter 扩展

### Phase 3：前端接入

- 新 composable / types
- 重构 `archive.vue`
- 国际化补齐

### Phase 4：验证与文档补充

- 联调
- 数据回填/脚本确认
- README / 数据库文档 / 架构文档更新

---

## 十五、待确认事项

以下几点建议在你审核文档时一并确认：

1. 已归档流程列表的可见范围
   - 方案默认按“用户有权限的归档流程类型 + OA 中已归档实例”展示
   - 不额外限定“必须是当前用户自己经手过的流程”
2. `archive_logs` 是否接受扩表
   - 若不扩表，异步任务、推理展示、错误回溯都会明显受限
3. 导出能力是否纳入第一阶段
   - 方案建议先把“真实复盘能力”做实，再补导出模板

---

## 十六、结论

归档复盘后端化不建议继续在现有审核工作台模块里“顺手扩展”，而应当以新增独立运行时模块的方式实现：

- 配置侧继续沿用现有 `/api/tenant/archive/*`
- 运行时新增 `/api/archive/*`
- 队列、任务状态、结果结构、前端 composable 全部独立
- 只复用认证、AI 调用、用户权限、OA 流程数据等基础能力

这样既能满足 `archive.vue` 真正后端化，也能为后续归档批量任务、归档统计、归档报表留下清晰扩展位。
