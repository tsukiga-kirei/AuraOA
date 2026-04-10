# 需求文档：代码规范化重构

## 简介

本项目是一个 OA 智能审核系统（Go 后端 + Nuxt 前端），包含两个核心业务模块：**审核工作台（audit）** 和 **归档复盘（archive）**。两个模块功能流程高度相似（配置管理、规则管理、AI 执行、流式任务、结果解析），但在命名、代码位置、模块边界等方面存在不一致，导致代码难以对比阅读和维护。本次重构旨在统一命名规范、修正代码位置、消除跨模块耦合、抽取公共功能。

## 术语表

- **Audit_Module**：审核工作台模块，负责对待审批 OA 流程进行 AI 辅助审核
- **Archive_Module**：归档复盘模块，负责对已归档 OA 流程进行 AI 合规复盘
- **Service_Layer**：后端 `go-service/internal/service/` 目录下的业务逻辑层
- **Model_Layer**：后端 `go-service/internal/model/` 目录下的数据模型层
- **DTO_Layer**：后端 `go-service/internal/dto/` 目录下的数据传输对象层
- **Handler_Layer**：后端 `go-service/internal/handler/` 目录下的 HTTP 处理层
- **Repository_Layer**：后端 `go-service/internal/repository/` 目录下的数据访问层
- **Composable_Layer**：前端 `frontend/composables/` 目录下的 API 调用封装层
- **Type_Layer**：前端 `frontend/types/` 目录下的 TypeScript 类型定义层
- **Stream_Worker**：基于 Redis Stream 的异步任务消费者
- **Prompt_Builder**：AI 提示词组装模块
- **Result_Parser**：AI 返回结果解析模块
- **Status_Constants**：任务异步状态常量（pending/assembling/reasoning/extracting/completed/failed）
- **Rule_Merge**：规则合并逻辑，将租户规则与用户个性化配置合并为最终生效规则列表

## 需求

### 需求 1：消除跨模块状态常量依赖

**用户故事：** 作为开发者，我希望 archive 模块和 audit 模块各自拥有独立的状态常量，或共享一组模块无关的公共常量，以便消除 archive 对 audit 内部实现的隐式依赖。

#### 验收标准

1. WHEN Archive_Module 的 Service_Layer 或 Repository_Layer 需要引用任务状态值时，THE Model_Layer SHALL 提供一组模块无关的公共状态常量（如 `JobStatusPending`、`JobStatusCompleted`），而非使用以 `Audit` 为前缀的常量
2. WHEN 公共状态常量被定义后，THE Audit_Module SHALL 使用该公共常量替代原有的 `AuditStatusPending` 等常量
3. WHEN 公共状态常量被定义后，THE Archive_Module SHALL 使用该公共常量替代对 `model.AuditStatusPending` 等的直接引用
4. THE Model_Layer SHALL 将公共状态常量定义在一个独立的文件中（如 `job_status.go`），而非嵌入在 `audit_log.go` 中

### 需求 2：统一后端 Service 层命名对称性

**用户故事：** 作为开发者，我希望 audit 和 archive 模块在 Service_Layer 的文件命名和结构上保持对称，以便快速对比阅读两个模块的实现。

#### 验收标准

1. THE Service_Layer SHALL 将 `audit_execute_service.go` 重命名为 `audit_review_service.go`，使其与 `archive_review_service.go` 形成对称命名
2. THE Service_Layer SHALL 确保 audit 模块拥有独立的 `audit_prompt_builder.go` 文件，与 `archive_prompt_builder.go` 形成对称命名，而非使用通用名称 `prompt_builder.go`
3. THE Service_Layer SHALL 确保 audit 模块拥有独立的 `audit_result_parser.go` 文件，与 `archive_result_parser.go` 形成对称命名，将 `ParseAuditResult` 函数从 `prompt_builder.go` 中迁移出来
4. WHEN 重命名完成后，THE Service_Layer 中 audit 和 archive 的文件命名 SHALL 遵循相同的 `{module}_{职责}.go` 模式

