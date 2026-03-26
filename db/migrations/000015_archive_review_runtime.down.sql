-- 000015_archive_review_runtime.down.sql

DROP INDEX IF EXISTS idx_arcl_process_created_at;
DROP INDEX IF EXISTS idx_arcl_process_status;
DROP INDEX IF EXISTS idx_arcl_tenant_status;

ALTER TABLE archive_logs DROP COLUMN IF EXISTS updated_at;
ALTER TABLE archive_logs DROP COLUMN IF EXISTS error_message;
ALTER TABLE archive_logs DROP COLUMN IF EXISTS parse_error;
ALTER TABLE archive_logs DROP COLUMN IF EXISTS raw_content;
ALTER TABLE archive_logs DROP COLUMN IF EXISTS confidence;
ALTER TABLE archive_logs DROP COLUMN IF EXISTS ai_reasoning;
ALTER TABLE archive_logs DROP COLUMN IF EXISTS duration_ms;
ALTER TABLE archive_logs DROP COLUMN IF EXISTS process_snapshot;
ALTER TABLE archive_logs DROP COLUMN IF EXISTS status;
