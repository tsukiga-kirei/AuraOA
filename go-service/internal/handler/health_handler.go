// 健康检查处理器，用于探活和就绪检查。
package handler

import (
	"github.com/gin-gonic/gin"

	"oa-smart-audit/go-service/internal/pkg/response"
)

// HealthHandler 处理健康检查相关的 HTTP 请求。
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器实例。
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health 返回服务存活状态。
// GET /api/health
// 返回：{"status": "ok"}。
func (h *HealthHandler) Health(c *gin.Context) {
	response.Success(c, gin.H{"status": "ok"})
}
