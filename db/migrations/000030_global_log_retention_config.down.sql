-- 000030_global_log_retention_config.down.sql
-- 回滚：删除全局日志文件保留天数配置项

DELETE FROM system_configs WHERE key = 'system.global_log_retention_days';
