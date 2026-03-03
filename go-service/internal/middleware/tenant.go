package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
)

// TenantContext injects tenant_id and is_system_admin into the gin.Context.
//
// For system_admin users the tenant_id is read from the "tenant_id" query
// parameter (may be empty) and is_system_admin is set to true.
// For all other roles the tenant_id comes from the JWT ActiveRole claim.
func TenantContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("jwt_claims")
		if !exists {
			response.Error(c, http.StatusUnauthorized, 40100, "未提供认证令牌")
			c.Abort()
			return
		}

		claims, ok := claimsVal.(*jwtpkg.JWTClaims)
		if !ok {
			response.Error(c, http.StatusInternalServerError, 50000, "服务器内部错误")
			c.Abort()
			return
		}

		if claims.ActiveRole.Role == "system_admin" {
			tenantID := c.Query("tenant_id")
			if tenantID != "" {
				c.Set("tenant_id", tenantID)
			}
			c.Set("is_system_admin", true)
		} else {
			if claims.ActiveRole.TenantID != nil {
				c.Set("tenant_id", *claims.ActiveRole.TenantID)
			}
			c.Set("is_system_admin", false)
		}

		c.Next()
	}
}
