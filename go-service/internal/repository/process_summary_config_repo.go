package repository

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
)

// ProcessSummaryConfigRepo 提供流程总结配置的数据访问方法，按租户隔离。
type ProcessSummaryConfigRepo struct {
	*BaseRepo
}

func NewProcessSummaryConfigRepo(db *gorm.DB) *ProcessSummaryConfigRepo {
	return &ProcessSummaryConfigRepo{BaseRepo: NewBaseRepo(db)}
}

func (r *ProcessSummaryConfigRepo) Create(c *gin.Context, cfg *model.ProcessSummaryConfig) error {
	return r.WithTenant(c).Create(cfg).Error
}

func (r *ProcessSummaryConfigRepo) GetByID(c *gin.Context, id uuid.UUID) (*model.ProcessSummaryConfig, error) {
	var cfg model.ProcessSummaryConfig
	if err := r.WithTenant(c).Where("id = ?", id).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *ProcessSummaryConfigRepo) ListByTenant(c *gin.Context) ([]model.ProcessSummaryConfig, error) {
	var configs []model.ProcessSummaryConfig
	if err := r.WithTenant(c).Order("created_at ASC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// ListAllTenants 查询所有租户的总结配置，供系统启动时重建持久化调度记录。
func (r *ProcessSummaryConfigRepo) ListAllTenants(ctx context.Context) ([]model.ProcessSummaryConfig, error) {
	var configs []model.ProcessSummaryConfig
	if err := r.DB.
		WithContext(ctx).
		Order("tenant_id ASC, created_at ASC").
		Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *ProcessSummaryConfigRepo) UpdateFields(c *gin.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.WithTenant(c).Model(&model.ProcessSummaryConfig{}).Where("id = ?", id).Updates(fields).Error
}

func (r *ProcessSummaryConfigRepo) Delete(c *gin.Context, id uuid.UUID) error {
	return r.WithTenant(c).Where("id = ?", id).Delete(&model.ProcessSummaryConfig{}).Error
}

func (r *ProcessSummaryConfigRepo) GetByProcessType(c *gin.Context, processType string) (*model.ProcessSummaryConfig, error) {
	var cfg model.ProcessSummaryConfig
	if err := r.WithTenant(c).Where("process_type = ?", processType).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *ProcessSummaryConfigRepo) ExistsByProcessType(c *gin.Context, processType string) (bool, error) {
	var count int64
	if err := r.WithTenant(c).Model(&model.ProcessSummaryConfig{}).Where("process_type = ?", processType).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
