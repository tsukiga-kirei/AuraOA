package service

import (
	"context"

	"github.com/google/uuid"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/repository"
)

// AgentPermissionService 计算系统管理员配额、租户启用状态、组织角色授权等多层求交后的有效智能体与有效工具集
type AgentPermissionService struct {
	agentRepo  *repository.AgentRepo
	tenantRepo *repository.TenantRepo
	orgRepo    *repository.OrgRepo
}

// NewAgentPermissionService 创建 AgentPermissionService
func NewAgentPermissionService(
	agentRepo *repository.AgentRepo,
	tenantRepo *repository.TenantRepo,
	orgRepo *repository.OrgRepo,
) *AgentPermissionService {
	return &AgentPermissionService{
		agentRepo:  agentRepo,
		tenantRepo: tenantRepo,
		orgRepo:    orgRepo,
	}
}

// EffectivePermissions 包含用户可用的智能体与工具列表
type EffectivePermissions struct {
	Agents []model.AgentDefinition `json:"agents"`
	Tools  map[string]bool         `json:"tools"` // tool_code -> allowed
}

// CalculateEffectivePermissions 计算用户在当前租户下的有效智能体与有效工具
func (s *AgentPermissionService) CalculateEffectivePermissions(ctx context.Context, tenantID, userID uuid.UUID) (*EffectivePermissions, error) {
	// 1. 租户 chat_enabled 总开关与系统配额
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil || tenant == nil {
		return &EffectivePermissions{Tools: make(map[string]bool)}, nil
	}
	if !tenant.ChatEnabled {
		return &EffectivePermissions{Tools: make(map[string]bool)}, nil
	}

	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil || alloc == nil {
		return &EffectivePermissions{Tools: make(map[string]bool)}, nil
	}

	allowedAgentCodes := make(map[string]bool)
	for _, code := range repository.ParseJSONSlice(alloc.AgentCodes) {
		allowedAgentCodes[code] = true
	}

	allowedToolCodes := make(map[string]bool)
	for _, code := range repository.ParseJSONSlice(alloc.ToolCodes) {
		allowedToolCodes[code] = true
	}

	// 2. 获取用户的所有组织角色
	var roleIDs []uuid.UUID
	member, err := s.orgRepo.FindByUserAndTenant(userID, tenantID)
	if err == nil && member != nil {
		for _, r := range member.Roles {
			roleIDs = append(roleIDs, r.ID)
		}
	}

	// 3. 组织角色授予的智能体与工具
	grantedAgentCodes, err := s.agentRepo.ListRoleAgentGrants(roleIDs)
	if err != nil {
		return nil, err
	}
	grantedToolCodes, err := s.agentRepo.ListRoleToolGrants(roleIDs)
	if err != nil {
		return nil, err
	}

	// 如果角色未显式配置任何智能体授权（初始默认体验），则默认授予租户配额内的基础智能体
	roleAgentSet := make(map[string]bool)
	if len(grantedAgentCodes) == 0 {
		for code := range allowedAgentCodes {
			roleAgentSet[code] = true
		}
	} else {
		for _, code := range grantedAgentCodes {
			roleAgentSet[code] = true
		}
	}

	roleToolSet := make(map[string]bool)
	if len(grantedToolCodes) == 0 {
		for code := range allowedToolCodes {
			roleToolSet[code] = true
		}
	} else {
		for _, code := range grantedToolCodes {
			roleToolSet[code] = true
		}
	}

	// 4. 计算有效智能体 = 租户配额 ∩ 角色授权 ∩ 智能体启用
	allAgents, err := s.agentRepo.ListAgentsByTenant(tenantID)
	if err != nil {
		return nil, err
	}

	var effectiveAgents []model.AgentDefinition
	for _, ag := range allAgents {
		if !ag.Enabled {
			continue
		}
		if allowedAgentCodes[ag.AgentCode] && roleAgentSet[ag.AgentCode] {
			effectiveAgents = append(effectiveAgents, ag)
		}
	}

	// 5. 计算有效工具 = 租户配额 ∩ 角色授权
	effectiveTools := make(map[string]bool)
	for code := range allowedToolCodes {
		if roleToolSet[code] {
			effectiveTools[code] = true
		}
	}

	return &EffectivePermissions{
		Agents: effectiveAgents,
		Tools:  effectiveTools,
	}, nil
}

// CalculateEffectiveToolsForAgent 计算特定智能体执行时模型可见并允许调用的有效工具集合：
// effective_tools = 租户配额 ∩ 智能体绑定 ∩ 角色工具 ∩ 启用状态
func (s *AgentPermissionService) CalculateEffectiveToolsForAgent(ctx context.Context, tenantID, userID uuid.UUID, agent *model.AgentDefinition) (map[string]bool, error) {
	perms, err := s.CalculateEffectivePermissions(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	effective := make(map[string]bool)
	for _, binding := range agent.ToolBindings {
		if perms.Tools[binding.ToolCode] {
			effective[binding.ToolCode] = true
		}
	}

	return effective, nil
}
