package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
)

// TenantContext 返回租户上下文注入中间件。
// 从 JWT claims 中提取租户信息并注入 gin.Context，供后续业务处理器使用。
//
// 注入规则：
//   - system_admin 角色：is_system_admin=true，tenant_id 从 query 参数读取（可为空，允许跨租户操作）。
//   - 其他角色：is_system_admin=false，tenant_id 从 JWT ActiveRole 中读取，代表该用户所属租户。
func TenantContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中获取 JWT claims（由 JWT 中间件注入）
		claimsVal, exists := c.Get("jwt_claims")
		if !exists {
			response.Error(c, http.StatusUnauthorized, 40100, "未提供认证令牌")
			c.Abort()
			return
		}

		// 类型断言，确保 claims 格式正确
		claims, ok := claimsVal.(*jwtpkg.JWTClaims)
		if !ok {
			response.Error(c, http.StatusInternalServerError, 50000, "服务器内部错误")
			c.Abort()
			return
		}

		if claims.ActiveRole.Role == "system_admin" {
			// 系统管理员可通过 query 参数指定目标租户，实现跨租户管理
			tenantID := c.Query("tenant_id")
			if tenantID != "" {
				c.Set("tenant_id", tenantID)
			}
			c.Set("is_system_admin", true)
		} else {
			// 普通租户用户，tenant_id 固定为 JWT 中记录的所属租户
			if claims.ActiveRole.TenantID != nil {
				c.Set("tenant_id", *claims.ActiveRole.TenantID)
			}
			c.Set("is_system_admin", false)
		}

		c.Next()
	}
}
