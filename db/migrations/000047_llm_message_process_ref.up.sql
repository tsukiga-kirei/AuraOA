-- 000047_llm_message_process_ref.up.sql
-- AI 调用记录关联业务流程

ALTER TABLE tenant_llm_message_logs
    ADD COLUMN process_id      VARCHAR(100),
    ADD COLUMN process_title   VARCHAR(500),
    ADD COLUMN business_log_id UUID;

CREATE INDEX idx_tllm_process
    ON tenant_llm_message_logs(tenant_id, process_id, created_at DESC)
    WHERE process_id IS NOT NULL AND process_id <> '';

COMMENT ON COLUMN tenant_llm_message_logs.process_id IS '关联 OA 流程编号';
COMMENT ON COLUMN tenant_llm_message_logs.process_title IS '关联流程标题（冗余，便于列表展示）';
COMMENT ON COLUMN tenant_llm_message_logs.business_log_id IS '关联业务日志 ID（audit_log / archive_log / process_summary_log）';
