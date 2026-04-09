-- 000029_llm_call_type.up.sql
-- tenant_llm_message_logs 新增 call_type 列，区分推理调用与结构化调用

ALTER TABLE tenant_llm_message_logs
  ADD COLUMN call_type VARCHAR(20) NOT NULL DEFAULT 'reasoning';

COMMENT ON COLUMN tenant_llm_message_logs.call_type
  IS 'LLM 调用类型：reasoning=推理调用 / structured=结构化调用';

CREATE INDEX idx_tllm_call_type
  ON tenant_llm_message_logs(tenant_id, call_type);
