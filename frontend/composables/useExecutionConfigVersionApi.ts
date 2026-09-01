export type ExecutionConfigModule = 'audit' | 'archive' | 'summary'

export interface ExecutionConfigVersionStatus {
  status: 'current' | 'updated' | 'unversioned'
  current_version_no?: number
  latest_version_no?: number
  has_pending_changes: boolean
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

  return { getStatus, publish }
}
