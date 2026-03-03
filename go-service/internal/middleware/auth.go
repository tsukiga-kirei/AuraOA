package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
)

// JWT returns a Gin middleware that validates Bearer tokens, checks the Redis
// blacklist, and injects jwt_claims / user_id / username into the context.
func JWT(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract Bearer token from Authorization header
		token := extractBearerToken(c)
		if token == "" {
			response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
			c.Abort()
			return
		}

		// Parse and validate the token
		claims, err := jwtpkg.ParseToken(token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "认证令牌无效或已过期")
			c.Abort()
			return
		}

		// Check Redis blacklist for JTI
		blacklistKey := fmt.Sprintf("blacklist:%s", claims.JTI)
		exists, err := rdb.Exists(context.Background(), blacklistKey).Result()
		if err != nil {
			// If Redis is unreachable, reject for safety
			response.Error(c, http.StatusInternalServerError, errcode.ErrRedisConn, "Redis 连接错误")
			c.Abort()
			return
		}
		if exists > 0 {
			response.Error(c, http.StatusUnauthorized, errcode.ErrTokenRevoked, "认证令牌已失效")
			c.Abort()
			return
		}

		// Inject claims into context
		c.Set("jwt_claims", claims)
		c.Set("user_id", claims.Sub)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// extractBearerToken pulls the token from the "Authorization: Bearer <token>" header.
func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
