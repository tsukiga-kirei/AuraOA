package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
)

//RequireRole 返回一个中间件，用于检查调用者是否
//active_role（来自 JWT 声明）是允许的角色之一。
//如果没有，则会以 403 / 40300 中止。
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("jwt_claims")
		if !exists {
			response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
			c.Abort()
			return
		}

		claims, ok := claimsVal.(*jwtpkg.JWTClaims)
		if !ok {
			response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
			c.Abort()
			return
		}

		for _, r := range roles {
			if claims.ActiveRole.Role == r {
				c.Next()
				return
			}
		}

		response.Error(c, http.StatusForbidden, errcode.ErrInsufficientPerms, "权限不足")
		c.Abort()
	}
}
