package handler

import (
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

// AuditDataQueryHandler 提供 /api/tenant/audit-data-queries 查询库、检查、预览和 AI 辅助接口。
type AuditDataQueryHandler struct {
	service *service.AuditDataQueryService
}

// NewAuditDataQueryHandler 创建数据记忆接口处理器。
func NewAuditDataQueryHandler(s *service.AuditDataQueryService) *AuditDataQueryHandler {
	return &AuditDataQueryHandler{s}
}

func queryID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "查询ID无效")
		return uuid.Nil, false
	}
	return id, true
}

func queryBody(c *gin.Context, value interface{}) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256<<10)
	if err := c.ShouldBindJSON(value); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "请求内容无效或超出大小限制")
		return false
	}
	return true
}

func queryReply(c *gin.Context, data interface{}, err error) {
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// List 分页查询当前租户的查询库，返回 items/total/page/page_size。
func (h *AuditDataQueryHandler) List(c *gin.Context) {
	data, err := h.service.List(c, c.Query("keyword"), parseIntQuery(c, "page", 1), parseIntQuery(c, "page_size", 20))
	queryReply(c, data, err)
}

// Get GET /:id，读取当前租户的查询草稿。
func (h *AuditDataQueryHandler) Get(c *gin.Context) {
	id, ok := queryID(c)
	if !ok {
		return
	}
	data, err := h.service.Get(c, id)
	queryReply(c, data, err)
}

// Save POST / 或 PUT /:id，保存查询定义与可视化配置。
func (h *AuditDataQueryHandler) Save(c *gin.Context) {
	id := uuid.Nil
	if c.Param("id") != "" {
		var ok bool
		id, ok = queryID(c)
		if !ok {
			return
		}
	}
	var req struct {
		Category   string                         `json:"category"`
		Definition model.AuditDataQueryDefinition `json:"definition"`
	}
	if !queryBody(c, &req) {
		return
	}
	data, err := h.service.Save(c, id, req.Category, req.Definition)
	queryReply(c, data, err)
}

// Publish POST /:id/publish，验证成功预览后生成不可变版本。
func (h *AuditDataQueryHandler) Publish(c *gin.Context) {
	id, ok := queryID(c)
	if !ok {
		return
	}
	data, err := h.service.Publish(c, id)
	queryReply(c, data, err)
}

// Versions GET /:id/versions，读取查询的发布历史。
func (h *AuditDataQueryHandler) Versions(c *gin.Context) {
	id, ok := queryID(c)
	if !ok {
		return
	}
	data, err := h.service.Versions(c, id)
	queryReply(c, data, err)
}

// Delete DELETE /:id，校验无当前引用后删除库条目，保留历史发布版本。
func (h *AuditDataQueryHandler) Delete(c *gin.Context) {
	id, ok := queryID(c)
	if !ok {
		return
	}
	queryReply(c, map[string]bool{"deleted": true}, h.service.Delete(c, id))
}

// References GET /:id/references，读取当前规则引用。
func (h *AuditDataQueryHandler) References(c *gin.Context) {
	id, ok := queryID(c)
	if !ok {
		return
	}
	data, err := h.service.References(c, id)
	queryReply(c, data, err)
}

// RemoveReferences DELETE /:id/references，按模块和流程取消全部当前规则引用。
func (h *AuditDataQueryHandler) RemoveReferences(c *gin.Context) {
	id, ok := queryID(c)
	if !ok {
		return
	}
	var req struct {
		Module      string `json:"module"`
		ProcessType string `json:"process_type"`
	}
	if !queryBody(c, &req) {
		return
	}
	count, err := h.service.RemoveReferences(c, id, req.Module, req.ProcessType)
	queryReply(c, map[string]int64{"removed_count": count}, err)
}

// Check POST /check，检查只读结构、参数、中文列名和表范围。
func (h *AuditDataQueryHandler) Check(c *gin.Context) {
	var req model.AuditDataQueryDefinition
	if !queryBody(c, &req) {
		return
	}
	queryReply(c, h.service.Check(req), nil)
}

// Preview POST /preview，使用测试参数或真实流程绑定查询并返回 AI 数据预览。
func (h *AuditDataQueryHandler) Preview(c *gin.Context) {
	var req service.QueryPreviewRequest
	if !queryBody(c, &req) {
		return
	}
	data, err := h.service.Preview(c, req)
	queryReply(c, data, err)
}

// Schema GET /schema?table=表名，返回物理字段与类型。
func (h *AuditDataQueryHandler) Schema(c *gin.Context) {
	data, err := h.service.Schema(c, c.Query("table"))
	queryReply(c, data, err)
}

// Assist POST /assist，生成、解释或修复 SQL，返回候选结果。
func (h *AuditDataQueryHandler) Assist(c *gin.Context) {
	var req service.QueryAssistRequest
	if !queryBody(c, &req) {
		return
	}
	data, err := h.service.Assist(c, req)
	queryReply(c, data, err)
}

// Prompts GET/PUT /prompts，读取或更新租户 SQL 辅助提示词。
func (h *AuditDataQueryHandler) Prompts(c *gin.Context) {
	if c.Request.Method == http.MethodPut {
		var req model.AuditDataQueryPrompts
		if !queryBody(c, &req) {
			return
		}
		data, err := h.service.SavePrompts(c, req)
		queryReply(c, data, err)
	} else {
		data, err := h.service.Prompts(c)
		queryReply(c, data, err)
	}
}

// Source GET /api/tenant/audit-data-queries/source，返回租户数据源名称和方言。
func (h *AuditDataQueryHandler) Source(c *gin.Context) {
	data, err := h.service.Source(c)
	queryReply(c, data, err)
}
