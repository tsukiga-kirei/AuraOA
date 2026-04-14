package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
)

// RequireRole 返回角色权限校验中间件。
// 从 gin.Context 中读取 JWT 解析后的 claims，判断当前用户的 active_role 是否在允许的角色列表中。
// 若角色不匹配，返回 403 权限不足并中止请求；若 claims 不存在，返回 401 未认证。
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中获取 JWT claims（由 JWT 中间件注入）
		claimsVal, exists := c.Get("jwt_claims")
		if !exists {
			response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
			c.Abort()
			return
		}

		// 类型断言，确保 claims 格式正确
		claims, ok := claimsVal.(*jwtpkg.JWTClaims)
		if !ok {
			response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
			c.Abort()
			return
		}

		// 遍历允许的角色列表，匹配则放行
		for _, r := range roles {
			if claims.ActiveRole.Role == r {
				c.Next()
				return
			}
		}

		// 当前角色不在允许列表中，拒绝访问
		response.Error(c, http.StatusForbidden, errcode.ErrInsufficientPerms, "权限不足")
		c.Abort()
	}
}
