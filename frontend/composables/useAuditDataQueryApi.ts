import type {
  AuditDataQuery,
  QueryDefinition,
  QueryVersion,
  QueryCheck,
  QueryEvidence,
  QueryBinding,
  QueryPrompts,
  QueryAssist,
  QueryColumn,
  QueryUsageReference,
} from '~/types/audit-data-query'

/**
 * useAuditDataQueryApi — 租户记忆库的数据记忆能力。
 * 路由前缀 /api/tenant/audit-data-queries：CRUD、versions、publish、references、check、preview、source、schema、assist、prompts。
 */
export const useAuditDataQueryApi = () => {
  const { authFetch } = useAuth()
  const root = '/api/tenant/audit-data-queries'
  const list = (keyword = '', page = 1, page_size = 20) =>
    authFetch<{ items: AuditDataQuery[]; total: number; page: number; page_size: number }>(root, {
      query: { keyword, page, page_size },
    })
  // 左侧配置目录不展示分页，分批读取全部匹配条目，避免只显示首批数据。
  const listAll = async (keyword = '') => {
    const items: AuditDataQuery[] = []
    for (let page = 1; ; page++) {
      const result = await list(keyword, page, 100)
      items.push(...result.items)
      if (!result.items.length || items.length >= result.total) return items
    }
  }
  return {
    source: () =>
      authFetch<{ name: string; dialect: QueryDefinition['dialect'] }>(`${root}/source`),
    list,
    listAll,
    get: (id: string) => authFetch<AuditDataQuery>(`${root}/${id}`),
    save: (id: string | undefined, category: string, definition: QueryDefinition) =>
      authFetch<AuditDataQuery>(id ? `${root}/${id}` : root, {
        method: id ? 'PUT' : 'POST',
        body: { category, definition },
      }),
    versions: (id: string) => authFetch<QueryVersion[]>(`${root}/${id}/versions`),
    publish: (id: string) => authFetch<QueryVersion>(`${root}/${id}/publish`, { method: 'POST' }),
    remove: (id: string) => authFetch<{ deleted: boolean }>(`${root}/${id}`, { method: 'DELETE' }),
    references: (id: string) => authFetch<QueryUsageReference[]>(`${root}/${id}/references`),
    removeReferences: (id: string, module: QueryUsageReference['module'], process_type: string) =>
      authFetch<{ removed_count: number }>(`${root}/${id}/references`, {
        method: 'DELETE',
        body: { module, process_type },
      }),
    check: (definition: QueryDefinition) =>
      authFetch<QueryCheck>(`${root}/check`, { method: 'POST', body: definition }),
    preview: (body: {
      query_id?: string
      version_no?: number
      definition?: QueryDefinition
      parameters?: Record<string, unknown>
      process_id?: string
      bindings?: Record<string, QueryBinding>
    }) => authFetch<QueryEvidence>(`${root}/preview`, { method: 'POST', body }),
    schema: (table: string) =>
      authFetch<{ table: string; dialect: string; columns: QueryColumn[] }>(`${root}/schema`, {
        query: { table },
      }),
    assist: (
      action: 'generate' | 'explain' | 'repair',
      definition: QueryDefinition,
      instruction: string,
      error: string,
    ) =>
      authFetch<QueryAssist>(`${root}/assist`, {
        method: 'POST',
        body: { action, definition, instruction, error },
      }),
    prompts: () => authFetch<QueryPrompts>(`${root}/prompts`),
    savePrompts: (body: QueryPrompts) =>
      authFetch<QueryPrompts>(`${root}/prompts`, { method: 'PUT', body }),
  }
}
