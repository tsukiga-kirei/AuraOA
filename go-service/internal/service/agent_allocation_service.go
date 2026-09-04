package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/agenttools"
	"auraoa/go-service/internal/pkg/crypto"
	"auraoa/go-service/internal/repository"
)

// AgentAllocationService 负责平台目录展示、系统管理员分配及租户管理员再分配管理
type AgentAllocationService struct {
	agentRepo  *repository.AgentRepo
	tenantRepo *repository.TenantRepo
	orgRepo    *repository.OrgRepo
}

// NewAgentAllocationService 初始化 AgentAllocationService
func NewAgentAllocationService(
	agentRepo *repository.AgentRepo,
	tenantRepo *repository.TenantRepo,
	orgRepo *repository.OrgRepo,
) *AgentAllocationService {
	return &AgentAllocationService{
		agentRepo:  agentRepo,
		tenantRepo: tenantRepo,
		orgRepo:    orgRepo,
	}
}

// GetAgentCatalog 获取平台目录（系统工具、内置智能体、内置 Skills）
func (s *AgentAllocationService) GetAgentCatalog(ctx context.Context) (*dto.AgentCatalogItemDTO, error) {
	// 1. 系统工具目录
	toolSpecs := agenttools.GetAllToolSpecs()
	toolCatalog := make([]dto.SystemToolCatalogItem, 0, len(toolSpecs))
	for _, spec := range toolSpecs {
		paramBytes, _ := json.Marshal(spec.Parameters)
		toolCatalog = append(toolCatalog, dto.SystemToolCatalogItem{
			ToolCode:    spec.Code,
			Name:        spec.Name,
			Description: spec.Description,
			UIKind:      spec.UIKind,
			OARequired:  spec.OARequired,
			Risk:        spec.Risk,
			Parameters:  string(paramBytes),
		})
	}

	// 2. 平台种子智能体
	allSeeds, err := s.agentRepo.ListAgentsByTenant(uuid.Nil)
	if err != nil {
		return nil, err
	}
	agentCatalog := make([]dto.AgentDefinitionDTO, 0)
	for _, a := range allSeeds {
		if a.IsSystem {
			toolCodes := make([]string, 0, len(a.ToolBindings))
			for _, b := range a.ToolBindings {
				toolCodes = append(toolCodes, b.ToolCode)
			}
			agentCatalog = append(agentCatalog, dto.AgentDefinitionDTO{
				ID:           a.ID,
				AgentCode:    a.AgentCode,
				Name:         a.Name,
				Description:  a.Description,
				SystemPrompt: a.SystemPrompt,
				Enabled:      a.Enabled,
				IsSystem:     a.IsSystem,
				ToolCodes:    toolCodes,
				CreatedAt:    a.CreatedAt,
				UpdatedAt:    a.UpdatedAt,
			})
		}
	}

	// 3. 平台内置 Skills
	allSkills, _ := s.agentRepo.ListSkills(uuid.Nil)
	skillCatalog := make([]dto.SkillItemDTO, 0)
	for _, sk := range allSkills {
		if sk.TenantID == nil {
			skillCatalog = append(skillCatalog, dto.SkillItemDTO{
				ID:          sk.ID,
				SkillCode:   sk.SkillCode,
				Name:        sk.Name,
				Description: sk.Description,
				Content:     sk.Content,
				Enabled:     sk.Enabled,
				IsSystem:    true,
				CreatedAt:   sk.CreatedAt,
			})
		}
	}

	return &dto.AgentCatalogItemDTO{
		ToolCatalog:  toolCatalog,
		AgentCatalog: agentCatalog,
		SkillCatalog: skillCatalog,
	}, nil
}

