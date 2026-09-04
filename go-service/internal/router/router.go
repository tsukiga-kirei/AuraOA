// Package router 负责注册所有 HTTP 路由及全局中间件。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"auraoa/go-service/internal/handler"
	"auraoa/go-service/internal/middleware"
	"auraoa/go-service/internal/pkg/systemflags"
	"auraoa/go-service/internal/repository"
)

// SetupRouter 在给定的 Gin 引擎上挂载全局中间件并注册所有路由分组。
func SetupRouter(
	r *gin.Engine,
	rdb *redis.Client,
	logger *zap.Logger,
	allowedOrigins []string,
	authHandler *handler.AuthHandler,
	basicSSOHandler *handler.BasicSSOHandler,
	orgHandler *handler.OrgHandler,
	tenantHandler *handler.TenantHandler,
	systemHandler *handler.SystemHandler,
	healthHandler *handler.HealthHandler,
	configHandler *handler.ProcessAuditConfigHandler,
	ruleHandler *handler.AuditRuleHandler,
	userConfigHandler *handler.UserPersonalConfigHandler,
	userConfigMgmtHandler *handler.UserConfigManagementHandler,
	llmLogHandler *handler.LLMMessageLogHandler,
	cronHandler *handler.CronConfigHandler,
	cronTaskHandler *handler.CronTaskHandler,
	archiveConfigHandler *handler.ArchiveConfigHandler,
	archiveRuleHandler *handler.ArchiveRuleHandler,
	summaryConfigHandler *handler.ProcessSummaryConfigHandler,
	executionConfigSourceHandler *handler.ExecutionConfigSourceHandler,
	externalContextHandler *handler.ExternalContextHandler,
	auditHandler *handler.AuditHandler,
	archiveReviewHandler *handler.ArchiveReviewHandler,
	summaryHandler *handler.ProcessSummaryHandler,
	embedEventHandler *handler.EmbedEventHandler,
	dashboardOverviewHandler *handler.DashboardOverviewHandler,
	userNotificationHandler *handler.UserNotificationHandler,
	cacheAdminHandler *handler.CacheAdminHandler,
	sysFlags *systemflags.Resolver,
	operationAuditRepo *repository.OperationAuditLogRepo,
	tenantRepo *repository.TenantRepo,
) {
	// 挂载全局中间件：结构化请求日志、panic 恢复、跨域（CORS）
	r.Use(middleware.Logger(logger))
	r.Use(middleware.AuditTrail(sysFlags, operationAuditRepo))
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.CORS(allowedOrigins))

	// 公开接口：无需登录即可访问，用于健康检查、初始化引导及登录流程
	r.GET("/api/health", healthHandler.Health)
	r.GET("/api/auth/bootstrap-status", authHandler.GetBootstrapStatus)
	r.POST("/api/auth/bootstrap", authHandler.BootstrapAdmin)
	r.POST("/api/auth/login", authHandler.Login)
	r.POST("/api/auth/refresh", authHandler.Refresh)
	r.GET("/api/auth/sso/basic-redirection", basicSSOHandler.Redirect)
	r.GET("/api/auth/sso/basic-consume", basicSSOHandler.Consume)
	r.GET("/api/tenants/list", tenantHandler.ListPublicTenants)

	// 认证相关接口（需要 JWT 验证）：登出、角色切换、菜单获取、密码修改、个人信息及站内通知
	auth := r.Group("/api/auth")
	auth.Use(middleware.JWT(rdb))
	{
		auth.POST("/logout", authHandler.Logout)
		auth.PUT("/switch-role", authHandler.SwitchRole)
		auth.GET("/menu", authHandler.GetMenu)
		auth.PUT("/change-password", authHandler.ChangePassword)
		auth.GET("/me", authHandler.GetMe)
		auth.PUT("/locale", authHandler.UpdateLocale)
		auth.PUT("/profile", authHandler.UpdateProfile)

		auth.GET("/notifications/unread-count", userNotificationHandler.UnreadCount)
		auth.GET("/notifications", userNotificationHandler.List)
		auth.PUT("/notifications/read-all", userNotificationHandler.MarkAllRead)
		auth.PUT("/notifications/:id/read", userNotificationHandler.MarkRead)
	}

	// 租户组织架构管理（需要 JWT + 租户上下文 + tenant_admin 角色）：部门、角色、成员的增删改查
	tenantOrg := r.Group("/api/tenant/org")
	tenantOrg.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		tenantOrg.GET("/departments", orgHandler.ListDepartments)
		tenantOrg.POST("/departments", orgHandler.CreateDepartment)
		tenantOrg.PUT("/departments/:id", orgHandler.UpdateDepartment)
		tenantOrg.DELETE("/departments/:id", orgHandler.DeleteDepartment)

		tenantOrg.GET("/roles", orgHandler.ListRoles)
		tenantOrg.POST("/roles", orgHandler.CreateRole)
		tenantOrg.PUT("/roles/:id", orgHandler.UpdateRole)
		tenantOrg.DELETE("/roles/:id", orgHandler.DeleteRole)

		tenantOrg.GET("/members", orgHandler.ListMembers)
		tenantOrg.POST("/members", orgHandler.CreateMember)
		tenantOrg.PUT("/members/:id", orgHandler.UpdateMember)
		tenantOrg.DELETE("/members/:id", orgHandler.DeleteMember)
		tenantOrg.POST("/members/import", orgHandler.ImportMembers)
		tenantOrg.GET("/members/import-template", orgHandler.DownloadImportTemplate)
	}

	// 系统管理员路由组（需要 JWT + 租户上下文 + system_admin 角色）
	admin := r.Group("/api/admin")
	admin.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("system_admin"))
	{
		// 租户管理
		admin.GET("/tenants", tenantHandler.ListTenants)
		admin.POST("/tenants", tenantHandler.CreateTenant)
		admin.PUT("/tenants/:id", tenantHandler.UpdateTenant)
		admin.DELETE("/tenants/:id", tenantHandler.DeleteTenant)
		admin.GET("/tenants/:id/stats", tenantHandler.GetTenantStats)
		admin.GET("/tenants/:id/members", tenantHandler.ListTenantMembers)
		admin.POST("/tenants/:id/embed-token", tenantHandler.RotateEmbedToken)

		// 系统设置：枚举选项、OA 数据库连接、AI 模型配置、系统 KV 配置
		system := admin.Group("/system")
		{
			system.GET("/options/oa-types", systemHandler.ListOATypes)
			system.GET("/options/db-drivers", systemHandler.ListDBDrivers)
			system.GET("/options/ai-deploy-types", systemHandler.ListAIDeployTypes)
			system.GET("/options/ai-providers", systemHandler.ListAIProviders)

			// OA 数据库连接
			system.GET("/oa-connections", systemHandler.ListOAConnections)
			system.POST("/oa-connections", systemHandler.CreateOAConnection)
			system.POST("/oa-connections/test", systemHandler.TestOAConnectionParams)
			system.PUT("/oa-connections/:id", systemHandler.UpdateOAConnection)
			system.DELETE("/oa-connections/:id", systemHandler.DeleteOAConnection)
			system.POST("/oa-connections/:id/test", systemHandler.TestOAConnection)

			// AI 模型配置
			system.GET("/ai-models", systemHandler.ListAIModels)
			system.POST("/ai-models", systemHandler.CreateAIModel)
			system.POST("/ai-models/test", systemHandler.TestAIModelConnection)
			system.PUT("/ai-models/:id", systemHandler.UpdateAIModel)
			system.DELETE("/ai-models/:id", systemHandler.DeleteAIModel)
			system.POST("/ai-models/:id/test", systemHandler.TestAIModelConnectionById)

			// 系统配置 (KV)
			system.GET("/configs", systemHandler.GetSystemConfigs)
			system.PUT("/configs", systemHandler.UpdateSystemConfigs)

			// 附件解析服务连接测试（不上传真实附件）
			system.POST("/attachment-recognition/test", systemHandler.TestAttachmentRecognition)
			system.POST("/attachment-recognition/test-compat", systemHandler.TestAttachmentCompatibility)
		}

		// 系统管理员 — Token 消耗统计
		admin.GET("/stats/token-usage", llmLogHandler.QueryAllTenantsTokenUsage)

		admin.GET("/dashboard-overview", dashboardOverviewHandler.GetPlatformOverview)
		admin.GET("/system-monitor", dashboardOverviewHandler.GetSystemMonitor)
	}

	// 缓存管理路由（仅超级管理员 system_admin）
	if cacheAdminHandler != nil {
		cacheAdmin := admin.Group("/cache")
		{
			cacheAdmin.GET("/stats", cacheAdminHandler.GetStats)
			cacheAdmin.DELETE("/tenant/:tenant_id", cacheAdminHandler.ClearTenantCache)
			cacheAdmin.DELETE("/module/:module", cacheAdminHandler.ClearModuleCache)
			cacheAdmin.POST("/toggle", cacheAdminHandler.ToggleCache)
		}
	}

	// 租户管理员 — 流程审核规则配置（需要 JWT + 租户上下文 + tenant_admin 角色）
	tenantRules := r.Group("/api/tenant/rules")
	tenantRules.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		// 流程审核配置
		tenantRules.GET("/configs", configHandler.List)
		tenantRules.POST("/configs", configHandler.Create)
		tenantRules.GET("/configs/:id", configHandler.GetByID)
		tenantRules.PUT("/configs/:id", configHandler.Update)
		tenantRules.DELETE("/configs/:id", configHandler.Delete)
		tenantRules.POST("/configs/test-connection", configHandler.TestConnection)
		tenantRules.POST("/configs/:id/fetch-fields", configHandler.FetchFields)

		// 审核规则
		tenantRules.GET("/audit-rules", ruleHandler.List)
		tenantRules.POST("/audit-rules", ruleHandler.Create)
		tenantRules.GET("/audit-rules/import-capability", ruleHandler.ImportCapability)
		tenantRules.POST("/audit-rules/import-preview", ruleHandler.PreviewImport)
		tenantRules.POST("/audit-rules/import-text-preview", ruleHandler.PreviewPastedImport)
		tenantRules.POST("/audit-rules/import-confirm", ruleHandler.ConfirmImport)
		tenantRules.POST("/audit-rules/batch-delete", ruleHandler.BatchDelete)
		tenantRules.PUT("/audit-rules/:id", ruleHandler.Update)
		tenantRules.DELETE("/audit-rules/:id", ruleHandler.Delete)
		tenantRules.POST("/context/test", externalContextHandler.Test)
		tenantRules.POST("/context/workflow-fields", externalContextHandler.FetchWorkflowFields)
		tenantRules.POST("/context/workflow-search", externalContextHandler.SearchWorkflows)

		// 系统提示词模板（只读）
		tenantRules.GET("/prompt-templates", configHandler.ListPromptTemplates)
	}

	// 租户管理员 — 当前配置与不可变执行版本的对应状态。
	executionConfigVersions := r.Group("/api/tenant/execution-config-versions")
	executionConfigVersions.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		executionConfigVersions.GET("/status", executionConfigSourceHandler.GetStatus)
		executionConfigVersions.POST("/publish", executionConfigSourceHandler.Publish)
		executionConfigVersions.GET("/history", executionConfigSourceHandler.ListHistory)
		executionConfigVersions.POST("/activate", executionConfigSourceHandler.Activate)
		executionConfigVersions.POST("/save-version", executionConfigSourceHandler.SaveVersion)
	}

	// 定时任务类型配置 — 只读（所有已登录租户用户均可访问，用于前端展示已启用的任务类型）
	tenantCronRO := r.Group("/api/tenant/cron")
	tenantCronRO.Use(middleware.JWT(rdb), middleware.TenantContext())
	{
		tenantCronRO.GET("/configs", cronHandler.ListConfigs)
	}

	// 定时任务类型配置 — 写操作（仅租户管理员可修改或重置任务类型配置）
	tenantCronAdmin := r.Group("/api/tenant/cron")
	tenantCronAdmin.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		tenantCronAdmin.PUT("/configs/:taskType", cronHandler.SaveConfig)
		tenantCronAdmin.DELETE("/configs/:taskType", cronHandler.ResetConfig)
	}

	// 定时任务实例管理（需要 JWT + 租户上下文，无角色限制）：任务的增删改查、手动触发及日志查看
	cronTasks := r.Group("/api/tenant/cron/tasks")
	cronTasks.Use(middleware.JWT(rdb), middleware.TenantContext())
	{
		cronTasks.GET("", cronTaskHandler.ListTasks)
		cronTasks.POST("", cronTaskHandler.CreateTask)
		cronTasks.PUT("/:id", cronTaskHandler.UpdateTask)
		cronTasks.DELETE("/:id", cronTaskHandler.DeleteTask)
		cronTasks.POST("/:id/toggle", cronTaskHandler.ToggleTask)
		cronTasks.POST("/:id/execute", cronTaskHandler.ExecuteNow)
		cronTasks.GET("/:id/logs", cronTaskHandler.ListLogs)
	}

	// 归档复盘配置管理（需要 JWT + 租户上下文 + tenant_admin 角色）：归档数据源配置及归档规则管理
	tenantArchive := r.Group("/api/tenant/archive")
	tenantArchive.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		tenantArchive.GET("/configs", archiveConfigHandler.List)
		tenantArchive.POST("/configs", archiveConfigHandler.Create)
		tenantArchive.POST("/configs/test-connection", archiveConfigHandler.TestConnection)
		tenantArchive.GET("/configs/:id", archiveConfigHandler.GetByID)
		tenantArchive.PUT("/configs/:id", archiveConfigHandler.Update)
		tenantArchive.DELETE("/configs/:id", archiveConfigHandler.Delete)
		tenantArchive.POST("/configs/:id/fetch-fields", archiveConfigHandler.FetchFields)
		tenantArchive.GET("/rules", archiveRuleHandler.List)
		tenantArchive.POST("/rules", archiveRuleHandler.Create)
		tenantArchive.GET("/rules/import-capability", archiveRuleHandler.ImportCapability)
		tenantArchive.POST("/rules/import-preview", archiveRuleHandler.PreviewImport)
		tenantArchive.POST("/rules/import-text-preview", archiveRuleHandler.PreviewPastedImport)
		tenantArchive.POST("/rules/import-confirm", archiveRuleHandler.ConfirmImport)
		tenantArchive.POST("/rules/batch-delete", archiveRuleHandler.BatchDelete)
		tenantArchive.PUT("/rules/:id", archiveRuleHandler.Update)
		tenantArchive.DELETE("/rules/:id", archiveRuleHandler.Delete)
		tenantArchive.POST("/context/test", externalContextHandler.Test)
		tenantArchive.POST("/context/workflow-fields", externalContextHandler.FetchWorkflowFields)
		tenantArchive.POST("/context/workflow-search", externalContextHandler.SearchWorkflows)
		tenantArchive.GET("/prompt-templates", archiveConfigHandler.ListPromptTemplates)
	}

	// 租户管理员 — 流程总结配置（需要 JWT + 租户上下文 + tenant_admin 角色）
	tenantSummary := r.Group("/api/tenant/summary")
	tenantSummary.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		tenantSummary.GET("/configs", summaryConfigHandler.List)
		tenantSummary.POST("/configs", summaryConfigHandler.Create)
		tenantSummary.POST("/configs/test-connection", summaryConfigHandler.TestConnection)
		tenantSummary.GET("/configs/:id", summaryConfigHandler.GetByID)
		tenantSummary.PUT("/configs/:id", summaryConfigHandler.Update)
		tenantSummary.DELETE("/configs/:id", summaryConfigHandler.Delete)
		tenantSummary.POST("/configs/:id/fetch-fields", summaryConfigHandler.FetchFields)
		tenantSummary.POST("/context/test", externalContextHandler.Test)
		tenantSummary.POST("/context/workflow-fields", externalContextHandler.FetchWorkflowFields)
		tenantSummary.POST("/context/workflow-search", externalContextHandler.SearchWorkflows)
	}

	// 租户管理员 — 用户个人配置管理（需要 JWT + 租户上下文 + tenant_admin 角色）
	tenantUserConfigs := r.Group("/api/tenant/user-configs")
	tenantUserConfigs.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		tenantUserConfigs.GET("", userConfigMgmtHandler.ListUserConfigs)
		tenantUserConfigs.GET("/export", userConfigMgmtHandler.ExportUserConfigs)
		tenantUserConfigs.GET("/:userId", userConfigMgmtHandler.GetUserConfig)
	}

	// 租户管理员 — Token 消耗统计
	tenantStats := r.Group("/api/tenant/stats")
	tenantStats.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		tenantStats.GET("/token-usage", llmLogHandler.QueryTokenUsage)
	}

	// 租户管理员 — AI 调用记录（数据管理页）
	tenantLLMLogs := r.Group("/api/tenant/llm-logs")
	tenantLLMLogs.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		tenantLLMLogs.GET("/processes", llmLogHandler.ListProcesses)
		tenantLLMLogs.GET("/stats", llmLogHandler.GetLogStats)
		tenantLLMLogs.GET("/:processId/chain", llmLogHandler.GetProcessChain)
		tenantLLMLogs.GET("/calls/:id", llmLogHandler.GetLogDetail)
	}

	// 业务用户个人设置（需要 JWT + 租户上下文，无角色限制）：流程配置、定时任务偏好、归档配置及仪表盘偏好
	tenantSettings := r.Group("/api/tenant/settings")
	tenantSettings.Use(middleware.JWT(rdb), middleware.TenantContext())
	{
		// 审核工作台个人配置
		tenantSettings.GET("/processes", userConfigHandler.GetProcessList)
		tenantSettings.GET("/processes/:processType", userConfigHandler.GetByProcessType)
		tenantSettings.PUT("/processes/:processType", userConfigHandler.UpdateByProcessType)
		tenantSettings.GET("/processes/:processType/full", userConfigHandler.GetFullProcessConfig)
		tenantSettings.GET("/processes/:processType/version-diff", userConfigHandler.GetAuditBaselineVersionDiff)

		// 定时任务个人偏好（默认邮箱等）
		tenantSettings.GET("/cron-prefs", userConfigHandler.GetCronPrefs)
		tenantSettings.PUT("/cron-prefs", userConfigHandler.UpdateCronPrefs)

		// 归档复盘个人配置
		tenantSettings.GET("/archive-configs", userConfigHandler.GetArchiveConfigList)
		tenantSettings.GET("/archive-configs/:processType/full", userConfigHandler.GetFullArchiveConfig)
		tenantSettings.PUT("/archive-configs/:processType", userConfigHandler.UpdateArchiveConfig)
		tenantSettings.GET("/archive-configs/:processType/version-diff", userConfigHandler.GetArchiveBaselineVersionDiff)

		// 流程总结个人展示偏好
		tenantSettings.GET("/summary-configs", userConfigHandler.GetSummaryConfigList)
		tenantSettings.GET("/summary-configs/:processType/full", userConfigHandler.GetFullSummaryPreference)
		tenantSettings.PUT("/summary-configs/:processType", userConfigHandler.UpdateSummaryPreference)

		// 仪表板偏好
		tenantSettings.GET("/dashboard-prefs", userConfigHandler.GetDashboardPrefs)
		tenantSettings.PUT("/dashboard-prefs", userConfigHandler.UpdateDashboardPrefs)

		// 仪表盘聚合数据
		tenantSettings.GET("/dashboard-overview", dashboardOverviewHandler.GetOverview)
	}

	// 流程总结工作台（需要 JWT + 租户上下文，无角色限制）。
	summary := r.Group("/api/summary")
	summary.Use(middleware.JWT(rdb), middleware.TenantContext())
	{
		summary.GET("/processes", summaryHandler.ListWorkbenchProcesses)
		summary.GET("/stats", summaryHandler.GetWorkbenchStats)
		summary.POST("/execute", summaryHandler.ExecuteWorkbench)
		summary.GET("/jobs/:id", summaryHandler.GetWorkbenchJobStatus)
		summary.GET("/stream/:id", summaryHandler.GetWorkbenchJobStream)
		summary.GET("/history/:processId", summaryHandler.GetWorkbenchHistory)
	}

	// 审核工作台（需要 JWT + 租户上下文，无角色限制）：发起审核、查询进度、流式输出、批量审核
	audit := r.Group("/api/audit")
	audit.Use(middleware.JWT(rdb), middleware.TenantContext())
	{
		audit.GET("/processes", auditHandler.ListProcesses)
		audit.GET("/processes/export", auditHandler.ExportProcesses)
		audit.GET("/stats", auditHandler.GetStats)
		audit.POST("/execute", auditHandler.Execute)
		audit.POST("/cancel/:id", auditHandler.CancelJob)
		audit.GET("/jobs/:id", auditHandler.GetJobStatus)
		audit.GET("/stream/:id", auditHandler.GetJobStream)
		audit.POST("/batch", auditHandler.BatchExecute)
		audit.GET("/chain/:processId", auditHandler.GetAuditChain)
	}

	// OA 嵌入审核（固定展示页，无需用户 JWT；由嵌入令牌 + 租户编码鉴权）
	embed := r.Group("/api/embed")
	embed.Use(middleware.EmbedAccess(tenantRepo))
	{
		embed.POST("/events", embedEventHandler.Schedule)
		embed.GET("/context", auditHandler.GetEmbedContext)
		embed.POST("/execute", auditHandler.ExecuteEmbed)
		embed.GET("/jobs/:id", auditHandler.GetJobStatus)
		embed.GET("/stream/:id", auditHandler.GetJobStream)

		embed.GET("/summary/context", summaryHandler.GetEmbedContext)
		embed.POST("/summary/execute", summaryHandler.ExecuteEmbed)
		embed.GET("/summary/jobs/:id", summaryHandler.GetJobStatus)
		embed.GET("/summary/stream/:id", summaryHandler.GetJobStream)
	}

	// 审核日志数据管理（仅 tenant_admin）：日志列表、统计及导出
	auditAdmin := r.Group("/api/audit/logs")
	auditAdmin.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		auditAdmin.GET("", auditHandler.ListLogs)
		auditAdmin.GET("/stats", auditHandler.GetLogStats)
		auditAdmin.GET("/export", auditHandler.ExportLogs)
	}

	// 审核快照数据管理（仅 tenant_admin）：快照列表、统计及审核链路查询
	auditSnapshotAdmin := r.Group("/api/audit/snapshots")
	auditSnapshotAdmin.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		auditSnapshotAdmin.GET("", auditHandler.ListSnapshots)
		auditSnapshotAdmin.GET("/stats", auditHandler.GetSnapshotStats)
		auditSnapshotAdmin.GET("/export", auditHandler.ExportSnapshots)
		auditSnapshotAdmin.GET("/:processId/chain", auditHandler.GetSnapshotChain)
	}

	// 流程总结数据管理（仅 tenant_admin）：快照列表、统计及总结链路查询
	summarySnapshotAdmin := r.Group("/api/summary/snapshots")
	summarySnapshotAdmin.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		summarySnapshotAdmin.GET("", summaryHandler.ListSnapshots)
		summarySnapshotAdmin.GET("/stats", summaryHandler.GetSnapshotStats)
		summarySnapshotAdmin.GET("/export", summaryHandler.ExportSnapshots)
		summarySnapshotAdmin.GET("/:processId/chain", summaryHandler.GetSnapshotChain)
	}

	// 归档复盘运行时（需要 JWT + 租户上下文，无角色限制）：发起复盘、查询进度、流式输出、历史记录
	archive := r.Group("/api/archive")
	archive.Use(middleware.JWT(rdb), middleware.TenantContext())
	{
		archive.GET("/processes", archiveReviewHandler.ListProcesses)
		archive.GET("/processes/export", archiveReviewHandler.ExportProcesses)
		archive.GET("/stats", archiveReviewHandler.GetStats)
		archive.POST("/execute", archiveReviewHandler.Execute)
		archive.POST("/batch", archiveReviewHandler.BatchExecute)
		archive.POST("/cancel/:id", archiveReviewHandler.CancelJob)
		archive.GET("/jobs/:id", archiveReviewHandler.GetJobStatus)
		archive.GET("/stream/:id", archiveReviewHandler.GetJobStream)
		archive.GET("/history/:processId", archiveReviewHandler.GetHistory)
		archive.GET("/result/:id", archiveReviewHandler.GetResult)
	}

	// 归档日志数据管理（仅 tenant_admin）：日志列表、统计及导出
	archiveAdmin := r.Group("/api/archive/logs")
	archiveAdmin.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		archiveAdmin.GET("", archiveReviewHandler.ListLogs)
		archiveAdmin.GET("/stats", archiveReviewHandler.GetLogStats)
		archiveAdmin.GET("/export", archiveReviewHandler.ExportLogs)
	}

	// 归档快照数据管理（仅 tenant_admin）：快照列表、统计及复盘链路查询
	archiveSnapshotAdmin := r.Group("/api/archive/snapshots")
	archiveSnapshotAdmin.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		archiveSnapshotAdmin.GET("", archiveReviewHandler.ListSnapshots)
		archiveSnapshotAdmin.GET("/stats", archiveReviewHandler.GetSnapshotStats)
		archiveSnapshotAdmin.GET("/:processId/chain", archiveReviewHandler.GetSnapshotChain)
	}

	// 定时任务全量日志数据管理（仅 tenant_admin）：跨任务日志列表、统计及导出
	cronLogsAdmin := r.Group("/api/tenant/cron/logs")
	cronLogsAdmin.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		cronLogsAdmin.GET("", cronTaskHandler.ListAllLogs)
		cronLogsAdmin.GET("/stats", cronTaskHandler.GetAllLogsStats)
		cronLogsAdmin.GET("/export", cronTaskHandler.ExportAllLogs)
	}
}
