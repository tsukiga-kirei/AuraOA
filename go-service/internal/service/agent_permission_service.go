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

	// 3. 组织角色授予的智能体；任一角色未勾选则视为该角色可用全部已启用智能体。
	grantedAgentCodes, err := s.agentRepo.ListRoleAgentGrants(roleIDs)
	if err != nil {
		return nil, err
	}
	roleAgentSet := make(map[string]bool)
	for _, code := range grantedAgentCodes {
		roleAgentSet[code] = true
	}

	// 4. 有效智能体 = 角色授权 ∩ 已启用
	allAgents, err := s.agentRepo.ListAgentsByTenant(tenantID)
	if err != nil {
		return nil, err
	}
	if len(roleAgentSet) == 0 {
		for _, ag := range allAgents {
			if ag.Enabled {
				roleAgentSet[ag.AgentCode] = true
			}
		}
	}

	var effectiveAgents []model.AgentDefinition
	effectiveTools := make(map[string]bool)
	for _, ag := range allAgents {
		if !ag.Enabled || !roleAgentSet[ag.AgentCode] {
			continue
		}
		effectiveAgents = append(effectiveAgents, ag)
		for _, binding := range ag.ToolBindings {
			if s.capabilityAllowed(alloc, tenantID, binding.ToolCode) {
				effectiveTools[binding.ToolCode] = true
			}
		}
	}

	return &EffectivePermissions{
		Agents: effectiveAgents,
		Tools:  effectiveTools,
	}, nil
}

// capabilityAllowed 判断工具/MCP/Skill 是否仍受租户大开关约束。
func (s *AgentPermissionService) capabilityAllowed(alloc *model.TenantChatAllocation, tenantID uuid.UUID, toolCode string) bool {
	if strings.HasPrefix(toolCode, "skill:") {
		code := strings.TrimPrefix(toolCode, "skill:")
		skill, err := s.agentRepo.GetSkillByCode(tenantID, code)
		if err != nil || skill == nil || !skill.Enabled {
			return false
		}
		if skill.TenantID != nil {
			return alloc.AllowCustomSkills
		}
		return true
	}
	if strings.HasPrefix(toolCode, "mcp:") {
		if !alloc.AllowTenantMCP {
			return false
		}
		parts := strings.SplitN(toolCode, ":", 3)
		if len(parts) < 2 {
			return false
		}
		servers, err := s.agentRepo.ListMCPServers(tenantID)
		if err != nil {
			return false
		}
		for _, server := range servers {
			if server.Enabled && server.ServerCode == parts[1] {
				return true
			}
		}
		return false
	}
	return true
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
	return effective, nil
}
