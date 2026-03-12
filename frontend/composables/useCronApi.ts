// useCronApi — 定时任务类型配置 API 调用封装

import type { CronTaskConfig, SaveCronTaskConfigRequest } from '~/types/rules'

export const useCronApi = () => {
  const { authFetch } = useAuth()

  /**
   * 获取所有 6 个任务类型的当前配置（预设+租户覆盖合并）
   * 返回：is_enabled=false 表示该任务类型未启用，配置值为系统预设
   */
  async function listConfigs(): Promise<CronTaskConfig[]> {
    return await authFetch<CronTaskConfig[]>('/api/tenant/cron/configs')
  }

  /**
   * 启用或更新指定任务类型配置（Upsert）
   * @param taskType 任务类型编码（如 audit_batch / archive_daily）
   * @param config 要保存的配置（推送格式、内容模板、批处理限制）
   */
  async function saveConfig(taskType: string, config: SaveCronTaskConfigRequest): Promise<CronTaskConfig> {
    return await authFetch<CronTaskConfig>(`/api/tenant/cron/configs/${taskType}`, {
      method: 'PUT',
      body: config,
    })
  }

  /**
   * 重置指定任务类型为系统预设（删除租户覆盖配置）
   * 重置后 is_enabled 变为 false，content_template 恢复预设
   */
  async function resetConfig(taskType: string): Promise<CronTaskConfig> {
    return await authFetch<CronTaskConfig>(`/api/tenant/cron/configs/${taskType}`, {
      method: 'DELETE',
    })
  }

  return {
    listConfigs,
    saveConfig,
    resetConfig,
  }
}
