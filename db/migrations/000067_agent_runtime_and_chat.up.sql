-- 000067_agent_runtime_and_chat.up.sql

-- 1. 扩展 tenants 表，支持对话开关、保留天数及专用模型配置
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS chat_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS chat_retention_days INT NOT NULL DEFAULT 90,
    ADD COLUMN IF NOT EXISTS chat_primary_model_id UUID REFERENCES ai_model_configs(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS chat_fallback_model_id UUID REFERENCES ai_model_configs(id) ON DELETE SET NULL;

COMMENT ON COLUMN tenants.chat_enabled IS '是否启用 AI 对话工作台';
COMMENT ON COLUMN tenants.chat_retention_days IS '会话保留天数（默认90天）';
COMMENT ON COLUMN tenants.chat_primary_model_id IS '对话首选 AI 模型 ID（为空时使用租户默认主模型）';
COMMENT ON COLUMN tenants.chat_fallback_model_id IS '对话降级 AI 模型 ID（为空时使用租户默认备选模型）';

-- 2. 系统全局默认配置
INSERT INTO system_configs (key, value, remark)
VALUES ('tenant.default_chat_retention_days', '90', '租户默认会话保留天数')
ON CONFLICT (key) DO NOTHING;

-- 3. 智能体定义表 (平台种子 + 租户自定义)
CREATE TABLE IF NOT EXISTS agent_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    agent_code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    system_prompt TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_agent_definitions_tenant_code 
    ON agent_definitions (COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid), agent_code);
CREATE INDEX IF NOT EXISTS idx_agent_definitions_tenant ON agent_definitions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_agent_definitions_code ON agent_definitions(agent_code);

-- 4. 智能体工具绑定表 (agent ⊆ tools/mcp/skills)
CREATE TABLE IF NOT EXISTS agent_tool_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agent_definitions(id) ON DELETE CASCADE,
    tool_type VARCHAR(32) NOT NULL DEFAULT 'system', -- system | mcp | skill
    tool_code VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_agent_tool_binding UNIQUE (agent_id, tool_code)
);

CREATE INDEX IF NOT EXISTS idx_agent_tool_bindings_agent ON agent_tool_bindings(agent_id);

-- 5. 系统管理员分配给租户的配额表
CREATE TABLE IF NOT EXISTS tenant_chat_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    agent_codes JSONB NOT NULL DEFAULT '["oa_query", "oa_assist"]'::jsonb,
    tool_codes JSONB NOT NULL DEFAULT '["list_my_todos", "get_process", "get_approval_flow", "get_latest_audit", "get_latest_summary", "draft_comment", "run_audit", "run_summary", "resolve_oa_url"]'::jsonb,
    skill_codes JSONB NOT NULL DEFAULT '[]'::jsonb,
    allow_custom_skills BOOLEAN NOT NULL DEFAULT FALSE,
    allow_tenant_mcp BOOLEAN NOT NULL DEFAULT FALSE,
    max_mcp_servers INT NOT NULL DEFAULT 0,
    mcp_template_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. 组织角色再分配表 (角色所被授予的智能体与工具)
CREATE TABLE IF NOT EXISTS org_role_agent_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES org_roles(id) ON DELETE CASCADE,
    agent_code VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_role_agent UNIQUE (role_id, agent_code)
);

CREATE TABLE IF NOT EXISTS org_role_tool_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES org_roles(id) ON DELETE CASCADE,
    tool_code VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_role_tool UNIQUE (role_id, tool_code)
);

CREATE INDEX IF NOT EXISTS idx_org_role_agent_grants_role ON org_role_agent_grants(role_id);
CREATE INDEX IF NOT EXISTS idx_org_role_tool_grants_role ON org_role_tool_grants(role_id);

-- 7. MCP 服务器配置表
CREATE TABLE IF NOT EXISTS mcp_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL 表示系统公共模板
    server_code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    transport_type VARCHAR(32) NOT NULL DEFAULT 'http', -- http | sse
    endpoint_url TEXT NOT NULL,
    headers_encrypted TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cached_tools JSONB NOT NULL DEFAULT '[]'::jsonb,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_mcp_servers_tenant_code 
    ON mcp_servers (COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid), server_code);

-- 8. Skills 指令包表
CREATE TABLE IF NOT EXISTS agent_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL 表示平台内置 Skill
    skill_code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    content TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_agent_skills_tenant_code 
    ON agent_skills (COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid), skill_code);

