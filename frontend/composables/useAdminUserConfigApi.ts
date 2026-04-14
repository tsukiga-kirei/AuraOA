/**
 * useAdminUserConfigApi — 租户管理端用户配置查看 API 封装
 * 对接后端路由：
 *   GET /api/tenant/user-configs        获取租户内所有用户配置摘要列表
 *   GET /api/tenant/user-configs/:id    获取单个用户的配置详情
 */

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

  // 用户配置列表（响应式，供模板直接绑定）
  const configs = ref<AdminUserConfigItem[]>([])
  // 加载状态标志
  const loading = ref(false)
  // 错误信息（null 表示无错误）
  const error = ref<string | null>(null)

  /**
   * 获取租户内所有用户的个人配置摘要列表。
   * 包含每个用户的审核流程配置、定时任务偏好、自定义规则等摘要信息。
   * @returns 用户配置摘要列表
   */
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

  /**
   * 获取单个用户的完整配置详情。
   * @param userId 目标用户 ID
   * @returns 用户配置详情（含流程配置、规则覆盖等）
   */
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
