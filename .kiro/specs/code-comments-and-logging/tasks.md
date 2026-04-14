# 实现计划：代码注释规范化与日志系统完善

## 概述

本计划分两大方向：先完成日志系统基础设施搭建（新建 logger 包、扩展配置、数据库迁移、清理服务），再按目录分组对后端和前端代码进行注释规范化改造。日志基础设施任务优先执行，因为后续注释改造中的新建文件依赖它。

## 任务

- [x] 1. 搭建日志系统基础设施
  - [x] 1.1 新建 `go-service/internal/pkg/logger/config.go`
    - 定义 `LogConfig` 结构体，字段含 `Level`、`Dir`、`MaxSizeMB`、`MaxBackups`、`Compress`、`GlobalRetentionDays`，使用 `mapstructure` tag
    - 全部使用中文注释
    - _需求：3.6、3.7、5.5_

  - [x] 1.2 新建 `go-service/internal/pkg/logger/logger.go`
    - 实现 `Init(cfg LogConfig) error`：初始化全局 logger，使用 lumberjack 写入 `logs/app.log`，同时输出到 stdout
    - 实现 `Global() *zap.Logger`：返回全局 logger 实例
    - 实现 `GetTenantLogger(tenantCode string) *zap.Logger`：用 `sync.Map` 缓存租户 logger，写入 `logs/tenants/{tenantCode}/tenant.log` 并同时写全局文件
    - 实现 `Sync()`：刷新所有 logger 缓冲区
    - 全部使用中文注释
    - _需求：3.1、4.1、4.2、4.3、4.4、4.5_

  - [x] 1.3 新建 `go-service/internal/pkg/logger/cleanup.go`
    - 实现 `CleanupGlobalLogs(retentionDays int) (deletedCount int, freedBytes int64, err error)`
    - 实现 `CleanupTenantLogs(retentionMap map[string]int) (deletedCount int, freedBytes int64, err error)`
    - 仅清理带时间戳的轮转备份文件，不删除当前写入文件
    - 单个文件删除失败时记录 WARN 并继续，不中断整体任务
    - 全部使用中文注释
    - _需求：6.3、6.4、6.5_

  - [x] 1.4 扩展 `go-service/internal/config/config.go`
    - 在 `Config` 结构体中新增 `Log LogConfig` 字段
    - 为所有配置字段补充中文注释，删除英文注释
    - _需求：3.6、5.5、需求 1.5_

  - [x] 1.5 扩展 `go-service/config.yaml`
    - 新增 `log` 配置节，包含 `level`、`dir`、`max_size_mb`、`max_backups`、`compress`、`global_retention_days` 字段及中文注释
    - _需求：3.6、3.7、5.5_

  - [x] 1.6 新建数据库迁移文件
    - 新建 `db/migrations/000030_global_log_retention_config.up.sql`：向 `system_configs` 表插入 `system.global_log_retention_days` 配置项，默认值 `30`
    - 新建 `db/migrations/000030_global_log_retention_config.down.sql`：回滚删除该配置项
    - _需求：9.1、9.2_

  - [x] 1.7 新建 `go-service/internal/service/log_cleanup_service.go`
    - 实现 `LogCleanupService` 结构体，依赖 `SystemConfigRepo` 和 `TenantRepo`
    - 实现 `RunCleanup(ctx context.Context) error`：读取全局保留天数 → 清理全局日志 → 遍历租户读取各自保留天数 → 清理租户日志 → 写入汇总日志
    - `system.global_log_retention_days` 不存在时降级使用 `config.yaml` 兜底值
    - 全部使用中文注释
    - _需求：6.1、6.2、6.3、6.4、6.5、9.3、9.4_

  - [x] 1.8 改造 `go-service/cmd/server/main.go`
    - 在最早阶段调用 `logger.Init`，失败时 `log.Fatalf` 终止进程
    - 程序退出前调用 `logger.Sync()`
    - 将 `LogCleanupService` 注册到 `CronScheduler`（每日凌晨执行）
    - 删除英文注释，补充各初始化阶段中文说明
    - _需求：3.2、3.3、需求 1.5_

- [x] 2. 检查点 —— 日志基础设施验证
  - 确保所有测试通过，如有疑问请向用户确认。

- [x] 3. 后端注释改造：middleware / router / dbmigrate
  - [x] 3.1 改造 `internal/middleware/` 下全部 6 个文件
    - `auth.go`：补充 JWT 校验流程中文注释
    - `cors.go`：补充跨域配置中文注释
    - `logger.go`：删除翻译腔注释，改为中文函数说明，切换为使用 `logger.Global()`
    - `recovery.go`：删除翻译腔注释，改为中文 panic 恢复说明，切换为使用 `logger.Global()`
    - `role.go`：补充角色权限校验中文注释
    - `tenant.go`：补充租户上下文注入中文注释
    - _需求：1.1、1.2、1.3、1.4、1.5_

  - [x] 3.2 改造 `internal/router/router.go` 和 `internal/dbmigrate/dbmigrate.go`
    - 补充路由分组、中间件挂载中文注释
    - 补充迁移执行逻辑中文注释
    - _需求：1.1、1.3_

