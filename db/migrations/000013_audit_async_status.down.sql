-- 000013_audit_async_status.down.sql

DROP INDEX IF EXISTS idx_al_completed_today;
DROP INDEX IF EXISTS idx_al_process_status;
DROP INDEX IF EXISTS idx_al_tenant_status;

ALTER TABLE audit_logs DROP COLUMN IF EXISTS updated_at;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS error_message;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS status;
