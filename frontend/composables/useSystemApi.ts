/**
 * useSystemApi — 系统管理相关 API 调用封装（仅系统管理员可用）
 * 对接后端路由组：
 *   /api/admin/system/configs          系统全局配置（KV 键值对）
 *   /api/admin/system/options/*        选项数据（OA 类型、数据库驱动、AI 部署类型等）
 *   /api/admin/system/oa-connections   OA 数据库连接管理
 *   /api/admin/system/ai-models        AI 模型配置管理
 *   /api/admin/tenants                 租户管理
 */

export const useSystemApi = () => {
  const { authFetch } = useAuth()

  // ============================================================
  // 系统全局配置（KV 键值对）
  // ============================================================

  /** 获取所有系统全局配置项 */
  async function getConfigs(): Promise<any> {
    return authFetch<any>('/api/admin/system/configs')
  }

  /**
   * 批量更新系统全局配置项。
   * @param configs 键值对映射（键为配置项名称，值为新配置值）
   */
  async function updateConfigs(configs: Record<string, any>): Promise<void> {
    await authFetch('/api/admin/system/configs', { method: 'PUT', body: configs })
  }

  // ============================================================
  // 选项数据（下拉框数据源）
  // ============================================================

  /** 获取支持的 OA 系统类型列表（用于 OA 连接配置下拉框） */
  async function listOATypes(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/options/oa-types')
  }

  /** 获取支持的数据库驱动类型列表（用于 OA 连接配置下拉框） */
  async function listDBDrivers(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/options/db-drivers')
  }

  /** 获取支持的 AI 部署类型列表（私有化/云端等） */
  async function listAIDeployTypes(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/options/ai-deploy-types')
  }

  /** 获取支持的 AI 服务商列表（OpenAI、Azure 等） */
  async function listAIProviders(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/options/ai-providers')
  }

  // ============================================================
  // OA 数据库连接管理
  // ============================================================

  /** 获取所有 OA 数据库连接配置列表 */
  async function listOAConnections(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/oa-connections')
  }

  /**
   * 创建新的 OA 数据库连接配置。
   * @param data 连接配置（类型、主机、端口、数据库名、账号等）
   */
  async function createOAConnection(data: Record<string, any>): Promise<any> {
    return authFetch<any>('/api/admin/system/oa-connections', { method: 'POST', body: data })
  }

  /**
   * 更新指定 OA 数据库连接配置。
   * @param id 连接配置 ID
   * @param data 要更新的字段
   */
  async function updateOAConnection(id: string, data: Record<string, any>): Promise<any> {
    return authFetch<any>(`/api/admin/system/oa-connections/${id}`, { method: 'PUT', body: data })
  }

  /**
   * 删除指定 OA 数据库连接配置。
   * @param id 连接配置 ID
   */
  async function deleteOAConnection(id: string): Promise<void> {
    await authFetch<null>(`/api/admin/system/oa-connections/${id}`, { method: 'DELETE' })
  }

  /**
   * 测试已保存的 OA 数据库连接是否可用。
   * @param id 连接配置 ID
   */
  async function testOAConnection(id: string): Promise<any> {
    return authFetch<any>(`/api/admin/system/oa-connections/${id}/test`, { method: 'POST' })
  }

  /**
   * 使用临时参数测试 OA 数据库连接（保存前预检）。
   * @param data 连接参数（与创建接口相同结构）
   */
  async function testOAConnectionParams(data: Record<string, any>): Promise<any> {
    return authFetch<any>('/api/admin/system/oa-connections/test', { method: 'POST', body: data })
  }

  // ============================================================
  // AI 模型配置管理
  // ============================================================

  /** 获取所有 AI 模型配置列表 */
  async function listAIModels(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/system/ai-models')
  }

  /**
   * 创建新的 AI 模型配置。
   * @param data 模型配置（服务商、部署类型、API Key、模型名称等）
   */
  async function createAIModel(data: Record<string, any>): Promise<any> {
    return authFetch<any>('/api/admin/system/ai-models', { method: 'POST', body: data })
  }

  /**
   * 更新指定 AI 模型配置。
   * @param id 模型配置 ID
   * @param data 要更新的字段
   */
  async function updateAIModel(id: string, data: Record<string, any>): Promise<any> {
    return authFetch<any>(`/api/admin/system/ai-models/${id}`, { method: 'PUT', body: data })
  }

  /**
   * 删除指定 AI 模型配置。
   * @param id 模型配置 ID
   */
  async function deleteAIModel(id: string): Promise<void> {
    await authFetch<null>(`/api/admin/system/ai-models/${id}`, { method: 'DELETE' })
  }

  /**
   * 使用临时参数测试 AI 模型连接（保存前预检）。
   * @param data 模型配置参数
   */
  async function testAIModelConnection(data: Record<string, any>): Promise<any> {
    return authFetch<any>('/api/admin/system/ai-models/test', { method: 'POST', body: data })
  }

  /**
   * 测试已保存的 AI 模型连接是否可用。
   * @param id 模型配置 ID
   */
  async function testAIModelConnectionById(id: string): Promise<any> {
    return authFetch<any>(`/api/admin/system/ai-models/${id}/test`, { method: 'POST' })
  }

  // ============================================================
  // 租户管理
  // ============================================================

  /** 获取所有租户列表 */
  async function listTenants(): Promise<any[]> {
    return authFetch<any[]>('/api/admin/tenants')
  }

  /**
   * 创建新租户（同时初始化租户管理员账号）。
   * @param data 租户信息（名称、编码、管理员账号等）
   */
  async function createTenant(data: Record<string, any>): Promise<any> {
    return authFetch<any>('/api/admin/tenants', { method: 'POST', body: data })
  }

  /**
   * 更新指定租户信息。
   * @param id 租户 ID
   * @param data 要更新的字段
   */
  async function updateTenant(id: string, data: Record<string, any>): Promise<any> {
    return authFetch<any>(`/api/admin/tenants/${id}`, { method: 'PUT', body: data })
  }

  /**
   * 删除指定租户（需要系统管理员密码二次确认）。
   * @param id 租户 ID
   * @param adminPassword 系统管理员密码（用于二次确认）
   */
  async function deleteTenant(id: string, adminPassword: string): Promise<void> {
    await authFetch<null>(`/api/admin/tenants/${id}`, { method: 'DELETE', body: { admin_password: adminPassword } })
  }

  /**
   * 获取指定租户的统计数据（成员数、审核数、归档数等）。
   * @param id 租户 ID
   */
  async function getTenantStats(id: string): Promise<any> {
    return authFetch<any>(`/api/admin/tenants/${id}/stats`)
  }

  /**
   * 获取指定租户的成员列表。
   * @param id 租户 ID
   */
  async function listTenantMembers(id: string): Promise<any[]> {
    return authFetch<any[]>(`/api/admin/tenants/${id}/members`)
  }

  return {
    // 系统配置
    getConfigs, updateConfigs,
    // 选项数据
    listOATypes, listDBDrivers, listAIDeployTypes, listAIProviders,
    // OA 连接
    listOAConnections, createOAConnection, updateOAConnection, deleteOAConnection, testOAConnection, testOAConnectionParams,
    // AI 模型
    listAIModels, createAIModel, updateAIModel, deleteAIModel, testAIModelConnection, testAIModelConnectionById,
    // 租户管理
    listTenants, createTenant, updateTenant, deleteTenant, getTenantStats, listTenantMembers,
  }
}
