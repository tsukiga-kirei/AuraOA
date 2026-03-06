package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// OAConnectionRepo 提供 OA 数据库连接的数据访问。
type OAConnectionRepo struct {
	*BaseRepo
}

func NewOAConnectionRepo(db *gorm.DB) *OAConnectionRepo {
	return &OAConnectionRepo{BaseRepo: NewBaseRepo(db)}
}

// List 返回所有 OA 数据库连接，按创建时间排序。
func (r *OAConnectionRepo) List() ([]model.OADatabaseConnection, error) {
	var items []model.OADatabaseConnection
	if err := r.DB.Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindByID 通过 UUID 查找 OA 连接。
func (r *OAConnectionRepo) FindByID(id uuid.UUID) (*model.OADatabaseConnection, error) {
	var item model.OADatabaseConnection
	if err := r.DB.Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// Create 创建新的 OA 连接记录。
func (r *OAConnectionRepo) Create(item *model.OADatabaseConnection) error {
	return r.DB.Create(item).Error
}

// Update 更新 OA 连接记录。
func (r *OAConnectionRepo) Update(id uuid.UUID, fields map[string]interface{}) error {
	return r.DB.Model(&model.OADatabaseConnection{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除 OA 连接记录。
func (r *OAConnectionRepo) Delete(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.OADatabaseConnection{}).Error
}
