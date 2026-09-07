/**
 * Chat & Agent 数据接口类型定义
 * 严格保持与后端 DTO 及 snake_case 规范一致
 */

export interface EffectiveAgentItem {
  id: string
  agent_code: string
  name: string
  description: string
  avatar_emoji?: string
  system_prompt?: string
  is_default?: boolean
  is_system: boolean
  sort_order?: number
  tool_codes?: string[]
}

export interface ChatSessionItem {
  id: string
  title: string
  agent_id: string
  agent_code: string
  agent_name?: string
  agent_avatar_emoji?: string
  pinned: boolean
  last_message_at?: string | null
  created_at: string
  updated_at: string
}

export interface ChatToolExecution {
  tool_code: string
  tool_call_id: string
  ui_kind: string
  status: 'running' | 'success' | 'error'
  arguments?: string
  payload?: any
  thought?: string
}

export interface ChatMessageItem {
  id: string
  session_id: string
  role: 'user' | 'assistant' | 'system'
  agent_code?: string
  content: string
  reasoning_content?: string
  tool_calls?: ChatToolExecution[]
  token_usage?: { input_tokens: number; output_tokens: number; total_tokens: number }
  status?: string
  error?: string
  created_at: string
  // 前端流式补充字段
  streaming?: boolean
  duration_ms?: number
}

export interface ChatSessionDetail {
  session: ChatSessionItem
  messages: ChatMessageItem[]
  agent?: EffectiveAgentItem
}

export interface SendMessageStreamRequest {
  content: string
}

export interface CreateSessionRequest {
  agent_code?: string
  title?: string
}

export interface UpdateSessionRequest {
  title?: string
  pinned?: boolean
}

// 智能体管理相关类型
export interface AgentDefinitionItem {
  enabled: boolean
  id: string
  tenant_id: string | null
  agent_code: string
  name: string
  description: string
  avatar_emoji?: string
  system_prompt: string
  is_default?: boolean
  is_system: boolean
  sort_order?: number
  tool_codes: string[]
  created_at: string
  updated_at: string
}

export interface SaveAgentRequest {
  enabled: boolean
  id?: string
  agent_code: string
  name: string
  description: string
  avatar_emoji?: string
  system_prompt: string
  is_default?: boolean
  sort_order?: number
  tool_codes: string[]
}

export interface MCPServerItem {
  id: string
  tenant_id: string | null
  server_code: string
  name: string
  description: string
  transport_type: string
  endpoint_url: string
  headers?: string
  enabled: boolean
  cached_tools: Array<{
    name: string
    description: string
    input_schema: Record<string, any>
  }>
  agent_codes?: string[]
  created_at: string
  updated_at: string
}

export interface SaveMCPServerRequest {
  id?: string
  server_code: string
  name: string
  description: string
  transport_type: string
  endpoint_url: string
  headers?: string
  enabled: boolean
  agent_codes?: string[]
}

export interface AgentSkillItem {
  enabled: boolean
  id: string
  tenant_id: string | null
  skill_code: string
  name: string
  description: string
  content: string
  input_schema?: Record<string, any>
  is_system: boolean
  agent_codes?: string[]
  created_at: string
  updated_at: string
}

export interface SaveSkillRequest {
  enabled: boolean
  id?: string
  skill_code: string
  name: string
  description: string
  content: string
  input_schema?: Record<string, any>
  agent_codes?: string[]
}

export interface SystemToolCatalogItem {
  tool_code: string
  name: string
  description: string
  parameters?: string
  ui_kind?: string
}

export interface AgentCatalogResponse {
  tool_catalog: SystemToolCatalogItem[]
  agent_catalog: AgentDefinitionItem[]
  skill_catalog: AgentSkillItem[]
  mcp_templates: MCPServerItem[]
}

export interface TenantChatAllocationItem {
  tenant_id: string
  chat_enabled: boolean
  chat_retention_days: number
  chat_primary_model_id: string | null
  chat_fallback_model_id: string | null
  agent_codes: string[]
  tool_codes: string[]
  skill_codes: string[]
  allow_custom_skills: boolean
  allow_tenant_mcp: boolean
  max_mcp_servers: number
  mcp_template_ids: string[]
}
