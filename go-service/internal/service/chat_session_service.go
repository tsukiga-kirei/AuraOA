package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/repository"
)

// ChatSessionService 会话管理服务
type ChatSessionService struct {
	chatRepo    *repository.ChatRepo
	agentRepo   *repository.AgentRepo
	tenantRepo  *repository.TenantRepo
	permService *AgentPermissionService
}

// NewChatSessionService 创建 ChatSessionService
func NewChatSessionService(
	chatRepo *repository.ChatRepo,
	agentRepo *repository.AgentRepo,
	tenantRepo *repository.TenantRepo,
	permService *AgentPermissionService,
) *ChatSessionService {
	return &ChatSessionService{
		chatRepo:    chatRepo,
		agentRepo:   agentRepo,
		tenantRepo:  tenantRepo,
		permService: permService,
	}
}

// GetEffectiveAgents 获取当前用户可用的有效智能体列表
func (s *ChatSessionService) GetEffectiveAgents(ctx context.Context, tenantID, userID uuid.UUID) ([]dto.EffectiveAgentDTO, error) {
	perms, err := s.permService.CalculateEffectivePermissions(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.EffectiveAgentDTO, 0, len(perms.Agents))
	for _, a := range perms.Agents {
		result = append(result, dto.EffectiveAgentDTO{
			ID:          a.ID,
			AgentCode:   a.AgentCode,
			Name:        a.Name,
			Description: a.Description,
			IsSystem:    a.IsSystem,
		})
	}
	return result, nil
}

// CreateSession 创建新会话
func (s *ChatSessionService) CreateSession(ctx context.Context, tenantID, userID uuid.UUID, req *dto.CreateSessionRequest) (*dto.ChatSessionItemDTO, error) {
	// 验证所选 agent 是否在有效权限内
	perms, err := s.permService.CalculateEffectivePermissions(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	var targetAgent *model.AgentDefinition
	for i := range perms.Agents {
		if perms.Agents[i].AgentCode == req.AgentCode {
			targetAgent = &perms.Agents[i]
			break
		}
	}
	if targetAgent == nil {
		return nil, fmt.Errorf("智能体「%s」不存在或您没有使用权限", req.AgentCode)
	}

	title := req.Title
	if title == "" {
		title = "新对话"
	}

	source := req.Source
	if source == "" {
		source = "standalone"
	}

	session := model.ChatSession{
		TenantID:  tenantID,
		UserID:    userID,
		AgentID:   targetAgent.ID,
		AgentCode: targetAgent.AgentCode,
		Title:     title,
		Source:    source,
		ProcessID: req.ProcessID,
		Pinned:    false,
	}

	if err := s.chatRepo.CreateSession(&session); err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	return &dto.ChatSessionItemDTO{
		ID:        session.ID,
		AgentID:   session.AgentID,
		AgentCode: session.AgentCode,
		Title:     session.Title,
		Source:    session.Source,
		ProcessID: session.ProcessID,
		Pinned:    session.Pinned,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}, nil
}

// ListSessions 分页获取当前用户的会话列表
func (s *ChatSessionService) ListSessions(ctx context.Context, tenantID, userID uuid.UUID, keyword string, page, pageSize int) (*dto.ChatSessionListResponse, error) {
	items, total, err := s.chatRepo.ListSessionsByUser(tenantID, userID, keyword, page, pageSize)
	if err != nil {
		return nil, err
	}

	dtos := make([]dto.ChatSessionItemDTO, 0, len(items))
	for _, it := range items {
		dtos = append(dtos, dto.ChatSessionItemDTO{
			ID:        it.ID,
			AgentID:   it.AgentID,
			AgentCode: it.AgentCode,
			Title:     it.Title,
			Source:    it.Source,
			ProcessID: it.ProcessID,
			Pinned:    it.Pinned,
			CreatedAt: it.CreatedAt,
			UpdatedAt: it.UpdatedAt,
		})
	}

	return &dto.ChatSessionListResponse{
		Items:    dtos,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetSessionDetail 获取会话详情与历史消息
func (s *ChatSessionService) GetSessionDetail(ctx context.Context, tenantID, userID, sessionID uuid.UUID) (*dto.ChatSessionDetailResponse, error) {
	session, err := s.chatRepo.GetSessionByID(tenantID, sessionID)
	if err != nil || session == nil {
		return nil, fmt.Errorf("会话不存在")
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("无权访问该会话")
	}

	messages, err := s.chatRepo.ListMessagesBySession(tenantID, sessionID)
	if err != nil {
		return nil, err
	}

	msgDTOs := make([]dto.ChatMessageDTO, 0, len(messages))
	for _, m := range messages {
		msgDTOs = append(msgDTOs, dto.ChatMessageDTO{
			ID:               m.ID,
			SessionID:        m.SessionID,
			Role:             m.Role,
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
			Status:           m.Status,
			ToolCalls:        m.ToolCalls,
			TokenUsage:       m.TokenUsage,
			CreatedAt:        m.CreatedAt,
		})
	}

	return &dto.ChatSessionDetailResponse{
		Session: dto.ChatSessionItemDTO{
			ID:        session.ID,
			AgentID:   session.AgentID,
			AgentCode: session.AgentCode,
			Title:     session.Title,
			Source:    session.Source,
			ProcessID: session.ProcessID,
			Pinned:    session.Pinned,
			CreatedAt: session.CreatedAt,
			UpdatedAt: session.UpdatedAt,
		},
		Messages: msgDTOs,
	}, nil
}

// UpdateSession 更新会话（重命名、置顶）
func (s *ChatSessionService) UpdateSession(ctx context.Context, tenantID, userID, sessionID uuid.UUID, req *dto.UpdateSessionRequest) error {
	session, err := s.chatRepo.GetSessionByID(tenantID, sessionID)
	if err != nil || session == nil {
		return fmt.Errorf("会话不存在")
	}
	if session.UserID != userID {
		return fmt.Errorf("无权修改该会话")
	}

	updates := map[string]interface{}{}
	if req.Title != nil && *req.Title != "" {
		updates["title"] = *req.Title
	}
	if req.Pinned != nil {
		updates["pinned"] = *req.Pinned
	}

	if len(updates) == 0 {
		return nil
	}

	return s.chatRepo.UpdateSession(tenantID, sessionID, updates)
}

// DeleteSession 删除会话
func (s *ChatSessionService) DeleteSession(ctx context.Context, tenantID, userID, sessionID uuid.UUID) error {
	session, err := s.chatRepo.GetSessionByID(tenantID, sessionID)
	if err != nil || session == nil {
		return fmt.Errorf("会话不存在")
	}
	if session.UserID != userID {
		return fmt.Errorf("无权删除该会话")
	}

	return s.chatRepo.DeleteSession(tenantID, sessionID)
}

// CleanExpiredSessions 硬删除超过指定保留天数的会话
func (s *ChatSessionService) CleanExpiredSessions(ctx context.Context, tenantID uuid.UUID, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	cutoff := apptime.Now().AddDate(0, 0, -retentionDays)
	return s.chatRepo.DeleteExpiredSessions(tenantID, cutoff)
}
