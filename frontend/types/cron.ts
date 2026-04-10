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


// ============================================================
// 定时任务类型配置（从 rules.ts 迁入）
// ============================================================

/** 定时任务内容模板 */
export interface CronContentTemplate {
  subject: string
  header: string
  body_template: string
  footer: string
  batch_limit?: number
}

/** 定时任务类型配置（合并预设+租户覆盖，后端返回） */
export interface CronTaskConfig {
  task_type: string                              // 任务类型编码（如 audit_batch / archive_daily）
  module: 'audit' | 'archive'                   // 所属模块
  label_zh: string                               // 中文显示名称
  label_en: string                               // 英文显示名称
  description_zh: string                         // 中文描述
  description_en: string                         // 英文描述
  default_cron: string                           // 预设默认 Cron 表达式（供参考）
  preset_push_format: string                     // 预设推送格式
  preset_content_template: CronContentTemplate   // 预设内容模板（用于"恢复默认"）
  sort_order: number
  // 租户当前状态
  is_enabled: boolean                            // 租户是否已启用该任务类型
  push_format: 'html' | 'markdown' | 'plain'    // 当前生效的推送格式
  content_template: CronContentTemplate          // 当前生效的内容模板
  batch_limit?: number                           // 当前批处理上限（null 表示使用默认）
}

/** 保存 Cron 任务类型配置请求体 */
export interface SaveCronTaskConfigRequest {
  push_format: string
  content_template: CronContentTemplate
  batch_limit?: number | null
}
