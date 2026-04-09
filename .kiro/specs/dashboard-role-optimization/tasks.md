# 实施计划：仪表盘角色差异化优化

## 概述

按照数据库 → 后端 → 前端的顺序，将仪表盘从日志表数据源迁移到快照表，重构三种角色（business / tenant_admin / system_admin）的差异化组件集合与数据结构，集成 ECharts 图表库，完善国际化支持。

## 任务

- [x] 1. 数据库迁移：tenant_llm_message_logs 新增 call_type 列
  - 创建 `db/migrations/000029_llm_call_type.up.sql`，为 `tenant_llm_message_logs` 表添加 `call_type VARCHAR(20) NOT NULL DEFAULT 'reasoning'` 列，添加列注释，创建 `idx_tllm_call_type` 复合索引 `(tenant_id, call_type)`
  - 创建 `db/migrations/000029_llm_call_type.down.sql`，回滚删除索引和列
  - 修改 `go-service/internal/model/tenant_llm_message_log.go`，在 `TenantLLMMessageLog` 结构体中新增 `CallType string` 字段，gorm tag 为 `size:20;not null;default:reasoning`
  - _需求: 12.3, 12.4_

- [x] 2. 后端 DTO 重构
  - [x] 2.1 重写 `go-service/internal/dto/dashboard_overview_dto.go` 中的 `DashboardOverviewResponse` 结构体
    - 移除旧的 `AuditSummary`、`PendingOACount`、`ArchiveRecent` 字段
    - 新增 `WeeklyOverview *WeeklyOverviewData`、`PendingTasks *PendingTasksData`、`WeeklyTrend []WeeklyTrendDayData`、`RecentActivity []ActivityItemEnriched`、`CronTasks []CronTaskPreview`
    - 保留 `DeptDistribution`（改为 `[]DeptDistributionData` 类型）、`UserActivity`
    - _需求: 18.1, 18.2, 18.3_

  - [x] 2.2 新增本周概览、待办任务、趋势、动态增强等 DTO 结构体
    - 新增 `WeeklyOverviewData`（total / audit_count / archive_count / cron_count）
    - 新增 `PendingTasksData`（audit_pending / archive_pending / total）
    - 新增 `WeeklyTrendDayData`（date / audit_count / cron_count / archive_count）
    - 新增 `ActivityItemEnriched`（含 recommendation / score / compliance / compliance_score / cron_status / task_label）
    - 新增 `CronTaskPreview`（id / task_label / task_type / description / cron_expression / is_active）
    - 新增 `DeptDistributionData`（department / audit_count / cron_count / archive_count / total）
    - _需求: 3.2, 4.1, 6.3, 6.4, 6.5, 7.1, 9.2_

  - [x] 2.3 重写 `PlatformDashboardOverviewResponse` 结构体
    - 移除旧的 `AuditSummary`、`WeeklyTrend`、`RecentActivity`、`ArchiveRecent`、`PendingOACount`、`TokenSummary` 字段
    - 新增 `TenantStats *PlatformTenantStatsData`、`AIPerformance *PlatformAIPerformanceData`、`TenantUsageList []TenantUsageRow`、`TenantRanking []PlatformTenantRankRowEnriched`
    - _需求: 18.4, 18.5, 18.6, 18.7_

  - [x] 2.4 新增系统管理员专用 DTO 结构体
    - 新增 `PlatformTenantStatsData`（tenant_total / tenant_active / active_criteria / tenants []TenantStatsRow）
    - 新增 `TenantStatsRow`（tenant_id / tenant_name / tenant_code / user_count / is_active）
    - 新增 `PlatformAIPerformanceData`（models []AIModelPerformanceRow）
    - 新增 `AIModelPerformanceRow`（model_config_id / model_name / display_name / provider / reasoning_stats / structured_stats / overall_success_rate / total_calls）
    - 新增 `AICallTypeStats`（calls / success_rate / avg_ms）
    - 新增 `TenantUsageRow`（tenant_id / tenant_name / tenant_code / token_used / token_quota）
    - 新增 `PlatformTenantRankRowEnriched`（含 audit_count / archive_count / cron_count / audit_failed / archive_failed）
    - _需求: 12.1, 12.2, 12.3, 13.1, 14.1, 14.2, 15.2, 15.3_

- [x] 3. 检查点 — 确认 DTO 编译通过
  - 确保所有 DTO 结构体编译通过，请用户确认是否有疑问。

