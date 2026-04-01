// useCronApi — 定时任务类型配置 & 任务实例 API 调用封装

import type { CronTaskConfig, SaveCronTaskConfigRequest } from '~/types/rules'
import type { CronTask, CreateCronTaskRequest, UpdateCronTaskRequest, CronLog } from '~/types/cron'

export const useCronApi = () => {
  const { authFetch } = useAuth()

  // ============================================================
  // 任务类型配置（租户管理员）
  // ============================================================

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

  // ============================================================
  // 任务实例 CRUD（业务用户）
  // ============================================================

  /** 获取当前租户所有任务实例列表 */
  async function listTasks(): Promise<CronTask[]> {
    return await authFetch<CronTask[]>('/api/tenant/cron/tasks')
  }

  /** 创建新任务实例 */
  async function createTask(req: CreateCronTaskRequest): Promise<CronTask> {
    return await authFetch<CronTask>('/api/tenant/cron/tasks', {
      method: 'POST',
      body: req,
    })
  }

  /** 更新任务实例（cron 表达式 / 标签 / 推送邮箱） */
  async function updateTask(id: string, req: UpdateCronTaskRequest): Promise<CronTask> {
    return await authFetch<CronTask>(`/api/tenant/cron/tasks/${id}`, {
      method: 'PUT',
      body: req,
    })
  }

  /** 删除任务实例（内置任务后端会拦截） */
  async function deleteTask(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/cron/tasks/${id}`, { method: 'DELETE' })
  }

  /** 切换任务启用/禁用状态 */
  async function toggleTask(id: string): Promise<CronTask> {
    return await authFetch<CronTask>(`/api/tenant/cron/tasks/${id}/toggle`, { method: 'POST' })
  }

  /** 立即触发任务执行（异步，后端 goroutine 执行） */
  async function executeTask(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/cron/tasks/${id}/execute`, { method: 'POST' })
  }

  /** 获取任务执行日志（最近 50 条） */
  async function listTaskLogs(id: string): Promise<CronLog[]> {
    return await authFetch<CronLog[]>(`/api/tenant/cron/tasks/${id}/logs`)
  }

  /** 中止当前正在运行的任务实例 */
  async function abortTask(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/cron/tasks/${id}/abort`, { method: 'POST' })
  }

  return {
    listConfigs,
    saveConfig,
    resetConfig,
    listTasks,
    createTask,
    updateTask,
    deleteTask,
    toggleTask,
    executeTask,
    abortTask,
    listTaskLogs,
  }
}
