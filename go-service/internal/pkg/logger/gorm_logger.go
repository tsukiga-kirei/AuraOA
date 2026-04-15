package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger 将 GORM 的日志输出桥接到 zap，确保数据库错误（如达梦驱动报错）
// 也能写入 app.log 和 stdout，而不是直接打印到 stderr。
type GormLogger struct {
	// SlowThreshold 慢查询阈值，超过此时长的 SQL 会以 WARN 级别记录，默认 200ms
	SlowThreshold time.Duration
	// IgnoreRecordNotFoundError 是否忽略 record not found 错误（通常是正常业务逻辑，不算真正的错误）
	IgnoreRecordNotFoundError bool
}

// NewGormLogger 创建一个桥接到 zap 的 GORM logger。
// slowThreshold 为慢查询阈值，ignoreNotFound 为是否忽略 record not found 错误。
func NewGormLogger(slowThreshold time.Duration, ignoreNotFound bool) *GormLogger {
	return &GormLogger{
		SlowThreshold:             slowThreshold,
		IgnoreRecordNotFoundError: ignoreNotFound,
	}
}

// LogMode 实现 gorm/logger.Interface，返回自身（日志级别由 zap 全局配置控制）。
func (l *GormLogger) LogMode(_ gormlogger.LogLevel) gormlogger.Interface {
	return l
}

// Info 记录 GORM 的 INFO 级别日志。
func (l *GormLogger) Info(_ context.Context, msg string, args ...interface{}) {
	Global().Sugar().Infof(msg, args...)
}

// Warn 记录 GORM 的 WARN 级别日志。
func (l *GormLogger) Warn(_ context.Context, msg string, args ...interface{}) {
	Global().Sugar().Warnf(msg, args...)
}

// Error 记录 GORM 的 ERROR 级别日志，包含完整错误信息。
func (l *GormLogger) Error(_ context.Context, msg string, args ...interface{}) {
	Global().Sugar().Errorf(msg, args...)
}

// Trace 记录每条 SQL 执行情况：慢查询用 WARN，错误用 ERROR，正常用 DEBUG。
func (l *GormLogger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}

	switch {
	case err != nil:
		// record not found 通常是正常业务逻辑，根据配置决定是否忽略
		if l.IgnoreRecordNotFoundError && err == gorm.ErrRecordNotFound {
			return
		}
		// 其他错误（如达梦驱动报错 Error -6128）以 ERROR 级别记录，包含完整错误信息
		Global().Error("GORM 查询错误", append(fields, zap.Error(err))...)

	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold:
		// 慢查询以 WARN 级别记录
		Global().Warn("GORM 慢查询",
			append(fields, zap.Duration("threshold", l.SlowThreshold))...)

	default:
		// 正常查询以 DEBUG 级别记录，生产环境 LOG_LEVEL=info 时不输出
		Global().Debug("GORM 查询", fields...)
	}
}
