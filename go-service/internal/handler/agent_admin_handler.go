package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/service"
)

// AgentAdminHandler 处理系统管理员配额与租户智能体管理的 HTTP 请求
type AgentAdminHandler struct {
	allocationService *service.AgentAllocationService
	mcpService        *service.MCPService
}

// NewAgentAdminHandler 创建 AgentAdminHandler 实例
func NewAgentAdminHandler(
	allocationService *service.AgentAllocationService,
	mcpService *service.MCPService,
) *AgentAdminHandler {
	return &AgentAdminHandler{
		allocationService: allocationService,
		mcpService:        mcpService,
	}
}

// GetAgentCatalog 获取平台目录（系统工具、内置智能体、内置 Skills）
// GET /api/admin/agent-catalog
func (h *AgentAdminHandler) GetAgentCatalog(c *gin.Context) {
	catalog, err := h.allocationService.GetAgentCatalog(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, catalog)
}

// GetTenantAllocationByAdmin 系统管理员获取某租户的对话配额
// GET /api/admin/tenants/:id/chat-allocation
func (h *AgentAdminHandler) GetTenantAllocationByAdmin(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户 ID 格式错误")
		return
	}

	alloc, err := h.allocationService.GetTenantAllocation(c.Request.Context(), tenantID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, alloc)
}

// UpdateTenantAllocationByAdmin 系统管理员更新某租户的对话配额
// PUT /api/admin/tenants/:id/chat-allocation
func (h *AgentAdminHandler) UpdateTenantAllocationByAdmin(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户 ID 格式错误")
		return
	}

	var req dto.UpdateTenantChatAllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	if err := h.allocationService.UpdateTenantAllocation(c.Request.Context(), tenantID, &req); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}

// GetTenantAllocationByTenant 租户管理员只读查询本租户配额
// GET /api/tenant/chat-allocation
func (h *AgentAdminHandler) GetTenantAllocationByTenant(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	alloc, err := h.allocationService.GetTenantAllocation(c.Request.Context(), tenantID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, alloc)
}

// ListTenantAgents 租户管理员查看本租户智能体列表
// GET /api/tenant/agents
func (h *AgentAdminHandler) ListTenantAgents(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	agents, err := h.allocationService.ListTenantAgents(c.Request.Context(), tenantID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, agents)
}

// CreateTenantAgent 租户管理员新建自定义智能体
// POST /api/tenant/agents
func (h *AgentAdminHandler) CreateTenantAgent(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	var req dto.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	agent, err := h.allocationService.CreateTenantAgent(c.Request.Context(), tenantID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, agent)
}

// UpdateTenantAgent 租户管理员更新智能体
// PUT /api/tenant/agents/:id
func (h *AgentAdminHandler) UpdateTenantAgent(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "智能体 ID 格式错误")
		return
	}

	var req dto.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	if err := h.allocationService.UpdateTenantAgent(c.Request.Context(), tenantID, agentID, &req); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}

// DeleteTenantAgent 租户管理员删除智能体
// DELETE /api/tenant/agents/:id
func (h *AgentAdminHandler) DeleteTenantAgent(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "智能体 ID 格式错误")
		return
	}

	if err := h.allocationService.DeleteTenantAgent(c.Request.Context(), tenantID, agentID); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ListMCPServers 租户管理员查看 MCP 服务列表
// GET /api/tenant/mcp-servers
func (h *AgentAdminHandler) ListMCPServers(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	servers, err := h.allocationService.ListMCPServers(c.Request.Context(), tenantID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, servers)
}

// SaveMCPServer 租户管理员新增 MCP 服务
// POST /api/tenant/mcp-servers
func (h *AgentAdminHandler) SaveMCPServer(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	var req dto.SaveMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	server, err := h.allocationService.SaveMCPServer(c.Request.Context(), tenantID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, server)
}

// TestAndRefreshMCPServer 测试 MCP 服务连接并刷新工具
// POST /api/tenant/mcp-servers/:id/test
func (h *AgentAdminHandler) TestAndRefreshMCPServer(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "服务 ID 格式错误")
		return
	}

	tools, err := h.mcpService.TestAndRefreshTools(c.Request.Context(), tenantID, serverID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrExternal, "测试连接失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"tools": tools})
}

// DeleteMCPServer 删除 MCP 服务
// DELETE /api/tenant/mcp-servers/:id
func (h *AgentAdminHandler) DeleteMCPServer(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "服务 ID 格式错误")
		return
	}

	if err := h.allocationService.DeleteMCPServer(c.Request.Context(), tenantID, serverID); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ListSkills 租户管理员查看 Skills 列表
// GET /api/tenant/skills
func (h *AgentAdminHandler) ListSkills(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	skills, err := h.allocationService.ListSkills(c.Request.Context(), tenantID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, skills)
}

// SaveSkill 租户管理员保存自定义 Skill
// POST /api/tenant/skills
func (h *AgentAdminHandler) SaveSkill(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	var req dto.SaveSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	skill, err := h.allocationService.SaveSkill(c.Request.Context(), tenantID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, skill)
}

// DeleteSkill 删除自定义 Skill
// DELETE /api/tenant/skills/:id
func (h *AgentAdminHandler) DeleteSkill(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户上下文丢失")
		return
	}

	skillID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "Skill ID 格式错误")
		return
	}

	if err := h.allocationService.DeleteSkill(c.Request.Context(), tenantID, skillID); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
