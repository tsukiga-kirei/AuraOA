package repository

import (
	"encoding/json"
	"errors"

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
		Order("is_system DESC, created_at ASC").
		Find(&agents).Error
	return agents, err
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
		if err := tx.Where("agent_id = ?", agentID).Delete(&model.AgentToolBinding{}).Error; err != nil {
			return err
		}
		for _, tc := range toolCodes {
			b := model.AgentToolBinding{
				TenantID: &tenantID,
				AgentID:  agentID,
				ToolType: "system",
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
		if err := tx.Where("role_id = ?", roleID).Delete(&model.OrgRoleAgentGrant{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", roleID).Delete(&model.OrgRoleToolGrant{}).Error; err != nil {
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
	return r.db.Create(skill).Error
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
