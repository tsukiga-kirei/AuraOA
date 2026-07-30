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

	"auraoa/go-service/internal/cache"
	"auraoa/go-service/internal/dbmigrate"
	"auraoa/go-service/internal/handler"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/crypto"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/pkg/oa"
	"auraoa/go-service/internal/pkg/systemflags"
	"auraoa/go-service/internal/repository"
	"auraoa/go-service/internal/router"
	"auraoa/go-service/internal/service"
)

func main() {
	// 第一步：加载配置文件
	if err := loadConfig(); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}
	ginMode := strings.TrimSpace(os.Getenv(gin.EnvGinMode))
	if ginMode == "" {
		ginMode = strings.TrimSpace(viper.GetString("server.mode"))
	}
	switch ginMode {
	case gin.DebugMode, gin.TestMode, gin.ReleaseMode:
		gin.SetMode(ginMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
	if err := apptime.Configure(viper.GetString("app.timezone")); err != nil {
		log.Fatalf("初始化应用时区失败: %v", err)
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
			apptime.Name(),
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

	// 第四步（补充）：初始化缓存组件
	cacheConfig := cache.Config{
		Enabled:          viper.GetBool("cache.enabled"),
		DefaultTTL:       viper.GetDuration("cache.default_ttl"),
		HitRateThreshold: viper.GetFloat64("cache.hit_rate_threshold"),
		TTL: cache.TTLConfig{
			AuditTodo:     viper.GetDuration("cache.ttl.audit_todo"),
			ArchiveList:   viper.GetDuration("cache.ttl.archive_list"),
			ProcessConfig: viper.GetDuration("cache.ttl.process_config"),
			Snapshot:      viper.GetDuration("cache.ttl.snapshot"),
			Stats:         viper.GetDuration("cache.ttl.stats"),
			Dashboard:     viper.GetDuration("cache.ttl.dashboard"),
		},
	}
	cacheConfig.ApplyDefaults()
	cacheManager := cache.NewCacheManager(rdb, pkglogger.Global(), cacheConfig)
	invalidationManager := cache.NewInvalidationManager(cacheManager, pkglogger.Global())

	// 第四步（补充）：初始化 AES 加密密钥
	encKey := viper.GetString("encryption.key")
	if encKey == "" {
		pkglogger.Global().Fatal("encryption.key 未配置")
	}
	if err := crypto.SetKey(encKey); err != nil {
		pkglogger.Global().Fatal("设置加密密钥失败", zap.Error(err))
	}
	oaConnectionManager := oa.NewConnectionManager(pkglogger.Global())
	defer func() {
		if err := oaConnectionManager.Close(); err != nil {
			pkglogger.Global().Warn("关闭 OA 数据库共享连接池失败", zap.Error(err))
		}
	}()

	// 第五步：初始化各数据访问层（Repository）
	userRepo := repository.NewUserRepo(db)
	orgRepo := repository.NewOrgRepo(db)
	tenantRepo := repository.NewTenantRepo(db)
	systemConfigRepo := repository.NewSystemConfigRepo(db)
	sysFlagsResolver := systemflags.NewResolver(systemConfigRepo)
	operationAuditLogRepo := repository.NewOperationAuditLogRepo(db)
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
	embedRefreshScheduleRepo := repository.NewEmbedRefreshScheduleRepo(db)
	archiveConfigRepo := repository.NewProcessArchiveConfigRepo(db)
	archiveRuleRepo := repository.NewArchiveRuleRepo(db)
	summaryConfigRepo := repository.NewProcessSummaryConfigRepo(db)

	auditLogRepo := repository.NewAuditLogRepo(db)
	archiveLogRepo := repository.NewArchiveLogRepo(db)
	summaryLogRepo := repository.NewProcessSummaryLogRepo(db)
	auditSnapshotRepo := repository.NewAuditProcessSnapshotRepo(db)
	archiveSnapshotRepo := repository.NewArchiveProcessSnapshotRepo(db)
	summarySnapshotRepo := repository.NewProcessSummarySnapshotRepo(db)

	// 第六步：初始化各业务服务层（Service）
	authService := service.NewAuthService(userRepo, rdb, db, systemConfigRepo)
	basicSSOService := service.NewBasicSSOService(tenantRepo, userRepo, authService, rdb)
	orgService := service.NewOrgService(orgRepo, userRepo, systemConfigRepo, db)
	tenantService := service.NewTenantService(tenantRepo, systemConfigRepo, userRepo, db, invalidationManager)
	systemConfigService := service.NewSystemConfigService(systemConfigRepo)
	optionService := service.NewOptionService(optionRepo)
	oaConnectionService := service.NewOAConnectionService(oaConnectionRepo, tenantRepo, invalidationManager, oaConnectionManager)
	aiModelService := service.NewAIModelService(aiModelRepo)
	processAuditConfigService := service.NewProcessAuditConfigService(processAuditConfigRepo, tenantRepo, oaConnectionRepo, promptTemplateRepo, db, invalidationManager, oaConnectionManager)
	auditRuleService := service.NewAuditRuleService(auditRuleRepo, invalidationManager)
	userPersonalConfigService := service.NewUserPersonalConfigService(userPersonalConfigRepo, processAuditConfigRepo, auditRuleRepo, archiveConfigRepo, archiveRuleRepo, orgRepo)
	llmMessageLogService := service.NewLLMMessageLogService(llmMessageLogRepo)
	cronConfigService := service.NewCronConfigService(cronPresetRepo, cronConfigRepo)
	archiveConfigService := service.NewProcessArchiveConfigService(archiveConfigRepo, tenantRepo, oaConnectionRepo, promptTemplateRepo, invalidationManager, oaConnectionManager)
	archiveRuleService := service.NewArchiveRuleService(archiveRuleRepo, invalidationManager)
	aiCallerService := service.NewAIModelCallerService(tenantRepo, llmMessageLogRepo, db, sysFlagsResolver)
	userNotificationService := service.NewUserNotificationService(userNotificationRepo, userRepo)
	// 附件识别服务依赖 system_configs，须在 audit/archive 之前初始化（两者会注入它）
	minerUTimeout := time.Duration(viper.GetInt("attachment.mineru_timeout_seconds")) * time.Second
	attachmentRecognitionService := service.NewAttachmentRecognitionService(systemConfigRepo, minerUTimeout)
	ruleImportService := service.NewRuleImportService(attachmentRecognitionService, tenantRepo, aiModelRepo, aiCallerService, processAuditConfigRepo, archiveConfigRepo, auditRuleRepo, archiveRuleRepo, invalidationManager)
	externalContextService := service.NewExternalContextService(oaConnectionRepo, attachmentRecognitionService, oaConnectionManager)
	auditExecuteService := service.NewAuditExecuteService(auditLogRepo, auditSnapshotRepo, processAuditConfigRepo, auditRuleRepo, userPersonalConfigRepo, tenantRepo, oaConnectionRepo, aiModelRepo, aiCallerService, attachmentRecognitionService, db, rdb, userNotificationService, cacheManager, invalidationManager, sysFlagsResolver, externalContextService, oaConnectionManager)
	summaryConfigService := service.NewProcessSummaryConfigService(summaryConfigRepo, tenantRepo, oaConnectionRepo, invalidationManager, oaConnectionManager)
	summaryService := service.NewProcessSummaryService(summaryLogRepo, summarySnapshotRepo, summaryConfigRepo, tenantRepo, oaConnectionRepo, aiModelRepo, aiCallerService, attachmentRecognitionService, db, rdb, sysFlagsResolver, externalContextService, oaConnectionManager)
	embedRefreshService := service.NewEmbedRefreshService(
		rdb,
		auditExecuteService,
		summaryService,
		processAuditConfigRepo,
		summaryConfigRepo,
		embedRefreshScheduleRepo,
		tenantRepo,
		pkglogger.Global(),
	)
	processAuditConfigService.SetEmbedRefreshScheduleManager(embedRefreshService)
	summaryConfigService.SetEmbedRefreshScheduleManager(embedRefreshService)
	dashboardOverviewService := service.NewDashboardOverviewService(
		auditSnapshotRepo, archiveSnapshotRepo, auditLogRepo, archiveLogRepo, cronLogRepo, cronTaskRepo, cronPresetRepo, llmMessageLogRepo, tenantRepo, orgRepo, cacheManager, invalidationManager,
	)
	archiveReviewService := service.NewArchiveReviewService(archiveLogRepo, archiveSnapshotRepo, archiveConfigRepo, archiveRuleRepo, userPersonalConfigRepo, tenantRepo, oaConnectionRepo, aiModelRepo, aiCallerService, attachmentRecognitionService, orgRepo, db, rdb, userNotificationService, cacheManager, invalidationManager, sysFlagsResolver, externalContextService, oaConnectionManager)
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

	backupDir := filepath.Clean(viper.GetString("backup.dir"))
	if !filepath.IsAbs(backupDir) {
		if wd, wdErr := os.Getwd(); wdErr == nil {
			backupDir = filepath.Join(wd, backupDir)
		}
	}
	dumpTO := viper.GetDuration("backup.dump_timeout")
	if dumpTO <= 0 {
		dumpTO = 45 * time.Minute
	}
	dbBackupService := service.NewDbBackupService(systemConfigRepo, service.DbBackupConfig{
		Host:                  viper.GetString("database.host"),
		Port:                  viper.GetInt("database.port"),
		User:                  viper.GetString("database.user"),
		Password:              viper.GetString("database.password"),
		DBName:                viper.GetString("database.dbname"),
		SSLMode:               viper.GetString("database.sslmode"),
		Dir:                   backupDir,
		RetentionFallbackDays: viper.GetInt("backup.retention_fallback_days"),
		DumpTimeout:           dumpTO,
	})
	if err := cronScheduler.RegisterCustomJob("0 * * * * *", func() {
		go func() {
			runTO := dbBackupService.DumpTimeout() + 15*time.Minute
			ctx, cancel := context.WithTimeout(context.Background(), runTO)
			defer cancel()
			dbBackupService.Tick(ctx)
		}()
	}); err != nil {
		pkglogger.Global().Warn("注册数据库备份定时任务失败", zap.Error(err))
	} else {
		pkglogger.Global().Info("数据库自动备份任务已注册",
			zap.String("backup_dir", backupDir),
			zap.String("note", "每分钟检查 system.backup_*；开启后按 Cron 执行 pg_dump"),
		)
	}

	if err := service.StartAuditStreamWorker(
		context.Background(),
		rdb,
		auditExecuteService,
		pkglogger.Global(),
		viper.GetInt("workers.audit_workbench_concurrency"),
		viper.GetInt("workers.audit_interactive_concurrency"),
		viper.GetInt("workers.audit_background_concurrency"),
		viper.GetInt("workers.audit_scheduled_concurrency"),
		viper.GetInt("workers.audit_total_concurrency"),
	); err != nil {
		pkglogger.Global().Warn("审计流处理器启动失败", zap.Error(err))
	}
	service.StartAuditStaleReconciler(context.Background(), auditExecuteService, pkglogger.Global(), 30*time.Second)
	if err := service.StartArchiveStreamWorker(context.Background(), rdb, archiveReviewService, pkglogger.Global(), 2); err != nil {
		pkglogger.Global().Warn("归档流处理器启动失败", zap.Error(err))
	}
	service.StartArchiveStaleReconciler(context.Background(), archiveReviewService, pkglogger.Global(), 30*time.Second)
	if err := service.StartSummaryStreamWorker(
		context.Background(),
		rdb,
		summaryService,
		pkglogger.Global(),
		viper.GetInt("workers.summary_interactive_concurrency"),
		viper.GetInt("workers.summary_background_concurrency"),
		viper.GetInt("workers.summary_scheduled_concurrency"),
		viper.GetInt("workers.summary_total_concurrency"),
	); err != nil {
		pkglogger.Global().Warn("总结流处理器启动失败", zap.Error(err))
	}
	service.StartSummaryStaleReconciler(context.Background(), summaryService, pkglogger.Global(), 30*time.Second)
	embedRefreshService.Start(context.Background())

	// 启动 Cron 调度器
	if err := cronScheduler.Start(context.Background()); err != nil {
		pkglogger.Global().Warn("Cron 调度器启动失败", zap.Error(err))
	}

	// 第七步：初始化各 HTTP 处理器（Handler）
	authHandler := handler.NewAuthHandler(authService, rdb)
	basicSSOHandler := handler.NewBasicSSOHandler(basicSSOService, viper.GetString("sso.public_base_url"))
	orgHandler := handler.NewOrgHandler(orgService)
	tenantHandler := handler.NewTenantHandler(tenantService)
	systemHandler := handler.NewSystemHandler(optionService, oaConnectionService, aiModelService, systemConfigService, attachmentRecognitionService)
	healthHandler := handler.NewHealthHandler()
	configHandler := handler.NewProcessAuditConfigHandler(processAuditConfigService)
	ruleHandler := handler.NewAuditRuleHandler(auditRuleService, ruleImportService)
	userConfigHandler := handler.NewUserPersonalConfigHandler(userPersonalConfigService, userDashboardPrefRepo)
	userConfigMgmtHandler := handler.NewUserConfigManagementHandler(userPersonalConfigRepo, cronTaskRepo, orgRepo, auditRuleRepo, archiveRuleRepo, processAuditConfigRepo, archiveConfigRepo)
	llmLogHandler := handler.NewLLMMessageLogHandler(llmMessageLogService)
	cronHandler := handler.NewCronConfigHandler(cronConfigService)
	cronTaskHandler := handler.NewCronTaskHandler(cronTaskService)
	archiveConfigHandler := handler.NewArchiveConfigHandler(archiveConfigService)
	archiveRuleHandler := handler.NewArchiveRuleHandler(archiveRuleService, ruleImportService)
	externalContextHandler := handler.NewExternalContextHandler(externalContextService, tenantRepo)
	auditHandler := handler.NewAuditHandler(auditExecuteService, auditSnapshotRepo, auditLogRepo)
	archiveReviewHandler := handler.NewArchiveReviewHandler(archiveReviewService, archiveSnapshotRepo, archiveLogRepo)
	summaryConfigHandler := handler.NewProcessSummaryConfigHandler(summaryConfigService)
	summaryHandler := handler.NewProcessSummaryHandler(summaryService)
	embedEventHandler := handler.NewEmbedEventHandler(embedRefreshService)
	systemMonitorService := service.NewSystemMonitorService(db, rdb)
	dashboardOverviewHandler := handler.NewDashboardOverviewHandler(dashboardOverviewService, systemMonitorService)
	userNotificationHandler := handler.NewUserNotificationHandler(userNotificationService)
	cacheAdminHandler := handler.NewCacheAdminHandler(cacheManager, invalidationManager)

	// 第八步：配置 Gin 路由及中间件
	r := gin.New()
	r.SetTrustedProxies(nil)
	r.ForwardedByClientIP = true
	allowedOrigins := viper.GetStringSlice("cors.allowed_origins")
	router.SetupRouter(r, rdb, pkglogger.Global(), allowedOrigins, authHandler, basicSSOHandler, orgHandler, tenantHandler, systemHandler, healthHandler, configHandler, ruleHandler, userConfigHandler, userConfigMgmtHandler, llmLogHandler, cronHandler, cronTaskHandler, archiveConfigHandler, archiveRuleHandler, summaryConfigHandler, externalContextHandler, auditHandler, archiveReviewHandler, summaryHandler, embedEventHandler, dashboardOverviewHandler, userNotificationHandler, cacheAdminHandler, sysFlagsResolver, operationAuditLogRepo, tenantRepo)

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
	viper.SetDefault("app.timezone", apptime.DefaultTimeZone)
	viper.SetDefault("server.mode", gin.ReleaseMode)

	// 缓存配置默认值
	viper.SetDefault("cache.enabled", true)
	viper.SetDefault("cache.default_ttl", "5m")
	viper.SetDefault("cache.hit_rate_threshold", 0.5)
	viper.SetDefault("cache.ttl.audit_todo", "3m")
	viper.SetDefault("cache.ttl.archive_list", "5m")
	viper.SetDefault("cache.ttl.process_config", "10m")
	viper.SetDefault("cache.ttl.snapshot", "5m")
	viper.SetDefault("cache.ttl.stats", "5m")
	viper.SetDefault("cache.ttl.dashboard", "2m")

	viper.SetDefault("backup.dir", "backups")
	viper.SetDefault("backup.retention_fallback_days", 30)
	viper.SetDefault("backup.dump_timeout", "45m")
	viper.SetDefault("attachment.mineru_timeout_seconds", 300)
	viper.SetDefault("workers.audit_workbench_concurrency", 2)
	viper.SetDefault("workers.audit_interactive_concurrency", 1)
	viper.SetDefault("workers.audit_background_concurrency", 1)
	viper.SetDefault("workers.audit_scheduled_concurrency", 1)
	viper.SetDefault("workers.audit_total_concurrency", 3)
	viper.SetDefault("workers.summary_interactive_concurrency", 1)
	viper.SetDefault("workers.summary_background_concurrency", 1)
	viper.SetDefault("workers.summary_scheduled_concurrency", 1)
	viper.SetDefault("workers.summary_total_concurrency", 2)
	viper.SetDefault("sso.public_base_url", "")

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
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		viper.GetString("database.dbname"),
		viper.GetString("database.sslmode"),
		apptime.Name(),
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
