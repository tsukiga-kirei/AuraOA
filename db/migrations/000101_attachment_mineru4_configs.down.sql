-- 回退后恢复旧版引擎选择，避免遗留不可用的提供商标识。
UPDATE system_configs SET value = 'mineru' WHERE key = 'attachment.ocr_provider' AND value = 'mineru4';
DELETE FROM system_configs WHERE key IN (
'attachment.mineru4_endpoint', 'attachment.mineru4_api_key',
'attachment.mineru4_tier', 'attachment.mineru4_ocr_mode');
