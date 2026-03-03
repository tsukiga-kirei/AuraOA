package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
)

// RequireRole returns a middleware that checks whether the caller's
// active_role (from JWT claims) is one of the allowed roles.
// If not, it aborts with 403 / 40300.
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
