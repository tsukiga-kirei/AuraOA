-- 000007_audit_configs_rules_presets.down.sql
-- 回滚：删除审核尺度预设表、审核规则表、流程审核配置表

DROP TABLE IF EXISTS system_prompt_templates;
DROP TABLE IF EXISTS audit_rules;
DROP TABLE IF EXISTS process_audit_configs;
