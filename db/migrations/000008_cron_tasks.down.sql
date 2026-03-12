-- 000008_cron_tasks.down.sql
-- 回滚：删除定时任务相关表（顺序与 up 相反）

DROP TABLE IF EXISTS cron_task_type_configs;
DROP TABLE IF EXISTS cron_task_type_presets;
DROP TABLE IF EXISTS cron_tasks;
