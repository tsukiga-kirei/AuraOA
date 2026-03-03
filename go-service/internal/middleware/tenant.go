package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
)

//TenantContext 将tenant_id 和is_system_admin 注入gin.Context 中。
//
//对于 system_admin 用户，tenant_id 是从“tenant_id”查询中读取的
//参数（可能为空）并且 is_system_admin 设置为 true。
//对于所有其他角色，tenant_id 来自 JWT ActiveRole 声明。
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
