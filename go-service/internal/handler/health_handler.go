package handler

import (
	"github.com/gin-gonic/gin"

	"oa-smart-audit/go-service/internal/pkg/response"
)

//HealthHandler 处理健康检查 HTTP 请求。
type HealthHandler struct{}

//NewHealthHandler 创建一个新的 HealthHandler 实例。
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

//Health 处理 GET /api/health
func (h *HealthHandler) Health(c *gin.Context) {
	response.Success(c, gin.H{"status": "ok"})
}
