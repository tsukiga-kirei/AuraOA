-- 000062_ai_model_and_execution_thinking.up.sql
-- 支持思考模型（Thinking / Reasoning Models）全链路存储

-- 1. AI 模型配置增加 supports_thinking 字段
ALTER TABLE ai_model_configs
    ADD COLUMN IF NOT EXISTS supports_thinking boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN ai_model_configs.supports_thinking IS '是否支持深度思考模式 (Thinking/Reasoning)';

-- 2. 大模型调用 Payload 增加 reasoning_content 字段
ALTER TABLE tenant_llm_message_payloads
    ADD COLUMN IF NOT EXISTS reasoning_content text NOT NULL DEFAULT '';

COMMENT ON COLUMN tenant_llm_message_payloads.reasoning_content IS '模型返回的深度思考过程/推理链路内容';

-- 3. 审核日志增加 deep_thinking 字段
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS deep_thinking text NOT NULL DEFAULT '';

COMMENT ON COLUMN audit_logs.deep_thinking IS '审核执行阶段产生的深度思考过程内容';

-- 4. 归档复盘日志增加 deep_thinking 字段
ALTER TABLE archive_logs
    ADD COLUMN IF NOT EXISTS deep_thinking text NOT NULL DEFAULT '';

COMMENT ON COLUMN archive_logs.deep_thinking IS '归档复盘执行阶段产生的深度思考过程内容';
