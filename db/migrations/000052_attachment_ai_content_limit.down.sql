DELETE FROM system_configs
WHERE key IN (
    'attachment.ai_content_limit_mode',
    'attachment.ai_content_max_bytes'
);
