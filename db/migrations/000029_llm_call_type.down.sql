-- 000029_llm_call_type.down.sql
-- 回滚：删除 call_type 索引和列

DROP INDEX IF EXISTS idx_tllm_call_type;

ALTER TABLE tenant_llm_message_logs
  DROP COLUMN IF EXISTS call_type;
