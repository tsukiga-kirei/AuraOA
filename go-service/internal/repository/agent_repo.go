package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
)

// AgentRepo 负责智能体定义、工具绑定、配额与角色授权的仓储
type AgentRepo struct {
	db *gorm.DB
}

// NewAgentRepo 创建 AgentRepo
func NewAgentRepo(db *gorm.DB) *AgentRepo {
	return &AgentRepo{db: db}
}

// --------------------- AgentDefinition ---------------------

// GetAgentByID 根据 ID 查询智能体（含绑定工具）
func (r *AgentRepo) GetAgentByID(id uuid.UUID) (*model.AgentDefinition, error) {
	var agent model.AgentDefinition
	err := r.db.Preload("ToolBindings").Where("id = ?", id).First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// GetAgentByCode 查询智能体（平台种子或租户自定义）
func (r *AgentRepo) GetAgentByCode(tenantID uuid.UUID, code string) (*model.AgentDefinition, error) {
	var agent model.AgentDefinition
	err := r.db.Preload("ToolBindings").
		Where("(tenant_id = ? OR tenant_id IS NULL) AND agent_code = ?", tenantID, code).
		Order("tenant_id DESC NULLS LAST").
		First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// ListAgentsByTenant 获取租户可用的智能体列表（平台种子 + 租户自建）
func (r *AgentRepo) ListAgentsByTenant(tenantID uuid.UUID) ([]model.AgentDefinition, error) {
	var agents []model.AgentDefinition
	err := r.db.Preload("ToolBindings").
		Where("tenant_id = ? OR tenant_id IS NULL", tenantID).
		Order("tenant_id DESC NULLS LAST, is_system DESC, created_at ASC").
		Find(&agents).Error
	seen := make(map[string]bool)
	unique := make([]model.AgentDefinition, 0, len(agents))
	for _, agent := range agents {
		if !seen[agent.AgentCode] {
			seen[agent.AgentCode] = true
			unique = append(unique, agent)
		}
	}
	return unique, err
}

// CreateAgent 创建智能体
func (r *AgentRepo) CreateAgent(agent *model.AgentDefinition) error {
	return r.db.Create(agent).Error
}

// UpdateAgent 更新智能体
func (r *AgentRepo) UpdateAgent(tenantID, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&model.AgentDefinition{}).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Updates(updates).Error
}

// DeleteAgent 删除智能体（平台种子不可删）
func (r *AgentRepo) DeleteAgent(tenantID, id uuid.UUID) error {
	return r.db.Where("tenant_id = ? AND id = ? AND is_system = FALSE", tenantID, id).
		Delete(&model.AgentDefinition{}).Error
}

// ReplaceAgentToolBindings 覆盖更新智能体的工具绑定
func (r *AgentRepo) ReplaceAgentToolBindings(tenantID, agentID uuid.UUID, toolCodes []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var owner model.AgentDefinition
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, agentID).First(&owner).Error; err != nil {
			return fmt.Errorf("无权修改该智能体: %w", err)
		}
		if err := tx.Where("tenant_id = ? AND agent_id = ?", tenantID, agentID).Delete(&model.AgentToolBinding{}).Error; err != nil {
			return err
		}
		seen := make(map[string]bool)
		for _, tc := range toolCodes {
			if seen[tc] {
				continue
			}
			seen[tc] = true
			kind := "system"
			if strings.HasPrefix(tc, "skill:") {
				kind = "skill"
			}
			if strings.HasPrefix(tc, "mcp:") {
				kind = "mcp"
			}
			b := model.AgentToolBinding{
				TenantID: &tenantID,
				AgentID:  agentID,
				ToolType: kind,
				ToolCode: tc,
			}
			if err := tx.Create(&b).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// --------------------- TenantChatAllocation ---------------------

// GetTenantAllocation 获取租户的系统管理员配额
func (r *AgentRepo) GetTenantAllocation(tenantID uuid.UUID) (*model.TenantChatAllocation, error) {
	var alloc model.TenantChatAllocation
	err := r.db.Where("tenant_id = ?", tenantID).First(&alloc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 自动初始化默认配额
		defaultAlloc := model.TenantChatAllocation{
			TenantID:          tenantID,
			AgentCodes:        datatypes.JSON([]byte(`["oa_query", "oa_assist"]`)),
			ToolCodes:         datatypes.JSON([]byte(`["list_my_todos", "get_process", "get_approval_flow", "get_latest_audit", "get_latest_summary", "draft_comment", "run_audit", "run_summary", "resolve_oa_url"]`)),
			SkillCodes:        datatypes.JSON([]byte(`[]`)),
			AllowCustomSkills: false,
			AllowTenantMCP:    false,
			MaxMCPServers:     0,
			MCPTemplateIDs:    datatypes.JSON([]byte(`[]`)),
		}
		if createErr := r.db.Create(&defaultAlloc).Error; createErr == nil {
			return &defaultAlloc, nil
		}
	}
	if err != nil {
		return nil, err
	}
	return &alloc, nil
}

// SaveTenantAllocation 保存租户配额
func (r *AgentRepo) SaveTenantAllocation(alloc *model.TenantChatAllocation) error {
	return r.db.Save(alloc).Error
}

// SaveTenantChatSettings 同事务保存聊天开关、模型设置与能力配额。
func (r *AgentRepo) SaveTenantChatSettings(alloc *model.TenantChatAllocation, updates map[string]interface{}) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&model.Tenant{}).Where("id = ?", alloc.TenantID).Updates(updates).Error; err != nil {
				return err
			}
		}
		return tx.Save(alloc).Error
	})
}

