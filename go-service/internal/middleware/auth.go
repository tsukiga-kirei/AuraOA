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

//JWT 返回一个 Gin 中间件，用于验证 Bearer 令牌、检查 Redis
//黑名单，并将 jwt_claims / user_id / username 注入上下文中。
func JWT(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		//从授权标头中提取承载令牌
		token := extractBearerToken(c)
		if token == "" {
			response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
			c.Abort()
			return
		}

		//解析并验证令牌
		claims, err := jwtpkg.ParseToken(token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "认证令牌无效或已过期")
			c.Abort()
			return
		}

		//检查 JTI 的 Redis 黑名单
		blacklistKey := fmt.Sprintf("blacklist:%s", claims.JTI)
		exists, err := rdb.Exists(context.Background(), blacklistKey).Result()
		if err != nil {
			//如果 Redis 无法访问，为了安全而拒绝
			response.Error(c, http.StatusInternalServerError, errcode.ErrRedisConn, "Redis 连接错误")
			c.Abort()
			return
		}
		if exists > 0 {
			response.Error(c, http.StatusUnauthorized, errcode.ErrTokenRevoked, "认证令牌已失效")
			c.Abort()
			return
		}

		//将声明注入上下文
		c.Set("jwt_claims", claims)
		c.Set("user_id", claims.Sub)
		c.Set("username", claims.Username)
		c.Next()
	}
}

//extractBearerToken 从“Authorization: Bearer <token>”标头中提取令牌。
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
