package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

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
	chatRepo   *repository.ChatRepo
}

// NewAgentAllocationService 初始化 AgentAllocationService
func NewAgentAllocationService(
	agentRepo *repository.AgentRepo,
	tenantRepo *repository.TenantRepo,
	orgRepo *repository.OrgRepo,
	chatRepo *repository.ChatRepo,
) *AgentAllocationService {
	return &AgentAllocationService{
		agentRepo:  agentRepo,
		tenantRepo: tenantRepo,
		orgRepo:    orgRepo,
		chatRepo:   chatRepo,
	}
}

func parseQuickQuestions(data datatypes.JSON) []dto.QuickQuestionItem {
	var items []dto.QuickQuestionItem
	if len(data) > 0 {
		_ = json.Unmarshal(data, &items)
	}
	if items == nil {
		items = []dto.QuickQuestionItem{}
	}
	return items
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
				ID:             a.ID,
				AgentCode:      a.AgentCode,
				Name:           a.Name,
				Description:    a.Description,
				SystemPrompt:   a.SystemPrompt,
				Enabled:        a.Enabled,
				IsSystem:       a.IsSystem,
				QuickQuestions: parseQuickQuestions(a.QuickQuestions),
				ToolCodes:      toolCodes,
				CreatedAt:      a.CreatedAt,
				UpdatedAt:      a.UpdatedAt,
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

	if req.ChatRetentionDays != nil && (*req.ChatRetentionDays < 1 || *req.ChatRetentionDays > 3650) {
		return fmt.Errorf("会话保留天数须为 1–3650")
	}
	if req.MaxMCPServers != nil && (*req.MaxMCPServers < 0 || *req.MaxMCPServers > 100) {
		return fmt.Errorf("MCP 服务数量上限须为 0–100")
	}
	// 1. 更新 tenant 表的 chat 字段
	tenantUpdates := map[string]interface{}{}
	if req.ChatEnabled != nil {
		tenantUpdates["chat_enabled"] = *req.ChatEnabled
	}
	if req.ChatRetentionDays != nil {
		tenantUpdates["chat_retention_days"] = *req.ChatRetentionDays
	}
	if len(req.PrimaryModelID) > 0 {
		var id *uuid.UUID
		if err := json.Unmarshal(req.PrimaryModelID, &id); err != nil {
			return fmt.Errorf("模型 ID 无效")
		}
		tenantUpdates["chat_primary_model_id"] = id
	}
	if len(req.FallbackModelID) > 0 {
		var id *uuid.UUID
		if err := json.Unmarshal(req.FallbackModelID, &id); err != nil {
			return fmt.Errorf("模型 ID 无效")
		}
		tenantUpdates["chat_fallback_model_id"] = id
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

	return s.agentRepo.SaveTenantChatSettings(alloc, tenantUpdates)
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
			ID:             a.ID,
			TenantID:       a.TenantID,
			AgentCode:      a.AgentCode,
			Name:           a.Name,
			Description:    a.Description,
			SystemPrompt:   a.SystemPrompt,
			Enabled:        a.Enabled,
			IsSystem:       a.IsSystem,
			QuickQuestions: parseQuickQuestions(a.QuickQuestions),
			ToolCodes:      toolCodes,
			CreatedAt:      a.CreatedAt,
			UpdatedAt:      a.UpdatedAt,
		})
	}
	return res, nil
}

