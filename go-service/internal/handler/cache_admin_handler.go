// 缓存管理处理器，提供缓存统计查询、租户/模块缓存清除和缓存开关功能。
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/cache"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
)

// CacheAdminHandler 处理缓存管理相关的 HTTP 请求。
type CacheAdminHandler struct {
	cache       *cache.CacheManager
	invalidator *cache.InvalidationManager
}

// NewCacheAdminHandler 创建缓存管理处理器实例。
func NewCacheAdminHandler(cache *cache.CacheManager, invalidator *cache.InvalidationManager) *CacheAdminHandler {
	return &CacheAdminHandler{
		cache:       cache,
		invalidator: invalidator,
	}
}

// GetStats 获取缓存统计信息。
// GET /api/admin/cache/stats
// 返回：缓存命中率、命中次数、未命中次数、错误次数、键数量等指标。
func (h *CacheAdminHandler) GetStats(c *gin.Context) {
	stats := h.cache.GetStats()
	response.Success(c, stats)
}

// ClearTenantCache 清除指定租户的全部缓存。
// DELETE /api/admin/cache/tenant/:tenant_id
// 路径参数：tenant_id（UUID 格式）
// 返回：null（成功时）。
func (h *CacheAdminHandler) ClearTenantCache(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "无效的租户ID")
		return
	}
	if err := h.invalidator.InvalidateTenantCache(c.Request.Context(), tenantID); err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "清除租户缓存失败")
		return
	}
	response.Success(c, nil)
}

// ClearModuleCache 清除指定模块的全部缓存。
// DELETE /api/admin/cache/module/:module
// 路径参数：module（audit、archive 或 dashboard）
// 返回：null（成功时）。
func (h *CacheAdminHandler) ClearModuleCache(c *gin.Context) {
	module := c.Param("module")
	if module == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "模块名称不能为空")
		return
	}
	if err := h.invalidator.InvalidateModuleCache(c.Request.Context(), module); err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "清除模块缓存失败")
		return
	}
	response.Success(c, nil)
}

// ToggleCache 开启或关闭缓存功能。
// POST /api/admin/cache/toggle
// 请求体：{"enabled": true/false}
// 返回：null（成功时）。
func (h *CacheAdminHandler) ToggleCache(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	h.cache.SetEnabled(req.Enabled)
	response.Success(c, nil)
}
