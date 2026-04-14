package repository

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BaseRepo 封装公共数据库操作，提供租户隔离支持。
// 所有业务 Repo 均嵌入此结构体以复用 DB 实例和租户过滤逻辑。
type BaseRepo struct {
	DB *gorm.DB
}

// NewBaseRepo 创建 BaseRepo 实例，注入 gorm.DB 连接。
func NewBaseRepo(db *gorm.DB) *BaseRepo {
	return &BaseRepo{DB: db}
}

// WithTenant 返回已附加当前租户过滤条件的 *gorm.DB。
// 若 gin.Context 中存在 tenant_id，则自动追加 WHERE tenant_id = ? 条件；
// 若 tenant_id 为空（如 system_admin 跨租户操作），则返回不带过滤的原始 DB。
func (r *BaseRepo) WithTenant(c *gin.Context) *gorm.DB {
	tenantID, exists := c.Get("tenant_id")
	if exists && tenantID != "" {
		return r.DB.Where("tenant_id = ?", tenantID)
	}
	return r.DB
}
