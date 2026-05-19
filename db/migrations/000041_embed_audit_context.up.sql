-- 嵌入 OA 审核：触发来源 + OA 上下文锚点 + 流程级 embed 配置

ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS trigger_source VARCHAR(30) NOT NULL DEFAULT 'workbench_manual',
    ADD COLUMN IF NOT EXISTS oa_context_anchor JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN audit_logs.trigger_source IS '审核触发来源：workbench_manual/workbench_batch/embed_auto/embed_manual/cron_scheduled';
COMMENT ON COLUMN audit_logs.oa_context_anchor IS '审核完成时 OA 流程上下文锚点（退回/版本/表单指纹），用于嵌入页判断结论是否过期';

CREATE INDEX IF NOT EXISTS idx_al_trigger_source ON audit_logs (tenant_id, trigger_source, created_at DESC);

ALTER TABLE process_audit_configs
    ADD COLUMN IF NOT EXISTS embed_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS embed_config JSONB NOT NULL DEFAULT '{"auto_audit_on_open":true,"auto_audit_on_stale":true}'::jsonb;

COMMENT ON COLUMN process_audit_configs.embed_enabled IS '是否在 OA 嵌入页启用 AI 辅助审核';
COMMENT ON COLUMN process_audit_configs.embed_config IS '嵌入页行为配置（auto_audit_on_open/auto_audit_on_stale 等）';