- [x] 4. 后端 Repository 层新增仪表盘查询方法
  - [x] 4.1 `audit_process_snapshot_repo.go` 新增方法
    - 实现 `CountThisWeek(c, userID)` — 本周快照条数，通过 JOIN audit_logs 按 user_id 过滤
    - 实现 `WeeklyTrendByDay(c, userID)` — 本周每天快照条数，使用 generate_series 填充无数据日期
    - 实现 `RecentEnriched(c, limit, userID)` — 最近 N 条快照，返回 recommendation + score + 操作人信息
    - 实现 `CountByDepartment(c)` — 按部门统计快照数（tenant_admin 用）
    - 实现 `CountByUserRanking(c, limit)` — 按用户统计有效快照数排名
    - 实现 `CountByTenantGlobal()` — 全平台按租户统计快照数
    - _需求: 1.1, 3.1, 4.1, 6.3, 9.1, 10.2, 15.2_

  - [x] 4.2 `archive_process_snapshot_repo.go` 新增方法
    - 实现 `CountThisWeek(c, userID)` — 本周归档快照条数
    - 实现 `WeeklyTrendByDay(c, userID)` — 本周每天归档快照条数
    - 实现 `RecentEnriched(c, limit, userID)` — 最近 N 条归档快照，返回 compliance + compliance_score
    - 实现 `CountByDepartment(c)` — 按部门统计归档快照数
    - 实现 `CountByTenantGlobal()` — 全平台按租户统计归档快照数
    - 实现 `CountFailedByTenantGlobal()` — 全平台按租户统计归档失败数（从 archive_logs 查 status=failed）
    - _需求: 1.2, 3.1, 4.1, 6.4, 9.1, 15.2, 15.3_

  - [x] 4.3 `cron_log_repo.go` 新增方法
    - 实现 `CountThisWeek(c, userID)` — 本周定时任务执行次数，business 按 task_owner_user_id 过滤
    - 实现 `WeeklyTrendByDay(c, userID)` — 本周每天定时任务执行次数
    - 实现 `RecentEnriched(tenantID, limit, userID)` — 最近 N 条日志，返回 status + task_label
    - 实现 `CountByDepartment(c)` — 按部门统计定时任务执行数（通过 task_owner_user_id JOIN）
    - 实现 `CountByTenantGlobal()` — 全平台按租户统计定时任务执行数
    - _需求: 1.3, 1.4, 3.1, 4.1, 6.5, 9.2, 15.2_

  - [x] 4.4 `llm_message_log_repo.go` 新增方法
    - 实现 `DashboardAIPerformanceByModel()` — 按 model_config_id + call_type 分组聚合，JOIN ai_model_configs 获取模型名称/provider，返回调用次数和平均耗时
    - _需求: 12.1, 12.2, 12.3, 12.4_

  - [x] 4.5 `tenant_repo.go` 新增方法
    - 实现 `DashboardTenantListWithUserCount()` — 所有租户列表，JOIN org_members COUNT 获取每个租户的注册人员数量
    - 实现 `DashboardActiveTenantIDs()` — 查询近 30 天内有审核或归档快照记录的租户 ID 集合（UNION audit_process_snapshots + archive_process_snapshots）
    - 实现 `DashboardTenantTokenList()` — 按租户分列返回 token_used / token_quota（替代原 DashboardPlatformTokenSum 的汇总方式）
    - _需求: 13.1, 14.1, 14.2, 14.3_

- [x] 5. 检查点 — 确认 Repository 编译通过
  - 确保所有新增 Repository 方法编译通过，请用户确认是否有疑问。

