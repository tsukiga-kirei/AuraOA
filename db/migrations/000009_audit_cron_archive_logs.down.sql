-- 000009_audit_cron_archive_logs.down.sql
-- 回滚：删除归档复盘日志表、定时任务执行日志表、AI审核日志表

DROP TABLE IF EXISTS archive_logs;
DROP TABLE IF EXISTS cron_logs;
DROP TABLE IF EXISTS audit_logs;
