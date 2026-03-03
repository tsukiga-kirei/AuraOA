package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//CORS 返回一个设置跨源资源共享标头的中间件。
//allowedOrigins 是允许访问 API 的源列表。
//OPTIONS 预检请求得到 204 No Content 应答。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	originsStr := strings.Join(allowedOrigins, ", ")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		//检查请求来源是否在允许列表中
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			//不允许 - 仍将标头设置为第一个配置的
			//来源，因此浏览器会发现不匹配并阻止请求。
			c.Header("Access-Control-Allow-Origin", originsStr)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		//处理预检
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
