package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

//ActiveRoleClaim 代表 JWT 令牌中当前活动的角色。
type ActiveRoleClaim struct {
	ID         string  `json:"id"`
	Role       string  `json:"role"`
	TenantID   *string `json:"tenant_id"`
	TenantName *string `json:"tenant_name"`
	Label      string  `json:"label"`
}

//JWTClaims 是嵌入在访问令牌中的自定义声明结构。
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

//GenerateAccessToken 使用给定的声明创建一个签名的访问令牌。
//TTL从配置中读取（jwt.access_token_ttl），默认2h。
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

//GenerateRefreshToken 为给定用户创建一个签名的刷新令牌。
//TTL从配置中读取（jwt.refresh_token_ttl），默认7d。
//返回签名的令牌字符串和使用的 JTI。
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

//ParseToken 验证并解析令牌字符串，返回自定义 JWTClaims。
func ParseToken(tokenString string) (*JWTClaims, error) {
	secret := viper.GetString("jwt.secret")

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
