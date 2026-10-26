-- 回滚会移除复核留痕，先保留所需历史数据。
ALTER TABLE audit_logs DROP COLUMN ai_recheck;
ALTER TABLE archive_logs DROP COLUMN ai_recheck;
DELETE FROM system_prompt_templates WHERE prompt_key IN ('audit_system_recheck','audit_user_recheck','archive_system_recheck','archive_user_recheck');
COMMENT ON COLUMN tenant_llm_message_logs.call_type IS 'LLM调用类型：reasoning推理 / structured结构化';
