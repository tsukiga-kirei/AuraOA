package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
)

// EmbedRefreshScheduleRepo 管理 OA 嵌入流程级持久化调度记录。
type EmbedRefreshScheduleRepo struct {
	db *gorm.DB
}

// NewEmbedRefreshScheduleRepo 创建嵌入刷新调度 Repository。
func NewEmbedRefreshScheduleRepo(db *gorm.DB) *EmbedRefreshScheduleRepo {
	return &EmbedRefreshScheduleRepo{db: db}
}

// Upsert 按 module + config_id 幂等保存调度记录，并返回数据库中的完整记录。
func (r *EmbedRefreshScheduleRepo) Upsert(ctx context.Context, schedule *model.EmbedRefreshSchedule) error {
	db := r.db.WithContext(ctx)
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "module"}, {Name: "config_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"tenant_id",
			"process_type",
			"is_active",
			"lookback_days",
			"interval_minutes",
			"cron_expression",
			"updated_at",
		}),
	}).Model(&model.EmbedRefreshSchedule{}).
		Create(embedRefreshScheduleValues(schedule)).Error; err != nil {
		return err
	}
	var persisted model.EmbedRefreshSchedule
	if err := db.Where("module = ? AND config_id = ?", schedule.Module, schedule.ConfigID).
		First(&persisted).Error; err != nil {
		return err
	}
	*schedule = persisted
	return nil
}

// embedRefreshScheduleValues 显式生成调度写入值，确保 false 不被 GORM 字段默认值替换。
func embedRefreshScheduleValues(schedule *model.EmbedRefreshSchedule) map[string]interface{} {
	now := apptime.Now()
	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}
	createdAt := schedule.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	return map[string]interface{}{
		"id":               schedule.ID,
		"tenant_id":        schedule.TenantID,
		"module":           schedule.Module,
		"config_id":        schedule.ConfigID,
		"process_type":     schedule.ProcessType,
		"is_active":        schedule.IsActive,
		"lookback_days":    schedule.LookbackDays,
		"interval_minutes": schedule.IntervalMinutes,
		"cron_expression":  schedule.CronExpression,
		"created_at":       createdAt,
		"updated_at":       now,
	}
}

// ListActive 查询所有租户启用中的嵌入刷新调度，供服务启动恢复。
func (r *EmbedRefreshScheduleRepo) ListActive(ctx context.Context) ([]model.EmbedRefreshSchedule, error) {
	var schedules []model.EmbedRefreshSchedule
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("tenant_id ASC, module ASC, created_at ASC").
		Find(&schedules).Error
	return schedules, err
}

// GetByID 查询一条调度记录。
func (r *EmbedRefreshScheduleRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.EmbedRefreshSchedule, error) {
	var schedule model.EmbedRefreshSchedule
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schedule).Error; err != nil {
		return nil, err
	}
	return &schedule, nil
}

// GetByConfig 查询指定模块和流程配置对应的调度记录。
func (r *EmbedRefreshScheduleRepo) GetByConfig(
	ctx context.Context,
	module string,
	configID uuid.UUID,
) (*model.EmbedRefreshSchedule, error) {
	var schedule model.EmbedRefreshSchedule
	if err := r.db.WithContext(ctx).
		Where("module = ? AND config_id = ?", module, configID).
		First(&schedule).Error; err != nil {
		return nil, err
	}
	return &schedule, nil
}

// DeleteByConfig 删除指定流程配置对应的调度记录。
func (r *EmbedRefreshScheduleRepo) DeleteByConfig(
	ctx context.Context,
	module string,
	configID uuid.UUID,
) error {
	return r.db.WithContext(ctx).
		Where("module = ? AND config_id = ?", module, configID).
		Delete(&model.EmbedRefreshSchedule{}).Error
}

// DeleteOrphans 清理流程配置已经不存在的调度记录。
func (r *EmbedRefreshScheduleRepo) DeleteOrphans(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`
		DELETE FROM embed_refresh_schedules s
		WHERE (s.module = 'audit' AND NOT EXISTS (
			SELECT 1 FROM process_audit_configs c WHERE c.id = s.config_id
		))
		OR (s.module = 'summary' AND NOT EXISTS (
			SELECT 1 FROM process_summary_configs c WHERE c.id = s.config_id
		))
	`).Error
}

// UpdateNextRun 更新下一次计划执行时间。
func (r *EmbedRefreshScheduleRepo) UpdateNextRun(
	ctx context.Context,
	id uuid.UUID,
	nextRunAt *time.Time,
) error {
	return r.db.WithContext(ctx).
		Model(&model.EmbedRefreshSchedule{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"next_run_at": nextRunAt,
			"updated_at":  apptime.Now(),
		}).Error
}

// UpdateRunResult 更新最近一次执行时间、结果和下次执行时间。
func (r *EmbedRefreshScheduleRepo) UpdateRunResult(
	ctx context.Context,
	id uuid.UUID,
	lastRunAt time.Time,
	nextRunAt *time.Time,
	status, lastError string,
) error {
	return r.db.WithContext(ctx).
		Model(&model.EmbedRefreshSchedule{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_run_at": lastRunAt,
			"next_run_at": nextRunAt,
			"last_status": status,
			"last_error":  lastError,
			"updated_at":  apptime.Now(),
		}).Error
}
