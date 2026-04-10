# 实施计划：代码规范化重构

## 概述

按依赖关系排序实施：先完成基础层变更（公共常量、公共工具函数、泛型化接口），再进行依赖这些变更的文件重命名、拆分和前端同步更新。后端变更优先于前端变更，确保 API 契约先稳定。

## 任务

- [x] 1. 创建公共状态常量并替换所有引用
  - [x] 1.1 新建 `go-service/internal/model/job_status.go`，定义 `JobStatusPending`、`JobStatusAssembling`、`JobStatusReasoning`、`JobStatusExtracting`、`JobStatusCompleted`、`JobStatusFailed` 常量
    - 常量值必须与原 `AuditStatus*` 常量完全一致
    - _需求: 1.1, 1.4_
  - [x] 1.2 将 `go-service/internal/model/audit_log.go` 中的 `AuditStatus*` 常量块删除，替换为对 `JobStatus*` 的引用或直接删除（因为新常量在同一个 package 中）
    - _需求: 1.2_
  - [x] 1.3 全局替换 `go-service/internal/repository/audit_log_repo.go` 中所有 `model.AuditStatusXxx` 引用为 `model.JobStatusXxx`
    - _需求: 1.2_
  - [x] 1.4 全局替换 `go-service/internal/repository/archive_log_repo.go` 中所有 `model.AuditStatusXxx` 引用为 `model.JobStatusXxx`
    - _需求: 1.3_
  - [x] 1.5 全局搜索并替换项目中其余所有 `AuditStatusPending`、`AuditStatusAssembling`、`AuditStatusReasoning`、`AuditStatusExtracting`、`AuditStatusCompleted`、`AuditStatusFailed` 引用为对应的 `JobStatus*` 常量
    - 包括 service 层、handler 层等所有引用处
    - _需求: 1.2, 1.3_
  - [ ]* 1.6 编写单元测试验证常量值等价性
    - 在 `go-service/internal/model/` 下新建 `job_status_test.go`，断言每个 `JobStatus*` 常量的字符串值正确
    - _需求: 1.1_

- [x] 2. 抽取公共工具函数到 `ai_utils.go`
  - [x] 2.1 新建 `go-service/internal/service/ai_utils.go`，从 `prompt_builder.go` 中迁入以下函数和类型：`cleanJSONResponse`、`extractJSONFromMarkdownFence`、`stripLeadingEllipsisPrefix`、`pickOverallScoreInt`、`clampPercentInt`、`coalesceStringSlice`、`firstNonEmpty`、`filterFields`、`filterRowFields`、`formatMainData`、`formatGroupedDetailData`、`truncate`、`SelectedFieldSet` 类型定义
    - 从 `prompt_builder.go` 中删除已迁移的函数
    - _需求: 5.1_
  - [x] 2.2 将 `normalizeAuditRecommendation` 和 `mapComplianceAliasToRecommendation` 函数迁入 `ai_utils.go`
    - 从 `prompt_builder.go` 中删除已迁移的函数
    - _需求: 5.2_
  - [x] 2.3 验证 `archive_result_parser.go` 和 `archive_prompt_builder.go` 对迁移后函数的调用无需修改（同一 package 内）
    - 确保 `go build ./...` 编译通过
    - _需求: 5.3, 5.4_
  - [ ]* 2.4 编写属性测试：AI 工具函数契约不变量
    - **Property 1: AI 工具函数契约不变量**
    - 测试 `cleanJSONResponse` 幂等性、`clampPercentInt` 范围约束 [0,100]、`truncate` 长度约束
    - 在 `go-service/internal/service/` 下新建 `ai_utils_test.go`
    - **验证: 需求 5.1**
  - [ ]* 2.5 编写属性测试：推荐结论归一化输出域约束
    - **Property 2: 推荐结论归一化输出域约束**
    - 测试 `normalizeAuditRecommendation` 输出属于 `{"", "approve", "return", "review"}` 或 `strings.ToLower(strings.TrimSpace(s))`
    - 测试 `mapComplianceAliasToRecommendation` 输出属于 `{"", "approve", "return", "review"}`
    - **验证: 需求 5.2**

