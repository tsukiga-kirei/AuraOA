-- 000011_tenant_llm_message_logs.up.sql
-- 创建租户大模型消息记录表（Token用量追踪）

-- ============================================================
-- tenant_llm_message_logs — 租户大模型调用记录表
-- ============================================================
CREATE TABLE tenant_llm_message_logs (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    tenant_id       UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- 所属租户ID（用于配额统计）
    user_id         UUID         REFERENCES users(id),                              -- 发起请求的用户ID（NULL表示系统自动触发）
    model_config_id UUID         REFERENCES ai_model_configs(id),                  -- 使用的AI模型配置ID（NULL表示模型已被删除）
    request_type    VARCHAR(50)  NOT NULL DEFAULT 'audit',                          -- 请求类型：audit=审核/archive=归档复盘/other=其他
    input_tokens    INT          NOT NULL DEFAULT 0,                                -- 输入Token消耗数
    output_tokens   INT          NOT NULL DEFAULT 0,                                -- 输出Token消耗数
    total_tokens    INT          NOT NULL DEFAULT 0,                                -- 总Token消耗数（input + output）
    duration_ms     INT          NOT NULL DEFAULT 0,                                -- 本次AI调用耗时（毫秒）
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()                             -- 记录创建时间
);

CREATE INDEX idx_tllm_tenant_id  ON tenant_llm_message_logs(tenant_id);
CREATE INDEX idx_tllm_created_at ON tenant_llm_message_logs(tenant_id, created_at DESC);
CREATE INDEX idx_tllm_model      ON tenant_llm_message_logs(model_config_id);
