package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	CtxKeyClaims = "claims"
)

// JWTMiddleware validates the Bearer token and injects claims into context.
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(CtxKeyClaims, claims)
		c.Next()
	}
}

// RequireRole checks that the authenticated user has one of the allowed roles.
func RequireRole(roles ...Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get(CtxKeyClaims)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		claims := val.(*JWTClaims)
		for _, r := range roles {
			if claims.Role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

// GetClaims extracts JWT claims from gin context.
func GetClaims(c *gin.Context) *JWTClaims {
	val, exists := c.Get(CtxKeyClaims)
	if !exists {
		return nil
	}
	return val.(*JWTClaims)
}
