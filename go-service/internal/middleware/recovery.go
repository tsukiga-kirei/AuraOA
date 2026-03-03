package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
)

//恢复返回一个捕获恐慌的中间件，记录堆栈跟踪
//通过 zap，并返回 500 / 50000 错误响应。
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