- [x] 3. Service 层文件重命名与拆分
  - [x] 3.1 将 `audit_execute_service.go` 重命名为 `audit_review_service.go`
    - _需求: 2.1_
  - [x] 3.2 将 `prompt_builder.go` 重命名为 `audit_prompt_builder.go`，确保仅保留 audit 模块特有的 prompt 构建逻辑（`BuildReasoningPrompt`、`BuildExtractionPrompt` 等）
    - _需求: 2.2, 5.3_
  - [x] 3.3 新建 `audit_result_parser.go`，从 `audit_prompt_builder.go`（原 `prompt_builder.go`）中迁出 `ParseAuditResult` 及相关辅助函数（`extractionPayload`、`coalesceRuleResults`）
    - _需求: 2.3_
  - [x] 3.4 将 `archive_config_service.go` 中的 `ArchiveRuleService`（`NewArchiveRuleService` 及 `Create`、`Update`、`Delete`、`ListByConfigIDFilter` 方法）拆分到新文件 `archive_rule_service.go`
    - 拆分后 `archive_config_service.go` 仅保留 `ProcessArchiveConfigService`
    - _需求: 4.1, 4.2, 4.3_
  - [x] 3.5 迁移 `prompt_builder_test.go` 为 `audit_prompt_builder_test.go`（或根据测试内容拆分到对应的 `ai_utils_test.go` 和 `audit_result_parser_test.go`）
    - 确保所有现有测试通过
    - _需求: 2.4_

- [x] 4. 检查点 - 后端 Service 层编译验证
  - 运行 `go build ./...` 和 `go vet ./...` 确保编译通过且无静态分析警告
  - 运行现有测试套件确保无回归
  - 如有问题请询问用户

- [x] 5. 泛型化 rule_merge 模块
  - [x] 5.1 在 `go-service/internal/service/rule_merge.go` 中定义 `MergeableRule` 接口（`GetID()`、`GetRuleContent()`、`GetRuleScope()`、`IsEnabled()` 方法）和 `UserRuleOverride` 结构体
    - _需求: 9.3_
  - [x] 5.2 修改 `MergeRules` 函数签名，将 `[]model.AuditRule` 参数替换为 `[]MergeableRule`，内部字段访问替换为接口方法调用
    - _需求: 9.1, 9.3_
  - [x] 5.3 为 `model.AuditRule` 实现 `MergeableRule` 接口方法（在 `go-service/internal/model/audit_rule.go` 中添加）
    - _需求: 9.1_
  - [x] 5.4 为 `model.ArchiveRule` 实现 `MergeableRule` 接口方法（在 `go-service/internal/model/archive_rule.go` 中添加）
    - _需求: 9.1_
  - [x] 5.5 更新所有调用 `MergeRules` 的代码，适配新的接口参数
    - _需求: 9.1_
  - [ ]* 5.6 编写属性测试：规则合并排序与启用状态不变量
    - **Property 3: 规则合并排序与启用状态不变量**
    - 测试 mandatory 规则始终 Enabled=true、结果按优先级排序、toggle 覆盖正确性
    - **验证: 需求 9.1**

- [x] 6. Handler 层文件重命名与拆分
  - [x] 6.1 将 `go-service/internal/handler/audit_handler.go` 重命名为 `audit_review_handler.go`
    - _需求: 10.1_
  - [x] 6.2 将 `archive_config_handler.go` 中的 `ArchiveRuleHandler`（及其 `List`、`Create`、`Update`、`Delete` 方法）拆分到新文件 `archive_rule_handler.go`
    - 拆分后 `archive_config_handler.go` 仅保留 `ArchiveConfigHandler`
    - _需求: 10.2, 10.3_

- [x] 7. 修正 archive 模块 API 路由命名
  - [x] 7.1 修改 `go-service/internal/router/router.go`，将 archive 模块下 4 条 `audit-rules` 路由路径改为 `rules`
    - 即 `tenantArchive.GET("/audit-rules", ...)` → `tenantArchive.GET("/rules", ...)`，POST/PUT/DELETE 同理
    - _需求: 3.1_
  - [x] 7.2 更新 `archive_rule_handler.go`（拆分后的文件）中方法注释里的路由路径描述
    - _需求: 3.1_

