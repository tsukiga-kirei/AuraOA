-- 新增 AI 调用重试次数系统配置，默认 3 次。
-- 主模型失败后切换到备用模型，备用模型也按此次数重试。
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM system_configs WHERE key = 'system.ai_retry_count') THEN
        INSERT INTO system_configs (key, value, remark)
        VALUES ('system.ai_retry_count', '3', 'AI 模型调用失败后的重试次数（主模型和备用模型各自重试该次数）');
    END IF;
END $$;