// GetTenantAllocation 查询某租户的系统分配配额
func (s *AgentAllocationService) GetTenantAllocation(ctx context.Context, tenantID uuid.UUID) (*dto.TenantChatAllocationDTO, error) {
	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil {
		return nil, err
	}
	tenant, _ := s.tenantRepo.FindByID(tenantID)

	chatEnabled := true
	chatRetentionDays := 90
	var primaryModelID, fallbackModelID *uuid.UUID
	if tenant != nil {
		chatEnabled = tenant.ChatEnabled
		chatRetentionDays = tenant.ChatRetentionDays
		primaryModelID = tenant.ChatPrimaryModelID
		fallbackModelID = tenant.ChatFallbackModelID
	}

	return &dto.TenantChatAllocationDTO{
		TenantID:          tenantID,
		ChatEnabled:       chatEnabled,
		ChatRetentionDays: chatRetentionDays,
		PrimaryModelID:    primaryModelID,
		FallbackModelID:   fallbackModelID,
		AgentCodes:        repository.ParseJSONSlice(alloc.AgentCodes),
		ToolCodes:         repository.ParseJSONSlice(alloc.ToolCodes),
		SkillCodes:        repository.ParseJSONSlice(alloc.SkillCodes),
		AllowCustomSkills: alloc.AllowCustomSkills,
		AllowTenantMCP:    alloc.AllowTenantMCP,
		MaxMCPServers:     alloc.MaxMCPServers,
		MCPTemplateIDs:    repository.ParseJSONSlice(alloc.MCPTemplateIDs),
	}, nil
}

// UpdateTenantAllocation 系统管理员配置租户配额
func (s *AgentAllocationService) UpdateTenantAllocation(ctx context.Context, tenantID uuid.UUID, req *dto.UpdateTenantChatAllocationRequest) error {
	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil {
		return err
	}

	// 1. 更新 tenant 表的 chat 字段
	tenantUpdates := map[string]interface{}{}
	if req.ChatEnabled != nil {
		tenantUpdates["chat_enabled"] = *req.ChatEnabled
	}
	if req.ChatRetentionDays != nil {
		tenantUpdates["chat_retention_days"] = *req.ChatRetentionDays
	}
	if req.PrimaryModelID != nil {
		tenantUpdates["chat_primary_model_id"] = *req.PrimaryModelID
	}
	if req.FallbackModelID != nil {
		tenantUpdates["chat_fallback_model_id"] = *req.FallbackModelID
	}
	if len(tenantUpdates) > 0 {
		if err := s.tenantRepo.UpdateFields(tenantID, tenantUpdates); err != nil {
			return fmt.Errorf("更新租户对话配置失败: %w", err)
		}
	}

	// 2. 更新配额表
	if req.AgentCodes != nil {
		b, _ := json.Marshal(req.AgentCodes)
		alloc.AgentCodes = datatypes.JSON(b)
	}
	if req.ToolCodes != nil {
		b, _ := json.Marshal(req.ToolCodes)
		alloc.ToolCodes = datatypes.JSON(b)
	}
	if req.SkillCodes != nil {
		b, _ := json.Marshal(req.SkillCodes)
		alloc.SkillCodes = datatypes.JSON(b)
	}
	if req.AllowCustomSkills != nil {
		alloc.AllowCustomSkills = *req.AllowCustomSkills
	}
	if req.AllowTenantMCP != nil {
		alloc.AllowTenantMCP = *req.AllowTenantMCP
	}
	if req.MaxMCPServers != nil {
		alloc.MaxMCPServers = *req.MaxMCPServers
	}
	if req.MCPTemplateIDs != nil {
		b, _ := json.Marshal(req.MCPTemplateIDs)
		alloc.MCPTemplateIDs = datatypes.JSON(b)
	}

	return s.agentRepo.SaveTenantAllocation(alloc)
}

// ---------------- 租户管理员：智能体 CRUD ----------------

// ListTenantAgents 租户管理员查看本租户智能体列表
func (s *AgentAllocationService) ListTenantAgents(ctx context.Context, tenantID uuid.UUID) ([]dto.AgentDefinitionDTO, error) {
	agents, err := s.agentRepo.ListAgentsByTenant(tenantID)
	if err != nil {
		return nil, err
	}

	res := make([]dto.AgentDefinitionDTO, 0, len(agents))
	for _, a := range agents {
		toolCodes := make([]string, 0, len(a.ToolBindings))
		for _, b := range a.ToolBindings {
			toolCodes = append(toolCodes, b.ToolCode)
		}
		res = append(res, dto.AgentDefinitionDTO{
			ID:           a.ID,
			TenantID:     a.TenantID,
			AgentCode:    a.AgentCode,
			Name:         a.Name,
			Description:  a.Description,
			SystemPrompt: a.SystemPrompt,
			Enabled:      a.Enabled,
			IsSystem:     a.IsSystem,
			ToolCodes:    toolCodes,
			CreatedAt:    a.CreatedAt,
			UpdatedAt:    a.UpdatedAt,
		})
	}
	return res, nil
}

