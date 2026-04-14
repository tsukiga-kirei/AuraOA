-- 000030_global_log_retention_config.up.sql
-- 向 system_configs 表新增全局日志文件保留天数配置项

INSERT INTO system_configs (key, value, remark)
VALUES ('system.global_log_retention_days', '30', '全局系统日志文件保留天数')
ON CONFLICT DO NOTHING;