- [x] 6. 后端 Service 层重写
  - [x] 6.1 更新 `DashboardOverviewService` 结构体依赖注入
    - 新增 `auditSnapshotRepo *repository.AuditProcessSnapshotRepo` 和 `archiveSnapshotRepo *repository.ArchiveProcessSnapshotRepo` 字段
    - 移除 `auditExecuteSvc *AuditExecuteService` 依赖（不再需要 OA 待办）
    - 更新 `NewDashboardOverviewService` 构造函数签名和实现
    - 同步更新调用 `NewDashboardOverviewService` 的 DI 注册代码（如 wire 或手动注入处）
    - _需求: 1.1, 1.2_

  - [x] 6.2 重写 `BuildOverview` 方法
    - 根据 activeRole 确定 userScope（business 按 user_id 过滤，tenant_admin 为 nil）
    - 构建本周概览：调用三个 Repo 的 `CountThisWeek` 方法，组装 `WeeklyOverviewData`
    - 构建审核趋势：调用三个 Repo 的 `WeeklyTrendByDay` 方法，合并为 `[]WeeklyTrendDayData`
    - 构建最近动态：从三个数据源各取数据，合并排序取前 10 条，组装 `[]ActivityItemEnriched`
    - business 专属：构建待办任务（`PendingTasksData`）和定时任务预览（`[]CronTaskPreview`）
    - tenant_admin 专属：构建部门分布（`[]DeptDistributionData`）和用户活跃排名
    - 移除所有旧的 auditLogRepo / archiveLogRepo 统计调用
    - _需求: 2.1, 3.1, 3.2, 3.3, 4.1, 6.1, 6.2, 6.3, 6.4, 6.5, 7.1, 7.2, 7.3, 8.1, 8.3, 8.4, 9.1, 9.2, 10.1, 10.2, 10.3_

  - [x] 6.3 重写 `BuildPlatformOverview` 方法
    - 构建租户规模：调用 `TenantRepo.DashboardTenantListWithUserCount` + `DashboardActiveTenantIDs`，组装 `PlatformTenantStatsData`
    - 构建 AI 模型表现：调用 `LLMMessageLogRepo.DashboardAIPerformanceByModel`，计算成功率，组装 `PlatformAIPerformanceData`
    - 构建租户资源用量：调用 `TenantRepo.DashboardTenantTokenList`，组装 `[]TenantUsageRow`
    - 构建租户审核排名：联合查询快照表和日志表，组装 `[]PlatformTenantRankRowEnriched`
    - 移除所有旧的全平台统计调用（CountStatsGlobal、WeeklyCompletedTrendGlobal 等）
    - _需求: 11.2, 12.1, 12.2, 12.3, 13.1, 13.2, 14.1, 14.2, 14.3, 15.1, 15.2, 15.3, 15.4_

  - [x] 6.4 移除不再使用的辅助方法
    - 删除 `fillRecentActivity` 和 `fillRecentActivityPlatform` 方法
    - 删除 `activitySort` 结构体
    - 清理不再引用的 import（如 `model.AuditStatusFailed`）
    - _需求: 18.1, 18.4_

- [x] 7. 检查点 — 确认后端编译通过
  - 确保 Service 层重写后整个 go-service 编译通过，请用户确认是否有疑问。

- [x] 8. 前端类型定义更新
  - 重写 `frontend/types/dashboard-overview.ts`
  - 新增接口：`WeeklyOverviewData`、`PendingTasksData`、`WeeklyTrendDayData`、`ActivityItemEnriched`、`CronTaskPreview`、`DeptDistributionData`
  - 重写 `DashboardOverview` 接口，字段与后端 `DashboardOverviewResponse` 对齐
  - 新增系统管理员接口：`PlatformTenantStatsData`、`TenantStatsRow`、`PlatformAIPerformanceData`、`AIModelPerformanceRow`、`AICallTypeStats`、`TenantUsageRow`、`PlatformTenantRankRowEnriched`
  - 重写 `PlatformDashboardOverview` 接口，字段与后端 `PlatformDashboardOverviewResponse` 对齐
  - 移除不再使用的旧接口（`DashboardAuditSummary`、`DashboardArchiveRecentRow`、`PlatformTokenSummary`、旧 `PlatformTenantRankRow`）
  - _需求: 18.1, 18.2, 18.3, 18.4, 18.5, 18.6, 18.7_

- [x] 9. 前端组件注册表更新
  - 修改 `frontend/constants/overviewWidgets.ts`
  - 将 `audit_summary` 替换为 `weekly_overview`，移除 `archive_review`
  - 更新 `OverviewWidgetId` 类型联合
  - 按设计文档中的角色-组件映射表更新每个 widget 的 `requiredPermissions`
  - 新增 `WIDGET_PAGE_PERMISSION_MAP` 常量，定义 business 角色的组件与页面权限映射
  - _需求: 2.2, 2.3, 3.4, 8.2, 11.1_

- [x] 10. 前端 ECharts 集成
  - [x] 10.1 安装 ECharts 依赖
    - 在 `frontend/` 目录下安装 `echarts` 和 `vue-echarts` 包
    - _需求: 16.1_

  - [x] 10.2 创建 `frontend/components/charts/StackedBarChart.vue`
    - 按需引入 ECharts 模块（BarChart、GridComponent、TooltipComponent、LegendComponent、CanvasRenderer）
    - 接收 `categories`（X 轴标签）和 `series`（数据系列）props
    - 支持 `height` prop 和 autoresize
    - _需求: 4.2, 16.2, 16.3_

  - [x] 10.3 创建 `frontend/components/charts/DeptDistributionChart.vue`
    - 水平堆叠条形图，每个部门一行，区分审核/定时任务/归档三种颜色
    - _需求: 9.3, 16.2_

