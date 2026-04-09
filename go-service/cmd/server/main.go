package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/dbmigrate"
	"oa-smart-audit/go-service/internal/handler"
	"oa-smart-audit/go-service/internal/pkg/crypto"
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

	// 3. PostgreSQL schema migrations (schema_migrations)，再建立 GORM 连接
	if viper.GetBool("migrations.enabled") {
		dir := resolveMigrationsPath(viper.GetString("migrations.path"))
		if dir == "" {
			logger.Fatal("migrations.enabled is true but migrations.path is empty and no default db/migrations directory was found")
		}
		if err := dbmigrate.Up(
			dir,
			viper.GetString("database.host"),
			viper.GetInt("database.port"),
			viper.GetString("database.user"),
			viper.GetString("database.password"),
			viper.GetString("database.dbname"),
			viper.GetString("database.sslmode"),
		); err != nil {
			logger.Fatal("Database migrations failed", zap.String("dir", dir), zap.Error(err))
		}
		logger.Info("Database migrations applied", zap.String("dir", dir))
	}

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

	// 4.5 Initialize AES encryption key
	encKey := viper.GetString("encryption.key")
	if encKey == "" {
		logger.Fatal("encryption.key is not configured")
	}
	if err := crypto.SetKey(encKey); err != nil {
		logger.Fatal("Failed to set encryption key", zap.Error(err))
	}

	// 5. Initialize repositories
	userRepo := repository.NewUserRepo(db)
	orgRepo := repository.NewOrgRepo(db)
	tenantRepo := repository.NewTenantRepo(db)
	systemConfigRepo := repository.NewSystemConfigRepo(db)
	optionRepo := repository.NewOptionRepo(db)
	oaConnectionRepo := repository.NewOAConnectionRepo(db)
	aiModelRepo := repository.NewAIModelRepo(db)
	processAuditConfigRepo := repository.NewProcessAuditConfigRepo(db)
	auditRuleRepo := repository.NewAuditRuleRepo(db)
	promptTemplateRepo := repository.NewSystemPromptTemplateRepo(db)
	userPersonalConfigRepo := repository.NewUserPersonalConfigRepo(db)
	userDashboardPrefRepo := repository.NewUserDashboardPrefRepo(db)
	userNotificationRepo := repository.NewUserNotificationRepo(db)
	llmMessageLogRepo := repository.NewLLMMessageLogRepo(db)
	cronPresetRepo := repository.NewCronTaskTypePresetRepo(db)
	cronConfigRepo := repository.NewCronTaskTypeConfigRepo(db)
	cronTaskRepo := repository.NewCronTaskRepo(db)
	cronLogRepo := repository.NewCronLogRepo(db)
	archiveConfigRepo := repository.NewProcessArchiveConfigRepo(db)
	archiveRuleRepo := repository.NewArchiveRuleRepo(db)

	auditLogRepo := repository.NewAuditLogRepo(db)
	archiveLogRepo := repository.NewArchiveLogRepo(db)
	auditSnapshotRepo := repository.NewAuditProcessSnapshotRepo(db)
	archiveSnapshotRepo := repository.NewArchiveProcessSnapshotRepo(db)

	// 6. Initialize services
	authService := service.NewAuthService(userRepo, rdb, db)
	orgService := service.NewOrgService(orgRepo, userRepo, systemConfigRepo, db)
	tenantService := service.NewTenantService(tenantRepo, systemConfigRepo, userRepo, db)
	systemConfigService := service.NewSystemConfigService(systemConfigRepo)
	optionService := service.NewOptionService(optionRepo)
	oaConnectionService := service.NewOAConnectionService(oaConnectionRepo)
	aiModelService := service.NewAIModelService(aiModelRepo)
	processAuditConfigService := service.NewProcessAuditConfigService(processAuditConfigRepo, tenantRepo, oaConnectionRepo, promptTemplateRepo, db)
	auditRuleService := service.NewAuditRuleService(auditRuleRepo)
	userPersonalConfigService := service.NewUserPersonalConfigService(userPersonalConfigRepo, processAuditConfigRepo, auditRuleRepo, archiveConfigRepo, archiveRuleRepo, orgRepo)
	llmMessageLogService := service.NewLLMMessageLogService(llmMessageLogRepo)
	cronConfigService := service.NewCronConfigService(cronPresetRepo, cronConfigRepo)
	archiveConfigService := service.NewProcessArchiveConfigService(archiveConfigRepo, tenantRepo, oaConnectionRepo, promptTemplateRepo)
	archiveRuleService := service.NewArchiveRuleService(archiveRuleRepo)
	aiCallerService := service.NewAIModelCallerService(tenantRepo, llmMessageLogRepo, db)
	auditExecuteService := service.NewAuditExecuteService(auditLogRepo, auditSnapshotRepo, processAuditConfigRepo, auditRuleRepo, userPersonalConfigRepo, tenantRepo, oaConnectionRepo, aiModelRepo, aiCallerService, db, rdb)
	dashboardOverviewService := service.NewDashboardOverviewService(
		auditSnapshotRepo, archiveSnapshotRepo, auditLogRepo, archiveLogRepo, cronLogRepo, llmMessageLogRepo, tenantRepo, orgRepo,
	)
	userNotificationService := service.NewUserNotificationService(userNotificationRepo, userRepo)
	archiveReviewService := service.NewArchiveReviewService(archiveLogRepo, archiveSnapshotRepo, archiveConfigRepo, archiveRuleRepo, userPersonalConfigRepo, tenantRepo, oaConnectionRepo, aiModelRepo, aiCallerService, orgRepo, db, rdb)
	reportCalculatorService := service.NewReportCalculatorService(auditLogRepo, archiveLogRepo, tenantRepo)
	mailService := service.NewMailService(systemConfigRepo)

	// Cron 任务实例服务（延迟注入调度器）
	cronTaskService := service.NewCronTaskService(cronTaskRepo, cronLogRepo, cronPresetRepo, cronConfigRepo, userRepo, tenantRepo, auditExecuteService, archiveReviewService, reportCalculatorService, mailService)
	cronScheduler := service.NewCronScheduler(cronTaskRepo, cronTaskService, logger)
	cronTaskService.SetScheduler(cronScheduler)

	if err := service.StartAuditStreamWorker(context.Background(), rdb, auditExecuteService, logger, 2); err != nil {
		logger.Warn("audit stream worker not started", zap.Error(err))
	}
	service.StartAuditStaleReconciler(context.Background(), auditExecuteService, logger, 30*time.Second)
	if err := service.StartArchiveStreamWorker(context.Background(), rdb, archiveReviewService, logger, 2); err != nil {
		logger.Warn("archive stream worker not started", zap.Error(err))
	}
	service.StartArchiveStaleReconciler(context.Background(), archiveReviewService, logger, 30*time.Second)

	// 启动 cron 调度器
	if err := cronScheduler.Start(context.Background()); err != nil {
		logger.Warn("cron scheduler not started", zap.Error(err))
	}

	// 7. Initialize handlers
	authHandler := handler.NewAuthHandler(authService, rdb)
	orgHandler := handler.NewOrgHandler(orgService)
	tenantHandler := handler.NewTenantHandler(tenantService)
	systemHandler := handler.NewSystemHandler(optionService, oaConnectionService, aiModelService, systemConfigService)
	healthHandler := handler.NewHealthHandler()
	configHandler := handler.NewProcessAuditConfigHandler(processAuditConfigService)
	ruleHandler := handler.NewAuditRuleHandler(auditRuleService)
	userConfigHandler := handler.NewUserPersonalConfigHandler(userPersonalConfigService, userDashboardPrefRepo)
	userConfigMgmtHandler := handler.NewUserConfigManagementHandler(userPersonalConfigRepo, cronTaskRepo, orgRepo, auditRuleRepo, archiveRuleRepo, processAuditConfigRepo, archiveConfigRepo)
	llmLogHandler := handler.NewLLMMessageLogHandler(llmMessageLogService)
	cronHandler := handler.NewCronConfigHandler(cronConfigService)
	cronTaskHandler := handler.NewCronTaskHandler(cronTaskService)
	archiveConfigHandler := handler.NewArchiveConfigHandler(archiveConfigService)
	archiveRuleHandler := handler.NewArchiveRuleHandler(archiveRuleService)
	auditHandler := handler.NewAuditHandler(auditExecuteService, auditSnapshotRepo, auditLogRepo)
	archiveReviewHandler := handler.NewArchiveReviewHandler(archiveReviewService, archiveSnapshotRepo, archiveLogRepo)
	dashboardOverviewHandler := handler.NewDashboardOverviewHandler(dashboardOverviewService)
	userNotificationHandler := handler.NewUserNotificationHandler(userNotificationService)

	// 8. Setup Gin router with middleware and routes
	r := gin.New()
	r.SetTrustedProxies(nil)
	r.ForwardedByClientIP = true
	allowedOrigins := viper.GetStringSlice("cors.allowed_origins")
	router.SetupRouter(r, rdb, logger, allowedOrigins, authHandler, orgHandler, tenantHandler, systemHandler, healthHandler, configHandler, ruleHandler, userConfigHandler, userConfigMgmtHandler, llmLogHandler, cronHandler, cronTaskHandler, archiveConfigHandler, archiveRuleHandler, auditHandler, archiveReviewHandler, dashboardOverviewHandler, userNotificationHandler)

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
	viper.SetDefault("migrations.enabled", true)
	viper.SetDefault("migrations.path", "")
	return viper.ReadInConfig()
}

// resolveMigrationsPath 返回迁移 SQL 所在目录；优先配置/环境变量，否则在当前工作目录下尝试常见相对路径（便于本地 go run）。
func resolveMigrationsPath(configured string) string {
	candidates := []string{}
	if configured != "" {
		candidates = append(candidates, configured)
	}
	if env := os.Getenv("MIGRATIONS_PATH"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates, "db/migrations", "../db/migrations", "../../db/migrations")

	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		p := c
		if !filepath.IsAbs(p) {
			p = filepath.Join(wd, c)
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			if _, err := os.Stat(filepath.Join(p, "000001_init_extensions.up.sql")); err == nil {
				return p
			}
		}
	}
	return ""
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
