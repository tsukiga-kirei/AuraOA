package tenant

import (
	"net/http"
	"sync"

	"oa-smart-audit/internal/auth"

	"github.com/gin-gonic/gin"
)

// Handler exposes tenant management HTTP endpoints.
type Handler struct {
	// In-memory concurrency semaphores per tenant
	semaphores sync.Map // tenantID -> chan struct{}
}

func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes registers tenant admin routes.
func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	admin.GET("/tenants", h.ListTenants)
	admin.POST("/tenants", h.CreateTenant)
	admin.PUT("/tenants/:id/quota", h.UpdateQuota)
	admin.GET("/tenants/:id/config", h.GetConfig)
	admin.PUT("/tenants/:id/kb-mode", h.SetKBMode)
}

func (h *Handler) ListTenants(c *gin.Context) {
	// TODO: query from database
	c.JSON(http.StatusOK, gin.H{"tenants": []interface{}{}})
}

func (h *Handler) CreateTenant(c *gin.Context) {
	var input TenantInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	// TODO: persist to database
	c.JSON(http.StatusCreated, gin.H{"message": "tenant created", "name": input.Name})
}

func (h *Handler) UpdateQuota(c *gin.Context) {
	var quota QuotaConfig
	if err := c.ShouldBindJSON(&quota); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	tenantID := c.Param("id")
	// TODO: update in database
	c.JSON(http.StatusOK, gin.H{"message": "quota updated", "tenant_id": tenantID})
}

func (h *Handler) GetConfig(c *gin.Context) {
	tenantID := c.Param("id")
	// TODO: query from database
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "config": TenantConfig{}})
}

func (h *Handler) SetKBMode(c *gin.Context) {
	var body struct {
		ProcessType string `json:"process_type" binding:"required"`
		KBMode      KBMode `json:"kb_mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	tenantID := c.Param("id")
	// TODO: persist to database
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "process_type": body.ProcessType, "kb_mode": body.KBMode})
}

// QuotaCheckMiddleware checks token quota before processing audit requests.
func QuotaCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.GetClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		// TODO: check token_used < token_quota from database
		// For now, pass through
		c.Next()
	}
}

// ConcurrencyLimitMiddleware limits concurrent audit requests per tenant.
func (h *Handler) ConcurrencyLimitMiddleware(maxDefault int) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.GetClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}

		sem := h.getOrCreateSemaphore(claims.TenantID, maxDefault)
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		default:
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "concurrent limit exceeded"})
		}
	}
}

func (h *Handler) getOrCreateSemaphore(tenantID string, maxConcurrency int) chan struct{} {
	val, loaded := h.semaphores.LoadOrStore(tenantID, make(chan struct{}, maxConcurrency))
	if !loaded {
		return val.(chan struct{})
	}
	return val.(chan struct{})
}
