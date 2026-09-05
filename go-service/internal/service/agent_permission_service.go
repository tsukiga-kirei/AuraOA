package service

import (
	"context"
	"fmt"
	"strings"

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
	chatPageAllowed := false
	member, err := s.orgRepo.FindByUserAndTenant(userID, tenantID)
	if err == nil && member != nil {
		for _, r := range member.Roles {
			roleIDs = append(roleIDs, r.ID)
			for _, page := range repository.ParseJSONSlice(r.PagePermissions) {
				if page == "/chat" {
					chatPageAllowed = true
				}
			}
		}
	}

	if !chatPageAllowed {
		return &EffectivePermissions{Tools: make(map[string]bool)}, nil
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

	// 空授权表示拒绝；撤销最后一项授权不能扩大权限。
	roleAgentSet := make(map[string]bool)
	for _, code := range grantedAgentCodes {
		roleAgentSet[code] = true
	}

	roleToolSet := make(map[string]bool)
	for _, code := range grantedToolCodes {
		roleToolSet[code] = true
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
		if (allowedAgentCodes[ag.AgentCode] || !ag.IsSystem) && roleAgentSet[ag.AgentCode] {
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
	// 自建 MCP 的授权仍需同时满足角色授权和租户开关。
	if alloc.AllowTenantMCP {
		servers, err := s.agentRepo.ListMCPServers(tenantID)
		if err != nil {
			return nil, err
		}
		for _, server := range servers {
			if !server.Enabled || server.TenantID == nil || *server.TenantID != tenantID {
				continue
			}
			for _, def := range ConvertMCPToolsToDefinitions(server.ServerCode, server.CachedTools) {
				if roleToolSet[def.Function.Name] {
					effectiveTools[def.Function.Name] = true
				}
			}
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
	allowed := false
	for _, ag := range perms.Agents {
		if ag.ID == agent.ID {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("智能体已停用或当前角色没有对话权限")
	}

	effective := make(map[string]bool)
	for _, binding := range agent.ToolBindings {
		if perms.Tools[binding.ToolCode] {
			effective[binding.ToolCode] = true
		}
	}
	alloc, err := s.agentRepo.GetTenantAllocation(tenantID)
	if err != nil {
		return nil, err
	}
	for _, binding := range agent.ToolBindings {
		if binding.ToolType != "skill" {
			continue
		}
		code := strings.TrimPrefix(binding.ToolCode, "skill:")
		skill, err := s.agentRepo.GetSkillByCode(tenantID, code)
		if err != nil || !skill.Enabled {
			continue
		}
		granted := skill.TenantID != nil && alloc.AllowCustomSkills
		for _, c := range repository.ParseJSONSlice(alloc.SkillCodes) {
			if c == code {
				granted = true
			}
		}
		if granted {
			effective[binding.ToolCode] = true
		}
	}

	return effective, nil
}
