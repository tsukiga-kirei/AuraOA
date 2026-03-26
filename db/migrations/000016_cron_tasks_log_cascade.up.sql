-- 000016_cron_tasks_log_cascade.up.sql
-- 修复 cron_logs 外键约束（支持任务删除级联）、补充日志字段

-- ============================================================
-- 1. 修复 cron_logs.task_id FK：改为 ON DELETE CASCADE
--    原始 FK 无 ON DELETE 子句（默认 RESTRICT），导致有日志的任务无法删除
-- ============================================================
ALTER TABLE cron_logs DROP CONSTRAINT IF EXISTS cron_logs_task_id_fkey;

ALTER TABLE cron_logs
    ADD CONSTRAINT cron_logs_task_id_fkey
        FOREIGN KEY (task_id)
        REFERENCES cron_tasks(id)
        ON DELETE CASCADE;

-- ============================================================
-- 2. 补充 task_label 字段（执行日志展示时使用，冗余存储）
-- ============================================================
ALTER TABLE cron_logs ADD COLUMN IF NOT EXISTS task_label VARCHAR(200) NOT NULL DEFAULT '';

-- ============================================================
-- 3. 补充查询索引
-- ============================================================
-- 按任务查询最近执行日志（日志列表分页）
CREATE INDEX IF NOT EXISTS idx_cl_task_started ON cron_logs(task_id, started_at DESC);

-- 按租户查询最近执行日志（运营视图）
CREATE INDEX IF NOT EXISTS idx_cl_tenant_started ON cron_logs(tenant_id, started_at DESC);

-- ============================================================
-- 数据库注释
-- ============================================================
COMMENT ON COLUMN cron_logs.task_label IS '任务显示名称（冗余存储，任务删除后日志仍保留历史名称）';

