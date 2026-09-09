package repository

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
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
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := r.db.Model(&model.ChatSession{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID)

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(`title ILIKE ? OR EXISTS (
			SELECT 1 FROM chat_messages m
			WHERE m.session_id = chat_sessions.id AND m.tenant_id = chat_sessions.tenant_id AND m.content ILIKE ?
		)`, like, like)
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

// UpdateSession 更新会话（标题、置顶状态等），并在修改标题时同步更新相关 AI 调用的流程标题
func (r *ChatRepo) UpdateSession(tenantID, sessionID uuid.UUID, updates map[string]interface{}) error {
	if err := r.db.Model(&model.ChatSession{}).
		Where("tenant_id = ? AND id = ?", tenantID, sessionID).
		Updates(updates).Error; err != nil {
		return err
	}
	if title, ok := updates["title"].(string); ok && title != "" {
		_ = r.db.Model(&model.TenantLLMMessageLog{}).
			Where("tenant_id = ? AND business_log_id = ?", tenantID, sessionID).
			Update("process_title", title).Error
	}
	return nil
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
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		return tx.Model(&model.ChatSession{}).Where("tenant_id = ? AND id = ?", msg.TenantID, msg.SessionID).Update("updated_at", msg.CreatedAt).Error
	})
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

// UpdateMessageFeedback 更新单条消息的点赞/点踩反馈及改进建议
func (r *ChatRepo) UpdateMessageFeedback(tenantID, messageID uuid.UUID, feedback *string, comment *string) error {
	updates := map[string]interface{}{}
	if feedback != nil && (*feedback == "like" || *feedback == "dislike") {
		updates["feedback"] = feedback
		now := apptime.Now()
		updates["feedback_at"] = &now
		if *feedback == "dislike" {
			updates["feedback_comment"] = comment
		} else {
			updates["feedback_comment"] = nil
		}
	} else {
		updates["feedback"] = nil
		updates["feedback_at"] = nil
		updates["feedback_comment"] = nil
	}
	return r.db.Model(&model.ChatMessage{}).
		Where("tenant_id = ? AND id = ?", tenantID, messageID).
		Updates(updates).Error
}

// ListSessionsByTenant 分页查询租户下的智能体会话明细（数据管理页使用）
func (r *ChatRepo) ListSessionsByTenant(
	tenantID uuid.UUID,
	keyword, agentCode, userName, startDate, endDate string,
	page, pageSize int,
) ([]dto.TenantAgentSessionItemDTO, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseQuery := r.db.Table("chat_sessions s").
		Joins("JOIN users u ON u.id = s.user_id").
		Joins("LEFT JOIN agent_definitions ad ON ad.id = s.agent_id").
		Where("s.tenant_id = ?", tenantID)

	if keyword != "" {
		like := "%" + keyword + "%"
		baseQuery = baseQuery.Where("s.title ILIKE ?", like)
	}
	if agentCode != "" {
		baseQuery = baseQuery.Where("s.agent_code = ?", agentCode)
	}
	if userName != "" {
		like := "%" + userName + "%"
		baseQuery = baseQuery.Where("(u.display_name ILIKE ? OR u.username ILIKE ?)", like, like)
	}
	if startDate != "" {
		baseQuery = baseQuery.Where("s.created_at >= ?", startDate)
	}
	if endDate != "" {
		baseQuery = baseQuery.Where("s.created_at <= ?", endDate)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	type sessionRow struct {
		ID           uuid.UUID `gorm:"column:id"`
		AgentCode    string    `gorm:"column:agent_code"`
		AgentName    string    `gorm:"column:agent_name"`
		UserID       uuid.UUID `gorm:"column:user_id"`
		UserName     string    `gorm:"column:user_name"`
		Title        string    `gorm:"column:title"`
		MessageCount int64     `gorm:"column:message_count"`
		TokenCount   int64     `gorm:"column:token_count"`
		LikeCount    int64     `gorm:"column:like_count"`
		DislikeCount int64     `gorm:"column:dislike_count"`
		CreatedAt    time.Time `gorm:"column:created_at"`
		UpdatedAt    time.Time `gorm:"column:updated_at"`
	}

	var rows []sessionRow
	err := baseQuery.Select(`
		s.id,
		s.agent_code,
		COALESCE(NULLIF(ad.name, ''), s.agent_code) AS agent_name,
		s.user_id,
		COALESCE(NULLIF(u.display_name, ''), u.username, '未知用户') AS user_name,
		s.title,
		s.created_at,
		s.updated_at,
		(SELECT COUNT(*) FROM chat_messages m WHERE m.session_id = s.id AND m.tenant_id = s.tenant_id) AS message_count,
		(SELECT COUNT(*) FROM chat_messages m WHERE m.session_id = s.id AND m.tenant_id = s.tenant_id AND m.feedback = 'like') AS like_count,
		(SELECT COUNT(*) FROM chat_messages m WHERE m.session_id = s.id AND m.tenant_id = s.tenant_id AND m.feedback = 'dislike') AS dislike_count,
		COALESCE((SELECT SUM(l.total_tokens) FROM tenant_llm_message_logs l WHERE l.tenant_id = s.tenant_id AND l.request_type = 'chat' AND l.business_log_id = s.id), 0) AS token_count
	`).
		Order("s.updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	items := make([]dto.TenantAgentSessionItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.TenantAgentSessionItemDTO{
			ID:           row.ID,
			AgentCode:    row.AgentCode,
			AgentName:    row.AgentName,
			UserID:       row.UserID,
			UserName:     row.UserName,
			Title:        row.Title,
			MessageCount: row.MessageCount,
			TokenCount:   row.TokenCount,
			LikeCount:    row.LikeCount,
			DislikeCount: row.DislikeCount,
			CreatedAt:    apptime.FormatRFC3339(row.CreatedAt),
			UpdatedAt:    apptime.FormatRFC3339(row.UpdatedAt),
		})
	}
	return items, total, nil
}

