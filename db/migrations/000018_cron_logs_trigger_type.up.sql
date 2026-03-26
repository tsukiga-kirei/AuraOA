-- 000018_cron_logs_trigger_type.up.sql
-- 将 triggered_by 拆分为 trigger_type（触发类型）+ created_by（创建人/触发人）

-- 1) 新增 trigger_type 列
ALTER TABLE cron_logs
    ADD COLUMN IF NOT EXISTS trigger_type VARCHAR(20) NOT NULL DEFAULT 'scheduled';

COMMENT ON COLUMN cron_logs.trigger_type IS '触发类型：manual = 手动执行，scheduled = 定时调度';

-- 2) 新增 created_by 列（任务创建人/手动触发人姓名）
ALTER TABLE cron_logs
    ADD COLUMN IF NOT EXISTS created_by VARCHAR(100) NOT NULL DEFAULT '';

COMMENT ON COLUMN cron_logs.created_by IS '创建人/触发人：手动触发时为操作用户名，定时触发时为 system';

-- 3) 迁移旧数据：triggered_by = 'system' → trigger_type='scheduled', 其余 → trigger_type='manual'
UPDATE cron_logs SET trigger_type = 'scheduled', created_by = 'system' WHERE triggered_by = 'system';
UPDATE cron_logs SET trigger_type = 'manual', created_by = triggered_by WHERE triggered_by != 'system' AND triggered_by != '';

-- 4) 删除旧列和旧索引
DROP INDEX IF EXISTS idx_cl_triggered_by;
ALTER TABLE cron_logs DROP COLUMN IF EXISTS triggered_by;

-- 5) 新索引
CREATE INDEX IF NOT EXISTS idx_cl_trigger_type ON cron_logs(tenant_id, trigger_type);
CREATE INDEX IF NOT EXISTS idx_cl_created_by ON cron_logs(tenant_id, created_by);

