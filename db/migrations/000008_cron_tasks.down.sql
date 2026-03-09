-- 000008_cron_tasks.down.sql
-- 回滚：删除定时任务类型配置表、定时任务表

DROP TABLE IF EXISTS cron_task_type_configs;
DROP TABLE IF EXISTS cron_tasks;