- [x] 11. 前端 overview.vue 重写
  - [x] 11.1 重写页面数据层逻辑
    - 更新 `EMPTY_OVERVIEW` 常量，匹配新的 `DashboardOverview` 接口
    - 更新 `mergePlatformToDash` 函数（或移除，改为直接使用 `PlatformDashboardOverview`）
    - 更新 `availableWidgets` computed，为 business 角色增加 `page_permissions` 过滤逻辑
    - 移除 `cronTasksList` 相关的独立 API 调用（定时任务数据改由后端 API 统一返回）
    - _需求: 2.2, 8.2_

  - [x] 11.2 实现本周概览组件（weekly_overview）
    - 替代原 `audit_summary` 组件模板
    - 展示总数 + 审核工作台/归档复盘/定时任务三个分项
    - _需求: 3.1, 3.2, 3.3, 3.4_

  - [x] 11.3 实现待办任务组件（pending_tasks）
    - 区分审核工作台待办和归档复盘待办两类
    - 展示各类待办数量和总数
    - _需求: 7.1_

  - [x] 11.4 实现审核趋势组件（weekly_trend）
    - 使用 `StackedBarChart` 组件渲染堆叠柱状图
    - 传入按日按功能分组的数据（审核/定时任务/归档）
    - 移除旧的纯 CSS 柱状图模板
    - _需求: 4.1, 4.2, 4.3_

  - [x] 11.5 实现定时任务组件（cron_tasks）优化
    - 在每个任务行中新增说明列
    - 无自定义说明时显示任务类型的 i18n 翻译
    - _需求: 5.1, 5.2_

  - [x] 11.6 实现最近动态组件（recent_activity）优化
    - 最多展示 10 条有效数据
    - 审核类型标注 recommendation + score
    - 归档类型标注 compliance + compliance_score
    - 定时任务类型标注执行状态 + 任务说明
    - _需求: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x] 11.7 实现部门分布组件（dept_distribution）优化
    - 使用 `DeptDistributionChart` 组件替代旧的纯 CSS 条形图
    - 区分审核/定时任务/归档三个功能的各自数量
    - _需求: 9.1, 9.2, 9.3_

  - [x] 11.8 实现用户活跃排名组件（user_activity）优化
    - 基于快照数据的排名展示
    - _需求: 10.2, 10.3_

  - [x] 11.9 实现系统管理员组件重写
    - 重写 `platform_tenant_stats`：展示每个租户的人员数量、活跃状态、活跃标准说明文案
    - 重写 `ai_performance`：按模型分组展示，区分推理/结构化调用的调用次数、成功率、平均响应时间
    - 重写 `tenant_usage`：按租户分列展示 Token 使用量和配额，移除全平台汇总
    - 重写 `platform_tenant_ranking`：展示每个租户的审核快照数、归档快照数、定时任务执行数、失败记录数
    - _需求: 11.1, 11.2, 12.1, 12.2, 12.3, 13.1, 13.2, 14.1, 14.2, 14.3, 15.1, 15.2, 15.3_

  - [x] 11.10 移除不再使用的组件模板和辅助函数
    - 删除旧的 `audit_summary` 模板
    - 删除旧的 `archive_review` 模板
    - 清理不再引用的 computed 属性（如 `trendMax`、`deptMax`、`aiBarMaxMs`、`storagePct`）和辅助函数
    - _需求: 3.4, 8.2, 11.1_

- [x] 12. 前端国际化更新
  - [x] 12.1 更新 `frontend/locales/zh-CN.ts`
    - 新增本周概览相关键（widgetTitle.weekly_overview、weeklyTotal、auditWorkbench、archiveReview、cronTasks）
    - 新增待办任务相关键（auditPending、archivePending、totalPending）
    - 新增最近动态标注键（recommendation.approve/return/review、compliance.*、cronStatus.*）
    - 新增系统管理员相关键（tenantUserCount、activeCriteria、modelPerformance、reasoningCalls、structuredCalls 等）
    - 移除不再使用的旧翻译键
    - _需求: 17.1, 17.2_

  - [x] 12.2 更新 `frontend/locales/en-US.ts`
    - 新增与 zh-CN 对应的所有英文翻译键
    - 移除不再使用的旧翻译键
    - _需求: 17.1, 17.2_

- [x] 13. 最终检查点 — 确保前后端对齐
  - 确保所有代码编译/类型检查通过，前后端 DTO 字段完全对齐，请用户确认是否有疑问。

## 备注

- 所有任务按数据库 → 后端 → 前端的依赖顺序排列，确保增量可编译
- 每个检查点用于验证阶段性成果，避免错误累积
- 定时任务数据继续从 cron_logs 表查询（无快照表），符合需求 1.4
- 现有记录的 call_type 默认为 `reasoning`，Python AI 服务需同步传入该字段（不在本次任务范围内）
