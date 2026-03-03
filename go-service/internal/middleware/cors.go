package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that sets Cross-Origin Resource Sharing headers.
// allowedOrigins is the list of origins permitted to access the API.
// An OPTIONS preflight request is answered with 204 No Content.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	originsStr := strings.Join(allowedOrigins, ", ")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if the request origin is in the allowed list
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
			// Not allowed — still set the header to the first configured
			// origin so the browser sees a mismatch and blocks the request.
			c.Header("Access-Control-Allow-Origin", originsStr)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
