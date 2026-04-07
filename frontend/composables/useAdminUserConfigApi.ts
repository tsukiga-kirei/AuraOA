// useAdminUserConfigApi — 封装租户管理端用户配置查看 API

import type {
  AdminUserConfigItem,
  AdminProcessDetail,
  AdminCronTaskDetail,
  AdminCustomRule,
  AdminRuleToggleItem,
} from '~/types/user-config'

export type {
  AdminUserConfigItem,
  AdminProcessDetail,
  AdminCronTaskDetail,
  AdminCustomRule,
  AdminRuleToggleItem,
}

export const useAdminUserConfigApi = () => {
  const { authFetch } = useAuth()

  const configs = ref<AdminUserConfigItem[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  /** 获取租户内所有用户的个人配置摘要列表 */
  async function listUserConfigs(): Promise<AdminUserConfigItem[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<AdminUserConfigItem[]>('/api/tenant/user-configs')
      configs.value = data ?? []
      return configs.value
    }
    catch (e: any) {
      error.value = e.message || '加载用户配置失败'
      console.error('[useAdminUserConfigApi] listUserConfigs failed', e)
      throw e
    }
    finally { loading.value = false }
  }

  /** 获取单个用户的配置详情 */
  async function getUserConfig(userId: string): Promise<AdminUserConfigItem> {
    return await authFetch<AdminUserConfigItem>(`/api/tenant/user-configs/${userId}`)
  }

  return {
    configs,
    loading,
    error,
    listUserConfigs,
    getUserConfig,
  }
}
