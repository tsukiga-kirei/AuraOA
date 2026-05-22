-- 000043_llm_message_payloads.up.sql
-- 单次 LLM 调用输入输出内容拆表存储，避免统计日志表承载大文本字段

CREATE TABLE tenant_llm_message_payloads (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    llm_message_log_id UUID        NOT NULL UNIQUE REFERENCES tenant_llm_message_logs(id) ON DELETE CASCADE,
    tenant_id          UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    system_prompt      TEXT        NOT NULL DEFAULT '',
    user_prompt        TEXT        NOT NULL DEFAULT '',
    response_content   TEXT        NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tllm_payload_tenant_created
    ON tenant_llm_message_payloads(tenant_id, created_at DESC);

COMMENT ON TABLE tenant_llm_message_payloads IS '租户大模型调用输入输出内容表（大文本，与调用统计表拆分）';
COMMENT ON COLUMN tenant_llm_message_payloads.id IS '主键UUID';
COMMENT ON COLUMN tenant_llm_message_payloads.llm_message_log_id IS '关联 tenant_llm_message_logs.id，一次调用一份输入输出内容';
COMMENT ON COLUMN tenant_llm_message_payloads.tenant_id IS '所属租户ID（冗余，便于权限过滤和清理）';
COMMENT ON COLUMN tenant_llm_message_payloads.system_prompt IS '实际发送给模型的系统提示词';
COMMENT ON COLUMN tenant_llm_message_payloads.user_prompt IS '实际发送给模型的用户提示词';
COMMENT ON COLUMN tenant_llm_message_payloads.response_content IS '模型返回的原始文本内容';
COMMENT ON COLUMN tenant_llm_message_payloads.created_at IS '记录创建时间';
