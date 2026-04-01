// types/cron.ts — 定时任务实例相关类型（与后端 CronTaskResponse 对齐）

/** 定时任务实例（后端返回） */
export interface CronTask {
  id: string
  tenant_id: string
  owner_user_id: string       // 任务归属用户（当前登录用户）
  task_type: string           // audit_batch / audit_daily / audit_weekly / archive_batch / archive_daily / archive_weekly
  task_label: string          // 自定义显示名称
  module: string              // audit | archive
  cron_expression: string
  is_active: boolean
  is_builtin: boolean
  push_email: string          // 推送邮箱（报告类任务）
  workflow_ids?: string[]      // 流程多选
  date_range?: number         // 日期范围（天）
  current_log_id?: string | null // 当前运行中的日志 ID
  last_run_at: string | null  // ISO 时间字符串
  next_run_at: string | null
  success_count: number
  fail_count: number
  created_at: string
  updated_at: string
}

/** 创建定时任务请求 */
export interface CreateCronTaskRequest {
  task_type: string
  task_label?: string
  cron_expression: string
  push_email?: string
  workflow_ids?: string[]
  date_range?: number
}

/** 更新定时任务请求（push_email 为 null 时清空，undefined 时不修改） */
export interface UpdateCronTaskRequest {
  task_label?: string
  cron_expression?: string
  push_email?: string | null
  workflow_ids?: string[]
  date_range?: number
}

/** 定时任务执行日志 */
export interface CronLog {
  id: string
  tenant_id: string
  task_id: string
  task_type: string
  task_label: string
  trigger_type?: string
  created_by?: string
  task_owner_user_id?: string | null
  status: 'running' | 'success' | 'failed'
  message: string
  started_at: string
  finished_at: string | null
}