// CreateTenantAgent 租户管理员新建智能体
func (s *AgentAllocationService) CreateTenantAgent(ctx context.Context, tenantID uuid.UUID, req *dto.CreateAgentRequest) (*dto.AgentDefinitionDTO, error) {
	// 校验绑定的工具是否都在系统管理员分配给租户的配额内
	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil {
		return nil, err
	}
	quotaTools := make(map[string]bool)
	for _, code := range repository.ParseJSONSlice(alloc.ToolCodes) {
		quotaTools[code] = true
	}

	for _, tc := range req.ToolCodes {
		if !quotaTools[tc] {
			return nil, fmt.Errorf("工具「%s」超出系统管理员授予的租户配额", tc)
		}
	}

	agent := model.AgentDefinition{
		TenantID:     &tenantID,
		AgentCode:    req.AgentCode,
		Name:         req.Name,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Enabled:      req.Enabled,
		IsSystem:     false,
	}

	if err := s.agentRepo.CreateAgent(&agent); err != nil {
		return nil, fmt.Errorf("创建智能体失败: %w", err)
	}

	// 绑定工具
	_ = s.agentRepo.ReplaceAgentToolBindings(tenantID, agent.ID, req.ToolCodes)

	return &dto.AgentDefinitionDTO{
		ID:           agent.ID,
		TenantID:     agent.TenantID,
		AgentCode:    agent.AgentCode,
		Name:         agent.Name,
		Description:  agent.Description,
		SystemPrompt: agent.SystemPrompt,
		Enabled:      agent.Enabled,
		IsSystem:     agent.IsSystem,
		ToolCodes:    req.ToolCodes,
		CreatedAt:    agent.CreatedAt,
		UpdatedAt:    agent.UpdatedAt,
	}, nil
}