// --------------------- Org Role Grants ---------------------

// ListRoleAgentGrants 获取角色的智能体授权
func (r *AgentRepo) ListRoleAgentGrants(roleIDs []uuid.UUID) ([]string, error) {
	var codes []string
	if len(roleIDs) == 0 {
		return codes, nil
	}
	err := r.db.Model(&model.OrgRoleAgentGrant{}).
		Where("role_id IN ?", roleIDs).
		Distinct("agent_code").
		Pluck("agent_code", &codes).Error
	return codes, err
}

// ListRoleToolGrants 获取角色的工具授权
func (r *AgentRepo) ListRoleToolGrants(roleIDs []uuid.UUID) ([]string, error) {
	var codes []string
	if len(roleIDs) == 0 {
		return codes, nil
	}
	err := r.db.Model(&model.OrgRoleToolGrant{}).
		Where("role_id IN ?", roleIDs).
		Distinct("tool_code").
		Pluck("tool_code", &codes).Error
	return codes, err
}

// SaveRoleGrants 保存指定角色的智能体和工具授权
func (r *AgentRepo) SaveRoleGrants(tenantID, roleID uuid.UUID, agentCodes, toolCodes []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var role model.OrgRole
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, roleID).First(&role).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND role_id = ?", tenantID, roleID).Delete(&model.OrgRoleAgentGrant{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND role_id = ?", tenantID, roleID).Delete(&model.OrgRoleToolGrant{}).Error; err != nil {
			return err
		}
		for _, code := range agentCodes {
			ag := model.OrgRoleAgentGrant{
				TenantID:  tenantID,
				RoleID:    roleID,
				AgentCode: code,
			}
			if err := tx.Create(&ag).Error; err != nil {
				return err
			}
		}
		for _, code := range toolCodes {
			tg := model.OrgRoleToolGrant{
				TenantID: tenantID,
				RoleID:   roleID,
				ToolCode: code,
			}
			if err := tx.Create(&tg).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// --------------------- MCP Servers ---------------------

func (r *AgentRepo) ListMCPServers(tenantID uuid.UUID) ([]model.MCPServer, error) {
	var servers []model.MCPServer
	err := r.db.Where("tenant_id = ? OR tenant_id IS NULL", tenantID).
		Order("created_at ASC").
		Find(&servers).Error
	return servers, err
}

func (r *AgentRepo) GetMCPServerByID(tenantID, id uuid.UUID) (*model.MCPServer, error) {
	var server model.MCPServer
	err := r.db.Where("(tenant_id = ? OR tenant_id IS NULL) AND id = ?", tenantID, id).First(&server).Error
	return &server, err
}

func (r *AgentRepo) CreateMCPServer(server *model.MCPServer) error {
	return r.db.Create(server).Error
}

func (r *AgentRepo) UpdateMCPServer(tenantID, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&model.MCPServer{}).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(updates).Error
}

func (r *AgentRepo) DeleteMCPServer(tenantID, id uuid.UUID) error {
	return r.db.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&model.MCPServer{}).Error
}

// --------------------- Skills ---------------------

func (r *AgentRepo) ListSkills(tenantID uuid.UUID) ([]model.AgentSkill, error) {
	var skills []model.AgentSkill
	err := r.db.Where("tenant_id = ? OR tenant_id IS NULL", tenantID).
		Order("created_at ASC").
		Find(&skills).Error
	return skills, err
}

