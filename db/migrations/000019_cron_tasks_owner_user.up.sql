-- 000019_cron_tasks_owner_user.up.sql
-- 定时任务按用户归属：cron_tasks.owner_user_id；日志冗余 task_owner_user_id
--
-- 假定尚未上线：直接清空定时任务与执行日志，不做历史数据回填。
-- 若环境已有需保留的 cron 数据，请先自行备份或改为手工迁移。

DELETE FROM cron_logs;
DELETE FROM cron_tasks;

ALTER TABLE cron_tasks
    ADD COLUMN IF NOT EXISTS owner_user_id UUID REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE cron_tasks
    ALTER COLUMN owner_user_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_cron_tasks_tenant_owner ON cron_tasks (tenant_id, owner_user_id);

COMMENT ON COLUMN cron_tasks.owner_user_id IS '任务归属用户（定时任务按人隔离，执行 OA 待办/归档以该用户身份）';

ALTER TABLE cron_logs
    ADD COLUMN IF NOT EXISTS task_owner_user_id UUID REFERENCES users (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_cl_task_owner ON cron_logs (tenant_id, task_owner_user_id);

COMMENT ON COLUMN cron_logs.task_owner_user_id IS '任务归属用户ID（冗余，与执行时 cron_tasks.owner_user_id 一致）';
