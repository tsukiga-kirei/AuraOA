package main

import (
	"log"
	"os"

	"oa-smart-audit/internal/audit"
	"oa-smart-audit/internal/auth"
	"oa-smart-audit/internal/cron"
	"oa-smart-audit/internal/history"
	"oa-smart-audit/internal/observability"
	"oa-smart-audit/internal/rule"
	"oa-smart-audit/internal/security"
	"oa-smart-audit/internal/tenant"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Observability middleware
	metrics := observability.NewMetrics(observability.AlertThresholds{})
	r.Use(observability.TraceMiddleware())
	r.Use(observability.MetricsMiddleware(metrics))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "go-business-service"})
	})

	api := r.Group("/api")

	// Auth routes (public + protected)
	authHandler := auth.NewHandler()
	authHandler.RegisterRoutes(api)

	// Protected routes
	protected := api.Group("")
	protected.Use(auth.JWTMiddleware())
	{
		// Audit routes
		auditHandler := audit.NewHandler()
		auditHandler.RegisterRoutes(protected)

		// Rule / preference routes
		ruleHandler := rule.NewHandler()
		ruleHandler.RegisterUserRoutes(protected)

		// Cron routes
		cronHandler := cron.NewHandler()
		cronHandler.RegisterRoutes(protected)

		// History routes
		historyHandler := history.NewHandler()
		historyHandler.RegisterRoutes(protected)
	}

	// Admin routes (tenant admin + system admin)
	admin := api.Group("/admin")
	admin.Use(auth.JWTMiddleware(), auth.RequireRole(auth.RoleAdmin, auth.RoleTenantAdmin))
	{
		tenantHandler := tenant.NewHandler()
		tenantHandler.RegisterRoutes(admin)

		ruleHandler := rule.NewHandler()
		ruleHandler.RegisterAdminRoutes(admin)

		securityHandler := security.NewHandler()
		securityHandler.RegisterRoutes(admin)
	}

	// System admin only routes
	sysAdmin := api.Group("/admin")
	sysAdmin.Use(auth.JWTMiddleware(), auth.RequireRole(auth.RoleAdmin))
	{
		sysAdmin.GET("/monitor", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"system_health":        "healthy",
				"api_success_rate":     metrics.GetAPISuccessRate() * 100,
				"avg_model_response_ms": metrics.GetAvgModelResponseMs(),
				"alerts":              metrics.GetAlerts(),
			})
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Go business service starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func placeholder(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(501, gin.H{"message": name + " not implemented yet"})
	}
}
