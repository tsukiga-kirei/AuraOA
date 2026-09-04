-- 000067_agent_runtime_and_chat.down.sql

DROP TABLE IF EXISTS chat_messages CASCADE;
DROP TABLE IF EXISTS chat_sessions CASCADE;
DROP TABLE IF EXISTS agent_skills CASCADE;
DROP TABLE IF EXISTS mcp_servers CASCADE;
DROP TABLE IF EXISTS org_role_tool_grants CASCADE;
DROP TABLE IF EXISTS org_role_agent_grants CASCADE;
DROP TABLE IF EXISTS tenant_chat_allocations CASCADE;
DROP TABLE IF EXISTS agent_tool_bindings CASCADE;
DROP TABLE IF EXISTS agent_definitions CASCADE;

DELETE FROM system_configs WHERE key = 'tenant.default_chat_retention_days';

ALTER TABLE tenants
    DROP COLUMN IF EXISTS chat_fallback_model_id,
    DROP COLUMN IF EXISTS chat_primary_model_id,
    DROP COLUMN IF EXISTS chat_retention_days,
    DROP COLUMN IF EXISTS chat_enabled;
