export type ExecutionConfigModule = 'audit' | 'archive' | 'summary'

export interface ExecutionConfigVersionStatus {
  status: 'current' | 'updated' | 'unversioned'
  active_version_no?: number
  current_version_no?: number
  latest_version_no?: number
  has_pending_changes: boolean
}

export interface TenantConfigVersionHistoryItem {
  id: string
  version_no: number
  fingerprint: string
  config_snapshot: Record<string, any>
  is_active: boolean
  created_at: string
  updated_at?: string
}

/** 查询租户当前配置与已生成执行版本之间的对应状态。 */
export const useExecutionConfigVersionApi = () => {
  const { authFetch } = useAuth()

  async function getStatus(
    module: ExecutionConfigModule,
    sourceConfigId: string,
  ): Promise<ExecutionConfigVersionStatus> {
    const params = new URLSearchParams({ module, source_config_id: sourceConfigId })
    return await authFetch<ExecutionConfigVersionStatus>(
      `/api/tenant/execution-config-versions/status?${params.toString()}`,
    )
  }

  async function publish(
    module: ExecutionConfigModule,
    sourceConfigId: string,
  ): Promise<ExecutionConfigVersionStatus> {
    return await authFetch<ExecutionConfigVersionStatus>(
      '/api/tenant/execution-config-versions/publish',
      {
        method: 'POST',
        body: { module, source_config_id: sourceConfigId },
      },
    )
  }

  /** 保存当前配置草稿，不生成发布版本。 */
  async function saveDraft(
    module: ExecutionConfigModule,
    sourceConfigId: string,
    snapshot: Record<string, any>,
  ): Promise<ExecutionConfigVersionStatus> {
    return await authFetch<ExecutionConfigVersionStatus>(
      '/api/tenant/execution-config-versions/save-draft',
      {
        method: 'POST',
        body: { module, source_config_id: sourceConfigId, snapshot },
      },
    )
  }

  async function getHistory(
    module: ExecutionConfigModule,
    sourceConfigId: string,
  ): Promise<TenantConfigVersionHistoryItem[]> {
    const params = new URLSearchParams({ module, source_config_id: sourceConfigId })
    return await authFetch<TenantConfigVersionHistoryItem[]>(
      `/api/tenant/execution-config-versions/history?${params.toString()}`,
    )
  }

  async function activate(
    module: ExecutionConfigModule,
    sourceConfigId: string,
    versionNo: number,
  ): Promise<ExecutionConfigVersionStatus> {
    return await authFetch<ExecutionConfigVersionStatus>(
      '/api/tenant/execution-config-versions/activate',
      {
        method: 'POST',
        body: { module, source_config_id: sourceConfigId, version_no: versionNo },
      },
    )
  }

  return { getStatus, saveDraft, publish, getHistory, activate }
}