// CreateTenantAgent 租户管理员新建智能体
func (s *AgentAllocationService) CreateTenantAgent(ctx context.Context, tenantID uuid.UUID, req *dto.CreateAgentRequest) (*dto.AgentDefinitionDTO, error) {
	if existing, _ := s.agentRepo.GetAgentByCode(tenantID, req.AgentCode); existing != nil {
		return nil, fmt.Errorf("智能体编码已存在，请编辑已有配置或使用新的方案编码")
	}

	// 校验绑定的工具是否都在系统管理员分配给租户的配额内
	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.validateBindings(tenantID, alloc, req.ToolCodes); err != nil {
		return nil, err
	}

	qqJSON, _ := json.Marshal(req.QuickQuestions)
	if len(qqJSON) == 0 {
		qqJSON = []byte("[]")
	}

	agent := model.AgentDefinition{
		TenantID:       &tenantID,
		AgentCode:      req.AgentCode,
		Name:           req.Name,
		Description:    req.Description,
		SystemPrompt:   req.SystemPrompt,
		Enabled:        req.Enabled,
		IsSystem:       false,
		QuickQuestions: qqJSON,
	}

	if err := s.agentRepo.CreateAgentWithBindings(tenantID, &agent, req.ToolCodes); err != nil {
		return nil, fmt.Errorf("创建智能体失败: %w", err)
	}

	return &dto.AgentDefinitionDTO{
		ID:             agent.ID,
		TenantID:       agent.TenantID,
		AgentCode:      agent.AgentCode,
		Name:           agent.Name,
		Description:    agent.Description,
		SystemPrompt:   agent.SystemPrompt,
		Enabled:        agent.Enabled,
		IsSystem:       agent.IsSystem,
		QuickQuestions: req.QuickQuestions,
		ToolCodes:      req.ToolCodes,
		CreatedAt:      agent.CreatedAt,
		UpdatedAt:      agent.UpdatedAt,
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
	if req.QuickQuestions != nil {
		qqJSON, _ := json.Marshal(*req.QuickQuestions)
		updates["quick_questions"] = qqJSON
	}

	if req.ToolCodes != nil {
		alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
		if err != nil {
			return err
		}
		if err := s.validateBindings(tenantID, alloc, *req.ToolCodes); err != nil {
			return err
		}
	}
	return s.agentRepo.SaveTenantAgent(tenantID, agent, updates, req.ToolCodes)
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
			AgentCodes:    s.agentCodesForPrefix(tenantID, "mcp:"+sv.ServerCode),
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
	endpoint, parseErr := url.Parse(req.EndpointURL)
	if parseErr != nil || endpoint.Host == "" || endpoint.User != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, fmt.Errorf("MCP 地址必须是有效的 HTTP(S) 地址")
	}
	if req.TransportType != "" && req.TransportType != "http" {
		return nil, fmt.Errorf("目前仅支持 Streamable HTTP MCP 服务")
	}
	if req.Headers != "" {
		var headers map[string]string
		if json.Unmarshal([]byte(req.Headers), &headers) != nil {
			return nil, fmt.Errorf("请求头必须为 JSON 字符串对象")
		}
	}
	if req.ID != uuid.Nil {
		server, err := s.agentRepo.GetMCPServerByID(tenantID, req.ID)
		if err != nil || server.TenantID == nil || *server.TenantID != tenantID {
			return nil, fmt.Errorf("无权修改该 MCP 服务")
		}
		if server.ServerCode != req.ServerCode {
			return nil, fmt.Errorf("MCP 标识码创建后不能修改")
		}
		updates := map[string]interface{}{"name": req.Name, "description": req.Description, "endpoint_url": req.EndpointURL, "transport_type": "http", "enabled": req.Enabled}
		if req.Headers != "" {
			encrypted, err := crypto.Encrypt(req.Headers)
			if err != nil {
				return nil, err
			}
			updates["headers_encrypted"] = encrypted
		}
		if server.EndpointURL != req.EndpointURL || req.Headers != "" {
			updates["cached_tools"] = datatypes.JSON([]byte(`[]`))
			updates["last_synced_at"] = nil
		}
		if err := s.agentRepo.UpdateMCPServer(tenantID, server.ID, updates); err != nil {
			return nil, err
		}
		if err := s.syncMCPBindings(ctx, tenantID, server, req.AgentCodes); err != nil {
			return nil, err
		}
		items, err := s.ListMCPServers(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.ID == server.ID {
				return &item, nil
			}
		}
		return nil, fmt.Errorf("MCP 服务不存在")
	}

	servers, err := s.agentRepo.ListMCPServers(tenantID)
	if err != nil {
		return nil, err
	}
	count := 0
	for _, server := range servers {
		if server.TenantID != nil && *server.TenantID == tenantID {
			count++
		}
	}
	if count >= alloc.MaxMCPServers {
		return nil, fmt.Errorf("已达到租户 MCP 服务数量上限（%d）", alloc.MaxMCPServers)
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

	if err := s.agentRepo.CreateMCPServerWithinQuota(tenantID, &server); err != nil {
		return nil, err
	}
	if err := s.syncMCPBindings(ctx, tenantID, &server, req.AgentCodes); err != nil {
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
		AgentCodes:    req.AgentCodes,
		CreatedAt:     server.CreatedAt,
	}, nil
}

func (s *AgentAllocationService) DeleteMCPServer(ctx context.Context, tenantID, serverID uuid.UUID) error {
	server, err := s.agentRepo.GetMCPServerByID(tenantID, serverID)
	if err == nil && server != nil {
		_ = s.syncMCPBindings(ctx, tenantID, server, nil)
	}
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
			AgentCodes:  s.agentCodesForPrefix(tenantID, "skill:"+sk.SkillCode),
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
	if len(req.Content) > 65536 {
		return nil, fmt.Errorf("技能内容不能超过 64 KB")
	}
	if req.ID != uuid.Nil {
		skill, err := s.agentRepo.GetSkillByCode(tenantID, req.SkillCode)
		if err != nil || skill.ID != req.ID || skill.TenantID == nil || *skill.TenantID != tenantID {
			return nil, fmt.Errorf("无权修改该技能，标识码创建后不能修改")
		}
		if err := s.agentRepo.UpdateSkill(tenantID, skill.ID, map[string]interface{}{"name": req.Name, "description": req.Description, "content": req.Content, "enabled": req.Enabled}); err != nil {
			return nil, err
		}
		if err := s.syncSkillBindings(ctx, tenantID, skill.SkillCode, req.AgentCodes); err != nil {
			return nil, err
		}
		return &dto.SkillItemDTO{ID: skill.ID, SkillCode: skill.SkillCode, Name: req.Name, Description: req.Description, Content: req.Content, Enabled: req.Enabled, IsSystem: false, AgentCodes: req.AgentCodes, CreatedAt: skill.CreatedAt}, nil
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
	if err := s.syncSkillBindings(ctx, tenantID, skill.SkillCode, req.AgentCodes); err != nil {
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
		AgentCodes:  req.AgentCodes,
		CreatedAt:   skill.CreatedAt,
	}, nil
}

func (s *AgentAllocationService) DeleteSkill(ctx context.Context, tenantID, skillID uuid.UUID) error {
	skills, _ := s.agentRepo.ListSkills(tenantID)
	for _, sk := range skills {
		if sk.ID == skillID {
			_ = s.syncSkillBindings(ctx, tenantID, sk.SkillCode, nil)
			break
		}
	}
	return s.agentRepo.DeleteSkill(tenantID, skillID)
}

// GetTenantCatalog 返回租户实际可装配的系统工具、MCP 与技能目录。
func (s *AgentAllocationService) GetTenantCatalog(ctx context.Context, tenantID uuid.UUID) (*dto.AgentCatalogItemDTO, error) {
	catalog, err := s.GetAgentCatalog(ctx)
	if err != nil {
		return nil, err
	}
	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil {
		return nil, err
	}
	filtered := append([]dto.SystemToolCatalogItem{}, catalog.ToolCatalog...)
	if alloc.AllowTenantMCP {
		servers, err := s.agentRepo.ListMCPServers(tenantID)
		if err != nil {
			return nil, err
		}
		for _, server := range servers {
			if !server.Enabled || server.TenantID == nil {
				continue
			}
			for _, def := range ConvertMCPToolsToDefinitions(server.ServerCode, server.CachedTools) {
				filtered = append(filtered, dto.SystemToolCatalogItem{ToolCode: def.Function.Name, Name: server.Name + " / " + def.Function.Name, Description: def.Function.Description, UIKind: "mcp_generic"})
			}
		}
	}
	catalog.ToolCatalog = filtered
	skills, err := s.ListSkills(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	catalog.SkillCatalog = []dto.SkillItemDTO{}
	for _, sk := range skills {
		allowed := sk.IsSystem || alloc.AllowCustomSkills
		if allowed && sk.Enabled {
			catalog.SkillCatalog = append(catalog.SkillCatalog, sk)
		}
	}
	return catalog, nil
}

func (s *AgentAllocationService) validateBindings(tenantID uuid.UUID, alloc *model.TenantChatAllocation, codes []string) error {
	catalog, err := s.GetTenantCatalog(context.Background(), tenantID)
	if err != nil {
		return err
	}
	allowed := make(map[string]bool)
	for _, tool := range catalog.ToolCatalog {
		allowed[tool.ToolCode] = true
	}
	for _, skill := range catalog.SkillCatalog {
		allowed["skill:"+skill.SkillCode] = true
	}
	for _, code := range codes {
		trimmed := strings.TrimSpace(code)
		if allowed[trimmed] || strings.HasPrefix(trimmed, "mcp:") {
			continue
		}
		return fmt.Errorf("工具或技能「%s」未启用或超出当前租户配额", code)
	}
	return nil
}

func (s *AgentAllocationService) agentCodesForPrefix(tenantID uuid.UUID, prefix string) []string {
	agents, err := s.agentRepo.ListAgentsByTenant(tenantID)
	if err != nil {
		return []string{}
	}
	seen := map[string]bool{}
	var codes []string
	for _, agent := range agents {
		for _, binding := range agent.ToolBindings {
			if strings.HasPrefix(binding.ToolCode, prefix) && !seen[agent.AgentCode] {
				seen[agent.AgentCode] = true
				codes = append(codes, agent.AgentCode)
			}
		}
	}
	return codes
}

func (s *AgentAllocationService) mcpToolCodes(server *model.MCPServer) []string {
	var codes []string
	for _, def := range ConvertMCPToolsToDefinitions(server.ServerCode, server.CachedTools) {
		codes = append(codes, def.Function.Name)
	}
	if len(codes) == 0 {
		return []string{"mcp:" + server.ServerCode}
	}
	return codes
}

func (s *AgentAllocationService) syncMCPBindings(_ context.Context, tenantID uuid.UUID, server *model.MCPServer, agentCodes []string) error {
	return s.syncCapabilityBindings(tenantID, "mcp:"+server.ServerCode, s.mcpToolCodes(server), agentCodes)
}

func (s *AgentAllocationService) syncSkillBindings(_ context.Context, tenantID uuid.UUID, skillCode string, agentCodes []string) error {
	return s.syncCapabilityBindings(tenantID, "skill:"+skillCode, []string{"skill:" + skillCode}, agentCodes)
}

// RefreshMCPAgentBindings 测试连接发现工具后，按原挂载智能体重新写入绑定。
func (s *AgentAllocationService) RefreshMCPAgentBindings(ctx context.Context, tenantID, serverID uuid.UUID) error {
	server, err := s.agentRepo.GetMCPServerByID(tenantID, serverID)
	if err != nil || server == nil {
		return err
	}
	return s.syncMCPBindings(ctx, tenantID, server, s.agentCodesForPrefix(tenantID, "mcp:"+server.ServerCode))
}

func (s *AgentAllocationService) syncCapabilityBindings(tenantID uuid.UUID, prefix string, toolCodes []string, agentCodes []string) error {
	wanted := map[string]bool{}
	for _, code := range agentCodes {
		if code != "" {
			wanted[code] = true
		}
	}
	agents, err := s.agentRepo.ListAgentsByTenant(tenantID)
	if err != nil {
		return err
	}
	for i := range agents {
		agent := &agents[i]
		kept := make([]string, 0, len(agent.ToolBindings))
		had := false
		for _, binding := range agent.ToolBindings {
			if strings.HasPrefix(binding.ToolCode, prefix) {
				had = true
				continue
			}
			kept = append(kept, binding.ToolCode)
		}
		should := wanted[agent.AgentCode]
		if !should && !had {
			continue
		}
		next := kept
		if should {
			next = append(next, toolCodes...)
		}
		if err := s.agentRepo.SaveTenantAgent(tenantID, agent, map[string]interface{}{}, &next); err != nil {
			return err
		}
	}
	return nil
}

// GetAgentUsageStats 汇总租户内各智能体的会话、消息、工具/MCP/Skill 调用与 Token。
func (s *AgentAllocationService) GetAgentUsageStats(ctx context.Context, tenantID uuid.UUID) (*dto.AgentUsageStatsDTO, error) {
	return s.agentRepo.QueryAgentUsageStats(tenantID)
}

// ListTenantAgentSessions 租户管理数据信息页分页查询会话列表
func (s *AgentAllocationService) ListTenantAgentSessions(
	ctx context.Context,
	tenantID uuid.UUID,
	keyword, agentCode, userName, startDate, endDate string,
	page, pageSize int,
) (*dto.TenantAgentSessionListResponse, error) {
	items, total, err := s.chatRepo.ListSessionsByTenant(tenantID, keyword, agentCode, userName, startDate, endDate, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &dto.TenantAgentSessionListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetTenantSessionMessages 租户管理数据信息页查询指定会话的历史聊天记录
func (s *AgentAllocationService) GetTenantSessionMessages(
	ctx context.Context,
	tenantID, sessionID uuid.UUID,
) ([]dto.ChatMessageDTO, error) {
	msgs, err := s.chatRepo.ListMessagesBySession(tenantID, sessionID)
	if err != nil {
		return nil, err
	}
	res := make([]dto.ChatMessageDTO, 0, len(msgs))
	for _, m := range msgs {
		res = append(res, dto.ChatMessageDTO{
			ID:               m.ID,
			SessionID:        m.SessionID,
			Role:             m.Role,
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
			Status:           m.Status,
			ToolCalls:        m.ToolCalls,
			TokenUsage:       m.TokenUsage,
			Feedback:         m.Feedback,
			FeedbackAt:       m.FeedbackAt,
			CreatedAt:        m.CreatedAt,
		})
	}
	return res, nil
}
