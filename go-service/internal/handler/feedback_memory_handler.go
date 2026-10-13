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

// FeedbackMemoryHandler 提供租户记忆库的流程反馈接口。
type FeedbackMemoryHandler struct {
	service *service.FeedbackMemoryService
}

// NewFeedbackMemoryHandler 创建反馈记忆处理器。
func NewFeedbackMemoryHandler(s *service.FeedbackMemoryService) *FeedbackMemoryHandler {
	return &FeedbackMemoryHandler{service: s}
}

func memoryScope(c *gin.Context) (dto.FeedbackMemoryScope, bool) {
	var q dto.FeedbackMemoryScope
	if err := c.ShouldBindQuery(&q); err != nil {
		memoryBadRequest(c)
		return q, false
	}
	return q, true
}
func memoryBadRequest(c *gin.Context) {
	response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "反馈记忆参数无效")
}
func memoryResult(c *gin.Context, out interface{}, err error) {
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, out)
}
func memoryID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		memoryBadRequest(c)
		return id, false
	}
	return id, true
}

// Configs GET /api/tenant/feedback-memory/configs?module=audit，列出流程类型。
func (h *FeedbackMemoryHandler) Configs(c *gin.Context) {
	out, err := h.service.Configs(c, c.Query("module"))
	memoryResult(c, out, err)
}

// Workspace GET /api/tenant/feedback-memory，返回素材、当前发布版与只读生成提示词。
func (h *FeedbackMemoryHandler) Workspace(c *gin.Context) {
	q, ok := memoryScope(c)
	if !ok {
		return
	}
	out, err := h.service.Workspace(c, q)
	memoryResult(c, out, err)
}

// Comments GET /api/tenant/feedback-memory/comments，返回同流程的分页原始评论。
func (h *FeedbackMemoryHandler) Comments(c *gin.Context) {
	scope, ok := memoryScope(c)
	if !ok {
		return
	}
	q, ok := experienceQuery(c)
	if !ok {
		return
	}
	out, err := h.service.Comments(c, scope, q)
	memoryResult(c, out, err)
}

// ImportComments POST /api/tenant/feedback-memory/import，批量导入评论快照。
func (h *FeedbackMemoryHandler) ImportComments(c *gin.Context) {
	var req dto.FeedbackMemoryImportRequest
	if c.ShouldBindJSON(&req) != nil {
		memoryBadRequest(c)
		return
	}
	memoryResult(c, gin.H{"saved": true}, h.service.ImportComments(c, req))
}

// SaveEntry POST /entries 或 PUT /entries/:id，新增手工条件或修改素材。
func (h *FeedbackMemoryHandler) SaveEntry(c *gin.Context) {
	var req dto.FeedbackMemoryEntryRequest
	if c.ShouldBindJSON(&req) != nil {
		memoryBadRequest(c)
		return
	}
	id := uuid.Nil
	if c.Param("id") != "" {
		var ok bool
		id, ok = memoryID(c)
		if !ok {
			return
		}
	}
	memoryResult(c, gin.H{"saved": true}, h.service.SaveEntry(c, id, req))
}

// DeleteEntry DELETE /entries/:id，删除素材但保留历史版本。
func (h *FeedbackMemoryHandler) DeleteEntry(c *gin.Context) {
	q, ok := memoryScope(c)
	if !ok {
		return
	}
	id, ok := memoryID(c)
	if !ok {
		return
	}
	memoryResult(c, gin.H{"deleted": true}, h.service.DeleteEntry(c, q, id))
}

// Generate POST /generate，以当前素材和上一发布版调用 AI 生成新草稿。
func (h *FeedbackMemoryHandler) Generate(c *gin.Context) {
	var q dto.FeedbackMemoryScope
	if c.ShouldBindJSON(&q) != nil {
		memoryBadRequest(c)
		return
	}
	out, err := h.service.Generate(c, q)
	memoryResult(c, out, err)
}

// Versions GET /versions，分页返回历史版本索引。
func (h *FeedbackMemoryHandler) Versions(c *gin.Context) {
	scope, ok := memoryScope(c)
	if !ok {
		return
	}
	q, ok := experienceQuery(c)
	if !ok {
		return
	}
	out, err := h.service.Versions(c, scope, q)
	memoryResult(c, out, err)
}

// Version GET /versions/:id，返回完整生成输入输出和来源快照。
func (h *FeedbackMemoryHandler) Version(c *gin.Context) {
	q, ok := memoryScope(c)
	if !ok {
		return
	}
	id, ok := memoryID(c)
	if !ok {
		return
	}
	out, err := h.service.Version(c, q, id)
	memoryResult(c, out, err)
}

// Publish POST /versions/:id/publish，确认发布草稿。
func (h *FeedbackMemoryHandler) Publish(c *gin.Context) {
	var q dto.FeedbackMemoryScope
	if c.ShouldBindJSON(&q) != nil {
		memoryBadRequest(c)
		return
	}
	id, ok := memoryID(c)
	if !ok {
		return
	}
	memoryResult(c, gin.H{"published": true}, h.service.Publish(c, q, &id))
}

// Disable POST /disable，停用当前记忆并保留来源和历史。
func (h *FeedbackMemoryHandler) Disable(c *gin.Context) {
	var q dto.FeedbackMemoryScope
	if c.ShouldBindJSON(&q) != nil {
		memoryBadRequest(c)
		return
	}
	memoryResult(c, gin.H{"disabled": true}, h.service.Publish(c, q, nil))
}
