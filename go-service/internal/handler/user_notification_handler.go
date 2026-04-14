// 用户通知处理器，负责通知消息的查询、已读标记等操作。
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

// UserNotificationHandler 处理用户通知相关的 HTTP 请求。
type UserNotificationHandler struct {
	svc *service.UserNotificationService
}

// NewUserNotificationHandler 创建用户通知处理器实例。
func NewUserNotificationHandler(svc *service.UserNotificationService) *UserNotificationHandler {
	return &UserNotificationHandler{svc: svc}
}

// parseScope 从 JWT claims 中提取用户 ID 和当前角色分配 ID，用于通知的用户+角色维度隔离。
// 若 claims 缺失或解析失败，直接写入错误响应并返回 ok=false。
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

// List 获取当前用户在当前角色下的通知列表。
// GET /api/auth/notifications
// 查询参数：limit（默认 20）、offset（默认 0）、unread_only（1/true 仅返回未读）
// 返回：通知列表及未读总数。
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

// UnreadCount 获取当前用户在当前角色下的未读通知数量。
// GET /api/auth/notifications/unread-count
// 返回：{"count": int}。
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

// MarkAllRead 将当前用户在当前角色下的所有通知标记为已读。
// PUT /api/auth/notifications/read-all
// 返回：{"ok": true}。
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

// MarkRead 将指定通知标记为已读。
// PUT /api/auth/notifications/:id/read
// 路径参数：id（通知 UUID）
// 返回：{"ok": true}。
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
