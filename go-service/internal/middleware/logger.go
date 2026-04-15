package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	pkglogger "oa-smart-audit/go-service/internal/pkg/logger"
)

// Logger 返回 HTTP 请求日志中间件。
// 每个请求完成后，使用全局 logger 记录请求方法、路径（含查询参数）、HTTP 状态码、耗时和客户端 IP。
// 轮询类接口（任务状态查询、通知未读数、统计等）降级为 DEBUG，避免刷屏。
// log 参数保留以兼容现有调用方，内部实际使用 pkglogger.Global() 输出结构化日志。
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 先执行后续处理链，再记录日志，确保状态码已写入
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		// 若存在查询参数，拼接到路径后便于排查问题
		if query != "" {
			path = path + "?" + query
		}

		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		}

		// 轮询类接口降级为 DEBUG：生产环境 LOG_LEVEL=info 时不输出，调试时改为 debug 才可见
		if isPollingPath(path) {
			pkglogger.Global().Debug("HTTP 请求", fields...)
		} else {
			pkglogger.Global().Info("HTTP 请求", fields...)
		}
	}
}

// isPollingPath 判断是否为前端轮询类接口，这类接口频率高但价值低，降级为 DEBUG。
func isPollingPath(path string) bool {
	pollingPrefixes := []string{
		"/api/audit/jobs/",
		"/api/archive/jobs/",
		"/api/auth/notifications/unread-count",
		"/api/audit/stats",
		"/api/archive/stats",
	}
	for _, prefix := range pollingPrefixes {
		if strings.HasPrefix(path, prefix) || strings.Contains(path, prefix) {
			return true
		}
	}
	return false
}
