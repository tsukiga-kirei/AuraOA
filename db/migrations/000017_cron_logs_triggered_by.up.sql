-- 000017_cron_logs_triggered_by.up.sql
-- 为 cron_logs 表新增 triggered_by 字段，记录任务触发来源

ALTER TABLE cron_logs
    ADD COLUMN IF NOT EXISTS triggered_by VARCHAR(100) NOT NULL DEFAULT '';

COMMENT ON COLUMN cron_logs.triggered_by IS '触发来源：手动触发时为用户显示名/用户名，调度器自动触发时为 system';

-- 补充查询索引（按触发人快速筛选）
CREATE INDEX IF NOT EXISTS idx_cl_triggered_by ON cron_logs(tenant_id, triggered_by);

