// useSettingsApi — 封装个人设置相关 API 调用

import type {
  ProcessListItem,
  CustomRule,
  RuleToggleOverride,
  AuditDetailItem,
  DashboardPref,
  UserPermissions,
} from '~/types/user-config'

export type { ProcessListItem, CustomRule, RuleToggleOverride, AuditDetailItem, DashboardPref, UserPermissions }

export const useSettingsApi = () => {
  const { authFetch } = useAuth()

  const processes = ref<ProcessListItem[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // ============================================================
  // 流程列表（双重校验结果）
  // ============================================================

  async function listProcesses(): Promise<ProcessListItem[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<ProcessListItem[]>('/api/tenant/settings/processes')
      processes.value = data
      return data
    }
    catch (e: any) {
      error.value = e.message || '加载流程列表失败'
      throw e
    }
    finally { loading.value = false }
  }

  // ============================================================
  // 用户流程配置
  // ============================================================

  async function getProcessConfig(processType: string): Promise<AuditDetailItem> {
    return await authFetch<AuditDetailItem>(`/api/tenant/settings/processes/${encodeURIComponent(processType)}`)
  }

  async function updateProcessConfig(processType: string, config: Partial<AuditDetailItem>): Promise<void> {
    await authFetch<null>(`/api/tenant/settings/processes/${encodeURIComponent(processType)}`, {
      method: 'PUT',
      body: config,
    })
  }

  // ============================================================
  // 仪表板偏好
  // ============================================================

  async function getDashboardPrefs(): Promise<DashboardPref> {
    return await authFetch<DashboardPref>('/api/tenant/settings/dashboard-prefs')
  }

  async function updateDashboardPrefs(prefs: Partial<DashboardPref>): Promise<void> {
    await authFetch<null>('/api/tenant/settings/dashboard-prefs', {
      method: 'PUT',
      body: prefs,
    })
  }

  // ============================================================
  // 权限锁定状态计算
  // ============================================================

  function computePermissionLocks(permissions: UserPermissions | null | undefined) {
    const defaults: UserPermissions = {
      allow_custom_fields: true,
      allow_custom_rules: true,
      allow_modify_strictness: true,
    }
    const perms = permissions ?? defaults

    return {
      fieldsLocked: !perms.allow_custom_fields,
      rulesLocked: !perms.allow_custom_rules,
      strictnessLocked: !perms.allow_modify_strictness,
    }
  }

  return {
    processes, loading, error,
    listProcesses, getProcessConfig, updateProcessConfig,
    getDashboardPrefs, updateDashboardPrefs,
    computePermissionLocks,
  }
}
