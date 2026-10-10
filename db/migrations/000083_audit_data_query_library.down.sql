-- 回滚审核数据查询库及默认页面授权；会删除查询定义、版本、提示词和取数证据。
UPDATE org_roles
SET page_permissions = page_permissions - '/admin/tenant/memory';
ALTER TABLE archive_logs DROP COLUMN query_evidence;
ALTER TABLE audit_logs DROP COLUMN query_evidence;
ALTER TABLE archive_rules DROP COLUMN data_queries;
ALTER TABLE audit_rules DROP COLUMN data_queries;
DROP TABLE audit_data_query_prompts;
DROP TABLE audit_data_query_versions;
DROP TABLE audit_data_queries;
