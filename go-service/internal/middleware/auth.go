// Package middleware 提供 Gin HTTP 中间件，包括 JWT 鉴权、跨域、日志、panic 恢复、角色校验和租户上下文注入。
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

// JWT 返回 JWT 鉴权中间件。
// 校验流程：提取 Bearer Token → 解析并验证签名与有效期 → 查询 Redis 黑名单 → 将用户信息注入上下文。
// 任意步骤失败均中止请求并返回对应错误码。
func JWT(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization 头或 query 参数中提取 Token
		token := extractBearerToken(c)
		if token == "" {
			response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
			c.Abort()
			return
		}

		// 解析 JWT，验证签名与有效期
		claims, err := jwtpkg.ParseToken(token)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "认证令牌无效或已过期")
			c.Abort()
			return
		}

		// 查询 Redis 黑名单，判断该 JTI 是否已被吊销（如用户主动登出）
		blacklistKey := fmt.Sprintf("blacklist:%s", claims.JTI)
		exists, err := rdb.Exists(context.Background(), blacklistKey).Result()
		if err != nil {
			// Redis 不可用时，出于安全考虑拒绝请求
			response.Error(c, http.StatusInternalServerError, errcode.ErrRedisConn, "Redis 连接错误")
			c.Abort()
			return
		}
		if exists > 0 {
			// Token 已被加入黑名单，拒绝访问
			response.Error(c, http.StatusUnauthorized, errcode.ErrTokenRevoked, "认证令牌已失效")
			c.Abort()
			return
		}

		// 校验通过，将用户信息注入 gin.Context，供后续处理器使用
		c.Set("jwt_claims", claims)
		c.Set("user_id", claims.Sub)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// extractBearerToken 从请求中提取 Bearer Token。
// 优先读取 "Authorization: Bearer <token>" 请求头；
// 对于 SSE、WebSocket 等不便传递请求头的场景，回退到 query 参数 "token"。
func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// 兼容 SSE 或 WebSocket 等不易传递 Authorization 头的请求
	token := c.Query("token")
	if token != "" {
		return strings.TrimSpace(token)
	}

	return ""
}