func (r *AgentRepo) GetSkillByCode(tenantID uuid.UUID, code string) (*model.AgentSkill, error) {
	var skill model.AgentSkill
	err := r.db.Where("(tenant_id = ? OR tenant_id IS NULL) AND skill_code = ?", tenantID, code).
		Order("tenant_id DESC NULLS LAST").
		First(&skill).Error
	return &skill, err
}

func (r *AgentRepo) CreateSkill(skill *model.AgentSkill) error {
	enabled := skill.Enabled
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(skill).Error; err != nil {
			return err
		}
		skill.Enabled = enabled
		return tx.Model(skill).Update("enabled", enabled).Error
	})
}

func (r *AgentRepo) UpdateSkill(tenantID, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&model.AgentSkill{}).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(updates).Error
}

func (r *AgentRepo) DeleteSkill(tenantID, id uuid.UUID) error {
	return r.db.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&model.AgentSkill{}).Error
}

// Helper: ParseJSONSlice 解析 jsonb string slice
func ParseJSONSlice(data datatypes.JSON) []string {
	var res []string
	if len(data) > 0 {
		_ = json.Unmarshal(data, &res)
	}
	return res
}

// SaveTenantAgent 在同一事务内保存租户定义及绑定；修改平台模板时创建租户专属副本。
func (r *AgentRepo) SaveTenantAgent(tenantID uuid.UUID, agent *model.AgentDefinition, updates map[string]interface{}, toolCodes *[]string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		repo := NewAgentRepo(tx)
		if agent.TenantID == nil {
			var existing model.AgentDefinition
			err := tx.Where("tenant_id = ? AND agent_code = ?", tenantID, agent.AgentCode).First(&existing).Error
			if err == nil {
				agent = &existing
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				copy := *agent
				copy.ID = uuid.New()
				copy.TenantID = &tenantID
				copy.ToolBindings = nil
				if err := tx.Create(&copy).Error; err != nil {
					return err
				}
				if toolCodes == nil {
					codes := []string{}
					for _, b := range agent.ToolBindings {
						codes = append(codes, b.ToolCode)
					}
					toolCodes = &codes
				}
				agent = &copy
			} else {
				return err
			}
		} else if *agent.TenantID != tenantID {
			return fmt.Errorf("无权修改该智能体")
		}
		if err := repo.UpdateAgent(tenantID, agent.ID, updates); err != nil {
			return err
		}
		if toolCodes != nil {
			return repo.ReplaceAgentToolBindings(tenantID, agent.ID, *toolCodes)
		}
		return nil
	})
}

// CreateAgentWithBindings 原子创建定义与工具绑定，避免出现没有装配完成的智能体。
func (r *AgentRepo) CreateAgentWithBindings(tenantID uuid.UUID, agent *model.AgentDefinition, codes []string) error {
	enabled := agent.Enabled
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(agent).Error; err != nil {
			return err
		}
		agent.Enabled = enabled
		if err := tx.Model(agent).Update("enabled", enabled).Error; err != nil {
			return err
		}
		return NewAgentRepo(tx).ReplaceAgentToolBindings(tenantID, agent.ID, codes)
	})
}

// CreateMCPServerWithinQuota 在租户行锁内检查数量，避免并发注册突破配额。
func (r *AgentRepo) CreateMCPServerWithinQuota(tenantID uuid.UUID, server *model.MCPServer) error {
	enabled := server.Enabled
	return r.db.Transaction(func(tx *gorm.DB) error {
		var alloc model.TenantChatAllocation
		if err := tx.Raw("SELECT * FROM tenant_chat_allocations WHERE tenant_id = ? FOR UPDATE", tenantID).Scan(&alloc).Error; err != nil {
			return err
		}
		if !alloc.AllowTenantMCP {
			return fmt.Errorf("当前租户未获得 MCP 权限")
		}
		var count int64
		if err := tx.Model(&model.MCPServer{}).Where("tenant_id = ?", tenantID).Count(&count).Error; err != nil {
			return err
		}
		if count >= int64(alloc.MaxMCPServers) {
			return fmt.Errorf("已达到 MCP 服务数量上限")
		}
		if err := tx.Create(server).Error; err != nil {
			return err
		}
		server.Enabled = enabled
		return tx.Model(server).Update("enabled", enabled).Error
	})
}
