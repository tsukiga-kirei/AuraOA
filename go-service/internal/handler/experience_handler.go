package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/service"
)

// ExperienceHandler 提供嵌入互动与租户管理员反馈接口。
type ExperienceHandler struct{ service *service.ExperienceService }

// NewExperienceHandler 创建体验反馈处理器。
func NewExperienceHandler(s *service.ExperienceService) *ExperienceHandler {
	return &ExperienceHandler{service: s}
}

func experienceID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "审核 ID 格式错误")
		return uuid.Nil, false
	}
	return id, true
}
func experienceQuery(c *gin.Context) (dto.ExperienceQuery, bool) {
	var q dto.ExperienceQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "筛选参数无效")
		return q, false
	}
	return q, true
}

// Interactions GET /api/embed/audits/:id/interactions，返回分页评论与评价。
func (h *ExperienceHandler) Interactions(c *gin.Context) {
	id, ok := experienceID(c)
	if !ok {
		return
	}
	q, ok := experienceQuery(c)
	if !ok {
		return
	}
	data, err := h.service.Interactions(c, id, q)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// Comment POST /api/embed/audits/:id/comments，追加评论。
func (h *ExperienceHandler) Comment(c *gin.Context) {
	id, ok := experienceID(c)
	if !ok {
		return
	}
	var req dto.AuditCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "评论参数无效")
		return
	}
	data, err := h.service.Comment(c, id, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// ListAudit GET /api/tenant/experience/audit，仅列出有互动的审核。
func (h *ExperienceHandler) ListAudit(c *gin.Context) { h.list(c, "audit") }

// ListAgents GET /api/tenant/experience/agents，仅列出有评价的助手消息。
func (h *ExperienceHandler) ListAgents(c *gin.Context) { h.list(c, "agents") }
func (h *ExperienceHandler) list(c *gin.Context, kind string) {
	q, ok := experienceQuery(c)
	if !ok {
		return
	}
	data, err := h.service.List(c, kind, q)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// AuditDetail GET /api/tenant/experience/audit/:id，返回原审核结果和分页评论。
func (h *ExperienceHandler) AuditDetail(c *gin.Context) {
	id, ok := experienceID(c)
	if !ok {
		return
	}
	q, ok := experienceQuery(c)
	if !ok {
		return
	}
	data, err := h.service.AuditDetail(c, id, q)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// UpdateComment POST /api/embed/audits/:id/comments/:comment_id/update，编辑本人评论及满意度。
func (h *ExperienceHandler) UpdateComment(c *gin.Context) { h.manageComment(c, false) }

// DeleteComment POST /api/embed/audits/:id/comments/:comment_id/delete，删除本人评论。
func (h *ExperienceHandler) DeleteComment(c *gin.Context) { h.manageComment(c, true) }
func (h *ExperienceHandler) manageComment(c *gin.Context, remove bool) {
	id, ok := experienceID(c)
	if !ok {
		return
	}
	commentID, err := uuid.Parse(c.Param("comment_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "评论 ID 格式错误")
		return
	}
	var req *dto.AuditCommentRequest
	if !remove {
		req = &dto.AuditCommentRequest{}
		if err := c.ShouldBindJSON(req); err != nil {
			response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "评论参数无效")
			return
		}
	}
	if err := h.service.ManageComment(c, id, commentID, req); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}
