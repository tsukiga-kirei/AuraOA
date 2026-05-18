-- 回滚：删除 AI 调用重试次数系统配置
DELETE FROM system_configs WHERE key = 'system.ai_retry_count';
