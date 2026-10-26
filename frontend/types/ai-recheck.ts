export interface AIRecheckConfig {
  enabled: boolean
  system_prompt: string
  user_prompt: string
}

export interface AIRecheckRecord {
  retry_only?: boolean
  source_log_id?: string
  enabled?: boolean
  status?: 'running' | 'completed' | 'failed' | 'cancelled'
  changed?: boolean
  reuse_original?: boolean
  reason?: string
  error?: string
  duration_ms?: number
  original_result?: Record<string, any>
  final_result?: Record<string, any>
}
