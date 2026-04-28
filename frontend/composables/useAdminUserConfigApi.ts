/**
 * useAdminUserConfigApi — 租户管理端用户配置查看 API 封装
 * 对接后端路由：
 *   GET /api/tenant/user-configs        获取租户内所有用户配置摘要列表
 *   GET /api/tenant/user-configs/:id    获取单个用户的配置详情
 *   GET /api/tenant/user-configs/export 导出所有用户配置摘要为 Excel 文件
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
  const { authFetch, token } = useAuth()

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

  /**
   * 导出当前租户所有用户配置摘要为 Excel 文件，触发浏览器下载。
   */
  async function exportUserConfigs(): Promise<void> {
    const runtimeConfig = useRuntimeConfig()
    const baseURL = String(runtimeConfig.public.apiBase || '')
    const url = `${baseURL}/api/tenant/user-configs/export`

    const accessToken = token.value || (process.client ? localStorage.getItem('token') || '' : '')

    const res = await fetch(url, {
      headers: accessToken
        ? { Authorization: `Bearer ${accessToken}` }
        : {},
    })

    if (!res.ok) {
      throw new Error('导出失败')
    }

    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)

    try {
      const a = document.createElement('a')
      a.href = blobUrl

      const contentDisposition = res.headers.get('Content-Disposition') || ''
      const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
      const normalMatch = contentDisposition.match(/filename=\"?([^\";]+)\"?/i)

      const filename = utf8Match?.[1]
        ? decodeURIComponent(utf8Match[1])
        : normalMatch?.[1] || 'user_configs.xlsx'

      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
    }
    finally {
      URL.revokeObjectURL(blobUrl)
    }
  }

  return {
    configs,
    loading,
    error,
    listUserConfigs,
    getUserConfig,
    exportUserConfigs,
  }
}
