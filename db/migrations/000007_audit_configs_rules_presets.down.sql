-- 000007_audit_configs_rules_presets.down.sql
-- 回滚：删除归档规则表、归档配置表、系统提示词模板表、审核规则表、流程审核配置表

DROP TABLE IF EXISTS archive_rules;
DROP TABLE IF EXISTS process_archive_configs;
DROP TABLE IF EXISTS system_prompt_templates;
DROP TABLE IF EXISTS audit_rules;
DROP TABLE IF EXISTS process_audit_configs;