// CountThisWeek 本周（按应用配置时区周一 00:00 至今）智能体会话活跃次数。
func (r *ChatRepo) CountThisWeek(c *gin.Context, userID *uuid.UUID) (int64, error) {
	tenantID, _ := c.Get("tenant_id")
	var count int64
	q := r.db.Model(&model.ChatSession{}).
		Where("updated_at >= date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE ?)", apptime.Name()).
		Where("EXISTS (SELECT 1 FROM chat_messages cm WHERE cm.session_id = chat_sessions.id)")
	if tenantID != nil && tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if userID != nil {
		q = q.Where("user_id = ?", *userID)
	}
	err := q.Count(&count).Error
	return count, err
}

// WeeklyTrendByDay 本周每天的智能体会话条数（按实际发生消息的时间聚合，generate_series 填充）。
func (r *ChatRepo) WeeklyTrendByDay(c *gin.Context, userID *uuid.UUID) ([]DayCount, error) {
	tenantID, _ := c.Get("tenant_id")
	userFilter := ""
	args := []interface{}{apptime.Name(), apptime.Name(), apptime.Name(), tenantID, apptime.Name()}
	if userID != nil {
		userFilter = "AND s.user_id = ?"
		args = append(args, *userID)
	}

	sql := `
WITH days AS (
  SELECT generate_series(
    date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE ?)::date,
    (CURRENT_TIMESTAMP AT TIME ZONE ?)::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date,
       COALESCE(b.cnt, 0)::bigint AS count
FROM days
LEFT JOIN (
  SELECT DATE(m.created_at AT TIME ZONE ?) AS d,
         COUNT(DISTINCT s.id)::bigint AS cnt
  FROM chat_messages m
  JOIN chat_sessions s ON s.id = m.session_id
  WHERE s.tenant_id = ?
    AND m.created_at >= date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE ?)
    ` + userFilter + `
  GROUP BY DATE(m.created_at AT TIME ZONE ?)
) b ON b.d = days.d
ORDER BY days.d ASC`

	args = append(args, apptime.Name())
	var result []DayCount
	err := r.db.Raw(sql, args...).Scan(&result).Error
	return result, err
}

