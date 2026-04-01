package repository

import (
	"context"
	"time"

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

// ============================================================
// CronTaskRepo — 定时任务实例 CRUD（按租户隔离）
// ============================================================

// CronTaskRepo 提供 cron_tasks 表的数据访问方法，按租户隔离。
type CronTaskRepo struct {
	*BaseRepo
}

// NewCronTaskRepo 创建一个新的 CronTaskRepo 实例。
func NewCronTaskRepo(db *gorm.DB) *CronTaskRepo {
	return &CronTaskRepo{BaseRepo: NewBaseRepo(db)}
}

// DB 暴露底层 gorm.DB（供调度器跨租户查询使用）。
func (r *CronTaskRepo) DB() *gorm.DB { return r.BaseRepo.DB }

// ListByOwner 查询当前租户下指定归属用户的任务实例，按创建时间排序。
func (r *CronTaskRepo) ListByOwner(c *gin.Context, ownerUserID uuid.UUID) ([]model.CronTask, error) {
	var tasks []model.CronTask
	if err := r.WithTenant(c).Where("owner_user_id = ?", ownerUserID).Order("created_at ASC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// ListActiveByAllTenants 查询所有租户的活跃任务（调度器启动时使用，无 gin.Context）。
func (r *CronTaskRepo) ListActiveByAllTenants() ([]model.CronTask, error) {
	var tasks []model.CronTask
	if err := r.BaseRepo.DB.Where("is_active = true").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetByIDForOwner 查询指定 ID 的任务（租户 + 归属用户校验）。
func (r *CronTaskRepo) GetByIDForOwner(c *gin.Context, id uuid.UUID, ownerUserID uuid.UUID) (*model.CronTask, error) {
	var task model.CronTask
	if err := r.WithTenant(c).Where("id = ? AND owner_user_id = ?", id, ownerUserID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// Create 创建新任务实例。
func (r *CronTaskRepo) Create(task *model.CronTask) error {
	return r.BaseRepo.DB.Create(task).Error
}

// Update 更新任务实例（归属用户校验；不可改 owner_user_id）。
func (r *CronTaskRepo) Update(c *gin.Context, id uuid.UUID, ownerUserID uuid.UUID, fields map[string]interface{}) error {
	return r.WithTenant(c).Model(&model.CronTask{}).Where("id = ? AND owner_user_id = ?", id, ownerUserID).Updates(fields).Error
}

// UpdateFields 更新任务指定字段（不强制要求 gin.Context）。
func (r *CronTaskRepo) UpdateFields(c context.Context, id uuid.UUID, ownerUserID uuid.UUID, fields map[string]interface{}) error {
	return r.BaseRepo.DB.WithContext(c).Model(&model.CronTask{}).Where("id = ? AND owner_user_id = ?", id, ownerUserID).Updates(fields).Error
}

// Delete 删除任务实例（内置任务由调用层防护）。
func (r *CronTaskRepo) Delete(c *gin.Context, id uuid.UUID, ownerUserID uuid.UUID) error {
	return r.WithTenant(c).Where("id = ? AND owner_user_id = ?", id, ownerUserID).Delete(&model.CronTask{}).Error
}

// UpdateRunStats 更新任务的运行统计（last_run_at / next_run_at / success_count / fail_count）。
func (r *CronTaskRepo) UpdateRunStats(id uuid.UUID, lastRunAt time.Time, nextRunAt *time.Time, success bool) error {
	fields := map[string]interface{}{
		"last_run_at": lastRunAt,
		"updated_at":  time.Now(),
	}
	if nextRunAt != nil {
		fields["next_run_at"] = nextRunAt
	}
	if success {
		fields["success_count"] = gorm.Expr("success_count + 1")
	} else {
		fields["fail_count"] = gorm.Expr("fail_count + 1")
	}
	return r.BaseRepo.DB.Model(&model.CronTask{}).Where("id = ?", id).Updates(fields).Error
}
