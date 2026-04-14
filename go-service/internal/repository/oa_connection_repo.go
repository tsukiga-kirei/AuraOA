package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// OAConnectionRepo 提供 OA 数据库连接配置的数据访问方法（全局，无租户隔离）。
type OAConnectionRepo struct {
	*BaseRepo
}

// NewOAConnectionRepo 创建 OAConnectionRepo 实例。
func NewOAConnectionRepo(db *gorm.DB) *OAConnectionRepo {
	return &OAConnectionRepo{BaseRepo: NewBaseRepo(db)}
}

// List 查询所有 OA 数据库连接配置，按创建时间升序排列。
func (r *OAConnectionRepo) List() ([]model.OADatabaseConnection, error) {
	var items []model.OADatabaseConnection
	if err := r.DB.Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindByID 按 UUID 查询单个 OA 连接配置，不存在时返回 gorm.ErrRecordNotFound。
func (r *OAConnectionRepo) FindByID(id uuid.UUID) (*model.OADatabaseConnection, error) {
	var item model.OADatabaseConnection
	if err := r.DB.Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// Create 创建新的 OA 连接配置记录。
func (r *OAConnectionRepo) Create(item *model.OADatabaseConnection) error {
	return r.DB.Create(item).Error
}

// Update 按 ID 更新 OA 连接配置的指定字段。
func (r *OAConnectionRepo) Update(id uuid.UUID, fields map[string]interface{}) error {
	return r.DB.Model(&model.OADatabaseConnection{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 按 ID 删除 OA 连接配置记录。
func (r *OAConnectionRepo) Delete(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.OADatabaseConnection{}).Error
}
