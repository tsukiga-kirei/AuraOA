-- 000036_remove_unused_attachment_auth_and_appsecret.up.sql
-- 清理当前不再使用的通用 OA 认证配置与 weaver_appsecret 列
-- （weaver_api_url 建列在 000035 中完成）

DELETE FROM system_configs WHERE key IN (
    'attachment.oa_api_endpoint',
    'attachment.oa_api_auth_type',
    'attachment.oa_api_auth_token',
    'attachment.oa_api_auth_header_name',
    'system.max_upload_size_mb'
);

ALTER TABLE oa_database_connections
    DROP COLUMN IF EXISTS weaver_appsecret;
