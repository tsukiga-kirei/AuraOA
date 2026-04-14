package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/ai"
	"oa-smart-audit/go-service/internal/pkg/crypto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// AIModelService 负责 AI 模型配置的增删改查及连接测试。
type AIModelService struct {
	repo *repository.AIModelRepo
}

// NewAIModelService 初始化 AI 模型配置服务。
func NewAIModelService(repo *repository.AIModelRepo) *AIModelService {
	return &AIModelService{repo: repo}
}

// List 查询所有 AI 模型配置，返回脱敏后的响应列表（API Key 不回显）。
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

// Create 新增 AI 模型配置，API Key 加密存储，未传入时保留空值。
// 对 MaxTokens、ContextWindow、DeployType、Status 等字段设置合理默认值。
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

// Update 按需更新 AI 模型配置字段，仅更新请求中非零值的字段。
// 若传入新的 API Key，则重新加密后覆盖存储。
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

// Delete 删除指定 AI 模型配置，删除前校验记录是否存在。
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

// 将数据库模型转换为 API 响应 DTO，API Key 不回显，Capabilities 反序列化为字符串切片。
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

// TestConnection 根据已保存的模型 ID 测试连接可用性，并将 online/offline 状态持久化到数据库。
// 测试前自动解密 API Key，测试结果无论成功与否都会更新状态字段。
func (s *AIModelService) TestConnection(id uuid.UUID) error {
	m, err := s.repo.FindByID(id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "AI模型不存在")
	}

	// 解密 API Key
	if m.APIKey != "" {
		decrypted, err := crypto.Decrypt(m.APIKey)
		if err != nil {
			return newServiceError(errcode.ErrInternalServer, "API Key解密失败")
		}
		m.APIKey = decrypted
	}

	testErr := s.testAIModel(m)

	// 持久化连接状态
	newStatus := "online"
	if testErr != nil {
		newStatus = "offline"
	}
	_ = s.repo.Update(id, map[string]interface{}{"status": newStatus})

	return testErr
}

// TestConnectionByParams 使用前端传入的明文参数直接测试连接，无需先保存配置。
// 适用于新建或编辑模型时的"测试连接"按钮场景。
func (s *AIModelService) TestConnectionByParams(req *dto.CreateAIModelRequest) error {
	m := &model.AIModelConfig{
		Provider:   req.Provider,
		ModelName:  req.ModelName,
		DeployType: req.DeployType,
		Endpoint:   req.Endpoint,
		APIKey:     req.APIKey, // 前端传入的是明文
		MaxTokens:  req.MaxTokens,
	}
	if m.MaxTokens == 0 {
		m.MaxTokens = 8192
	}

	return s.testAIModel(m)
}

// testAIModel 实际执行连接测试：创建对应部署类型的调用器，发送探测请求，超时 10 秒。
func (s *AIModelService) testAIModel(m *model.AIModelConfig) error {
	caller, err := ai.NewAIModelCaller(m)
	if err != nil {
		return newServiceError(errcode.ErrAIDeployTypeUnsupported, err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := caller.TestConnection(ctx); err != nil {
		return newServiceError(errcode.ErrAIConnectionFailed, err.Error())
	}
	return nil
}