-- 9. 会话表与消息表
CREATE TABLE IF NOT EXISTS chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agent_definitions(id) ON DELETE RESTRICT,
    agent_code VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '新对话',
    source VARCHAR(32) NOT NULL DEFAULT 'standalone', -- standalone | embed
    process_id VARCHAR(128),
    pinned BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_tenant ON chat_sessions(tenant_id, user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL, -- user | assistant | system | tool
    content TEXT NOT NULL DEFAULT '',
    reasoning_content TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'success', -- success | error | interrupted
    tool_calls JSONB NOT NULL DEFAULT '[]'::jsonb,
    token_usage JSONB,
    llm_log_id UUID REFERENCES tenant_llm_message_logs(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id, created_at ASC);

-- ============================================================
-- 字段与表结构注释规范 (COMMENT ON TABLE / COLUMN)
-- ============================================================
COMMENT ON TABLE agent_definitions IS '智能体定义表，存储平台内置种子智能体与租户自定义智能体';
COMMENT ON COLUMN agent_definitions.id IS '智能体唯一标识 (UUID)';
COMMENT ON COLUMN agent_definitions.tenant_id IS '所属租户ID，NULL 表示平台公共内置智能体';
COMMENT ON COLUMN agent_definitions.agent_code IS '智能体唯一标识码（如 oa_query, oa_assist）';
COMMENT ON COLUMN agent_definitions.name IS '智能体展示名称';
COMMENT ON COLUMN agent_definitions.description IS '智能体功能描述与使用场景说明';
COMMENT ON COLUMN agent_definitions.system_prompt IS '智能体角色设定与系统提示词（System Prompt）';
COMMENT ON COLUMN agent_definitions.enabled IS '是否启用该智能体';
COMMENT ON COLUMN agent_definitions.is_system IS '是否为平台内置系统级智能体（内置智能体不可删除）';
COMMENT ON COLUMN agent_definitions.created_at IS '创建时间';
COMMENT ON COLUMN agent_definitions.updated_at IS '最后更新时间';

COMMENT ON TABLE agent_tool_bindings IS '智能体与可用工具/MCP/Skills的绑定关系表';
COMMENT ON COLUMN agent_tool_bindings.id IS '绑定记录唯一标识 (UUID)';
COMMENT ON COLUMN agent_tool_bindings.tenant_id IS '所属租户ID，NULL 表示系统内置绑定';
COMMENT ON COLUMN agent_tool_bindings.agent_id IS '关联的智能体定义ID';
COMMENT ON COLUMN agent_tool_bindings.tool_type IS '工具类型：system (内置系统工具) / mcp (MCP服务工具) / skill (提示词技能)';
COMMENT ON COLUMN agent_tool_bindings.tool_code IS '绑定的工具或技能唯一标识码';
COMMENT ON COLUMN agent_tool_bindings.created_at IS '绑定时间';

COMMENT ON TABLE tenant_chat_allocations IS '系统管理员向租户分配的对话权限与智能体/工具额度配置表';
COMMENT ON COLUMN tenant_chat_allocations.id IS '主键 (UUID)';
COMMENT ON COLUMN tenant_chat_allocations.tenant_id IS '租户ID';
COMMENT ON COLUMN tenant_chat_allocations.agent_codes IS '分配给该租户的可用智能体编码列表 (JSONB 数组)';
COMMENT ON COLUMN tenant_chat_allocations.tool_codes IS '分配给该租户的可用系统工具编码列表 (JSONB 数组)';
COMMENT ON COLUMN tenant_chat_allocations.skill_codes IS '分配给该租户的公共技能编码列表 (JSONB 数组)';
COMMENT ON COLUMN tenant_chat_allocations.allow_custom_skills IS '是否允许租户管理员自建提示词技能';
COMMENT ON COLUMN tenant_chat_allocations.allow_tenant_mcp IS '是否允许租户管理员接入私有 MCP 服务';
COMMENT ON COLUMN tenant_chat_allocations.max_mcp_servers IS '租户最多可配置的 MCP 服务数量上限';
COMMENT ON COLUMN tenant_chat_allocations.mcp_template_ids IS '分配给租户的平台公共 MCP 模版 ID 列表 (JSONB 数组)';
COMMENT ON COLUMN tenant_chat_allocations.created_at IS '创建时间';
COMMENT ON COLUMN tenant_chat_allocations.updated_at IS '最后更新时间';

COMMENT ON TABLE org_role_agent_grants IS '租户内组织角色与智能体的再分配授权表';
COMMENT ON COLUMN org_role_agent_grants.id IS '主键 (UUID)';
COMMENT ON COLUMN org_role_agent_grants.tenant_id IS '所属租户ID';
COMMENT ON COLUMN org_role_agent_grants.role_id IS '组织角色ID (关联 org_roles.id)';
COMMENT ON COLUMN org_role_agent_grants.agent_code IS '授予该角色的智能体编码';
COMMENT ON COLUMN org_role_agent_grants.created_at IS '授权时间';

COMMENT ON TABLE org_role_tool_grants IS '租户内组织角色与系统工具的再分配授权表';
COMMENT ON COLUMN org_role_tool_grants.id IS '主键 (UUID)';
COMMENT ON COLUMN org_role_tool_grants.tenant_id IS '所属租户ID';
COMMENT ON COLUMN org_role_tool_grants.role_id IS '组织角色ID (关联 org_roles.id)';
COMMENT ON COLUMN org_role_tool_grants.tool_code IS '授予该角色的工具编码';
COMMENT ON COLUMN org_role_tool_grants.created_at IS '授权时间';

COMMENT ON TABLE mcp_servers IS 'MCP (Model Context Protocol) 外部服务接入配置表';
COMMENT ON COLUMN mcp_servers.id IS '主键 (UUID)';
COMMENT ON COLUMN mcp_servers.tenant_id IS '所属租户ID，NULL 表示平台系统公共 MCP 模版';
COMMENT ON COLUMN mcp_servers.server_code IS 'MCP 服务唯一编码标识';
COMMENT ON COLUMN mcp_servers.name IS '服务展示名称';
COMMENT ON COLUMN mcp_servers.description IS '服务功能说明';
COMMENT ON COLUMN mcp_servers.transport_type IS '传输协议：http / sse';
COMMENT ON COLUMN mcp_servers.endpoint_url IS 'MCP 服务端点接入 URL 地址';
COMMENT ON COLUMN mcp_servers.headers_encrypted IS '认证请求头密文（AES-GCM 加密存储的 JSON 键值对）';
COMMENT ON COLUMN mcp_servers.enabled IS '是否启用该 MCP 服务';
COMMENT ON COLUMN mcp_servers.cached_tools IS '最近一次从 MCP 服务探测同步到的工具元数据列表 (JSONB)';
COMMENT ON COLUMN mcp_servers.last_synced_at IS '最近一次探测工具成功的时间戳';
COMMENT ON COLUMN mcp_servers.created_at IS '创建时间';
COMMENT ON COLUMN mcp_servers.updated_at IS '最后更新时间';

COMMENT ON TABLE agent_skills IS '智能体技能与结构化提示词指令包表';
COMMENT ON COLUMN agent_skills.id IS '主键 (UUID)';
COMMENT ON COLUMN agent_skills.tenant_id IS '所属租户ID，NULL 表示平台公共内置技能';
COMMENT ON COLUMN agent_skills.skill_code IS '技能唯一标识码';
COMMENT ON COLUMN agent_skills.name IS '技能名称';
COMMENT ON COLUMN agent_skills.description IS '技能触发场景与说明';
COMMENT ON COLUMN agent_skills.content IS '技能注入的具体提示词指令正文模板';
COMMENT ON COLUMN agent_skills.enabled IS '是否启用';
COMMENT ON COLUMN agent_skills.created_at IS '创建时间';
COMMENT ON COLUMN agent_skills.updated_at IS '最后更新时间';

COMMENT ON TABLE chat_sessions IS 'AI 对话工作台用户会话记录表';
COMMENT ON COLUMN chat_sessions.id IS '会话唯一标识 (UUID)';
COMMENT ON COLUMN chat_sessions.tenant_id IS '所属租户ID';
COMMENT ON COLUMN chat_sessions.user_id IS '发起会话的用户ID (关联 users.id)';
COMMENT ON COLUMN chat_sessions.agent_id IS '当前会话绑定的智能体定义ID';
COMMENT ON COLUMN chat_sessions.agent_code IS '智能体编码快照';
COMMENT ON COLUMN chat_sessions.title IS '会话标题';
COMMENT ON COLUMN chat_sessions.source IS '会话来源入口：standalone (独立工作台) / embed (OA内嵌侧边栏)';
COMMENT ON COLUMN chat_sessions.process_id IS '关联的 OA 流程实例 ID（内嵌模式下自动注入）';
COMMENT ON COLUMN chat_sessions.pinned IS '是否置顶会话';
COMMENT ON COLUMN chat_sessions.created_at IS '会话创建时间';
COMMENT ON COLUMN chat_sessions.updated_at IS '会话最新消息交互时间';

COMMENT ON TABLE chat_messages IS 'AI 对话消息流与工具调用记录表';
COMMENT ON COLUMN chat_messages.id IS '消息唯一标识 (UUID)';
COMMENT ON COLUMN chat_messages.session_id IS '所属会话ID (关联 chat_sessions.id)';
COMMENT ON COLUMN chat_messages.tenant_id IS '所属租户ID';
COMMENT ON COLUMN chat_messages.role IS '消息角色：user (用户) / assistant (智能助手) / system (系统提示) / tool (工具返回结果)';
COMMENT ON COLUMN chat_messages.content IS '消息正文内容 (Markdown 格式)';
COMMENT ON COLUMN chat_messages.reasoning_content IS '深度思考/推理模型思维链内容 (Thinking process)';
COMMENT ON COLUMN chat_messages.status IS '消息生成状态：success (成功) / error (失败) / interrupted (用户中断)';
COMMENT ON COLUMN chat_messages.tool_calls IS '本轮生成触发的系统工具与参数调用详情 (JSONB)';
COMMENT ON COLUMN chat_messages.token_usage IS '本次调用的 Token 消耗详情 (prompt_tokens, completion_tokens, total_tokens)';
COMMENT ON COLUMN chat_messages.llm_log_id IS '关联的统一 LLM 调用审计日志ID (关联 tenant_llm_message_logs.id)';
COMMENT ON COLUMN chat_messages.created_at IS '消息创建时间';

-- 10. 写入系统种子智能体 (oa_query 与 oa_assist)
INSERT INTO agent_definitions (tenant_id, agent_code, name, description, system_prompt, enabled, is_system)
VALUES
(
    NULL,
    'oa_query',
    'OA 查询助手',
    '专注于待办列表、流程详情、审批流历史以及历史审核/总结结果的只读查询。',
    '你是 AuraOA 的官方 OA 查询助手。当前时间：{{current_datetime}}（{{weekday}}）。你的职责是协助用户高效检索 OA 待办流程、表单业务数据和审批轨迹。请根据用户提问选择合适的系统工具进行查询。查询数据必须真实可靠，严禁编造流程数据。若缺少关键参数或流程不存在，请清晰向用户解释。',
    TRUE,
    TRUE
),
(
    NULL,
    'oa_assist',
    'OA 辅助办理助手',
    '全能 OA 助手，除具备查询能力外，还支持起草审批意见、触发 AI 审核与流程总结，并生成直达 OA 的办理跳转链接。',
    '你是 AuraOA 的 OA 辅助办理助手。当前时间：{{current_datetime}}（{{weekday}}）。你不仅能够查询待办流程和审批轨迹，还可以针对流程内容起草专业的同意或驳回审批意见、触发 AuraOA 的智能审核和流程总结任务，并为用户生成跳转至 OA 系统的直接办理链接。请注意，你只具有辅助办理能力，不能代替用户在 OA 中最终点击提交或审批。',
    TRUE,
    TRUE
)
ON CONFLICT DO NOTHING;

-- 绑定种子智能体的默认工具
INSERT INTO agent_tool_bindings (tenant_id, agent_id, tool_type, tool_code)
SELECT NULL, a.id, 'system', t.code
FROM agent_definitions a
CROSS JOIN (
    VALUES 
        ('list_my_todos'),
        ('get_process'),
        ('get_approval_flow'),
        ('get_latest_audit'),
        ('get_latest_summary')
) AS t(code)
WHERE a.agent_code = 'oa_query' AND a.tenant_id IS NULL
ON CONFLICT DO NOTHING;

INSERT INTO agent_tool_bindings (tenant_id, agent_id, tool_type, tool_code)
SELECT NULL, a.id, 'system', t.code
FROM agent_definitions a
CROSS JOIN (
    VALUES 
        ('list_my_todos'),
        ('get_process'),
        ('get_approval_flow'),
        ('get_latest_audit'),
        ('get_latest_summary'),
        ('draft_comment'),
        ('run_audit'),
        ('run_summary'),
        ('resolve_oa_url')
) AS t(code)
WHERE a.agent_code = 'oa_assist' AND a.tenant_id IS NULL
ON CONFLICT DO NOTHING;

-- 11. 为已有租户初始化默认配额
INSERT INTO tenant_chat_allocations (tenant_id)
SELECT id FROM tenants
ON CONFLICT (tenant_id) DO NOTHING;
