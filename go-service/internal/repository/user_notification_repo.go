package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// UserNotificationRepo 用户通知数据访问。
type UserNotificationRepo struct {
	*BaseRepo
}

// NewUserNotificationRepo 创建 UserNotificationRepo。
func NewUserNotificationRepo(db *gorm.DB) *UserNotificationRepo {
	return &UserNotificationRepo{BaseRepo: NewBaseRepo(db)}
}

// ListForAssignment 按当前角色分配列出通知；unreadOnly 为 true 时仅未读。
func (r *UserNotificationRepo) ListForAssignment(userID, roleAssignmentID uuid.UUID, limit, offset int, unreadOnly bool) ([]model.UserNotification, int64, error) {
	base := r.DB.Model(&model.UserNotification{}).
		Where("user_id = ? AND role_assignment_id = ?", userID, roleAssignmentID)
	if unreadOnly {
		base = base.Where("read_at IS NULL")
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := r.DB.Where("user_id = ? AND role_assignment_id = ?", userID, roleAssignmentID)
	if unreadOnly {
		q = q.Where("read_at IS NULL")
	}
	var rows []model.UserNotification
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// UnreadCount 当前角色分配下未读数量。
func (r *UserNotificationRepo) UnreadCount(userID, roleAssignmentID uuid.UUID) (int64, error) {
	var n int64
	err := r.DB.Model(&model.UserNotification{}).
		Where("user_id = ? AND role_assignment_id = ? AND read_at IS NULL", userID, roleAssignmentID).
		Count(&n).Error
	return n, err
}

// MarkRead 将单条标记为已读；仅当 user + assignment 匹配时更新。
func (r *UserNotificationRepo) MarkRead(id, userID, roleAssignmentID uuid.UUID) (int64, error) {
	now := time.Now()
	tx := r.DB.Model(&model.UserNotification{}).
		Where("id = ? AND user_id = ? AND role_assignment_id = ?", id, userID, roleAssignmentID).
		Update("read_at", now)
	return tx.RowsAffected, tx.Error
}

// MarkAllRead 当前角色分配下全部标为已读。
func (r *UserNotificationRepo) MarkAllRead(userID, roleAssignmentID uuid.UUID) error {
	now := time.Now()
	return r.DB.Model(&model.UserNotification{}).
		Where("user_id = ? AND role_assignment_id = ? AND read_at IS NULL", userID, roleAssignmentID).
		Update("read_at", now).Error
}

// Create 写入一条通知（供业务/定时任务调用）。
func (r *UserNotificationRepo) Create(n *model.UserNotification) error {
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	return r.DB.Create(n).Error
}
