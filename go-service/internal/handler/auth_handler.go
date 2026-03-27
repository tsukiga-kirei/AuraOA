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

//AuthHandler 处理与身份验证相关的 HTTP 请求。
type AuthHandler struct {
	authService *service.AuthService
	rdb         *redis.Client
}

//NewAuthHandler 创建一个新的 AuthHandler 实例。
func NewAuthHandler(authService *service.AuthService, rdb *redis.Client) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		rdb:         rdb,
	}
}

//logoutBody 是 POST /api/auth/logout 的可选请求正文。
type logoutBody struct {
	RefreshJTI string `json:"refresh_jti"`
}

//登录句柄 POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	resp, err := h.authService.Login(&req, c.ClientIP(), c.GetHeader("User-Agent"))
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

// GetBootstrapStatus GET /api/auth/bootstrap-status — 是否需要进行首次初始化（无用户）。
func (h *AuthHandler) GetBootstrapStatus(c *gin.Context) {
	resp, err := h.authService.BootstrapStatus()
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

// BootstrapAdmin POST /api/auth/bootstrap — 创建首个系统管理员（仅零用户时）。
func (h *AuthHandler) BootstrapAdmin(c *gin.Context) {
	var req dto.BootstrapAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.authService.BootstrapAdmin(&req); err != nil {
		httpStatus := mapServiceErrorToHTTP(err)
		if svcErr, ok := err.(*service.ServiceError); ok {
			response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}
	response.Success(c, nil)
}

//注销处理 POST /api/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	//从上下文中获取jwt_claims（由JWT中间件设置）
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

	//尝试从请求体中获取refresh_jti
	var body logoutBody
	_ = c.ShouldBindJSON(&body)

	refreshJTI := body.RefreshJTI

	//如果正文中未提供，请尝试从 Redis 会话中获取
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

//刷新句柄 POST /api/auth/refresh
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

//SwitchRole 处理 PUT /api/auth/switch-role
func (h *AuthHandler) SwitchRole(c *gin.Context) {
	var req dto.SwitchRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	//从上下文中获取 user_id 和 jwt_claims
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

//GetMenu 处理 GET /api/auth/menu
func (h *AuthHandler) GetMenu(c *gin.Context) {
	//从上下文中获取 jwt_claims
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

	//确定tenantID：从ActiveRole或system_admin的查询参数
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
//Helper：将 ServiceError 代码映射到 HTTP 状态
// ---------------------------------------------------------------------------

//mapServiceErrorToHTTP 将 ServiceError 的业务代码映射到 HTTP 状态代码。
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
	case code >= 50200 && code <= 50299:
		return http.StatusBadGateway
	case code >= 50300 && code <= 50399:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// ChangePassword handles PUT /api/auth/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

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

	if err := h.authService.ChangePassword(userID, &req); err != nil {
		httpStatus := mapServiceErrorToHTTP(err)
		if svcErr, ok := err.(*service.ServiceError); ok {
			response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	response.Success(c, nil)
}

// GetMe handles GET /api/auth/me
func (h *AuthHandler) GetMe(c *gin.Context) {
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

	resp, err := h.authService.GetMe(userID, claims.ActiveRole, claims.AllRoleIDs)
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

// UpdateProfile handles PUT /api/auth/profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

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

	if err := h.authService.UpdateProfile(userID, &req); err != nil {
		httpStatus := mapServiceErrorToHTTP(err)
		if svcErr, ok := err.(*service.ServiceError); ok {
			response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	response.Success(c, nil)
}

// UpdateLocale handles PUT /api/auth/locale
func (h *AuthHandler) UpdateLocale(c *gin.Context) {
	var req dto.UpdateLocaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

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

	if err := h.authService.UpdateLocale(userID, req.Locale); err != nil {
		httpStatus := mapServiceErrorToHTTP(err)
		if svcErr, ok := err.(*service.ServiceError); ok {
			response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
			return
		}
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}

	response.Success(c, nil)
}
