package middleware

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	pkglogger "auraoa/go-service/internal/pkg/logger"
)

// Logger 返回 HTTP 请求日志中间件。
// 每个请求完成后，使用全局 logger 记录请求方法、路径（敏感查询参数已脱敏）、HTTP 状态码、耗时和客户端 IP。
// 轮询类接口（任务状态查询、通知未读数、统计等）降级为 DEBUG，避免刷屏。
// log 参数保留以兼容现有调用方，内部实际使用 pkglogger.Global() 输出结构化日志。
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := redactSensitiveQuery(c.Request.URL.RawQuery)

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

		if codeVal, exists := c.Get("error_code"); exists {
			if code, ok := codeVal.(int); ok {
				fields = append(fields, zap.Int("error_code", code))
			}
		}
		if msgVal, exists := c.Get("error_message"); exists {
			if msg, ok := msgVal.(string); ok && msg != "" {
				fields = append(fields, zap.String("error_message", msg))
			}
		}

		if status >= 500 {
			pkglogger.Global().Error("HTTP 请求失败", fields...)
		} else if status >= 400 {
			pkglogger.Global().Warn("HTTP 请求客户端错误", fields...)
		} else if IsLowValuePollingPath(path) {
			pkglogger.Global().Debug("HTTP 请求", fields...)
		} else {
			pkglogger.Global().Info("HTTP 请求", fields...)
		}
	}
}

// redactSensitiveQuery 对日志中的查询参数做脱敏，请求本身仍使用原始参数。
func redactSensitiveQuery(raw string) string {
	if raw == "" {
		return ""
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return ""
	}
	for key := range values {
		switch strings.ToLower(key) {
		case "token", "access_token", "embed_token", "api_key":
			values.Set(key, "***")
		}
	}
	return values.Encode()
}
