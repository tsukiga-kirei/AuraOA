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
			system.PUT("/oa-connections/:id", systemHandler.UpdateOAConnection)
			system.DELETE("/oa-connections/:id", systemHandler.DeleteOAConnection)
			system.POST("/oa-connections/:id/test", systemHandler.TestOAConnection)

			// AI 模型配置
			system.GET("/ai-models", systemHandler.ListAIModels)
			system.POST("/ai-models", systemHandler.CreateAIModel)
			system.PUT("/ai-models/:id", systemHandler.UpdateAIModel)
			system.DELETE("/ai-models/:id", systemHandler.DeleteAIModel)

			// 系统配置 (KV)
			system.GET("/configs", systemHandler.GetSystemConfigs)
			system.PUT("/configs", systemHandler.UpdateSystemConfigs)
		}
	}
}
