package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/repository"
	"auraoa/go-service/internal/service"
)

// ExternalContextHandler 处理规则/总结块外部关联数据测试请求。
type ExternalContextHandler struct {
	ctxService *service.ExternalContextService
	tenantRepo *repository.TenantRepo
}

func NewExternalContextHandler(ctxService *service.ExternalContextService, tenantRepo *repository.TenantRepo) *ExternalContextHandler {
	return &ExternalContextHandler{ctxService: ctxService, tenantRepo: tenantRepo}
}

// Test 测试外部关联数据配置，返回实际会注入 AI 的文本。
func (h *ExternalContextHandler) Test(c *gin.Context) {
	if h.ctxService == nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "外部关联数据服务未初始化")
		return
	}
	tidVal, ok := c.Get("tenant_id")
	if !ok {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "租户ID缺失")
		return
	}
	tenantID, err := uuid.Parse(tidVal.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户ID格式无效")
		return
	}
	tenant, err := h.tenantRepo.FindByID(tenantID)
	if err != nil {
		response.Error(c, http.StatusNotFound, errcode.ErrConfigNotFound, "租户不存在")
		return
	}
	var req service.ExternalContextTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	result, err := h.ctxService.Test(c, tenant, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, result)
}
