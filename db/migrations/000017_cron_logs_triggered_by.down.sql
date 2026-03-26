-- 000017_cron_logs_triggered_by.down.sql
DROP INDEX IF EXISTS idx_cl_triggered_by;
ALTER TABLE cron_logs DROP COLUMN IF EXISTS triggered_by;

