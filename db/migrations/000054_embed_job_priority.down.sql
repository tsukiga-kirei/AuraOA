DROP INDEX IF EXISTS idx_psl_schedule_pending;
DROP INDEX IF EXISTS idx_psl_pending_priority;
DROP INDEX IF EXISTS idx_audit_logs_schedule_pending;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS schedule_config_id,
    DROP COLUMN IF EXISTS attempt_fingerprint,
    DROP COLUMN IF EXISTS trigger_detail;

ALTER TABLE process_summary_logs
    DROP COLUMN IF EXISTS schedule_config_id,
    DROP COLUMN IF EXISTS attempt_fingerprint,
    DROP COLUMN IF EXISTS priority,
    DROP COLUMN IF EXISTS trigger_detail;
