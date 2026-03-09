-- 000011_tenant_llm_message_logs.up.sql
-- 创建租户大模型消息记录表

-- ============================================================
-- tenant_llm_message_logs — 租户大模型消息记录表
-- ============================================================
CREATE TABLE tenant_llm_message_logs (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id         UUID         REFERENCES users(id),
    model_config_id UUID         REFERENCES ai_model_configs(id),
    request_type    VARCHAR(50)  NOT NULL DEFAULT 'audit',
    input_tokens    INT          NOT NULL DEFAULT 0,
    output_tokens   INT          NOT NULL DEFAULT 0,
    total_tokens    INT          NOT NULL DEFAULT 0,
    duration_ms     INT          NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_tllm_tenant_id ON tenant_llm_message_logs(tenant_id);
CREATE INDEX idx_tllm_created_at ON tenant_llm_message_logs(tenant_id, created_at DESC);
CREATE INDEX idx_tllm_model ON tenant_llm_message_logs(model_config_id);
