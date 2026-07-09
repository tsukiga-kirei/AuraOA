import type { ProcessInfo, ProcessFields } from '~/types/common'
import type { ProcessSummaryConfig } from '~/types/process-summary'
import type { ExternalContextMount, ExternalContextTestResponse } from '~/types/external-context'

export const useSummaryConfigApi = () => {
  const { authFetch } = useAuth()

  async function listConfigs(): Promise<ProcessSummaryConfig[]> {
    return await authFetch<ProcessSummaryConfig[]>('/api/tenant/summary/configs')
  }

  async function createConfig(config: Partial<ProcessSummaryConfig>): Promise<ProcessSummaryConfig> {
    return await authFetch<ProcessSummaryConfig>('/api/tenant/summary/configs', { method: 'POST', body: config })
  }

  async function updateConfig(id: string, config: Partial<ProcessSummaryConfig>): Promise<ProcessSummaryConfig> {
    return await authFetch<ProcessSummaryConfig>(`/api/tenant/summary/configs/${id}`, { method: 'PUT', body: config })
  }

  async function deleteConfig(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/summary/configs/${id}`, { method: 'DELETE' })
  }

  async function testConnection(processType: string, mainTableName?: string, processTypeLabel?: string): Promise<ProcessInfo> {
    return await authFetch<ProcessInfo>('/api/tenant/summary/configs/test-connection', {
      method: 'POST',
      body: { process_type: processType, main_table_name: mainTableName || '', process_type_label: processTypeLabel || '' },
    })
  }

  async function fetchFields(configId: string): Promise<ProcessFields> {
    return await authFetch<ProcessFields>(`/api/tenant/summary/configs/${configId}/fetch-fields`, { method: 'POST' })
  }

  async function testContext(mounts: ExternalContextMount[], processId?: string): Promise<ExternalContextTestResponse> {
    return await authFetch<ExternalContextTestResponse>('/api/tenant/summary/context/test', {
      method: 'POST',
      body: { ...(processId ? { process_id: processId } : {}), context_mounts: mounts },
    })
  }

  async function testContextConfig(mounts: ExternalContextMount[]): Promise<ExternalContextTestResponse> {
    return await authFetch<ExternalContextTestResponse>('/api/tenant/summary/context/test', {
      method: 'POST',
      body: { context_mounts: mounts },
    })
  }

  async function fetchWorkflowFields(processType: string, workflowId?: string): Promise<ProcessFields> {
    return await authFetch<ProcessFields>('/api/tenant/summary/context/workflow-fields', {
      method: 'POST',
      body: { process_type: processType, workflow_id: workflowId || '' },
    })
  }

  async function searchWorkflows(keyword: string): Promise<ProcessInfo[]> {
    return await authFetch<ProcessInfo[]>('/api/tenant/summary/context/workflow-search', {
      method: 'POST',
      body: { keyword },
    })
  }

  return { listConfigs, createConfig, updateConfig, deleteConfig, testConnection, fetchFields, testContext, testContextConfig, fetchWorkflowFields, searchWorkflows }
}
