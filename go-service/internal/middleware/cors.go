package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 返回跨域资源共享（CORS）中间件。
// allowedOrigins 为允许跨域访问的来源列表，支持通配符 "*"。
// 对于浏览器发起的 OPTIONS 预检请求，直接返回 204 No Content，不进入业务处理链。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	originsStr := strings.Join(allowedOrigins, ", ")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 检查请求来源是否在允许列表中
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed && origin != "" {
			// 来源在白名单内，回显请求来源以支持携带凭证的跨域请求
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
			// 配置为全量放行
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			// 来源不在白名单，设置为已配置的来源列表，浏览器会因不匹配而拦截请求
			c.Header("Access-Control-Allow-Origin", originsStr)
		}

		// 允许的请求方法
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		// 允许的请求头，包含鉴权所需的 Authorization 字段
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		// 允许前端 JS 读取的响应头
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		// 允许跨域请求携带 Cookie 等凭证
		c.Header("Access-Control-Allow-Credentials", "true")
		// 预检结果缓存时间（秒），减少重复预检请求
		c.Header("Access-Control-Max-Age", "86400")

		// OPTIONS 预检请求直接返回，不进入后续处理链
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
