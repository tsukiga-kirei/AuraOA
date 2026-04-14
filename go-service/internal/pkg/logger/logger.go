package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/lumberjack.v2"
)

// globalLogger 全局 logger 实例，通过 Init 初始化，通过 Global() 获取。
var globalLogger *zap.Logger

// tenantLoggers 租户 logger 缓存，key 为 tenantCode，value 为 *zap.Logger。
// 使用 sync.Map 保证并发安全，避免重复创建同一租户的 logger。
var tenantLoggers sync.Map

// globalLumberjack 全局日志文件轮转写入器，供租户 logger 复用以实现双写。
var globalLumberjack *lumberjack.Logger

// globalCfg 保存初始化时传入的配置，供 GetTenantLogger 创建租户 logger 时使用。
var globalCfg LogConfig

// Init 根据配置初始化全局 logger，必须在 main.go 最早调用。
// 同时输出到 stdout 和 {cfg.Dir}/app.log，支持日志等级配置。
// 若目录创建失败或 logger 构建失败，返回具体错误信息。
func Init(cfg LogConfig) error {
	// 填充缺省值，确保配置完整
	applyDefaults(&cfg)
	globalCfg = cfg

	// 确保日志根目录存在
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return fmt.Errorf("创建日志目录 %q 失败: %w", cfg.Dir, err)
	}

	// 解析日志等级
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("无效的日志等级 %q: %w", cfg.Level, err)
	}

	// 初始化 lumberjack 文件轮转写入器
	globalLumberjack = &lumberjack.Logger{
		Filename:   filepath.Join(cfg.Dir, "app.log"),
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		Compress:   cfg.Compress,
	}

	// 构建 zapcore：文件写入器 + stdout 双写
	encoderCfg := newEncoderConfig()
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(globalLumberjack),
		zap.NewAtomicLevelAt(level),
	)
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.AddSync(os.Stdout),
		zap.NewAtomicLevelAt(level),
	)

	globalLogger = zap.New(
		zapcore.NewTee(fileCore, consoleCore),
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	return nil
}

// Global 返回全局 *zap.Logger 实例（写入 logs/app.log + stdout）。
// 若 Init 尚未调用，返回 zap.NewNop() 避免 nil panic。
func Global() *zap.Logger {
	if globalLogger == nil {
		return zap.NewNop()
	}
	return globalLogger
}

// GetTenantLogger 返回指定租户的 *zap.Logger 实例。
// 内部维护 sync.Map 缓存，相同 tenantCode 复用同一实例，不会重复创建。
// 租户 logger 同时写入租户专属文件（{cfg.Dir}/tenants/{tenantCode}/tenant.log）
// 和全局文件，实现双写。
// 若租户目录创建失败，降级返回全局 logger 并记录 WARN，不 panic。
func GetTenantLogger(tenantCode string) *zap.Logger {
	// 优先从缓存中获取，避免重复创建
	if cached, ok := tenantLoggers.Load(tenantCode); ok {
		return cached.(*zap.Logger)
	}

	// 构建租户日志目录路径
	tenantDir := filepath.Join(globalCfg.Dir, "tenants", tenantCode)
	if err := os.MkdirAll(tenantDir, 0o755); err != nil {
		// 目录创建失败，降级使用全局 logger 并记录警告
		Global().Warn("创建租户日志目录失败，降级使用全局 logger",
			zap.String("tenantCode", tenantCode),
			zap.String("dir", tenantDir),
			zap.Error(err),
		)
		return Global()
	}

	// 解析日志等级（复用全局配置）
	level, err := zapcore.ParseLevel(globalCfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// 租户专属文件轮转写入器
	tenantLumberjack := &lumberjack.Logger{
		Filename:   filepath.Join(tenantDir, "tenant.log"),
		MaxSize:    globalCfg.MaxSizeMB,
		MaxBackups: globalCfg.MaxBackups,
		Compress:   globalCfg.Compress,
	}

	encoderCfg := newEncoderConfig()

	// 租户专属文件 core
	tenantFileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(tenantLumberjack),
		zap.NewAtomicLevelAt(level),
	)

	// 全局文件 core（双写，确保租户日志也出现在全局日志中）
	globalFileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(globalLumberjack),
		zap.NewAtomicLevelAt(level),
	)

	// 使用 zapcore.NewTee 同时写入租户文件和全局文件
	tenantLogger := zap.New(
		zapcore.NewTee(tenantFileCore, globalFileCore),
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(zap.String("tenantCode", tenantCode)),
	)

	// 写入缓存，若并发场景下已有其他 goroutine 先写入，则使用已缓存的实例
	actual, loaded := tenantLoggers.LoadOrStore(tenantCode, tenantLogger)
	if loaded {
		// 另一个 goroutine 已创建并缓存，同步关闭本次多余创建的 logger
		_ = tenantLogger.Sync()
		return actual.(*zap.Logger)
	}
	return tenantLogger
}

// Sync 刷新所有 logger 缓冲区，在程序退出前调用，确保日志不丢失。
func Sync() {
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
	// 遍历并刷新所有租户 logger
	tenantLoggers.Range(func(_, value any) bool {
		if l, ok := value.(*zap.Logger); ok {
			_ = l.Sync()
		}
		return true
	})
}

// applyDefaults 为 LogConfig 中未设置的字段填充默认值。
func applyDefaults(cfg *LogConfig) {
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Dir == "" {
		cfg.Dir = "logs"
	}
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 100
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = 5
	}
	if cfg.GlobalRetentionDays <= 0 {
		cfg.GlobalRetentionDays = 30
	}
}

// newEncoderConfig 返回统一的 zapcore 编码器配置。
func newEncoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "time"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	return cfg
}
