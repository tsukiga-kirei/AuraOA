package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	pkglogger "oa-smart-audit/go-service/internal/pkg/logger"
)

// Logger 返回 HTTP 请求日志中间件。
// 每个请求完成后，使用全局 logger 记录请求方法、路径（含查询参数）、HTTP 状态码、耗时和客户端 IP。
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

		// 使用全局 logger 记录请求摘要
		pkglogger.Global().Info("HTTP 请求",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		)
	}
}
