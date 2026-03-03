package handler

import (
	"github.com/gin-gonic/gin"

	"oa-smart-audit/go-service/internal/pkg/response"
)

// HealthHandler handles health check HTTP requests.
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler instance.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health handles GET /api/health
func (h *HealthHandler) Health(c *gin.Context) {
	response.Success(c, gin.H{"status": "ok"})
}
