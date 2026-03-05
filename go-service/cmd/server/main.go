package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/handler"
	"oa-smart-audit/go-service/internal/repository"
	"oa-smart-audit/go-service/internal/router"
	"oa-smart-audit/go-service/internal/service"
)

func main() {
	// 1. Load config via Viper
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// 3. Connect PostgreSQL via GORM
	db, err := initDatabase()
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	logger.Info("Database connected successfully")

	// 4. Connect Redis
	rdb, err := initRedis()
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	logger.Info("Redis connected successfully")

	// 5. Initialize repositories
	userRepo := repository.NewUserRepo(db)
	orgRepo := repository.NewOrgRepo(db)
	tenantRepo := repository.NewTenantRepo(db)
	systemConfigRepo := repository.NewSystemConfigRepo(db)

	// 6. Initialize services
	authService := service.NewAuthService(userRepo, rdb, db)
	orgService := service.NewOrgService(orgRepo, userRepo, db)
	tenantService := service.NewTenantService(tenantRepo, db)
	systemConfigService := service.NewSystemConfigService(systemConfigRepo)

	// 7. Initialize handlers
	authHandler := handler.NewAuthHandler(authService, rdb)
	orgHandler := handler.NewOrgHandler(orgService)
	tenantHandler := handler.NewTenantHandler(tenantService, systemConfigService)
	healthHandler := handler.NewHealthHandler()

	// 8. Setup Gin router with middleware and routes
	r := gin.New()
	// Trust all proxies so that X-Forwarded-For / X-Real-IP headers are respected
	// by c.ClientIP(), which fixes "::1" being recorded in login history when
	// requests come through Docker / Nuxt proxy.
	r.SetTrustedProxies(nil)
	r.ForwardedByClientIP = true
	allowedOrigins := viper.GetStringSlice("cors.allowed_origins")
	router.SetupRouter(r, rdb, logger, allowedOrigins, authHandler, orgHandler, tenantHandler, healthHandler)

	// 9. Start HTTP server
	port := viper.GetInt("server.port")
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	// 10. Graceful shutdown
	go func() {
		logger.Info("Server starting", zap.Int("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}
	logger.Info("Server exited gracefully")
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../../")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	return viper.ReadInConfig()
}

func initDatabase() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		viper.GetString("database.dbname"),
		viper.GetString("database.sslmode"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(viper.GetInt("database.max_open_conns"))
	sqlDB.SetMaxIdleConns(viper.GetInt("database.max_idle_conns"))

	return db, nil
}

func initRedis() (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", viper.GetString("redis.host"), viper.GetInt("redis.port")),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return rdb, nil
}
