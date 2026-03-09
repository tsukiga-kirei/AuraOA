package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// ProcessAuditConfigRepo 提供流程审核配置的数据访问方法，按租户隔离。
type ProcessAuditConfigRepo struct {
	*BaseRepo
}

// NewProcessAuditConfigRepo 创建一个新的 ProcessAuditConfigRepo 实例。
func NewProcessAuditConfigRepo(db *gorm.DB) *ProcessAuditConfigRepo {
	return &ProcessAuditConfigRepo{BaseRepo: NewBaseRepo(db)}
}

// Create 创建流程审核配置记录。
func (r *ProcessAuditConfigRepo) Create(c *gin.Context, cfg *model.ProcessAuditConfig) error {
	return r.WithTenant(c).Create(cfg).Error
}

// GetByID 通过 ID 查询单个配置，自动按租户隔离。
func (r *ProcessAuditConfigRepo) GetByID(c *gin.Context, id uuid.UUID) (*model.ProcessAuditConfig, error) {
	var cfg model.ProcessAuditConfig
	if err := r.WithTenant(c).Where("id = ?", id).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ListByTenant 查询当前租户的所有流程审核配置。
func (r *ProcessAuditConfigRepo) ListByTenant(c *gin.Context) ([]model.ProcessAuditConfig, error) {
	var configs []model.ProcessAuditConfig
	if err := r.WithTenant(c).Order("created_at ASC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// Update 更新流程审核配置。
func (r *ProcessAuditConfigRepo) Update(c *gin.Context, cfg *model.ProcessAuditConfig) error {
	return r.WithTenant(c).Model(cfg).Where("id = ?", cfg.ID).Updates(cfg).Error
}

// UpdateFields 通过 map 更新指定字段，支持零值更新。
func (r *ProcessAuditConfigRepo) UpdateFields(c *gin.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.WithTenant(c).Model(&model.ProcessAuditConfig{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除流程审核配置。
func (r *ProcessAuditConfigRepo) Delete(c *gin.Context, id uuid.UUID) error {
	return r.WithTenant(c).Where("id = ?", id).Delete(&model.ProcessAuditConfig{}).Error
}

// GetByProcessType 通过流程类型查询配置（租户内唯一）。
func (r *ProcessAuditConfigRepo) GetByProcessType(c *gin.Context, processType string) (*model.ProcessAuditConfig, error) {
	var cfg model.ProcessAuditConfig
	if err := r.WithTenant(c).Where("process_type = ?", processType).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ExistsByProcessType 检查当前租户下是否已存在指定流程类型的配置。
func (r *ProcessAuditConfigRepo) ExistsByProcessType(c *gin.Context, processType string) (bool, error) {
	var count int64
	if err := r.WithTenant(c).Model(&model.ProcessAuditConfig{}).Where("process_type = ?", processType).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
