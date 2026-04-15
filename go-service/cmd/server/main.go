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
	pkglogger "oa-smart-audit/go-service/internal/pkg/logger"
	"oa-smart-audit/go-service/internal/repository"
	"oa-smart-audit/go-service/internal/router"
	"oa-smart-audit/go-service/internal/service"
)

func main() {
	// 第一步：加载配置文件
	if err := loadConfig(); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 第二步：初始化全局日志系统
	logCfg := pkglogger.LogConfig{
		Level:               viper.GetString("log.level"),
		Dir:                 viper.GetString("log.dir"),
		MaxSizeMB:           viper.GetInt("log.max_size_mb"),
		MaxBackups:          viper.GetInt("log.max_backups"),
		Compress:            viper.GetBool("log.compress"),
		GlobalRetentionDays: viper.GetInt("log.global_retention_days"),
	}
	if err := pkglogger.Init(logCfg); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}
	defer pkglogger.Sync()

	// 第三步：执行数据库迁移（schema_migrations），再建立 GORM 连接
	if viper.GetBool("migrations.enabled") {
		dir := resolveMigrationsPath(viper.GetString("migrations.path"))
		if dir == "" {
			pkglogger.Global().Fatal("migrations.enabled 为 true，但未找到有效的迁移目录")
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
			pkglogger.Global().Fatal("数据库迁移失败", zap.String("dir", dir), zap.Error(err))
		}
		pkglogger.Global().Info("数据库迁移完成", zap.String("dir", dir))
	}

	db, err := initDatabase()
	if err != nil {
		pkglogger.Global().Fatal("数据库连接失败", zap.Error(err))
	}
	pkglogger.Global().Info("数据库连接成功")

	// 第四步：连接 Redis
	rdb, err := initRedis()
	if err != nil {
		pkglogger.Global().Fatal("Redis 连接失败", zap.Error(err))
	}
	pkglogger.Global().Info("Redis 连接成功")

	// 第四步（补充）：初始化 AES 加密密钥
	encKey := viper.GetString("encryption.key")
	if encKey == "" {
		pkglogger.Global().Fatal("encryption.key 未配置")
	}
	if err := crypto.SetKey(encKey); err != nil {
		pkglogger.Global().Fatal("设置加密密钥失败", zap.Error(err))
	}

	// 第五步：初始化各数据访问层（Repository）
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

	// 第六步：初始化各业务服务层（Service）
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
	userNotificationService := service.NewUserNotificationService(userNotificationRepo, userRepo)
	auditExecuteService := service.NewAuditExecuteService(auditLogRepo, auditSnapshotRepo, processAuditConfigRepo, auditRuleRepo, userPersonalConfigRepo, tenantRepo, oaConnectionRepo, aiModelRepo, aiCallerService, db, rdb, userNotificationService)
	dashboardOverviewService := service.NewDashboardOverviewService(
		auditSnapshotRepo, archiveSnapshotRepo, auditLogRepo, archiveLogRepo, cronLogRepo, cronTaskRepo, cronPresetRepo, llmMessageLogRepo, tenantRepo, orgRepo,
	)
	archiveReviewService := service.NewArchiveReviewService(archiveLogRepo, archiveSnapshotRepo, archiveConfigRepo, archiveRuleRepo, userPersonalConfigRepo, tenantRepo, oaConnectionRepo, aiModelRepo, aiCallerService, orgRepo, db, rdb, userNotificationService)
	reportCalculatorService := service.NewReportCalculatorService(auditLogRepo, archiveLogRepo, tenantRepo)
	mailService := service.NewMailService(systemConfigRepo)

	// 初始化 Cron 任务实例服务（调度器延迟注入）
	cronTaskService := service.NewCronTaskService(cronTaskRepo, cronLogRepo, cronPresetRepo, cronConfigRepo, userRepo, tenantRepo, auditExecuteService, archiveReviewService, reportCalculatorService, mailService, userNotificationService)
	cronScheduler := service.NewCronScheduler(cronTaskRepo, cronTaskService, pkglogger.Global())
	cronTaskService.SetScheduler(cronScheduler)

	// 注册日志清理定时任务（每日凌晨 00:00 执行）
	logCleanupService := service.NewLogCleanupService(systemConfigRepo, tenantRepo, viper.GetInt("log.global_retention_days"))
	if err := cronScheduler.RegisterCustomJob("0 0 * * *", func() {
		if cleanErr := logCleanupService.RunCleanup(context.Background()); cleanErr != nil {
			pkglogger.Global().Warn("日志清理任务执行失败", zap.Error(cleanErr))
		}
	}); err != nil {
		pkglogger.Global().Warn("注册日志清理定时任务失败", zap.Error(err))
	}

	if err := service.StartAuditStreamWorker(context.Background(), rdb, auditExecuteService, pkglogger.Global(), 2); err != nil {
		pkglogger.Global().Warn("审计流处理器启动失败", zap.Error(err))
	}
	service.StartAuditStaleReconciler(context.Background(), auditExecuteService, pkglogger.Global(), 30*time.Second)
	if err := service.StartArchiveStreamWorker(context.Background(), rdb, archiveReviewService, pkglogger.Global(), 2); err != nil {
		pkglogger.Global().Warn("归档流处理器启动失败", zap.Error(err))
	}
	service.StartArchiveStaleReconciler(context.Background(), archiveReviewService, pkglogger.Global(), 30*time.Second)

	// 启动 Cron 调度器
	if err := cronScheduler.Start(context.Background()); err != nil {
		pkglogger.Global().Warn("Cron 调度器启动失败", zap.Error(err))
	}

	// 第七步：初始化各 HTTP 处理器（Handler）
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

	// 第八步：配置 Gin 路由及中间件
	r := gin.New()
	r.SetTrustedProxies(nil)
	r.ForwardedByClientIP = true
	allowedOrigins := viper.GetStringSlice("cors.allowed_origins")
	router.SetupRouter(r, rdb, pkglogger.Global(), allowedOrigins, authHandler, orgHandler, tenantHandler, systemHandler, healthHandler, configHandler, ruleHandler, userConfigHandler, userConfigMgmtHandler, llmLogHandler, cronHandler, cronTaskHandler, archiveConfigHandler, archiveRuleHandler, auditHandler, archiveReviewHandler, dashboardOverviewHandler, userNotificationHandler)

	// 第九步：启动 HTTP 服务器
	port := viper.GetInt("server.port")
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	// 第十步：监听系统信号，优雅关闭服务
	go func() {
		pkglogger.Global().Info("服务器启动", zap.Int("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			pkglogger.Global().Fatal("服务器启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	pkglogger.Global().Info("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		pkglogger.Global().Fatal("服务器强制关闭", zap.Error(err))
	}
	pkglogger.Global().Info("服务器已优雅退出")
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

// resolveMigrationsPath 返回迁移 SQL 所在目录。
// 优先使用配置或环境变量中的路径，否则在当前工作目录下尝试常见相对路径（便于本地 go run）。
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// 使用自定义 zap logger，将 GORM 的 SQL 错误（含达梦驱动报错）写入 app.log
		// record not found 属于正常业务逻辑，忽略；慢查询阈值 200ms
		Logger: pkglogger.NewGormLogger(200*time.Millisecond, true),
	})
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
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
		return nil, fmt.Errorf("Redis 连接失败: %w", err)
	}

	return rdb, nil
}
