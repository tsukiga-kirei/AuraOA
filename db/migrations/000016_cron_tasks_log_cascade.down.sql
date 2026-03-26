-- 000016_cron_tasks_log_cascade.down.sql
-- 回滚：恢复 cron_logs 外键约束为无级联版本，移除补充字段与索引

DROP INDEX IF EXISTS idx_cl_task_started;
DROP INDEX IF EXISTS idx_cl_tenant_started;

ALTER TABLE cron_logs DROP COLUMN IF EXISTS task_label;

ALTER TABLE cron_logs DROP CONSTRAINT IF EXISTS cron_logs_task_id_fkey;

ALTER TABLE cron_logs
    ADD CONSTRAINT cron_logs_task_id_fkey
        FOREIGN KEY (task_id)
        REFERENCES cron_tasks(id);

