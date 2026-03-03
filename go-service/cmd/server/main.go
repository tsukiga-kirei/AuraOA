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
	//1.通过Viper加载配置
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	//2. 初始化记录器
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	//3.通过GORM连接PostgreSQL
	db, err := initDatabase()
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	logger.Info("Database connected successfully")

	//4.连接Redis
	rdb, err := initRedis()
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	logger.Info("Redis connected successfully")

	//5. 初始化存储库
	userRepo := repository.NewUserRepo(db)
	orgRepo := repository.NewOrgRepo(db)
	tenantRepo := repository.NewTenantRepo(db)

	//6. 初始化服务
	authService := service.NewAuthService(userRepo, rdb, db)
	orgService := service.NewOrgService(orgRepo, userRepo, db)
	tenantService := service.NewTenantService(tenantRepo, db)

	//7. 初始化处理程序
	authHandler := handler.NewAuthHandler(authService, rdb)
	orgHandler := handler.NewOrgHandler(orgService)
	tenantHandler := handler.NewTenantHandler(tenantService)
	healthHandler := handler.NewHealthHandler()

	//8. 使用中间件和路由设置 Gin 路由器
	r := gin.New()
	allowedOrigins := viper.GetStringSlice("cors.allowed_origins")
	router.SetupRouter(r, rdb, logger, allowedOrigins, authHandler, orgHandler, tenantHandler, healthHandler)

	//9. 在配置的端口上启动 HTTP 服务器
	port := viper.GetInt("server.port")
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	//10. 优雅关机
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

//loadConfig 使用 Viper 读取 config.yaml。
func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../../")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	return viper.ReadInConfig()
}

//initDatabase 使用 GORM 连接到 PostgreSQL。
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

//initRedis 连接到 Redis。
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
