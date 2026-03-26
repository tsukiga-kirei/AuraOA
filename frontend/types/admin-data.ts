// ─── 数据管理页面 — 公共分页 & 过滤类型 ───────────────────────────────────────

export interface PagedResult<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

// ─── 审核日志 ─────────────────────────────────────────────────────────────────

export interface AuditLogItem {
  id: string
  tenant_id: string
  user_id: string
  user_name: string          // JOIN 用户显示名
  process_id: string
  title: string
  process_type: string
  status: string             // pending / assembling / reasoning / extracting / completed / failed
  recommendation: string     // approve / return / review
  score: number
  confidence: number
  ai_reasoning: string
  duration_ms: number
  audit_result: any          // JSONB 完整审核结果
  parse_error: string
  created_at: string
  updated_at: string
}

export interface AuditLogStats {
  total: number
  pending_ai: number
  ai_done: number
  approve_count: number
  return_count: number
  review_count: number
}

export interface AuditLogFilter {
  status_group?: string       // '' | 'pending_ai' | 'ai_done'
  keyword?: string
  process_type?: string
  recommendation?: string     // '' | 'approve' | 'return' | 'review'
  start_date?: string         // YYYY-MM-DD
  end_date?: string
  page?: number
  page_size?: number
}

// ─── 归档复盘日志 ─────────────────────────────────────────────────────────────

export interface ArchiveLogItem {
  id: string
  tenant_id: string
  user_id: string
  user_name: string
  process_id: string
  title: string
  process_type: string
  status: string              // pending / assembling / reasoning / extracting / completed / failed
  compliance: string          // compliant / partially_compliant / non_compliant
  compliance_score: number
  confidence: number
  ai_reasoning: string
  archive_result: any         // JSONB 完整归档审核结果
  duration_ms: number
  parse_error: string
  created_at: string
  updated_at: string
}

export interface ArchiveLogStats {
  total: number
  compliant: number
  partial: number
  non_compliant: number
  pending_review: number
}

export interface ArchiveLogFilter {
  keyword?: string
  process_type?: string
  compliance?: string         // '' | 'compliant' | 'partially_compliant' | 'non_compliant'
  start_date?: string
  end_date?: string
  page?: number
  page_size?: number
}

// ─── 定时任务日志 ─────────────────────────────────────────────────────────────

export interface CronLogItem {
  id: string
  tenant_id: string
  task_id: string
  task_type: string
  task_label: string
  trigger_type: string        // manual = 手动执行, scheduled = 定时调度
  created_by: string          // 创建人/触发人
  status: string              // running / success / failed
  message: string
  started_at: string
  finished_at: string | null
}

export interface CronLogStats {
  total: number
  success: number
  failed: number
  running: number
}

export interface CronLogFilter {
  keyword?: string
  status?: string             // '' | 'running' | 'success' | 'failed'
  task_type?: string
  trigger_type?: string       // '' | 'manual' | 'scheduled'
  created_by?: string
  start_date?: string
  end_date?: string
  page?: number
  page_size?: number
}
