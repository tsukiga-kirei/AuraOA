package auth

import "context"

// Role represents user roles in the system.
type Role string

const (
	RoleAdmin       Role = "admin"
	RoleTenantAdmin Role = "tenant_admin"
	RoleUser        Role = "user"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	TenantID string `json:"tenant_id" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     Role   `json:"role"`
}

// AuthService handles user authentication and JWT management.
type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (TokenResponse, error)
	ValidateToken(ctx context.Context, token string) (Claims, error)
	RefreshToken(ctx context.Context, refreshToken string) (TokenResponse, error)
}
