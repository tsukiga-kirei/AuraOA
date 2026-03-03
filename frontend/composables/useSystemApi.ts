export const useSystemApi = () => {
  const { authFetch } = useAuth()

  async function getConfigs(): Promise<any> {
    return authFetch<any>('/api/admin/system/configs')
  }

  async function updateConfigs(configs: Record<string, any>): Promise<void> {
    await authFetch('/api/admin/system/configs', { method: 'PUT', body: configs })
  }

  return { getConfigs, updateConfigs }
}
