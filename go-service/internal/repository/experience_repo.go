package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
)

// ExperienceRepo 负责租户内审核互动和智能体反馈查询。
type ExperienceRepo struct{ *BaseRepo }

// NewExperienceRepo 创建反馈仓储。
func NewExperienceRepo(db *gorm.DB) *ExperienceRepo { return &ExperienceRepo{NewBaseRepo(db)} }

func normalizeExperiencePage(q *dto.ExperienceQuery) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// CreateComment 保存一条用户评论。
func (r *ExperienceRepo) CreateComment(c *gin.Context, comment *model.AuditComment) error {
	return r.WithTenant(c).Create(comment).Error
}

// UpdateOwnComment 在同一条 SQL 内校验租户、审核批次及 OA 作者，防止越权修改。
func (r *ExperienceRepo) UpdateOwnComment(c *gin.Context, auditID, commentID uuid.UUID, oaID string, req dto.AuditCommentRequest) (bool, error) {
	result := r.WithTenant(c).Model(&model.AuditComment{}).
		Where("audit_log_id = ? AND id = ? AND oa_user_id = ?", auditID, commentID, oaID).
		Updates(map[string]interface{}{"content": req.Content, "feedback": req.Feedback, "updated_at": apptime.Now()})
	return result.RowsAffected > 0, result.Error
}

// DeleteOwnComment 仅删除当前 OA 作者在指定审核批次下的评论。
func (r *ExperienceRepo) DeleteOwnComment(c *gin.Context, auditID, commentID uuid.UUID, oaID string) (bool, error) {
	result := r.WithTenant(c).Where("audit_log_id = ? AND id = ? AND oa_user_id = ?", auditID, commentID, oaID).Delete(&model.AuditComment{})
	return result.RowsAffected > 0, result.Error
}

// Interactions 分页读取指定审核的评论，同时统计当前评价。
func (r *ExperienceRepo) Interactions(c *gin.Context, id uuid.UUID, q dto.ExperienceQuery) (*dto.AuditInteractionResponse, error) {
	normalizeExperiencePage(&q)
	out := &dto.AuditInteractionResponse{ExperiencePage: dto.ExperiencePage[model.AuditComment]{Items: []model.AuditComment{}, Page: q.Page, PageSize: q.PageSize}}
	query := r.WithTenant(c).Model(&model.AuditComment{}).Where("audit_log_id = ?", id)
	if err := query.Session(&gorm.Session{}).Count(&out.Total).Error; err != nil {
		return nil, err
	}
	if err := query.Order("created_at ASC, id ASC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&out.Items).Error; err != nil {
		return nil, err
	}
	var counts struct {
		Likes    int64
		Dislikes int64
	}
	err := r.WithTenant(c).Model(&model.AuditComment{}).Where("audit_log_id = ?", id).
		Select("COUNT(*) FILTER (WHERE feedback = 'like') AS likes, COUNT(*) FILTER (WHERE feedback = 'dislike') AS dislikes").Scan(&counts).Error
	out.LikeCount, out.DislikeCount = counts.Likes, counts.Dislikes
	return out, err
}

// ListAudits 仅返回存在评价或评论的审核批次，筛选在数据库分页前执行。
func (r *ExperienceRepo) ListAudits(c *gin.Context, q dto.ExperienceQuery) (*dto.ExperiencePage[dto.AuditExperienceItem], error) {
	normalizeExperiencePage(&q)
	out := &dto.ExperiencePage[dto.AuditExperienceItem]{Items: []dto.AuditExperienceItem{}, Page: q.Page, PageSize: q.PageSize}
	query := r.WithTenant(c).Table("audit_logs").Where(`EXISTS (SELECT 1 FROM audit_comments m WHERE m.tenant_id=audit_logs.tenant_id AND m.audit_log_id=audit_logs.id)`)
	if q.Feedback == "comments" {
		query = query.Where("EXISTS (SELECT 1 FROM audit_comments m WHERE m.tenant_id=audit_logs.tenant_id AND m.audit_log_id=audit_logs.id AND m.content <> '')")
	} else if q.Feedback != "" {
		query = query.Where("EXISTS (SELECT 1 FROM audit_comments f WHERE f.tenant_id=audit_logs.tenant_id AND f.audit_log_id=audit_logs.id AND f.feedback=?)", q.Feedback)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		query = query.Where(`title ILIKE ? OR process_id ILIKE ? OR EXISTS (SELECT 1 FROM audit_comments m WHERE m.tenant_id=audit_logs.tenant_id AND m.audit_log_id=audit_logs.id AND (m.content ILIKE ? OR m.username ILIKE ?))`, like, like, like, like)
	}
	if err := query.Session(&gorm.Session{}).Count(&out.Total).Error; err != nil {
		return nil, err
	}
	err := query.Select(`id, process_id, title, process_type,
 (SELECT COUNT(*) FROM audit_comments f WHERE f.tenant_id=audit_logs.tenant_id AND f.audit_log_id=audit_logs.id AND feedback='like') AS like_count,
 (SELECT COUNT(*) FROM audit_comments f WHERE f.tenant_id=audit_logs.tenant_id AND f.audit_log_id=audit_logs.id AND feedback='dislike') AS dislike_count,
 (SELECT COUNT(*) FROM audit_comments m WHERE m.tenant_id=audit_logs.tenant_id AND m.audit_log_id=audit_logs.id) AS comment_count,
 (SELECT MAX(m.updated_at) FROM audit_comments m WHERE m.tenant_id=audit_logs.tenant_id AND m.audit_log_id=audit_logs.id) AS updated_at`).Order("updated_at DESC, id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Scan(&out.Items).Error
	return out, err
}

// ListAgents 查询被点赞或点踩的助手回复，联表始终限定相同租户。
func (r *ExperienceRepo) ListAgents(c *gin.Context, q dto.ExperienceQuery) (*dto.ExperiencePage[dto.AgentExperienceItem], error) {
	normalizeExperiencePage(&q)
	out := &dto.ExperiencePage[dto.AgentExperienceItem]{Items: []dto.AgentExperienceItem{}, Page: q.Page, PageSize: q.PageSize}
	// 先在独立子查询应用 WithTenant，避免联表 tenant_id 字段歧义。
	messages := r.WithTenant(c).Model(&model.ChatMessage{}).Where("role = 'assistant' AND feedback IN ('like','dislike')")
	query := r.DB.Table("(?) AS m", messages).Joins("JOIN chat_sessions s ON s.id=m.session_id AND s.tenant_id=m.tenant_id").Joins("JOIN users u ON u.id=s.user_id").Joins("LEFT JOIN agent_definitions a ON a.id=s.agent_id")
	if q.Feedback == "comments" {
		query = query.Where("COALESCE(m.feedback_comment,'') <> ''")
	} else if q.Feedback != "" {
		query = query.Where("m.feedback = ?", q.Feedback)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		query = query.Where("s.title ILIKE ? OR m.content ILIKE ? OR m.feedback_comment ILIKE ? OR u.username ILIKE ? OR u.display_name ILIKE ?", like, like, like, like, like)
	}
	if err := query.Session(&gorm.Session{}).Count(&out.Total).Error; err != nil {
		return nil, err
	}
	err := query.Select("m.id, m.session_id, s.title, COALESCE(a.name,s.agent_code) AS agent_name, COALESCE(NULLIF(u.display_name,''),u.username) AS username, m.content, m.feedback, COALESCE(m.feedback_comment,'') AS feedback_comment, COALESCE(m.feedback_at,m.created_at) AS updated_at").Order("updated_at DESC, m.id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Scan(&out.Items).Error
	return out, err
}
