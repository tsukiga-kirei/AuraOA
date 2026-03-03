package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
)

// Recovery returns a middleware that catches panics, logs the stack trace
// via zap, and returns a 500 / 50000 error response.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				log.Error("panic recovered",
					zap.Any("error", r),
					zap.ByteString("stack", stack),
				)

				response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
				c.Abort()
			}
		}()

		c.Next()
	}
}
