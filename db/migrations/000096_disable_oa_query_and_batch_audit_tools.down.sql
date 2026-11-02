-- 000096_disable_oa_query_and_batch_audit_tools.down.sql
UPDATE agent_definitions
SET enabled = TRUE,
    updated_at = NOW()
WHERE agent_code = 'oa_query' AND is_system = TRUE;

ALTER TABLE tenant_chat_allocations 
    ALTER COLUMN agent_codes SET DEFAULT '["oa_query", "oa_assist"]'::jsonb;
