/**
 * Chat & Agent 数据接口类型定义
 * 严格保持与后端 DTO 及 snake_case 规范一致
 */

export interface EffectiveAgentItem {
  id: string
  code: string
  name: string
  description: string
  avatar_emoji: string
  system_prompt_override: string
  is_default: boolean
  is_system: boolean
  sort_order: number
}

export interface ChatSessionItem {
  id: string
  title: string
  agent_id: string
  agent_code: string
  agent_name: string
  agent_avatar_emoji: string
  is_pinned: boolean
  last_message_at: string | null
  created_at: string
  updated_at: string
}

export interface ChatToolExecution {
  tool_code: string
  tool_name: string
  arguments: Record<string, any>
  result: any
  error?: string
  execution_ms: number
}

export interface ChatMessageItem {
  id: string
  session_id: string
  sender_type: 'user' | 'assistant' | 'system'
  agent_code?: string
  content: string
  reasoning_content?: string
  tool_executions?: ChatToolExecution[]
  token_cost?: number
  created_at: string
  // 前端流式补充字段
  streaming?: boolean
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
  is_pinned?: boolean
}

// 智能体管理相关类型
export interface AgentDefinitionItem {
  id: string
  tenant_id: string | null
  code: string
  name: string
  description: string
  avatar_emoji: string
  system_prompt_override: string
  is_default: boolean
  is_system: boolean
  sort_order: number
  tools: string[]
  created_at: string
  updated_at: string
}

export interface SaveAgentRequest {
  id?: string
  code: string
  name: string
  description: string
  avatar_emoji: string
  system_prompt_override: string
  is_default: boolean
  sort_order: number
  tools: string[]
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
  is_enabled: boolean
  discovered_tools: Array<{
    name: string
    description: string
    input_schema: Record<string, any>
  }>
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
  is_enabled: boolean
}

export interface AgentSkillItem {
  id: string
  tenant_id: string | null
  skill_code: string
  name: string
  description: string
  prompt_template: string
  input_schema?: Record<string, any>
  is_system: boolean
  created_at: string
  updated_at: string
}

export interface SaveSkillRequest {
  id?: string
  skill_code: string
  name: string
  description: string
  prompt_template: string
  input_schema?: Record<string, any>
}

export interface SystemToolCatalogItem {
  code: string
  name: string
  description: string
  parameters: Record<string, any>
}

export interface AgentCatalogResponse {
  system_tools: SystemToolCatalogItem[]
  system_agents: AgentDefinitionItem[]
  system_skills: AgentSkillItem[]
  mcp_templates: MCPServerItem[]
}

export interface TenantChatAllocationItem {
  tenant_id: string
  chat_enabled: boolean
  chat_retention_days: number
  primary_model_id: string | null
  fallback_model_id: string | null
  agent_codes: string[]
  tool_codes: string[]
  skill_codes: string[]
  allow_custom_skills: boolean
  allow_tenant_mcp: boolean
  max_mcp_servers: number
  mcp_template_ids: string[]
}
