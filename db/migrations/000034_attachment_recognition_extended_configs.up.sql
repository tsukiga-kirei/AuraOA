-- 000034_attachment_recognition_extended_configs.up.sql
-- 扩展附件识别相关配置：
--   1) MinerU 高级参数（backend / 公式 / 表格 / OCR / 语言）
--   2) OA 附件接口认证（none / bearer / basic / custom_header）
--   3) oa_database_connections 增加泛微 E9 原生 API 密钥字段
-- 关联文档：docs/oa-configurations/01-attachment-recognition.md

-- ============================================================
-- 1) MinerU 高级参数 + OA 接口认证（system_configs）
-- ============================================================
INSERT INTO system_configs (key, value, remark) VALUES
    ('attachment.mineru_backend',         'pipeline', 'MinerU 解析 backend：pipeline / vlm-auto-engine / vlm-http-client / hybrid-auto-engine / hybrid-http-client'),
    ('attachment.mineru_enable_formula',  'true',     'MinerU 是否启用公式识别'),
    ('attachment.mineru_enable_table',    'true',     'MinerU 是否启用表格识别'),
    ('attachment.mineru_enable_ocr',      'true',     'MinerU 是否启用 OCR'),
    ('attachment.mineru_language',        'ch',       'MinerU 解析语言（ch / en / 等，按 MinerU 服务支持的列表填写）'),
    ('attachment.oa_api_auth_type',       'none',     'OA 附件接口认证类型：none / bearer / basic / custom_header'),
    ('attachment.oa_api_auth_token',      '',         'OA 附件接口认证 Token：bearer 模式直接填令牌；basic 模式填 base64(user:pass)；custom_header 模式填头值'),
    ('attachment.oa_api_auth_header_name','X-API-Key','OA 附件接口认证 Header 名（仅 custom_header 模式生效）')
ON CONFLICT (key) DO NOTHING;

-- ============================================================
-- 2) oa_database_connections 增加泛微 E9 原生 API 密钥
-- 仅在 oa_type='weaver_e9' 时使用；其他 OA 类型保持空值即可。
-- ============================================================
ALTER TABLE oa_database_connections
    ADD COLUMN IF NOT EXISTS weaver_appid        VARCHAR(200) NOT NULL DEFAULT '', -- 泛微 E9 应用 ID
    ADD COLUMN IF NOT EXISTS weaver_appsecret    VARCHAR(500) NOT NULL DEFAULT '', -- 泛微 E9 应用密钥（明文，加密存储）
    ADD COLUMN IF NOT EXISTS weaver_default_user VARCHAR(200) NOT NULL DEFAULT ''; -- 调用泛微 E9 接口的默认用户 loginid

COMMENT ON COLUMN oa_database_connections.weaver_appid        IS '泛微 E9 应用 ID（仅 oa_type=weaver_e9 使用）';
COMMENT ON COLUMN oa_database_connections.weaver_appsecret    IS '泛微 E9 应用密钥（仅 oa_type=weaver_e9 使用，加密存储）';
COMMENT ON COLUMN oa_database_connections.weaver_default_user IS '调用泛微 E9 接口的默认用户 loginid（仅 oa_type=weaver_e9 使用）';
