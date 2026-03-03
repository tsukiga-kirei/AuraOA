package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService *service.AuthService
	rdb         *redis.Client
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService *service.AuthService, rdb *redis.Client) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		rdb:         rdb,
	}
}

// logoutBody is the optional request body for POST /api/auth/logout.
type logoutBody struct {
	RefreshJTI string `json:"refresh_jti"`
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	resp, err := h.authService.Login(&req)
	if err != nil {
		httpStatus := mapServiceErrorToHTTP(err)
		if svcErr, ok := err.(*service.ServiceError); ok {
			response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	response.Success(c, resp)
}

// Logout handles POST /api/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get jwt_claims from context (set by JWT middleware)
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	// Try to get refresh_jti from request body
	var body logoutBody
	_ = c.ShouldBindJSON(&body)

	refreshJTI := body.RefreshJTI

	// If not provided in body, try to get from Redis session
	if refreshJTI == "" {
		sessionKey := fmt.Sprintf("session:%s", claims.Sub)
		sessionJSON, err := h.rdb.Get(context.Background(), sessionKey).Result()
		if err == nil && sessionJSON != "" {
			var sessionData map[string]interface{}
			if jsonErr := json.Unmarshal([]byte(sessionJSON), &sessionData); jsonErr == nil {
				if jti, ok := sessionData["refresh_jti"].(string); ok {
					refreshJTI = jti
				}
			}
		}
	}

	logoutReq := &service.LogoutRequest{
		AccessJTI:  claims.JTI,
		RefreshJTI: refreshJTI,
		UserID:     claims.Sub,
	}

	if err := h.authService.Logout(logoutReq); err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	response.Success(c, nil)
}

// Refresh handles POST /api/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	resp, err := h.authService.Refresh(&req)
	if err != nil {
		httpStatus := mapServiceErrorToHTTP(err)
		if svcErr, ok := err.(*service.ServiceError); ok {
			response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	response.Success(c, resp)
}

// SwitchRole handles PUT /api/auth/switch-role
func (h *AuthHandler) SwitchRole(c *gin.Context) {
	var req dto.SwitchRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	// Get user_id and jwt_claims from context
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "认证令牌无效或已过期")
		return
	}

	resp, err := h.authService.SwitchRole(userID, req.RoleID, claims.JTI)
	if err != nil {
		httpStatus := mapServiceErrorToHTTP(err)
		if svcErr, ok := err.(*service.ServiceError); ok {
			response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	response.Success(c, resp)
}

// GetMenu handles GET /api/auth/menu
func (h *AuthHandler) GetMenu(c *gin.Context) {
	// Get jwt_claims from context
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	// Determine tenantID: from ActiveRole or query param for system_admin
	tenantID := ""
	if claims.ActiveRole.TenantID != nil {
		tenantID = *claims.ActiveRole.TenantID
	}

	resp, err := h.authService.GetMenu(claims.ActiveRole, claims.Sub, tenantID)
	if err != nil {
		httpStatus := mapServiceErrorToHTTP(err)
		if svcErr, ok := err.(*service.ServiceError); ok {
			response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	response.Success(c, resp)
}

// ---------------------------------------------------------------------------
// Helper: map ServiceError code to HTTP status
// ---------------------------------------------------------------------------

// mapServiceErrorToHTTP maps a ServiceError's business code to an HTTP status code.
func mapServiceErrorToHTTP(err error) int {
	svcErr, ok := err.(*service.ServiceError)
	if !ok {
		return http.StatusInternalServerError
	}

	code := svcErr.Code

	switch {
	case code == errcode.ErrParamValidation:
		return http.StatusBadRequest
	case code >= 40100 && code <= 40199:
		return http.StatusUnauthorized
	case code >= 40300 && code <= 40399:
		return http.StatusForbidden
	case code >= 40400 && code <= 40499:
		return http.StatusNotFound
	case code >= 40900 && code <= 40999:
		return http.StatusConflict
	case code >= 50000 && code <= 50099:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
