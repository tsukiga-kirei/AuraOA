export const useSystemApi = () => {
  const { authFetch } = useAuth()

  // ============================================================
  // System Configs
  // ============================================================

  async function getConfigs(): Promise<any> {
    return authFetch<any>('/api/admin/system/configs')
  }

  async function updateConfigs(configs: Record<string, any>): Promise<void> {
    await authFetch('/api/admin/system/configs', { method: 'PUT', body: configs })
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

  async function deleteTenant(id: string): Promise<void> {
    await authFetch<null>(`/api/admin/tenants/${id}`, { method: 'DELETE' })
  }

  async function getTenantStats(id: string): Promise<any> {
    return authFetch<any>(`/api/admin/tenants/${id}/stats`)
  }

  return { getConfigs, updateConfigs, listTenants, createTenant, updateTenant, deleteTenant, getTenantStats }
}
