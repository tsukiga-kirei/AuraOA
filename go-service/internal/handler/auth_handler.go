// 认证处理器，负责登录、登出、令牌刷新、角色切换及用户信息管理。
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

// AuthHandler 处理身份认证相关的 HTTP 请求。
type AuthHandler struct {
	authService *service.AuthService
	rdb         *redis.Client
}

// NewAuthHandler 创建认证处理器实例。
func NewAuthHandler(authService *service.AuthService, rdb *redis.Client) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		rdb:         rdb,
	}
}

// logoutBody 是 POST /api/auth/logout 的可选请求体，用于传递 refresh token 的 JTI。
type logoutBody struct {
	RefreshJTI string `json:"refresh_jti"`
}

// Login 用户登录，验证账号密码并签发 access/refresh token。
// POST /api/auth/login
// 请求体：LoginRequest（username、password、tenant_code）
// 返回：access_token、refresh_token 及用户角色信息。
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

// GetBootstrapStatus 查询系统是否需要进行首次初始化（无任何用户时返回 true）。
// GET /api/auth/bootstrap-status
// 返回：{"needs_bootstrap": bool}。
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

// BootstrapAdmin 创建首个系统管理员账号（仅在系统无用户时可调用）。
// POST /api/auth/bootstrap
// 请求体：BootstrapAdminRequest（username、password）
// 返回：null（成功时）。
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

// Logout 用户登出，将 access/refresh token 加入黑名单使其失效。
// POST /api/auth/logout
// 请求体（可选）：{"refresh_jti": "..."}
// 若请求体未提供 refresh_jti，则尝试从 Redis 会话中读取。
// 返回：null（成功时）。
func (h *AuthHandler) Logout(c *gin.Context) {
	// 从上下文中获取 JWT claims（由 JWT 中间件注入）
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

	// 尝试从请求体中获取 refresh_jti
	var body logoutBody
	_ = c.ShouldBindJSON(&body)

	refreshJTI := body.RefreshJTI

	// 若请求体未提供，则从 Redis 会话中查找
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

// Refresh 使用 refresh token 换取新的 access token。
// POST /api/auth/refresh
// 请求体：RefreshRequest（refresh_token）
// 返回：新的 access_token 及过期时间。
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

// SwitchRole 切换当前用户的活跃角色，重新签发携带新角色信息的 token。
// PUT /api/auth/switch-role
// 请求体：SwitchRoleRequest（role_id）
// 返回：新的 access_token 及角色信息。
func (h *AuthHandler) SwitchRole(c *gin.Context) {
	var req dto.SwitchRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	// 从上下文中获取当前用户的 JWT claims
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

// GetMenu 获取当前用户在当前角色下的菜单权限列表。
// GET /api/auth/menu
// 返回：菜单树结构，根据 active_role 动态生成。
func (h *AuthHandler) GetMenu(c *gin.Context) {
	// 从上下文中获取 JWT claims
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

	// system_admin 无租户上下文，其他角色从 active_role 中取 tenant_id
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

// mapServiceErrorToHTTP 将业务层 ServiceError 的错误码映射为对应的 HTTP 状态码。
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

// ChangePassword 修改当前登录用户的密码。
// PUT /api/auth/change-password
// 请求体：ChangePasswordRequest（old_password、new_password）
// 返回：null（成功时）。
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

// GetMe 获取当前登录用户的个人信息及角色列表。
// GET /api/auth/me
// 返回：用户基本信息、当前活跃角色、所有角色分配列表。
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

// UpdateProfile 更新当前登录用户的个人资料（显示名称等）。
// PUT /api/auth/profile
// 请求体：UpdateProfileRequest（display_name 等）
// 返回：null（成功时）。
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

// UpdateLocale 更新当前登录用户的界面语言偏好。
// PUT /api/auth/locale
// 请求体：UpdateLocaleRequest（locale，如 zh-CN / en-US）
// 返回：null（成功时）。
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
