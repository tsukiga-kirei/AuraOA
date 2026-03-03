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
	healthHandler *handler.HealthHandler,
) {
	// Global middleware
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.CORS(allowedOrigins))

	// Public routes (no auth required)
	r.GET("/api/health", healthHandler.Health)
	r.POST("/api/auth/login", authHandler.Login)
	r.GET("/api/tenants/list", tenantHandler.ListPublicTenants)

	// Auth routes (JWT required)
	auth := r.Group("/api/auth")
	auth.Use(middleware.JWT(rdb))
	{
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/refresh", authHandler.Refresh)
		auth.PUT("/switch-role", authHandler.SwitchRole)
		auth.GET("/menu", authHandler.GetMenu)
		auth.PUT("/change-password", authHandler.ChangePassword)
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
		admin.GET("/tenants", tenantHandler.ListTenants)
		admin.POST("/tenants", tenantHandler.CreateTenant)
		admin.PUT("/tenants/:id", tenantHandler.UpdateTenant)
		admin.DELETE("/tenants/:id", tenantHandler.DeleteTenant)
		admin.GET("/tenants/:id/stats", tenantHandler.GetTenantStats)

		admin.GET("/system/configs", tenantHandler.GetSystemConfigs)
		admin.PUT("/system/configs", tenantHandler.UpdateSystemConfigs)
	}
}
