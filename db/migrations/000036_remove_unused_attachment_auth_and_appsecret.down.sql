-- 000036_remove_unused_attachment_auth_and_appsecret.down.sql

ALTER TABLE oa_database_connections
    ADD COLUMN IF NOT EXISTS weaver_appsecret VARCHAR(500) NOT NULL DEFAULT '';

INSERT INTO system_configs (key, value, remark) VALUES
    ('attachment.oa_api_endpoint', '', '【已废弃】OA 附件接口地址（改为按 OA 连接配置）'),
    ('attachment.oa_api_auth_type', 'none', '【已废弃】OA 附件接口认证类型'),
    ('attachment.oa_api_auth_token', '', '【已废弃】OA 附件接口认证 Token'),
    ('attachment.oa_api_auth_header_name', 'X-API-Key', '【已废弃】OA 附件接口认证 Header 名'),
    ('system.max_upload_size_mb', '50', '上传文件最大体积限制（MB）')
ON CONFLICT (key) DO NOTHING;
