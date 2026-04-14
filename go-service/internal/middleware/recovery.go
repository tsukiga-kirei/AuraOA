package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"oa-smart-audit/go-service/internal/pkg/errcode"
	pkglogger "oa-smart-audit/go-service/internal/pkg/logger"
	"oa-smart-audit/go-service/internal/pkg/response"
)

// Recovery 返回 panic 恢复中间件。
// 当请求处理过程中发生 panic 时，捕获异常并通过全局 logger 记录错误信息和完整堆栈，
// 然后向客户端返回 500 内部服务器错误，避免进程崩溃。
// log 参数保留以兼容现有调用方，内部实际使用 pkglogger.Global() 输出日志。
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 获取完整堆栈信息，便于定位 panic 发生位置
				stack := debug.Stack()
				// 使用全局 logger 以 ERROR 级别记录 panic 详情和堆栈
				pkglogger.Global().Error("panic 已恢复",
					zap.Any("error", r),
					zap.ByteString("stack", stack),
				)

				// 向客户端返回统一的 500 错误响应
				response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
				c.Abort()
			}
		}()

		c.Next()
	}
}
