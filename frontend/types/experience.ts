import type { PagedResult } from '~/types/admin-data'
import type { AuditResult } from '~/types/audit'

export type Feedback = 'like' | 'dislike'
export interface AuditComment {
  id: string
  audit_log_id: string
  oa_user_id: string
  username: string
  content: string
  feedback: Feedback | null
  created_at: string
  updated_at: string
  can_manage: boolean
}
export interface AuditInteractions extends PagedResult<AuditComment> {
  like_count: number
  dislike_count: number
  can_interact: boolean
}
export interface AuditExperienceItem {
  id: string
  process_id: string
  title: string
  process_type: string
  like_count: number
  dislike_count: number
  comment_count: number
  updated_at: string
}
export interface AgentExperienceItem {
  id: string
  session_id: string
  title: string
  agent_name: string
  username: string
  content: string
  feedback: Feedback
  feedback_comment: string
  updated_at: string
}
export interface AuditExperienceDetail {
  audit_result: AuditResult
  interactions: AuditInteractions
}
export interface ExperienceQuery {
  keyword?: string
  feedback?: Feedback | 'comments'
  page: number
  page_size: number
}