### 需求 3：修正 archive 模块 API 路由命名

**用户故事：** 作为开发者，我希望 archive 模块的 API 路由使用正确的模块前缀，以避免与 audit 模块的路由产生语义混淆。

#### 验收标准

1. THE Router SHALL 将归档规则的 API 路径从 `/api/tenant/archive/audit-rules` 修改为 `/api/tenant/archive/rules`
2. WHEN 路由路径变更后，THE Composable_Layer 中的 `useArchiveApi.ts` SHALL 同步更新对应的请求路径
3. IF 存在其他前端代码引用了旧的 `/api/tenant/archive/audit-rules` 路径，THEN THE Composable_Layer SHALL 同步更新所有引用

### 需求 4：拆分 archive_config_service.go 中的混合职责

**用户故事：** 作为开发者，我希望每个 service 文件只包含单一职责的服务，以便代码组织清晰且与 audit 模块保持一致。

#### 验收标准

1. THE Service_Layer SHALL 将 `ArchiveRuleService` 从 `archive_config_service.go` 中拆分到独立的 `archive_rule_service.go` 文件中
2. WHEN 拆分完成后，`archive_config_service.go` SHALL 仅包含 `ProcessArchiveConfigService` 的逻辑
3. THE Service_Layer 中 `archive_rule_service.go` 的结构 SHALL 与 `audit_rule_service.go` 保持对称

### 需求 5：抽取公共工具函数

**用户故事：** 作为开发者，我希望 audit 和 archive 模块共用的工具函数被抽取到公共位置，以消除隐式的跨文件依赖并减少重复代码。

#### 验收标准

1. THE Service_Layer SHALL 将以下在 `prompt_builder.go` 中定义但被 `archive_result_parser.go` 和 `archive_prompt_builder.go` 共同使用的函数抽取到一个公共文件（如 `ai_utils.go`）中：`cleanJSONResponse`、`truncate`、`coalesceStringSlice`、`pickOverallScoreInt`、`clampPercentInt`、`filterFields`、`filterRowFields`、`formatMainData`、`formatGroupedDetailData`、`extractJSONFromMarkdownFence`、`stripLeadingEllipsisPrefix`、`firstNonEmpty`
2. THE Service_Layer SHALL 将 `normalizeAuditRecommendation` 和 `mapComplianceAliasToRecommendation` 函数抽取到公共文件中，因为 `archive_result_parser.go` 直接调用了 `normalizeAuditRecommendation`
3. WHEN 公共函数被抽取后，THE `prompt_builder.go`（重命名后为 `audit_prompt_builder.go`）和 `archive_prompt_builder.go` SHALL 仅包含各自模块特有的提示词组装逻辑
4. WHEN 公共函数被抽取后，THE `archive_result_parser.go` 和新的 `audit_result_parser.go` SHALL 仅包含各自模块特有的结果解析逻辑

### 需求 6：整理 DTO 层文件组织

**用户故事：** 作为开发者，我希望 DTO_Layer 中的文件按模块职责清晰划分，以便与审核模块的 DTO 组织保持一致。

#### 验收标准

1. THE DTO_Layer SHALL 将 `cron_archive_dto.go` 中混合的三类 DTO（Cron 任务类型配置、归档复盘配置、归档规则）拆分到各自独立的文件中
2. THE DTO_Layer SHALL 将归档复盘配置 DTO（`CreateProcessArchiveConfigRequest`、`UpdateProcessArchiveConfigRequest`）迁移到 `archive_config_dto.go` 中
3. THE DTO_Layer SHALL 将归档规则 DTO（`CreateArchiveRuleRequest`、`UpdateArchiveRuleRequest`、`ListArchiveRulesQuery`）迁移到 `archive_rule_dto.go` 中
4. WHEN 拆分完成后，THE DTO_Layer 中 audit 和 archive 的 DTO 文件 SHALL 遵循对称的命名模式：`audit_list_dto.go` / `archive_review_dto.go`（执行相关）、`rules_dto.go` / `archive_config_dto.go`（配置相关）

