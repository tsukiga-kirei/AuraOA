-- 000018_cron_logs_trigger_type.down.sql
-- 还原：重建 triggered_by 列，删除 trigger_type 和 created_by

ALTER TABLE cron_logs
    ADD COLUMN IF NOT EXISTS triggered_by VARCHAR(100) NOT NULL DEFAULT '';

UPDATE cron_logs SET triggered_by = created_by WHERE trigger_type = 'manual';
UPDATE cron_logs SET triggered_by = 'system' WHERE trigger_type = 'scheduled';

DROP INDEX IF EXISTS idx_cl_trigger_type;
DROP INDEX IF EXISTS idx_cl_created_by;

ALTER TABLE cron_logs DROP COLUMN IF EXISTS trigger_type;
ALTER TABLE cron_logs DROP COLUMN IF EXISTS created_by;

CREATE INDEX IF NOT EXISTS idx_cl_triggered_by ON cron_logs(tenant_id, triggered_by);

