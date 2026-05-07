-- 000035_weaver_api_url.up.sql
-- Ecology9 附件接口地址（按 OA 连接单独配置）

ALTER TABLE oa_database_connections
    ADD COLUMN IF NOT EXISTS weaver_api_url VARCHAR(500) NOT NULL DEFAULT '';

COMMENT ON COLUMN oa_database_connections.weaver_api_url
    IS '泛微 E9 附件接口 URL（仅 oa_type=weaver_e9 使用）';
