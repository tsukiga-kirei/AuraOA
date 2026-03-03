package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// ActiveRoleClaim represents the currently active role in a JWT token.
type ActiveRoleClaim struct {
	ID         string  `json:"id"`
	Role       string  `json:"role"`
	TenantID   *string `json:"tenant_id"`
	TenantName *string `json:"tenant_name"`
	Label      string  `json:"label"`
}

// JWTClaims is the custom claims structure embedded in access tokens.
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

// GenerateAccessToken creates a signed access token with the given claims.
// TTL is read from config (jwt.access_token_ttl), default 2h.
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

// GenerateRefreshToken creates a signed refresh token for the given user.
// TTL is read from config (jwt.refresh_token_ttl), default 7d.
// Returns the signed token string and the JTI used.
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

// ParseToken validates and parses a token string, returning the custom JWTClaims.
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
