-- 000027_cron_tasks_enhanced_fields.down.sql
ALTER TABLE cron_tasks DROP COLUMN IF EXISTS workflow_ids;
ALTER TABLE cron_tasks DROP COLUMN IF EXISTS date_range;
ALTER TABLE cron_tasks DROP COLUMN IF EXISTS current_log_id;

DELETE FROM system_configs WHERE key IN ('system.smtp_password', 'system.smtp_sender');