- [x] 8. 整理 DTO 层文件组织
  - [x] 8.1 将 `go-service/internal/dto/cron_archive_dto.go` 中的归档复盘配置 DTO（`CreateProcessArchiveConfigRequest`、`UpdateProcessArchiveConfigRequest`）迁移到新文件 `archive_config_dto.go`
    - _需求: 6.2_
  - [x] 8.2 将 `cron_archive_dto.go` 中的归档规则 DTO（`CreateArchiveRuleRequest`、`UpdateArchiveRuleRequest`、`ListArchiveRulesQuery`）迁移到新文件 `archive_rule_dto.go`
    - _需求: 6.3_
  - [x] 8.3 将 `cron_archive_dto.go` 重命名为 `cron_dto.go`，仅保留 Cron 任务相关 DTO
    - _需求: 6.1, 6.4_

- [x] 9. 检查点 - 后端全量编译与测试验证
  - 运行 `go build ./...`、`go vet ./...` 确保编译通过
  - 运行 `go test ./...` 确保所有测试通过
  - 全局搜索 `AuditStatus` 确认无遗漏引用
  - 全局搜索 `audit-rules`（在 archive 上下文中）确认后端无遗漏路径引用
  - 如有问题请询问用户

- [x] 10. 统一前端 Composable 命名
  - [x] 10.1 将 `frontend/composables/useRulesApi.ts` 重命名为 `useAuditConfigApi.ts`，同步更新文件内的函数名和导出名
    - _需求: 7.1_
  - [x] 10.2 更新 `frontend/pages/admin/tenant/rules.vue` 及其他所有引用 `useRulesApi` 的文件，改为引用 `useAuditConfigApi`
    - _需求: 7.2_

- [x] 11. 修正前端 archive API 路径
  - [x] 11.1 更新 `frontend/composables/useArchiveApi.ts` 中 4 个函数的请求路径，将 `/api/tenant/archive/audit-rules` 改为 `/api/tenant/archive/rules`
    - _需求: 3.2_
  - [x] 11.2 全局搜索前端代码中是否有其他引用 `/api/tenant/archive/audit-rules` 的地方，如有则同步更新
    - _需求: 3.3_

- [x] 12. 统一前端类型定义文件命名
  - [x] 12.1 从 `frontend/types/rules.ts` 中将审核配置类型（`ProcessAuditConfig`、`AuditRule`）拆分到新文件 `audit-config.ts`
    - _需求: 8.2_
  - [x] 12.2 从 `frontend/types/rules.ts` 中将归档配置类型（`ProcessArchiveConfig`、`ArchiveRule`、`AccessControl`）拆分到新文件 `archive-config.ts`
    - _需求: 8.2_
  - [x] 12.3 从 `frontend/types/rules.ts` 中将 Cron 相关类型（`CronContentTemplate`、`CronTaskConfig`、`SaveCronTaskConfigRequest`）迁移到已有的 `cron.ts` 中
    - _需求: 8.2_
  - [x] 12.4 将 `frontend/types/rules.ts` 重命名为 `common.ts`，仅保留公共类型（`ProcessField`、`DetailTableDef`、`SystemPromptTemplate`、`ProcessInfo`、`FieldDef`、`ProcessFields`）
    - _需求: 8.3_
  - [x] 12.5 更新所有引用 `~/types/rules` 的前端文件，改为引用对应的新类型文件路径
    - _需求: 8.1, 8.2, 8.3_

- [x] 13. 最终检查点 - 全量编译与回归验证
  - 运行 `go build ./...` 确保后端编译通过
  - 运行 `go test ./...` 确保后端所有测试通过
  - 运行 `npx nuxi build` 确保前端编译通过
  - 全局搜索确认无遗漏的旧命名引用
  - 如有问题请询问用户

## 备注

- 标记 `*` 的子任务为可选任务，可跳过以加速 MVP 交付
- 每个任务引用了具体的需求编号以确保可追溯性
- 检查点任务确保增量验证，避免问题累积
- 属性测试验证设计文档中定义的正确性属性
- 单元测试验证具体示例和边界条件
