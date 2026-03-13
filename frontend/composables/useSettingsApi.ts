// useSettingsApi — 封装个人设置相关 API 调用

import type {
  ProcessListItem,
  CustomRule,
  RuleToggleOverride,
  AuditDetailItem,
  DashboardPref,
  UserPermissions,
  FullAuditProcessConfig,
  CronPrefs,
  AccessibleArchiveConfig,
  FullArchiveConfig,
  UpdatePersonalConfigRequest,
} from '~/types/user-config'

export type {
  ProcessListItem, CustomRule, RuleToggleOverride, AuditDetailItem,
  DashboardPref, UserPermissions, FullAuditProcessConfig,
  CronPrefs, AccessibleArchiveConfig, FullArchiveConfig, UpdatePersonalConfigRequest,
}

export const useSettingsApi = () => {
  const { authFetch } = useAuth()

  const processes = ref<ProcessListItem[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // ============================================================
  // 审核工作台 — 流程列表
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
  // 审核工作台 — 完整配置（租户+用户合并）
  // ============================================================

  async function getFullProcessConfig(processType: string): Promise<FullAuditProcessConfig> {
    return await authFetch<FullAuditProcessConfig>(
      `/api/tenant/settings/processes/${encodeURIComponent(processType)}/full`,
    )
  }

  async function updateProcessConfig(processType: string, config: UpdatePersonalConfigRequest): Promise<void> {
    await authFetch<null>(`/api/tenant/settings/processes/${encodeURIComponent(processType)}`, {
      method: 'PUT',
      body: config,
    })
  }

  // ============================================================
  // 定时任务偏好（默认推送邮箱）
  // ============================================================

  async function getCronPrefs(): Promise<CronPrefs> {
    return await authFetch<CronPrefs>('/api/tenant/settings/cron-prefs')
  }

  async function updateCronPrefs(prefs: CronPrefs): Promise<void> {
    await authFetch<null>('/api/tenant/settings/cron-prefs', {
      method: 'PUT',
      body: prefs,
    })
  }

  // ============================================================
  // 归档复盘 — 可访问配置列表
  // ============================================================

  async function listArchiveConfigs(): Promise<AccessibleArchiveConfig[]> {
    return await authFetch<AccessibleArchiveConfig[]>('/api/tenant/settings/archive-configs')
  }

  // ============================================================
  // 归档复盘 — 完整配置（租户+用户合并）
  // ============================================================

  async function getFullArchiveConfig(processType: string): Promise<FullArchiveConfig> {
    return await authFetch<FullArchiveConfig>(
      `/api/tenant/settings/archive-configs/${encodeURIComponent(processType)}/full`,
    )
  }

  async function updateArchiveConfig(processType: string, config: UpdatePersonalConfigRequest): Promise<void> {
    await authFetch<null>(`/api/tenant/settings/archive-configs/${encodeURIComponent(processType)}`, {
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
    listProcesses, getFullProcessConfig, updateProcessConfig,
    getCronPrefs, updateCronPrefs,
    listArchiveConfigs, getFullArchiveConfig, updateArchiveConfig,
    getDashboardPrefs, updateDashboardPrefs,
    computePermissionLocks,
  }
}
