/** 用户体验 API：/api/tenant/experience（管理列表、审核详情）；/api/embed/audits（评论与可选满意度）。 */
import type { PagedResult } from '~/types/admin-data'
import type { AuditExperienceItem, AgentExperienceItem, AuditExperienceDetail, ExperienceQuery, AuditInteractions, AuditComment, Feedback } from '~/types/experience'

export function useExperienceApi() {
  const { authFetch } = useAuth()
  const { embedAuthHeaders } = useEmbedAuth()
  const queryString = (query: object) => new URLSearchParams(Object.entries(query).filter(([, v]) => v !== undefined && v !== '').map(([k, v]) => [k, String(v)])).toString()
  const embedFetch = <T>(id: string, path: string, body?: object) => $fetch<T>(`/api/embed/audits/${encodeURIComponent(id)}/${path}`, {
    method: body ? 'POST' : 'GET', body, headers: embedAuthHeaders(), credentials: 'include',
  })
  return {
    listAudits: (query: ExperienceQuery) => authFetch<PagedResult<AuditExperienceItem>>(`/api/tenant/experience/audit?${queryString(query)}`),
    listAgents: (query: ExperienceQuery) => authFetch<PagedResult<AgentExperienceItem>>(`/api/tenant/experience/agents?${queryString(query)}`),
    auditDetail: (id: string, page = 1) => authFetch<AuditExperienceDetail>(`/api/tenant/experience/audit/${encodeURIComponent(id)}?page=${page}&page_size=20`),
    interactions: (id: string, page = 1) => embedFetch<AuditInteractions>(id, `interactions?page=${page}&page_size=20`),
    updateComment: (id: string, commentId: string, content: string, feedback: Feedback | null) => embedFetch(id, `comments/${encodeURIComponent(commentId)}/update`, { content, feedback }),
    deleteComment: (id: string, commentId: string) => embedFetch(id, `comments/${encodeURIComponent(commentId)}/delete`, {}),
    comment: (id: string, content: string, feedback: Feedback | null) => embedFetch<AuditComment>(id, 'comments', { content, feedback }),
  }
}
