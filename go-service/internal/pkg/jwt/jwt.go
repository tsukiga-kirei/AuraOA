// Package jwt 提供 JWT 访问令牌和刷新令牌的签发与校验功能。
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// ActiveRoleClaim 表示 JWT 中当前激活的角色信息。
type ActiveRoleClaim struct {
	ID         string  `json:"id"`
	Role       string  `json:"role"`
	TenantID   *string `json:"tenant_id"`
	TenantName *string `json:"tenant_name"`
	Label      string  `json:"label"`
}

// JWTClaims 访问令牌的自定义声明结构，嵌入标准 RegisteredClaims。
type JWTClaims struct {
	Sub         string          `json:"sub"`
	Username    string          `json:"username"`
	DisplayName string          `json:"display_name"`
	ActiveRole  ActiveRoleClaim `json:"active_role"`
	Permissions []string        `json:"permissions"`
	AllRoleIDs  []string        `json:"all_role_ids"`
	JTI         string          `json:"jti"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 签发访问令牌。
// TTL 从配置项 jwt.access_token_ttl 读取，默认 2 小时。
func GenerateAccessToken(claims *JWTClaims) (string, error) {
	secret := viper.GetString("jwt.secret")
	ttl := viper.GetDuration("jwt.access_token_ttl")
	if ttl == 0 {
		ttl = 2 * time.Hour
	}

	jti := uuid.New().String()
	claims.JTI = jti

	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Subject:   claims.Sub,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		ID:        jti,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken 签发刷新令牌，返回签名字符串和 JTI。
// TTL 从配置项 jwt.refresh_token_ttl 读取，默认 7 天。
// 若传入非空 jti 则复用，否则自动生成新 UUID。
func GenerateRefreshToken(userID string, jti string) (string, string, error) {
	secret := viper.GetString("jwt.secret")
	ttl := viper.GetDuration("jwt.refresh_token_ttl")
	if ttl == 0 {
		ttl = 7 * 24 * time.Hour
	}

	if jti == "" {
		jti = uuid.New().String()
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		ID:        jti,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}
	return signed, jti, nil
}

// ParseToken 验证并解析访问令牌，返回自定义 JWTClaims。
func ParseToken(tokenString string) (*JWTClaims, error) {
	secret := viper.GetString("jwt.secret")

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名算法: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("令牌声明无效")
	}

	return claims, nil
}

// ParseRefreshToken 验证并解析刷新令牌，返回仅含 Sub 和 JTI 的 JWTClaims。
func ParseRefreshToken(tokenString string) (*JWTClaims, error) {
	secret := viper.GetString("jwt.secret")

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名算法: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	rc, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("刷新令牌声明无效")
	}

	return &JWTClaims{
		Sub:              rc.Subject,
		JTI:              rc.ID,
		RegisteredClaims: *rc,
	}, nil
}
