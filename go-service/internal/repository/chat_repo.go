package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
)

// ChatRepo 负责对话会话与消息的持久化
type ChatRepo struct {
	db *gorm.DB
}

// NewChatRepo 创建 ChatRepo
func NewChatRepo(db *gorm.DB) *ChatRepo {
	return &ChatRepo{db: db}
}

// CreateSession 创建新会话
func (r *ChatRepo) CreateSession(session *model.ChatSession) error {
	return r.db.Create(session).Error
}

// GetSessionByID 查询会话详情
func (r *ChatRepo) GetSessionByID(tenantID, sessionID uuid.UUID) (*model.ChatSession, error) {
	var session model.ChatSession
	err := r.db.Where("tenant_id = ? AND id = ?", tenantID, sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListSessionsByUser 分页查询用户在某租户下的会话列表
func (r *ChatRepo) ListSessionsByUser(tenantID, userID uuid.UUID, keyword string, page, pageSize int) ([]model.ChatSession, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := r.db.Model(&model.ChatSession{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID)

	if keyword != "" {
		query = query.Where("title ILIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.ChatSession
	err := query.Order("pinned DESC, updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// UpdateSession 更新会话（标题、置顶状态等）
func (r *ChatRepo) UpdateSession(tenantID, sessionID uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&model.ChatSession{}).
		Where("tenant_id = ? AND id = ?", tenantID, sessionID).
		Updates(updates).Error
}

// DeleteSession 删除会话（级联删除消息）
func (r *ChatRepo) DeleteSession(tenantID, sessionID uuid.UUID) error {
	return r.db.Where("tenant_id = ? AND id = ?", tenantID, sessionID).Delete(&model.ChatSession{}).Error
}

// DeleteExpiredSessions 硬删除超过指定保留天数的会话
func (r *ChatRepo) DeleteExpiredSessions(tenantID uuid.UUID, beforeTime time.Time) (int64, error) {
	res := r.db.Where("tenant_id = ? AND updated_at < ?", tenantID, beforeTime).Delete(&model.ChatSession{})
	return res.RowsAffected, res.Error
}

// CreateMessage 创建消息
func (r *ChatRepo) CreateMessage(msg *model.ChatMessage) error {
	return r.db.Create(msg).Error
}

// ListMessagesBySession 获取会话内的所有消息（按时间升序）
func (r *ChatRepo) ListMessagesBySession(tenantID, sessionID uuid.UUID) ([]model.ChatMessage, error) {
	var messages []model.ChatMessage
	err := r.db.Where("tenant_id = ? AND session_id = ?", tenantID, sessionID).
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}

// UpdateMessageStatus 更新消息状态与内容
func (r *ChatRepo) UpdateMessage(msgID uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&model.ChatMessage{}).Where("id = ?", msgID).Updates(updates).Error
}
