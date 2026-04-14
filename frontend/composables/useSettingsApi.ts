/**
 * useSettingsApi — 个人设置相关 API 调用封装
 * 对接后端路由组：
 *   /api/tenant/settings/processes          审核流程列表及个人配置
 *   /api/tenant/settings/cron-prefs         定时任务推送偏好
 *   /api/tenant/settings/archive-configs    归档复盘可访问配置
 *   /api/tenant/settings/dashboard-prefs    仪表板布局偏好
 */

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

  /** 流程列表（响应式，供模板直接绑定） */
  const processes = ref<ProcessListItem[]>([])
  /** 加载状态标志 */
  const loading = ref(false)
  /** 错误信息（null 表示无错误） */
  const error = ref<string | null>(null)

  // ============================================================
  // 审核工作台 — 流程列表
  // ============================================================

  /** 获取当前用户可访问的审核流程列表 */
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
  // 审核工作台 — 完整配置（租户规则 + 用户个人覆盖合并）
  // ============================================================

  /**
   * 获取指定流程的完整审核配置（租户默认配置与用户个人覆盖合并后的结果）。
   * @param processType 流程类型编码
   */
  async function getFullProcessConfig(processType: string): Promise<FullAuditProcessConfig> {
    return await authFetch<FullAuditProcessConfig>(
      `/api/tenant/settings/processes/${encodeURIComponent(processType)}/full`,
    )
  }

  /**
   * 保存用户对指定流程的个人配置覆盖（自定义字段、规则开关、严格度等）。
   * @param processType 流程类型编码
   * @param config 个人配置内容
   */
  async function updateProcessConfig(processType: string, config: UpdatePersonalConfigRequest): Promise<void> {
    await authFetch<null>(`/api/tenant/settings/processes/${encodeURIComponent(processType)}`, {
      method: 'PUT',
      body: config,
    })
  }

  // ============================================================
  // 定时任务偏好（默认推送邮箱）
  // ============================================================

  /** 获取当前用户的定时任务推送偏好（默认邮箱等） */
  async function getCronPrefs(): Promise<CronPrefs> {
    return await authFetch<CronPrefs>('/api/tenant/settings/cron-prefs')
  }

  /**
   * 保存当前用户的定时任务推送偏好。
   * @param prefs 偏好配置（推送邮箱等）
   */
  async function updateCronPrefs(prefs: CronPrefs): Promise<void> {
    await authFetch<null>('/api/tenant/settings/cron-prefs', {
      method: 'PUT',
      body: prefs,
    })
  }

  // ============================================================
  // 归档复盘 — 可访问配置列表
  // ============================================================

  /** 获取当前用户可访问的归档复盘配置列表 */
  async function listArchiveConfigs(): Promise<AccessibleArchiveConfig[]> {
    return await authFetch<AccessibleArchiveConfig[]>('/api/tenant/settings/archive-configs')
  }

  // ============================================================
  // 归档复盘 — 完整配置（租户规则 + 用户个人覆盖合并）
  // ============================================================

  /**
   * 获取指定归档流程的完整配置（租户默认与用户个人覆盖合并后的结果）。
   * @param processType 归档流程类型编码
   */
  async function getFullArchiveConfig(processType: string): Promise<FullArchiveConfig> {
    return await authFetch<FullArchiveConfig>(
      `/api/tenant/settings/archive-configs/${encodeURIComponent(processType)}/full`,
    )
  }

  /**
   * 保存用户对指定归档流程的个人配置覆盖。
   * @param processType 归档流程类型编码
   * @param config 个人配置内容
   */
  async function updateArchiveConfig(processType: string, config: UpdatePersonalConfigRequest): Promise<void> {
    await authFetch<null>(`/api/tenant/settings/archive-configs/${encodeURIComponent(processType)}`, {
      method: 'PUT',
      body: config,
    })
  }

  // ============================================================
  // 仪表板布局偏好
  // ============================================================

  /** 获取当前用户的仪表板布局偏好（组件顺序、显示/隐藏等） */
  async function getDashboardPrefs(): Promise<DashboardPref> {
    return await authFetch<DashboardPref>('/api/tenant/settings/dashboard-prefs')
  }

  /**
   * 保存当前用户的仪表板布局偏好。
   * @param prefs 偏好配置（部分更新）
   */
  async function updateDashboardPrefs(prefs: Partial<DashboardPref>): Promise<void> {
    await authFetch<null>('/api/tenant/settings/dashboard-prefs', {
      method: 'PUT',
      body: prefs,
    })
  }

  // ============================================================
  // 权限锁定状态计算
  // ============================================================

  /**
   * 根据用户权限配置计算各功能的锁定状态。
   * 租户管理员可通过权限配置限制用户自定义能力。
   * @param permissions 用户权限配置（为空时默认全部开放）
   * @returns 各功能的锁定状态标志
   */
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
