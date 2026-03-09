-- 000011_tenant_llm_message_logs.down.sql
-- 回滚：删除租户大模型消息记录表

DROP TABLE IF EXISTS tenant_llm_message_logs;
