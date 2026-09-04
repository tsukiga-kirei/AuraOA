-- 000066_oa_connection_jump_url.down.sql
-- 回滚：删除 OA Web 访问地址与流程跳转 URL 模板字段

ALTER TABLE oa_database_connections
    DROP COLUMN IF EXISTS oa_base_url,
    DROP COLUMN IF EXISTS process_url_template;
