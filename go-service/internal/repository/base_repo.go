package repository

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

//BaseRepo 提供常见的数据库操作和租户隔离支持。
type BaseRepo struct {
	DB *gorm.DB
}

//NewBaseRepo 创建一个新的 BaseRepo 实例。
func NewBaseRepo(db *gorm.DB) *BaseRepo {
	return &BaseRepo{DB: db}
}

//WithTenant 返回一个范围为当前租户的 *gorm.DB。
//如果上下文中存在tenant_id，则会添加WHEREtenant_id = ?。
//如果tenant_id为空（例如没有特定租户的system_admin），则返回未过滤的数据库。
func (r *BaseRepo) WithTenant(c *gin.Context) *gorm.DB {
	tenantID, exists := c.Get("tenant_id")
	if exists && tenantID != "" {
		return r.DB.Where("tenant_id = ?", tenantID)
	}
	return r.DB
}
