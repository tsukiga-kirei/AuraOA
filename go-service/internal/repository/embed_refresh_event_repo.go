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

// EmbedRefreshEventRepo 管理 OA 保存/提交事件及 requestid 解析状态。
type EmbedRefreshEventRepo struct {
	db *gorm.DB
}

// NewEmbedRefreshEventRepo 创建嵌入刷新事件 Repository。
func NewEmbedRefreshEventRepo(db *gorm.DB) *EmbedRefreshEventRepo {
	return &EmbedRefreshEventRepo{db: db}
}

// CreateOrGet 按 tenant_id + event_id 幂等写入，并返回实际持久化记录及是否新建。
func (r *EmbedRefreshEventRepo) CreateOrGet(ctx context.Context, event *model.EmbedRefreshEvent) (*model.EmbedRefreshEvent, bool, error) {
	now := apptime.Now()
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = now
	}
	event.CreatedAt = now
	event.UpdatedAt = now
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "event_id"}},
		DoNothing: true,
	}).Create(event)
	if result.Error != nil {
		return nil, false, result.Error
	}
	var persisted model.EmbedRefreshEvent
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND event_id = ?", event.TenantID, event.EventID).
		First(&persisted).Error; err != nil {
		return nil, false, err
	}
	return &persisted, result.RowsAffected == 1, nil
}

// GetByEventID 查询租户内指定外部事件。
func (r *EmbedRefreshEventRepo) GetByEventID(ctx context.Context, tenantID uuid.UUID, eventID string) (*model.EmbedRefreshEvent, error) {
	var event model.EmbedRefreshEvent
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND event_id = ?", tenantID, eventID).
		First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

// UpdateResolution 更新 requestid 解析结果。
func (r *EmbedRefreshEventRepo) UpdateResolution(
	ctx context.Context,
	id uuid.UUID,
	status, processID string,
	attempt int,
	nextAttemptAt *time.Time,
	lastError string,
	resolvedAt *time.Time,
) error {
	return r.db.WithContext(ctx).
		Model(&model.EmbedRefreshEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":          status,
			"process_id":      processID,
			"attempt":         attempt,
			"next_attempt_at": nextAttemptAt,
			"last_error":      lastError,
			"resolved_at":     resolvedAt,
			"updated_at":      apptime.Now(),
		}).Error
}

// ListPending 查询仍需恢复的 requestid 解析事件。
func (r *EmbedRefreshEventRepo) ListPending(ctx context.Context, limit int) ([]model.EmbedRefreshEvent, error) {
	if limit < 1 || limit > 5000 {
		limit = 1000
	}
	var events []model.EmbedRefreshEvent
	err := r.db.WithContext(ctx).
		Where("status = ?", model.EmbedRefreshEventPending).
		Order("received_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}
