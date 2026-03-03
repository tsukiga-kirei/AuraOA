package repository

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BaseRepo provides common database operations with tenant isolation support.
type BaseRepo struct {
	DB *gorm.DB
}

// NewBaseRepo creates a new BaseRepo instance.
func NewBaseRepo(db *gorm.DB) *BaseRepo {
	return &BaseRepo{DB: db}
}

// WithTenant returns a *gorm.DB scoped to the current tenant.
// If tenant_id is present in the context, it adds WHERE tenant_id = ?.
// If tenant_id is empty (e.g. system_admin without a specific tenant), returns unfiltered DB.
func (r *BaseRepo) WithTenant(c *gin.Context) *gorm.DB {
	tenantID, exists := c.Get("tenant_id")
	if exists && tenantID != "" {
		return r.DB.Where("tenant_id = ?", tenantID)
	}
	return r.DB
}
