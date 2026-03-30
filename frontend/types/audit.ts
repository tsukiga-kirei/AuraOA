// types/audit.ts — 审核工作台相关类型定义

/** 异步审核阶段（与后端 audit_logs.status 一致） */
export type AuditRunStatus = 'pending' | 'assembling' | 'reasoning' | 'extracting' | 'completed' | 'failed'

/** OA 流程列表项（后端聚合 OA 待办 + AI 审核状态返回） */
export interface OAProcessItem {
  process_id: string
  title: string
  applicant: string
  department: string
  process_type: string
  process_type_label: string
  current_node: string
  submit_time: string
  urgency: 'high' | 'medium' | 'low'
  has_audit: boolean
  /** 当前最新一条审核记录的状态（含进行中的异步任务） */
  audit_status?: AuditRunStatus
  audit_result?: AuditResult | null
}

/** AI 审核结构化结果（对应提取阶段 JSON Schema） */
export interface AuditResult {
  id?: string
  trace_id: string
  process_id: string
  /** 异步审核：进行中 / 失败时由后端填充 */
  status?: AuditRunStatus
  error_message?: string
  /** GET /api/audit/jobs/:id 返回的进度步骤 */
  progress_steps?: { key: string; label: string; done?: boolean; current?: boolean; failed?: boolean }[]
  updated_at?: string
  recommendation?: 'approve' | 'return' | 'review'
  overall_score?: number
  rule_results: RuleResultItem[]
  risk_points: string[]
  suggestions: string[]
  confidence: number
  ai_reasoning: string
  duration_ms: number
  created_at?: string
  /** 非空时表示 JSON 解析异常，前端需做降级展示 */
  parse_error?: string
  /** 原始 AI 回复（parse_error 时用于展示） */
  raw_content?: string
}

/** 单条规则校验结果 */
export interface RuleResultItem {
  rule_content: string
  passed: boolean
  reason: string
}

/** 审核链记录（租户级，所有用户共享） */
export interface AuditChainItem {
  id: string
  process_id: string
  process_type: string
  title: string
  user_id: string
  user_name: string
  recommendation: 'approve' | 'return' | 'review'
  score: number
  /** 与 audit_result 并列返回，来自 audit_logs.ai_reasoning 列；JSONB audit_result 内不含推理正文 */
  ai_reasoning?: string
  audit_result: AuditResult
  duration_ms: number
  created_at: string
}

/** 批量审核响应 */
export interface BatchAuditResponse {
  results: AuditResult[]
  total: number
  success: number
  failed: number
}

/** 审核工作台页签 */
export type AuditTab = 'pending_ai' | 'ai_done' | 'completed'

/** 审核执行请求 */
export interface AuditExecuteRequest {
  process_id: string
  process_type: string
  title: string
}

/** 批量审核请求 */
export interface BatchAuditRequest {
  items: AuditExecuteRequest[]
}

/** 审核工作台统计 */
export interface AuditStats {
  pending_ai_count: number
  ai_done_count: number
  completed_count: number
  /** 今日审核成功条数（status=completed 且当日） */
  today_completed_count: number
}

/** POST /api/audit/execute 立即返回 */
export interface AuditSubmitResponse {
  status: AuditRunStatus
  id: string
  trace_id: string
  process_id: string
  created_at: string
}
