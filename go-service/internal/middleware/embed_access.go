package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/repository"
)

// EmbedAccess OA 嵌入页专用鉴权：校验租户级共享令牌，并将对应租户注入上下文。
// 令牌由租户管理页面生成，运行时不依赖全局 EMBED_* 环境变量。
func EmbedAccess(tenantRepo *repository.TenantRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader("X-Embed-Token"))
		if token == "" {
			response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "缺少嵌入访问令牌")
			c.Abort()
			return
		}

		sum := sha256.Sum256([]byte(token))
		tokenHash := hex.EncodeToString(sum[:])
		tenant, err := tenantRepo.FindByEmbedTokenHash(tokenHash)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "嵌入访问令牌无效")
			} else {
				response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "查询租户失败")
			}
			c.Abort()
			return
		}
		if tenant.Status != "active" {
			response.Error(c, http.StatusUnauthorized, errcode.ErrTenantNotFound, "租户不存在或已停用")
			c.Abort()
			return
		}
		if !tenant.EmbedEnabled {
			response.Error(c, http.StatusForbidden, errcode.ErrPermissionDenied, "租户未启用 OA 嵌入")
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
