package service

import (
	"encoding/json"

	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/crypto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// AIModelService 处理 AI 模型配置的业务逻辑。
type AIModelService struct {
	repo *repository.AIModelRepo
}

func NewAIModelService(repo *repository.AIModelRepo) *AIModelService {
	return &AIModelService{repo: repo}
}

// List 返回所有 AI 模型配置。
func (s *AIModelService) List() ([]dto.AIModelResponse, error) {
	items, err := s.repo.List()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.AIModelResponse, len(items))
	for i := range items {
		result[i] = toAIModelResponse(&items[i])
	}
	return result, nil
}

// Create 创建新的 AI 模型配置。
func (s *AIModelService) Create(req *dto.CreateAIModelRequest) (*dto.AIModelResponse, error) {
	capsJSON, _ := json.Marshal(req.Capabilities)
	if req.Capabilities == nil {
		capsJSON = []byte("[]")
	}

	m := &model.AIModelConfig{
		ID:               uuid.New(),
		Provider:         req.Provider,
		ProviderLabel:    req.ProviderLabel,
		ModelName:        req.ModelName,
		DisplayName:      req.DisplayName,
		DeployType:       req.DeployType,
		Endpoint:         req.Endpoint,
		APIKeyConfigured: req.APIKey != "",
		MaxTokens:        req.MaxTokens,
		ContextWindow:    req.ContextWindow,
		CostPer1kTokens:  req.CostPer1kTokens,
		Enabled:          req.Enabled,
		Description:      req.Description,
		Capabilities:     capsJSON,
	}

	// 加密 API Key
	if req.APIKey != "" {
		encrypted, err := crypto.Encrypt(req.APIKey)
		if err != nil {
			return nil, newServiceError(errcode.ErrInternalServer, "加密失败")
		}
		m.APIKey = encrypted
	}

	// 默认值
	if m.MaxTokens == 0 {
		m.MaxTokens = 8192
	}
	if m.ContextWindow == 0 {
		m.ContextWindow = 131072
	}
	if m.DeployType == "" {
		m.DeployType = "local"
	}
	if m.Status == "" {
		m.Status = "offline"
	}

	if err := s.repo.Create(m); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	resp := toAIModelResponse(m)
	return &resp, nil
}

// Update 更新 AI 模型配置。
func (s *AIModelService) Update(id uuid.UUID, req *dto.UpdateAIModelRequest) (*dto.AIModelResponse, error) {
	_, err := s.repo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "AI模型不存在")
	}

	fields := make(map[string]interface{})
	if req.Provider != "" {
		fields["provider"] = req.Provider
	}
	if req.ProviderLabel != "" {
		fields["provider_label"] = req.ProviderLabel
	}
	if req.ModelName != "" {
		fields["model_name"] = req.ModelName
	}
	if req.DisplayName != "" {
		fields["display_name"] = req.DisplayName
	}
	if req.DeployType != "" {
		fields["deploy_type"] = req.DeployType
	}
	if req.Endpoint != "" {
		fields["endpoint"] = req.Endpoint
	}
	if req.APIKey != "" {
		encrypted, err := crypto.Encrypt(req.APIKey)
		if err != nil {
			return nil, newServiceError(errcode.ErrInternalServer, "加密失败")
		}
		fields["api_key"] = encrypted
		fields["api_key_configured"] = true
	}
	if req.MaxTokens != 0 {
		fields["max_tokens"] = req.MaxTokens
	}
	if req.ContextWindow != 0 {
		fields["context_window"] = req.ContextWindow
	}
	if req.CostPer1kTokens != nil {
		fields["cost_per_1k_tokens"] = *req.CostPer1kTokens
	}
	if req.Enabled != nil {
		fields["enabled"] = *req.Enabled
	}
	if req.Status != "" {
		fields["status"] = req.Status
	}
	if req.Description != "" {
		fields["description"] = req.Description
	}
	if req.Capabilities != nil {
		capsJSON, _ := json.Marshal(req.Capabilities)
		fields["capabilities"] = capsJSON
	}

	if len(fields) > 0 {
		if err := s.repo.Update(id, fields); err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}

	m, err := s.repo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	resp := toAIModelResponse(m)
	return &resp, nil
}

// Delete 删除 AI 模型配置。
func (s *AIModelService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "AI模型不存在")
	}
	if err := s.repo.Delete(id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
}

func toAIModelResponse(m *model.AIModelConfig) dto.AIModelResponse {
	var caps []string
	_ = json.Unmarshal(m.Capabilities, &caps)
	if caps == nil {
		caps = []string{}
	}
	return dto.AIModelResponse{
		ID:               m.ID.String(),
		Provider:         m.Provider,
		ProviderLabel:    m.ProviderLabel,
		ModelName:        m.ModelName,
		DisplayName:      m.DisplayName,
		DeployType:       m.DeployType,
		Endpoint:         m.Endpoint,
		APIKeyConfigured: m.APIKeyConfigured,
		MaxTokens:        m.MaxTokens,
		ContextWindow:    m.ContextWindow,
		CostPer1kTokens:  m.CostPer1kTokens,
		Status:           m.Status,
		Enabled:          m.Enabled,
		Description:      m.Description,
		Capabilities:     caps,
		CreatedAt:        m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
