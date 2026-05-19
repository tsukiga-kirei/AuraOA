package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/repository"
)

// EmbedAccess OA 嵌入页专用鉴权：校验共享令牌 + 租户编码，无需用户 JWT 登录。
// 配置项（config.yaml）：
//   embed.access_token — 与 Nuxt 服务端代理使用的令牌一致
//   embed.tenant_code  — 默认租户 code（可被 X-Tenant-Code 覆盖）
func EmbedAccess(tenantRepo *repository.TenantRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := viper.GetString("embed.access_token")
		if expected == "" {
			response.Error(c, http.StatusForbidden, errcode.ErrPermissionDenied, "嵌入审核未启用")
			c.Abort()
			return
		}

		token := c.GetHeader("X-Embed-Token")
		if token == "" {
			response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "缺少嵌入访问令牌")
			c.Abort()
			return
		}
		if token != expected {
			response.Error(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "嵌入访问令牌无效")
			c.Abort()
			return
		}

		tenantCode := c.GetHeader("X-Tenant-Code")
		if tenantCode == "" {
			tenantCode = viper.GetString("embed.tenant_code")
		}
		if tenantCode == "" {
			response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "缺少租户编码")
			c.Abort()
			return
		}

		tenant, err := tenantRepo.FindByCode(tenantCode)
		if err != nil || tenant == nil || tenant.Status != "active" {
			response.Error(c, http.StatusBadRequest, errcode.ErrTenantNotFound, "租户不存在或已停用")
			c.Abort()
			return
		}

		if tenant.AdminUserID == nil {
			response.Error(c, http.StatusForbidden, errcode.ErrPermissionDenied, "租户未配置管理员，无法执行嵌入审核")
			c.Abort()
			return
		}

		c.Set("tenant_id", tenant.ID.String())
		c.Set("embed_user_id", *tenant.AdminUserID)
		c.Set("embed_mode", true)
		c.Set("is_system_admin", false)
		c.Next()
	}
}
