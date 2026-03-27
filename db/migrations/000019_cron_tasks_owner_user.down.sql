-- 000019_cron_tasks_owner_user.down.sql

DROP INDEX IF EXISTS idx_cl_task_owner;
ALTER TABLE cron_logs DROP COLUMN IF EXISTS task_owner_user_id;

DROP INDEX IF EXISTS idx_cron_tasks_tenant_owner;
ALTER TABLE cron_tasks DROP COLUMN IF EXISTS owner_user_id;
