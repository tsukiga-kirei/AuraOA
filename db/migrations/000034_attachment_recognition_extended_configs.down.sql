-- 000034_attachment_recognition_extended_configs.down.sql

DELETE FROM system_configs WHERE key IN (
    'attachment.mineru_backend',
    'attachment.mineru_enable_formula',
    'attachment.mineru_enable_table',
    'attachment.mineru_enable_ocr',
    'attachment.mineru_language',
    'attachment.oa_api_auth_type',
    'attachment.oa_api_auth_token',
    'attachment.oa_api_auth_header_name'
);

ALTER TABLE oa_database_connections
    DROP COLUMN IF EXISTS weaver_appid,
    DROP COLUMN IF EXISTS weaver_appsecret,
    DROP COLUMN IF EXISTS weaver_default_user;