// GetDashboardAgentOverview 获取仪表盘智能体专属统计数据（按近一周统计）
func (r *ChatRepo) GetDashboardAgentOverview(tenantID uuid.UUID, userScope *uuid.UUID) (*dto.DashboardAgentOverviewData, error) {
	since := apptime.Now().AddDate(0, 0, -7)
	if userScope != nil {
		// 个人视角 (business，近一周)
		var mySessions int64
		_ = r.db.Model(&model.ChatSession{}).Where("tenant_id = ? AND user_id = ? AND created_at >= ?", tenantID, *userScope, since).Count(&mySessions).Error

		var myMessages int64
		_ = r.db.Table("chat_messages m").
			Joins("JOIN chat_sessions s ON s.id = m.session_id").
			Where("s.tenant_id = ? AND s.user_id = ? AND m.role = 'user' AND m.created_at >= ?", tenantID, *userScope, since).
			Count(&myMessages).Error

		var myLikes int64
		_ = r.db.Table("chat_messages m").
			Joins("JOIN chat_sessions s ON s.id = m.session_id").
			Where("s.tenant_id = ? AND s.user_id = ? AND m.feedback = 'like' AND m.created_at >= ?", tenantID, *userScope, since).
			Count(&myLikes).Error

		var favAgent string
		_ = r.db.Raw(`
			SELECT COALESCE(NULLIF(ad.name, ''), s.agent_code) AS agent_name
			FROM chat_sessions s
			LEFT JOIN agent_definitions ad ON ad.id = s.agent_id
			WHERE s.tenant_id = ? AND s.user_id = ? AND s.created_at >= ?
			GROUP BY s.agent_code, ad.name
			ORDER BY COUNT(*) DESC
			LIMIT 1`, tenantID, *userScope, since).Scan(&favAgent).Error

		type recSession struct {
			ID        uuid.UUID `gorm:"column:id"`
			AgentCode string    `gorm:"column:agent_code"`
			AgentName string    `gorm:"column:agent_name"`
			Title     string    `gorm:"column:title"`
			CreatedAt time.Time `gorm:"column:created_at"`
		}
		var recs []recSession
		_ = r.db.Raw(`
			SELECT s.id, s.agent_code, COALESCE(NULLIF(ad.name, ''), s.agent_code) AS agent_name, s.title, s.created_at
			FROM chat_sessions s
			LEFT JOIN agent_definitions ad ON ad.id = s.agent_id
			WHERE s.tenant_id = ? AND s.user_id = ? AND s.created_at >= ?
			ORDER BY s.updated_at DESC
			LIMIT 5`, tenantID, *userScope, since).Scan(&recs).Error

		recentSessions := make([]dto.DashboardRecentSessionDTO, 0, len(recs))
		for _, row := range recs {
			recentSessions = append(recentSessions, dto.DashboardRecentSessionDTO{
				ID:        row.ID,
				AgentCode: row.AgentCode,
				AgentName: row.AgentName,
				Title:     row.Title,
				CreatedAt: apptime.FormatRFC3339(row.CreatedAt),
			})
		}

		return &dto.DashboardAgentOverviewData{
			Role:            "business",
			MySessionsCount: mySessions,
			MyMessagesCount: myMessages,
			MyLikesCount:    myLikes,
			FavoriteAgent:   favAgent,
			RecentSessions:  recentSessions,
		}, nil
	}

	// 租户视角 (tenant_admin，近一周)
	var totalSessions int64
	_ = r.db.Model(&model.ChatSession{}).Where("tenant_id = ? AND created_at >= ?", tenantID, since).Count(&totalSessions).Error

	var totalMessages int64
	_ = r.db.Model(&model.ChatMessage{}).Where("tenant_id = ? AND role = 'user' AND created_at >= ?", tenantID, since).Count(&totalMessages).Error

	var activeUsers int64
	_ = r.db.Model(&model.ChatSession{}).Where("tenant_id = ? AND created_at >= ?", tenantID, since).Select("COUNT(DISTINCT user_id)").Scan(&activeUsers).Error

	var totalAssistantMessages int64
	_ = r.db.Model(&model.ChatMessage{}).Where("tenant_id = ? AND role = 'assistant' AND created_at >= ?", tenantID, since).Count(&totalAssistantMessages).Error

	var totalLikes, totalDislikes int64
	_ = r.db.Model(&model.ChatMessage{}).Where("tenant_id = ? AND feedback = 'like' AND created_at >= ?", tenantID, since).Count(&totalLikes).Error
	_ = r.db.Model(&model.ChatMessage{}).Where("tenant_id = ? AND feedback = 'dislike' AND created_at >= ?", tenantID, since).Count(&totalDislikes).Error

	// 解答满意率：未主动点踩的回复均视为默认认可满意
	rate := 100.0
	if totalAssistantMessages > 0 {
		satisfiedCount := totalAssistantMessages - totalDislikes
		if satisfiedCount < 0 {
			satisfiedCount = 0
		}
		rate = float64(satisfiedCount) / float64(totalAssistantMessages) * 100.0
	}

	type rankRow struct {
		AgentCode    string `gorm:"column:agent_code"`
		AgentName    string `gorm:"column:agent_name"`
		SessionCount int64  `gorm:"column:session_count"`
		MessageCount int64  `gorm:"column:message_count"`
	}
	var ranks []rankRow
	_ = r.db.Raw(`
		SELECT s.agent_code,
		       COALESCE(NULLIF(MAX(ad.name), ''), s.agent_code) AS agent_name,
		       COUNT(DISTINCT s.id) AS session_count,
		       COUNT(m.id) FILTER (WHERE m.role = 'user') AS message_count
		FROM chat_sessions s
		LEFT JOIN agent_definitions ad ON ad.id = s.agent_id
		LEFT JOIN chat_messages m ON m.session_id = s.id AND m.tenant_id = s.tenant_id
		WHERE s.tenant_id = ? AND s.created_at >= ?
		GROUP BY s.agent_code
		ORDER BY session_count DESC
		LIMIT 5`, tenantID, since).Scan(&ranks).Error

	agentRanks := make([]dto.DashboardAgentRankDTO, 0, len(ranks))
	for _, row := range ranks {
		agentRanks = append(agentRanks, dto.DashboardAgentRankDTO{
			AgentCode:    row.AgentCode,
			AgentName:    row.AgentName,
			SessionCount: row.SessionCount,
			MessageCount: row.MessageCount,
		})
	}

	return &dto.DashboardAgentOverviewData{
		Role:             "tenant_admin",
		TotalSessions:    totalSessions,
		TotalMessages:    totalMessages,
		ActiveUsersCount: activeUsers,
		TotalLikes:       totalLikes,
		TotalDislikes:    totalDislikes,
		SatisfactionRate: rate,
		AgentUsageRank:   agentRanks,
	}, nil
}