- [x] 4. 后端注释改造：model / dto
  - [x] 4.1 改造 `internal/model/` 下全部 29 个文件
    - 删除英文注释，为结构体及关键字段补充中文说明
    - 涵盖：`ai_deploy_type_option.go`、`ai_model_config.go`、`ai_provider_option.go`、`archive_process_snapshot.go`、`archive_rule.go`、`audit_log.go`、`audit_process_snapshot.go`、`audit_rule.go`、`cron_task_type_preset.go`、`cron_task.go`、`db_driver_option.go`、`department.go`、`job_status.go`、`login_history.go`、`oa_database_connection.go`、`oa_type_option.go`、`org_member.go`、`org_role.go`、`process_archive_config.go`、`process_audit_config.go`、`system_config.go`、`system_prompt_template.go`、`tenant_llm_message_log.go`、`tenant.go`、`user_dashboard_pref.go`、`user_notification.go`、`user_personal_config.go`、`user_role_assignment.go`、`user.go`
    - _需求：1.1、1.2、1.4_

  - [x] 4.2 改造 `internal/dto/` 下全部 14 个文件
    - 删除英文注释，为请求/响应结构体补充中文说明
    - 涵盖：`archive_config_dto.go`、`archive_review_dto.go`、`archive_rule_dto.go`、`audit_list_dto.go`、`auth_dto.go`、`cron_dto.go`、`dashboard_overview_dto.go`、`org_dto.go`、`rules_dto.go`、`settings_dto.go`、`system_dto.go`、`tenant_dto.go`、`user_config_admin_dto.go`、`user_notification_dto.go`
    - _需求：1.1、1.2、1.4_

- [x] 5. 后端注释改造：repository / handler
  - [x] 5.1 改造 `internal/repository/` 下全部 22 个文件
    - 删除英文注释，为每个公开方法补充中文说明（查询条件、返回值含义）
    - 涵盖：`ai_model_repo.go`、`archive_config_repo.go`、`archive_log_repo.go`、`archive_process_snapshot_repo.go`、`audit_log_repo.go`、`audit_process_snapshot_repo.go`、`audit_rule_repo.go`、`base_repo.go`、`cron_log_repo.go`、`cron_repo.go`、`llm_message_log_repo.go`、`oa_connection_repo.go`、`option_repo.go`、`org_repo.go`、`process_audit_config_repo.go`、`system_config_repo.go`、`system_prompt_template_repo.go`、`tenant_repo.go`、`user_dashboard_pref_repo.go`、`user_notification_repo.go`、`user_personal_config_repo.go`、`user_repo.go`
    - _需求：1.1、1.3_

  - [x] 5.2 改造 `internal/handler/` 下全部 18 个文件
    - 删除英文注释，为每个 HTTP 处理函数补充中文说明（接口用途、参数来源、返回格式）
    - 涵盖：`archive_config_handler.go`、`archive_review_handler.go`、`archive_rule_handler.go`、`audit_review_handler.go`、`audit_rule_handler.go`、`auth_handler.go`、`cron_config_handler.go`、`cron_task_handler.go`、`dashboard_overview_handler.go`、`health_handler.go`、`llm_message_log_handler.go`、`org_handler.go`、`process_audit_config_handler.go`、`system_handler.go`、`tenant_handler.go`、`user_config_management_handler.go`、`user_notification_handler.go`、`user_personal_config_handler.go`
    - _需求：1.1、1.3、7.4_

- [x] 6. 后端注释改造：service / pkg
  - [x] 6.1 改造 `internal/service/` 下全部 31 个文件（含新建的 `log_cleanup_service.go`）
    - 删除英文注释，为核心业务方法补充中文说明（业务意图、关键分支逻辑）
    - 涵盖：`ai_caller_service.go`、`ai_model_service.go`、`ai_utils.go`、`archive_config_service.go`、`archive_prompt_builder.go`、`archive_result_parser.go`、`archive_review_service.go`、`archive_rule_service.go`、`archive_stream_worker.go`、`audit_prompt_builder.go`、`audit_result_parser.go`、`audit_review_service.go`、`audit_rule_service.go`、`audit_stream_worker.go`、`auth_service.go`、`cron_config_service.go`、`cron_scheduler.go`、`cron_task_service.go`、`dashboard_overview_service.go`、`llm_message_log_service.go`、`mail_service.go`、`oa_connection_service.go`、`option_service.go`、`org_service.go`、`process_audit_config_service.go`、`report_calculator_service.go`、`rule_merge.go`、`system_config_service.go`、`tenant_service.go`、`user_notification_service.go`、`user_personal_config_service.go`
    - _需求：1.1、1.3、8.1、8.2、8.3、8.4_

  - [x] 6.2 改造 `internal/pkg/` 下全部 13 个文件
    - 删除英文注释，补充各子包中文注释
    - `pkg/ai/`（`caller.go`、`factory.go`、`openai_compat.go`）：补充 AI 调用封装中文注释
    - `pkg/crypto/aes.go`：补充 AES 加解密中文注释
    - `pkg/errcode/errcode.go`：补充错误码定义中文注释
    - `pkg/hash/bcrypt.go`：补充哈希工具中文注释
    - `pkg/jwt/jwt.go`：补充 JWT 签发/校验中文注释
    - `pkg/mail/mail.go`：补充邮件发送中文注释
    - `pkg/oa/`（`adapter.go`、`ecology9.go`、`factory.go`）：补充 OA 数据源连接中文注释
    - `pkg/response/response.go`：补充统一响应格式中文注释
    - `pkg/sanitize/sanitize.go`：补充输入清洗中文注释
    - _需求：1.1、1.3_

