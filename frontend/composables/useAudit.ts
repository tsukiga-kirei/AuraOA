interface OAProcess {
  process_id: string
  title: string
  applicant: string
  submit_time: string
  process_type: string
  status: string
}

interface ChecklistResult {
  rule_id: string
  passed: boolean
  reasoning: string
}

interface AuditResult {
  trace_id: string
  process_id: string
  recommendation: 'approve' | 'reject' | 'revise'
  details: ChecklistResult[]
  ai_reasoning: string
}

export const useAudit = () => {
  const config = useRuntimeConfig()
  const { token } = useAuth()

  const todoList = useState<OAProcess[]>('audit_todo', () => [])
  const currentResult = useState<AuditResult | null>('audit_result', () => null)
  const loading = useState('audit_loading', () => false)

  const headers = computed(() => ({
    Authorization: `Bearer ${token.value}`,
  }))

  const getTodoList = async () => {
    try {
      const data = await $fetch<{ processes: OAProcess[] }>(`${config.public.apiBase}/api/audit/todo`, {
        headers: headers.value,
      })
      todoList.value = data.processes
    } catch {
      todoList.value = []
    }
  }

  const executeAudit = async (processId: string) => {
    loading.value = true
    try {
      const data = await $fetch<AuditResult>(`${config.public.apiBase}/api/audit/execute`, {
        method: 'POST',
        headers: headers.value,
        body: { process_id: processId },
      })
      currentResult.value = data
      return data
    } catch {
      return null
    } finally {
      loading.value = false
    }
  }

  const submitFeedback = async (processId: string, adopted: boolean, actionTaken: string) => {
    await $fetch(`${config.public.apiBase}/api/audit/feedback`, {
      method: 'POST',
      headers: headers.value,
      body: { process_id: processId, adopted, action_taken: actionTaken },
    })
  }

  return { todoList, currentResult, loading, getTodoList, executeAudit, submitFeedback }
}