// ListRecentChatActivities 获取最近的智能体对话动态
func (r *ChatRepo) ListRecentChatActivities(tenantID uuid.UUID, userScope *uuid.UUID, limit int) ([]dto.ActivityItemEnriched, error) {
	if limit <= 0 {
		limit = 10
	}
	type actRow struct {
		ID        uuid.UUID `gorm:"column:id"`
		Title     string    `gorm:"column:title"`
		AgentName string    `gorm:"column:agent_name"`
		UserName  string    `gorm:"column:user_name"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}
	var rows []actRow
	q := r.db.Table("chat_sessions s").
		Select(`
			s.id,
			s.title,
			COALESCE(NULLIF(ad.name, ''), s.agent_code) AS agent_name,
			COALESCE(NULLIF(u.display_name, ''), u.username, '用户') AS user_name,
			s.updated_at AS created_at
		`).
		Joins("JOIN users u ON u.id = s.user_id").
		Joins("LEFT JOIN agent_definitions ad ON ad.id = s.agent_id").
		Where("s.tenant_id = ?", tenantID)

	if userScope != nil {
		q = q.Where("s.user_id = ?", *userScope)
	}

	if err := q.Order("s.updated_at DESC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	acts := make([]dto.ActivityItemEnriched, 0, len(rows))
	for _, r := range rows {
		acts = append(acts, dto.ActivityItemEnriched{
			ID:        r.ID.String(),
			Kind:      "chat",
			Title:     r.Title,
			UserName:  r.UserName,
			AgentName: r.AgentName,
			CreatedAt: apptime.FormatRFC3339(r.CreatedAt),
		})
	}
	return acts, nil
}

// CountByDepartment 按用户所属部门统计智能体会话数（租户仪表盘使用）
func (r *ChatRepo) CountByDepartment(tenantID uuid.UUID) ([]DeptCount, error) {
	sql := `
SELECT COALESCE(d.name, '未分配') AS department,
       COUNT(DISTINCT cs.id)::bigint AS count
FROM chat_sessions cs
JOIN users u ON u.id = cs.user_id
LEFT JOIN org_members om ON om.user_id = u.id AND om.tenant_id = cs.tenant_id AND om.status = 'active'
LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = cs.tenant_id
WHERE cs.tenant_id = ?
  AND EXISTS (SELECT 1 FROM chat_messages cm WHERE cm.session_id = cs.id)
GROUP BY d.name
ORDER BY count DESC`

	var rows []DeptCount
	err := r.db.Raw(sql, tenantID).Scan(&rows).Error
	return rows, err
}
