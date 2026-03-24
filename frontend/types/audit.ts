// types/audit.ts — 审核工作台相关类型定义

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
  audit_result?: AuditResult | null
}

/** AI 审核结构化结果（对应提取阶段 JSON Schema） */
export interface AuditResult {
  id?: string
  trace_id: string
  process_id: string
  recommendation: 'approve' | 'return' | 'review'
  overall_score: number
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
}
