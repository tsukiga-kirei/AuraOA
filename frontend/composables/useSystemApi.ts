export const useSystemApi = () => {
  const { authFetch } = useAuth()

  // ============================================================
  // System Configs (KV)
  // ============================================================

  async function getConfigs(): Promise<any> {
    return authFetch<any>('/api/admin/system/configs')
  }

  async function updateConfigs(configs: Record<string, any>): Promise<void> {
    await authFetch('/api/admin/system/configs', { method: 'PUT', body: configs })
  }

  // ============================================================
  // 选项数据 (Options)
  // ============================================================

  async function listOATypes(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/options/oa-types')
  }

  async function listDBDrivers(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/options/db-drivers')
  }

  async function listAIDeployTypes(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/options/ai-deploy-types')
  }

  async function listAIProviders(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/options/ai-providers')
  }

  // ============================================================
  // OA 数据库连接
  // ============================================================

  async function listOAConnections(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/oa-connections')
  }

  async function createOAConnection(data: Record<string, any>): Promise<any> {
    return authFetch<any>('/api/admin/system/oa-connections', { method: 'POST', body: data })
  }

  async function updateOAConnection(id: string, data: Record<string, any>): Promise<any> {
    return authFetch<any>(`/api/admin/system/oa-connections/${id}`, { method: 'PUT', body: data })
  }

  async function deleteOAConnection(id: string): Promise<void> {
    await authFetch<null>(`/api/admin/system/oa-connections/${id}`, { method: 'DELETE' })
  }

  async function testOAConnection(id: string): Promise<any> {
    return authFetch<any>(`/api/admin/system/oa-connections/${id}/test`, { method: 'POST' })
  }

  async function testOAConnectionParams(data: Record<string, any>): Promise<any> {
    return authFetch<any>('/api/admin/system/oa-connections/test', { method: 'POST', body: data })
  }

  // ============================================================
  // AI 模型配置
  // ============================================================

  async function listAIModels(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/ai-models')
  }

  async function createAIModel(data: Record<string, any>): Promise<any> {
    return authFetch<any>('/api/admin/system/ai-models', { method: 'POST', body: data })
  }

  async function updateAIModel(id: string, data: Record<string, any>): Promise<any> {
    return authFetch<any>(`/api/admin/system/ai-models/${id}`, { method: 'PUT', body: data })
  }

  async function deleteAIModel(id: string): Promise<void> {
    await authFetch<null>(`/api/admin/system/ai-models/${id}`, { method: 'DELETE' })
  }

  async function testAIModelConnection(data: Record<string, any>): Promise<any> {
    return authFetch<any>('/api/admin/system/ai-models/test', { method: 'POST', body: data })
  }

  async function testAIModelConnectionById(id: string): Promise<any> {
    return authFetch<any>(`/api/admin/system/ai-models/${id}/test`, { method: 'POST' })
  }

  // ============================================================
  // Tenant Management
  // ============================================================

  async function listTenants(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/tenants')
  }

  async function createTenant(data: Record<string, any>): Promise<any> {
    return authFetch<any>('/api/admin/tenants', { method: 'POST', body: data })
  }

  async function updateTenant(id: string, data: Record<string, any>): Promise<any> {
    return authFetch<any>(`/api/admin/tenants/${id}`, { method: 'PUT', body: data })
  }

  async function deleteTenant(id: string, adminPassword: string): Promise<void> {
    await authFetch<null>(`/api/admin/tenants/${id}`, { method: 'DELETE', body: { admin_password: adminPassword } })
  }

  async function getTenantStats(id: string): Promise<any> {
    return authFetch<any>(`/api/admin/tenants/${id}/stats`)
  }

  async function listTenantMembers(id: string): Promise<any[]> {
    return authFetch<any[]>(`/api/admin/tenants/${id}/members`)
  }

  return {
    // System configs
    getConfigs, updateConfigs,
    // Options
    listOATypes, listDBDrivers, listAIDeployTypes, listAIProviders,
    // OA connections
    listOAConnections, createOAConnection, updateOAConnection, deleteOAConnection, testOAConnection, testOAConnectionParams,
    // AI models
    listAIModels, createAIModel, updateAIModel, deleteAIModel, testAIModelConnection, testAIModelConnectionById,
    // Tenants
    listTenants, createTenant, updateTenant, deleteTenant, getTenantStats, listTenantMembers,
  }
}
