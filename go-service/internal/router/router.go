package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"oa-smart-audit/go-service/internal/handler"
	"oa-smart-audit/go-service/internal/middleware"
)

// SetupRouter registers all routes and middleware on the given Gin engine.
func SetupRouter(
	r *gin.Engine,
	rdb *redis.Client,
	logger *zap.Logger,
	allowedOrigins []string,
	authHandler *handler.AuthHandler,
	orgHandler *handler.OrgHandler,
	tenantHandler *handler.TenantHandler,
	systemHandler *handler.SystemHandler,
	healthHandler *handler.HealthHandler,
	configHandler *handler.ProcessAuditConfigHandler,
	ruleHandler *handler.AuditRuleHandler,
	presetHandler *handler.StrictnessPresetHandler,
	userConfigHandler *handler.UserPersonalConfigHandler,
	userConfigMgmtHandler *handler.UserConfigManagementHandler,
	llmLogHandler *handler.LLMMessageLogHandler,
) {
	// Global middleware
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.CORS(allowedOrigins))

	// Public routes (no auth required)
	r.GET("/api/health", healthHandler.Health)
	r.POST("/api/auth/login", authHandler.Login)
	r.POST("/api/auth/refresh", authHandler.Refresh)
	r.GET("/api/tenants/list", tenantHandler.ListPublicTenants)

	// Auth routes (JWT required)
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
	}

	// Tenant org routes (JWT + TenantContext + tenant_admin)
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
	}

	// Admin routes (JWT + TenantContext + system_admin)
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

		// 系统设置 — 选项数据
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
		}

		// 系统管理员 — Token 消耗统计
		admin.GET("/stats/token-usage", llmLogHandler.QueryAllTenantsTokenUsage)
	}

	// 租户管理员路由组（JWT + TenantContext + tenant_admin）
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
		tenantRules.PUT("/audit-rules/:id", ruleHandler.Update)
		tenantRules.DELETE("/audit-rules/:id", ruleHandler.Delete)

		// 审核尺度预设
		tenantRules.GET("/strictness-presets", presetHandler.List)
		tenantRules.PUT("/strictness-presets/:strictness", presetHandler.Update)
	}

	// 租户管理员 — 用户配置管理
	tenantUserConfigs := r.Group("/api/tenant/user-configs")
	tenantUserConfigs.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		tenantUserConfigs.GET("", userConfigMgmtHandler.ListUserConfigs)
		tenantUserConfigs.GET("/:userId", userConfigMgmtHandler.GetUserConfig)
	}

	// 租户管理员 — Token 消耗统计
	tenantStats := r.Group("/api/tenant/stats")
	tenantStats.Use(middleware.JWT(rdb), middleware.TenantContext(), middleware.RequireRole("tenant_admin"))
	{
		tenantStats.GET("/token-usage", llmLogHandler.QueryTokenUsage)
	}

	// 业务用户路由组（JWT + TenantContext，无角色限制）
	tenantSettings := r.Group("/api/tenant/settings")
	tenantSettings.Use(middleware.JWT(rdb), middleware.TenantContext())
	{
		tenantSettings.GET("/processes", userConfigHandler.GetProcessList)
		tenantSettings.GET("/processes/:processType", userConfigHandler.GetByProcessType)
		tenantSettings.PUT("/processes/:processType", userConfigHandler.UpdateByProcessType)
		tenantSettings.GET("/dashboard-prefs", userConfigHandler.GetDashboardPrefs)
		tenantSettings.PUT("/dashboard-prefs", userConfigHandler.UpdateDashboardPrefs)
	}
}
