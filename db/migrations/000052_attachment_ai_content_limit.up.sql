INSERT INTO system_configs (key, value, remark)
VALUES
    ('attachment.ai_content_limit_mode', 'bytes', '发送给 AI 的单附件正文策略：bytes / unlimited'),
    ('attachment.ai_content_max_bytes', '10000', 'bytes 模式下发送给 AI 的单附件正文最大字节数')
ON CONFLICT (key) DO NOTHING;