- [x] 7. 检查点 —— 后端注释全量验证
  - 确保所有测试通过，如有疑问请向用户确认。

- [x] 8. 前端注释改造
  - [x] 8.1 改造 `frontend/middleware/auth.ts` 和 `frontend/composables/` 下全部 18 个文件
    - 删除英文注释及翻译腔注释，为关键函数、响应式变量、API 调用补充中文说明
    - `auth.ts`：补充路由守卫、token 校验流程详细中文注释
    - `useAuth.ts`：补充鉴权逻辑详细中文注释
    - 其余 composable 文件：补充函数用途、参数含义、返回值说明
    - _需求：2.1、2.2、2.4、2.5_

  - [x] 8.2 改造 `frontend/components/` 下全部 9 个文件
    - 删除英文注释，为 props、emit、关键计算逻辑补充中文说明
    - 涵盖：`AppHeader.vue`、`AppSidebar.vue`、`AuditPanel.vue`、`CronHistory.vue`、`RuleEditor.vue`、`RuleList.vue`、`SnapshotDetail.vue`、`charts/DeptDistributionChart.vue`、`charts/StackedBarChart.vue`
    - _需求：2.1、2.2、2.3_

  - [x] 8.3 改造 `frontend/pages/` 下全部 14 个页面文件
    - 删除英文注释，为页面初始化逻辑、数据加载、表单提交补充中文说明
    - 涵盖：`index.vue`、`login.vue`、`dashboard.vue`、`overview.vue`、`settings.vue`、`setup.vue`、`archive.vue`、`cron.vue`、`admin/system/settings.vue`、`admin/system/tenants.vue`、`admin/tenant/data.vue`、`admin/tenant/org.vue`、`admin/tenant/rules.vue`、`admin/tenant/user-configs.vue`
    - _需求：2.1、2.2、2.3、7.1、7.3、7.5_

  - [x] 8.4 改造 `frontend/constants/overviewWidgets.ts`
    - 补充仪表盘组件配置中文注释
    - _需求：2.1_

- [ ] 9. 属性测试：日志系统核心属性
  - [ ]* 9.1 为属性 2（日志等级过滤正确性）编写属性测试
    - 使用 `pgregory.net/rapid` 库，生成随机日志等级配置和随机日志条目，验证过滤行为
    - 测试文件：`go-service/internal/pkg/logger/logger_property_test.go`
    - **属性 2：日志等级过滤正确性**
    - **验证：需求 8.5**

  - [ ]* 9.2 为属性 3（日志清理保留天数边界）编写属性测试
    - 使用 `pgregory.net/rapid` 库，生成随机文件集合（随机修改时间）和随机保留天数，验证清理结果
    - 测试文件：`go-service/internal/pkg/logger/cleanup_property_test.go`
    - **属性 3：日志清理保留天数边界**
    - **验证：需求 6.3、6.4**

  - [ ]* 9.3 为属性 4（配置值合法性校验）编写属性测试
    - 使用 `pgregory.net/rapid` 库，生成随机整数，验证 [1, 3650] 内通过、范围外拒绝
    - 测试文件：`go-service/internal/handler/system_handler_property_test.go`
    - **属性 4：配置值合法性校验**
    - **验证：需求 7.4**

  - [ ]* 9.4 为属性 5（租户 logger 缓存幂等性）编写属性测试
    - 使用 `pgregory.net/rapid` 库，生成随机 tenantCode 序列，验证相同 code 返回相同指针
    - 测试文件：`go-service/internal/pkg/logger/logger_property_test.go`（与 9.1 合并）
    - **属性 5：租户 logger 缓存幂等性**
    - **验证：需求 4.5**

- [x] 10. 最终检查点 —— 确保所有测试通过
  - 确保所有测试通过，如有疑问请向用户确认。

## 备注

- 标有 `*` 的子任务为可选项，可跳过以加快 MVP 进度
- 每个任务均引用具体需求条款，保证可追溯性
- 日志基础设施（任务 1）必须先于注释改造任务执行，因为 `log_cleanup_service.go` 等新建文件依赖 logger 包
- 属性测试使用 `pgregory.net/rapid` 库，每个属性最少运行 100 次
- 注释质量验收标准：`grep -r "// [A-Z]" internal/` 结果为空；无翻译腔特征词；每个公开函数/结构体均有中文注释
