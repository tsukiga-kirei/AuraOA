package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// UserNotificationHandler /api/auth/notifications*
type UserNotificationHandler struct {
	svc *service.UserNotificationService
}

// NewUserNotificationHandler 创建处理器。
func NewUserNotificationHandler(svc *service.UserNotificationService) *UserNotificationHandler {
	return &UserNotificationHandler{svc: svc}
}

func (h *UserNotificationHandler) parseScope(c *gin.Context) (userID uuid.UUID, roleAssignmentID uuid.UUID, ok bool) {
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return uuid.Nil, uuid.Nil, false
	}
	claims, okClaims := claimsVal.(*jwtpkg.JWTClaims)
	if !okClaims {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return uuid.Nil, uuid.Nil, false
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "用户标识无效")
		return uuid.Nil, uuid.Nil, false
	}
	if claims.ActiveRole.ID == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "缺少当前角色上下文")
		return uuid.Nil, uuid.Nil, false
	}
	roleAssignmentID, err = uuid.Parse(claims.ActiveRole.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "角色分配标识无效")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, roleAssignmentID, true
}

// List GET /api/auth/notifications
func (h *UserNotificationHandler) List(c *gin.Context) {
	userID, assignmentID, ok := h.parseScope(c)
	if !ok {
		return
	}
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	unreadOnly := c.Query("unread_only") == "1" || c.Query("unread_only") == "true"

	data, err := h.svc.List(userID, assignmentID, limit, offset, unreadOnly)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// UnreadCount GET /api/auth/notifications/unread-count
func (h *UserNotificationHandler) UnreadCount(c *gin.Context) {
	userID, assignmentID, ok := h.parseScope(c)
	if !ok {
		return
	}
	data, err := h.svc.UnreadCount(userID, assignmentID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// MarkAllRead PUT /api/auth/notifications/read-all
func (h *UserNotificationHandler) MarkAllRead(c *gin.Context) {
	userID, assignmentID, ok := h.parseScope(c)
	if !ok {
		return
	}
	if err := h.svc.MarkAllRead(userID, assignmentID); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// MarkRead PUT /api/auth/notifications/:id/read
func (h *UserNotificationHandler) MarkRead(c *gin.Context) {
	userID, assignmentID, ok := h.parseScope(c)
	if !ok {
		return
	}
	nid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "通知 ID 无效")
		return
	}
	if err := h.svc.MarkRead(userID, assignmentID, nid); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
