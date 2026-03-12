package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// CronTaskTypePresetRepo 提供 Cron 任务类型系统预设的数据访问方法（全局，无租户隔离）。
type CronTaskTypePresetRepo struct {
	db *gorm.DB
}

// NewCronTaskTypePresetRepo 创建一个新的 CronTaskTypePresetRepo 实例。
func NewCronTaskTypePresetRepo(db *gorm.DB) *CronTaskTypePresetRepo {
	return &CronTaskTypePresetRepo{db: db}
}

// ListAll 查询所有预设记录，按 sort_order 排序。
func (r *CronTaskTypePresetRepo) ListAll() ([]model.CronTaskTypePreset, error) {
	var presets []model.CronTaskTypePreset
	if err := r.db.Order("sort_order ASC").Find(&presets).Error; err != nil {
		return nil, err
	}
	return presets, nil
}

// GetByTaskType 按 task_type 查询单条预设。
func (r *CronTaskTypePresetRepo) GetByTaskType(taskType string) (*model.CronTaskTypePreset, error) {
	var preset model.CronTaskTypePreset
	if err := r.db.Where("task_type = ?", taskType).First(&preset).Error; err != nil {
		return nil, err
	}
	return &preset, nil
}

// CronTaskTypeConfigRepo 提供租户 Cron 任务类型配置的数据访问方法，按租户隔离。
type CronTaskTypeConfigRepo struct {
	*BaseRepo
}

// NewCronTaskTypeConfigRepo 创建一个新的 CronTaskTypeConfigRepo 实例。
func NewCronTaskTypeConfigRepo(db *gorm.DB) *CronTaskTypeConfigRepo {
	return &CronTaskTypeConfigRepo{BaseRepo: NewBaseRepo(db)}
}

// ListByTenant 查询当前租户的所有已配置任务类型。
func (r *CronTaskTypeConfigRepo) ListByTenant(c *gin.Context) ([]model.CronTaskTypeConfig, error) {
	var configs []model.CronTaskTypeConfig
	if err := r.WithTenant(c).Order("created_at ASC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetByTaskType 查询当前租户指定任务类型的配置。
func (r *CronTaskTypeConfigRepo) GetByTaskType(c *gin.Context, taskType string) (*model.CronTaskTypeConfig, error) {
	var cfg model.CronTaskTypeConfig
	if err := r.WithTenant(c).Where("task_type = ?", taskType).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save 保存（创建或更新）租户任务类型配置（基于 tenant_id + task_type 唯一约束）。
func (r *CronTaskTypeConfigRepo) Save(c *gin.Context, cfg *model.CronTaskTypeConfig) error {
	return r.WithTenant(c).Save(cfg).Error
}

// Upsert 使用 ON CONFLICT 进行幂等保存。
func (r *CronTaskTypeConfigRepo) Upsert(c *gin.Context, cfg *model.CronTaskTypeConfig) error {
	return r.WithTenant(c).
		Where(model.CronTaskTypeConfig{TenantID: cfg.TenantID, TaskType: cfg.TaskType}).
		Assign(*cfg).
		FirstOrCreate(cfg).Error
}

// Delete 删除租户任务类型配置（删除即代表关闭该任务类型）。
func (r *CronTaskTypeConfigRepo) Delete(c *gin.Context, tenantID uuid.UUID, taskType string) error {
	return r.WithTenant(c).
		Where("task_type = ?", taskType).
		Delete(&model.CronTaskTypeConfig{}).Error
}
