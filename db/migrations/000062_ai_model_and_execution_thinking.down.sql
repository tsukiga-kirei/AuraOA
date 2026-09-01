-- 000062_ai_model_and_execution_thinking.down.sql

ALTER TABLE archive_logs
    DROP COLUMN IF EXISTS deep_thinking;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS deep_thinking;

ALTER TABLE tenant_llm_message_payloads
    DROP COLUMN IF EXISTS reasoning_content;

ALTER TABLE ai_model_configs
    DROP COLUMN IF EXISTS supports_thinking;
