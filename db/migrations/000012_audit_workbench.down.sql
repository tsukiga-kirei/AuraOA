-- Rollback migration 000012
DROP INDEX IF EXISTS idx_audit_logs_tenant_process;
DROP INDEX IF EXISTS idx_audit_logs_process_id;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS ai_reasoning,
    DROP COLUMN IF EXISTS confidence,
    DROP COLUMN IF EXISTS raw_content,
    DROP COLUMN IF EXISTS parse_error;
