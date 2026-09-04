-- 000066_oa_connection_jump_url.up.sql
-- 为系统级 OA 数据库连接增加 OA Web 访问地址与流程详情跳转 URL 模板

ALTER TABLE oa_database_connections
    ADD COLUMN IF NOT EXISTS oa_base_url VARCHAR(500) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS process_url_template VARCHAR(500) NOT NULL DEFAULT '';

COMMENT ON COLUMN oa_database_connections.oa_base_url IS 'OA系统Web访问基准域名或URL（用于流程跳转）';
COMMENT ON COLUMN oa_database_connections.process_url_template IS '流程详情跳转URL模板，支持{process_id}/{requestid}占位符；为空且为weaver_e9时默认/workflow/request/ViewRequestForwardSPA.jsp?requestid={process_id}';