### 需求 7：统一前端 Composable 命名模式

**用户故事：** 作为开发者，我希望前端 composable 的命名在 audit 和 archive 模块之间保持一致的模式，以便快速理解每个 composable 的职责。

#### 验收标准

1. THE Composable_Layer SHALL 确保 audit 模块的配置管理 API 封装（当前在 `useRulesApi.ts` 中）命名为 `useAuditConfigApi.ts`，与 `useArchiveApi.ts`（归档配置 API）形成对称
2. WHEN `useRulesApi.ts` 被重命名为 `useAuditConfigApi.ts` 后，THE Composable_Layer SHALL 更新所有引用该 composable 的前端文件
3. THE Composable_Layer SHALL 确保 audit 和 archive 的 composable 遵循相同的命名模式：`use{Module}ConfigApi.ts`（配置管理）和 `use{Module}ReviewApi.ts`（执行/复盘）

### 需求 8：统一前端类型定义文件命名

**用户故事：** 作为开发者，我希望前端类型定义文件的命名在 audit 和 archive 模块之间保持一致，以便快速定位类型定义。

#### 验收标准

1. THE Type_Layer SHALL 确保 audit 模块的类型文件命名与 archive 模块对称：`audit.ts` 对应 `archive-review.ts` 的命名不对称，应统一为 `audit-review.ts` 和 `archive-review.ts`，或 `audit.ts` 和 `archive.ts`
2. THE Type_Layer SHALL 将 `rules.ts` 中混合的审核配置类型（`ProcessAuditConfig`、`AuditRule`）和归档配置类型（`ProcessArchiveConfig`、`ArchiveRule`）拆分到各自模块的类型文件中
3. WHEN 类型文件拆分完成后，THE Type_Layer 中 `rules.ts` SHALL 仅保留真正公共的类型定义（如 `ProcessField`、`DetailTableDef`、`SystemPromptTemplate`、`ProcessInfo`、`ProcessFields`）

### 需求 9：统一 rule_merge 模块的适用范围

**用户故事：** 作为开发者，我希望规则合并逻辑能同时服务于 audit 和 archive 模块，或者各模块拥有独立的合并实现，以消除当前仅绑定 `model.AuditRule` 的局限性。

#### 验收标准

1. WHEN Archive_Module 需要规则合并功能时，THE Service_Layer SHALL 提供一个通用的规则合并接口或函数，同时支持 `AuditRule` 和 `ArchiveRule` 类型
2. IF 两个模块的规则合并逻辑存在差异，THEN THE Service_Layer SHALL 为每个模块提供独立的合并实现，并在文件命名上保持对称（如 `audit_rule_merge.go` 和 `archive_rule_merge.go`）
3. THE `rule_merge.go` 中的 `MergeRules` 函数 SHALL 不再硬编码依赖 `model.AuditRule` 类型，而是使用接口或泛型参数

### 需求 10：统一后端 Handler 层文件组织和命名

**用户故事：** 作为开发者，我希望 Handler_Layer 中 audit 和 archive 的 handler 文件命名和职责划分保持对称，以便快速定位对应的 HTTP 处理逻辑。

#### 验收标准

1. THE Handler_Layer SHALL 将 `audit_handler.go` 重命名为 `audit_review_handler.go`，使其与 `archive_review_handler.go` 形成对称命名
2. THE Handler_Layer SHALL 将 `archive_config_handler.go` 中混合的 `ArchiveRuleHandler` 拆分到独立的 `archive_rule_handler.go` 文件中，使其与 `audit_rule_handler.go` 形成对称
3. WHEN 拆分完成后，`archive_config_handler.go` SHALL 仅包含 `ArchiveConfigHandler` 的逻辑
4. THE Handler_Layer 中 audit 和 archive 的 handler 文件 SHALL 遵循相同的命名模式：`{module}_review_handler.go`（执行/复盘）、`{module}_config_handler.go`（配置管理）、`{module}_rule_handler.go`（规则管理）
