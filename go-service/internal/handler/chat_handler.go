package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/pkg/errcode"
	jwtpkg "auraoa/go-service/internal/pkg/jwt"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/service"
)

// ChatHandler 处理业务用户 AI 对话与会话相关的 HTTP 与 SSE 请求
type ChatHandler struct {
	sessionService *service.ChatSessionService
	runtimeService *service.AgentRuntimeService
}

// NewChatHandler 创建 ChatHandler 实例
func NewChatHandler(
	sessionService *service.ChatSessionService,
	runtimeService *service.AgentRuntimeService,
) *ChatHandler {
	return &ChatHandler{
		sessionService: sessionService,
		runtimeService: runtimeService,
	}
}

// GetEffectiveAgents 获取当前用户可用的有效智能体列表
// GET /api/chat/agents
func (h *ChatHandler) GetEffectiveAgents(c *gin.Context) {
	tenantID, userID, _, err := extractUserAndTenant(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "认证失效")
		return
	}

	agents, err := h.sessionService.GetEffectiveAgents(c.Request.Context(), tenantID, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, agents)
}

// ListSessions 分页查询用户在当前租户下的会话列表
// GET /api/chat/sessions
func (h *ChatHandler) ListSessions(c *gin.Context) {
	tenantID, userID, _, err := extractUserAndTenant(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "认证失效")
		return
	}

	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	res, err := h.sessionService.ListSessions(c.Request.Context(), tenantID, userID, keyword, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, res)
}

// CreateSession 创建新会话
// POST /api/chat/sessions
func (h *ChatHandler) CreateSession(c *gin.Context) {
	tenantID, userID, _, err := extractUserAndTenant(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "认证失效")
		return
	}

	var req dto.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "请求参数校验失败")
		return
	}

	res, err := h.sessionService.CreateSession(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, res)
}

// GetSessionDetail 获取会话详情与历史消息
// GET /api/chat/sessions/:id
func (h *ChatHandler) GetSessionDetail(c *gin.Context) {
	tenantID, userID, _, err := extractUserAndTenant(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "认证失效")
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "会话 ID 格式错误")
		return
	}

	res, err := h.sessionService.GetSessionDetail(c.Request.Context(), tenantID, userID, sessionID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, res)
}

// UpdateSession 更新会话（重命名、置顶等）
// PATCH /api/chat/sessions/:id
func (h *ChatHandler) UpdateSession(c *gin.Context) {
	tenantID, userID, _, err := extractUserAndTenant(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "认证失效")
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "会话 ID 格式错误")
		return
	}

	var req dto.UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "请求参数校验失败")
		return
	}

	if err := h.sessionService.UpdateSession(c.Request.Context(), tenantID, userID, sessionID, &req); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}

// DeleteSession 删除会话
// DELETE /api/chat/sessions/:id
func (h *ChatHandler) DeleteSession(c *gin.Context) {
	tenantID, userID, _, err := extractUserAndTenant(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "认证失效")
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "会话 ID 格式错误")
		return
	}

	if err := h.sessionService.DeleteSession(c.Request.Context(), tenantID, userID, sessionID); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// StreamMessage 发送消息并以 SSE 事件流返回智能体思考过程、工具调用与 Markdown 回答
// POST /api/chat/sessions/:id/messages/stream
func (h *ChatHandler) StreamMessage(c *gin.Context) {
	tenantID, userID, username, err := extractUserAndTenant(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "认证失效")
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "会话 ID 格式错误")
		return
	}

	var req dto.SendMessageStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "请输入有效的消息内容")
		return
	}

	// 设置 SSE 响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	sink := func(event string, data interface{}) error {
		dataBytes, _ := json.Marshal(data)
		_, wErr := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(dataBytes))
		if wErr != nil {
			return wErr
		}
		c.Writer.Flush()
		return nil
	}

	_ = h.runtimeService.ExecuteMessageStream(c, tenantID, userID, username, sessionID, req.Content, sink)
}

func extractUserAndTenant(c *gin.Context) (tenantID, userID uuid.UUID, username string, err error) {
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("missing claims")
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("invalid claims")
	}

	uID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("invalid user id in claims: %w", err)
	}

	tidVal, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("missing tenant_id")
	}
	tidStr, ok := tidVal.(string)
	if !ok {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("invalid tenant_id")
	}
	tID, err := uuid.Parse(tidStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, "", err
	}

	return tID, uID, claims.Username, nil
}
