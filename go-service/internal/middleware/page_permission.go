package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/pkg/errcode"
	jwtpkg "auraoa/go-service/internal/pkg/jwt"
	"auraoa/go-service/internal/pkg/response"
)

type pagePermissionChecker interface {
	UserHasPagePermission(c *gin.Context, userID uuid.UUID, page string) (bool, error)
}

// RequirePagePermission 返回组织角色页面权限校验中间件。
// 必须放在 JWT 与 TenantContext 之后，避免仅依赖前端菜单隐藏保护业务接口。
func RequirePagePermission(checker pagePermissionChecker, page string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("jwt_claims")
		claims, ok := claimsVal.(*jwtpkg.JWTClaims)
		if !exists || !ok {
			response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
			c.Abort()
			return
		}
		userID, err := uuid.Parse(claims.Sub)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "认证令牌无效")
			c.Abort()
			return
		}
		allowed, err := checker.UserHasPagePermission(c, userID, page)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "查询页面权限失败")
			c.Abort()
			return
		}
		if !allowed {
			response.Error(c, http.StatusForbidden, errcode.ErrInsufficientPerms, "权限不足")
			c.Abort()
			return
		}
		c.Next()
	}
}
