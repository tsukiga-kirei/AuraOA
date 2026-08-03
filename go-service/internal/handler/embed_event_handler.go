package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/service"
)

// EmbedEventHandler 处理 OA 父页面提交的无界面刷新事件。
type EmbedEventHandler struct {
	refreshService *service.EmbedRefreshService
}

// NewEmbedEventHandler 创建嵌入事件处理器。
func NewEmbedEventHandler(refreshService *service.EmbedRefreshService) *EmbedEventHandler {
	return &EmbedEventHandler{refreshService: refreshService}
}

// Schedule 接收 OA 保存/提交事件，只安排后台检查或 requestid 解析，不等待 AI。
// POST /api/embed/events
func (h *EmbedEventHandler) Schedule(c *gin.Context) {
	var req service.EmbedRefreshEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "租户ID无效")
		return
	}
	embedUser, ok := c.Get("embed_user_id")
	if !ok {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "嵌入用户无效")
		return
	}
	userID, ok := embedUser.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "嵌入用户无效")
		return
	}

	result, err := h.refreshService.ScheduleEvent(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidEmbedRefreshAction) ||
			errors.Is(err, service.ErrInvalidEmbedRefreshContext) {
			response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, err.Error())
			return
		}
		response.Error(c, http.StatusServiceUnavailable, errcode.ErrRedisConn, "后台刷新事件暂时无法接收")
		return
	}
	c.JSON(http.StatusAccepted, response.Response{
		Code:    0,
		Message: "accepted",
		Data:    result,
	})
}
