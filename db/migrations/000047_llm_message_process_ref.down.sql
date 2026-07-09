-- 000047_llm_message_process_ref.down.sql

DROP INDEX IF EXISTS idx_tllm_process;

ALTER TABLE tenant_llm_message_logs
    DROP COLUMN IF EXISTS business_log_id,
    DROP COLUMN IF EXISTS process_title,
    DROP COLUMN IF EXISTS process_id;