// UpdateTenantAgent 租户管理员更新智能体
func (s *AgentAllocationService) UpdateTenantAgent(ctx context.Context, tenantID, agentID uuid.UUID, req *dto.UpdateAgentRequest) error {
	agent, err := s.agentRepo.GetAgentByID(agentID)
	if err != nil || agent == nil {
		return fmt.Errorf("智能体不存在")
	}

	// 系统种子智能体不允许非租户操作，且不可删除
	if agent.TenantID != nil && *agent.TenantID != tenantID {
		return fmt.Errorf("无权修改该智能体")
	}

	updates := map[string]interface{}{}
	if req.Name != nil && *req.Name != "" {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.SystemPrompt != nil {
		updates["system_prompt"] = *req.SystemPrompt
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if len(updates) > 0 {
		if err := s.agentRepo.UpdateAgent(tenantID, agentID, updates); err != nil {
			return err
		}
	}

	if req.ToolCodes != nil {
		// 校验配额
		alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
		if err != nil {
			return err
		}
		quotaTools := make(map[string]bool)
		for _, code := range repository.ParseJSONSlice(alloc.ToolCodes) {
			quotaTools[code] = true
		}

		for _, tc := range *req.ToolCodes {
			if !quotaTools[tc] {
				return fmt.Errorf("工具「%s」超出系统管理员授予的租户配额", tc)
			}
		}

		if err := s.agentRepo.ReplaceAgentToolBindings(tenantID, agentID, *req.ToolCodes); err != nil {
			return err
		}
	}

	return nil
}

// DeleteTenantAgent 租户管理员删除自定义智能体
func (s *AgentAllocationService) DeleteTenantAgent(ctx context.Context, tenantID, agentID uuid.UUID) error {
	agent, err := s.agentRepo.GetAgentByID(agentID)
	if err != nil || agent == nil {
		return fmt.Errorf("智能体不存在")
	}
	if agent.IsSystem {
		return fmt.Errorf("系统内置种子智能体不可删除，如不需要可设为停用")
	}
	return s.agentRepo.DeleteAgent(tenantID, agentID)
}

// ---------------- 租户管理员：MCP Servers ----------------

func (s *AgentAllocationService) ListMCPServers(ctx context.Context, tenantID uuid.UUID) ([]dto.MCPServerDTO, error) {
	servers, err := s.agentRepo.ListMCPServers(tenantID)
	if err != nil {
		return nil, err
	}
	res := make([]dto.MCPServerDTO, 0, len(servers))
	for _, sv := range servers {
		res = append(res, dto.MCPServerDTO{
			ID:            sv.ID,
			ServerCode:    sv.ServerCode,
			Name:          sv.Name,
			Description:   sv.Description,
			TransportType: sv.TransportType,
			EndpointURL:   sv.EndpointURL,
			Enabled:       sv.Enabled,
			CachedTools:   sv.CachedTools,
			LastSyncedAt:  sv.LastSyncedAt,
			CreatedAt:     sv.CreatedAt,
		})
	}
	return res, nil
}

func (s *AgentAllocationService) SaveMCPServer(ctx context.Context, tenantID uuid.UUID, req *dto.SaveMCPServerRequest) (*dto.MCPServerDTO, error) {
	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil || alloc == nil || !alloc.AllowTenantMCP {
		return nil, fmt.Errorf("当前租户未获得自建 MCP 权限")
	}

	encHeaders := ""
	if req.Headers != "" {
		var encErr error
		encHeaders, encErr = crypto.Encrypt(req.Headers)
		if encErr != nil {
			return nil, fmt.Errorf("加密请求头失败: %w", encErr)
		}
	}

	trans := req.TransportType
	if trans == "" {
		trans = "http"
	}

	server := model.MCPServer{
		TenantID:         &tenantID,
		ServerCode:       req.ServerCode,
		Name:             req.Name,
		Description:      req.Description,
		TransportType:    trans,
		EndpointURL:      req.EndpointURL,
		HeadersEncrypted: encHeaders,
		Enabled:          req.Enabled,
		CachedTools:      datatypes.JSON([]byte(`[]`)),
	}

	if err := s.agentRepo.CreateMCPServer(&server); err != nil {
		return nil, err
	}

	return &dto.MCPServerDTO{
		ID:            server.ID,
		ServerCode:    server.ServerCode,
		Name:          server.Name,
		Description:   server.Description,
		TransportType: server.TransportType,
		EndpointURL:   server.EndpointURL,
		Enabled:       server.Enabled,
		CachedTools:   server.CachedTools,
		CreatedAt:     server.CreatedAt,
	}, nil
}

func (s *AgentAllocationService) DeleteMCPServer(ctx context.Context, tenantID, serverID uuid.UUID) error {
	return s.agentRepo.DeleteMCPServer(tenantID, serverID)
}

// ---------------- 租户管理员：Skills ----------------

func (s *AgentAllocationService) ListSkills(ctx context.Context, tenantID uuid.UUID) ([]dto.SkillItemDTO, error) {
	skills, err := s.agentRepo.ListSkills(tenantID)
	if err != nil {
		return nil, err
	}
	res := make([]dto.SkillItemDTO, 0, len(skills))
	for _, sk := range skills {
		res = append(res, dto.SkillItemDTO{
			ID:          sk.ID,
			SkillCode:   sk.SkillCode,
			Name:        sk.Name,
			Description: sk.Description,
			Content:     sk.Content,
			Enabled:     sk.Enabled,
			IsSystem:    sk.TenantID == nil,
			CreatedAt:   sk.CreatedAt,
		})
	}
	return res, nil
}

func (s *AgentAllocationService) SaveSkill(ctx context.Context, tenantID uuid.UUID, req *dto.SaveSkillRequest) (*dto.SkillItemDTO, error) {
	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil || alloc == nil || !alloc.AllowCustomSkills {
		return nil, fmt.Errorf("当前租户未获得自定义 Skills 权限")
	}

	skill := model.AgentSkill{
		TenantID:    &tenantID,
		SkillCode:   req.SkillCode,
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		Enabled:     req.Enabled,
	}

	if err := s.agentRepo.CreateSkill(&skill); err != nil {
		return nil, err
	}

	return &dto.SkillItemDTO{
		ID:          skill.ID,
		SkillCode:   skill.SkillCode,
		Name:        skill.Name,
		Description: skill.Description,
		Content:     skill.Content,
		Enabled:     skill.Enabled,
		IsSystem:    false,
		CreatedAt:   skill.CreatedAt,
	}, nil
}

func (s *AgentAllocationService) DeleteSkill(ctx context.Context, tenantID, skillID uuid.UUID) error {
	return s.agentRepo.DeleteSkill(tenantID, skillID)
}
