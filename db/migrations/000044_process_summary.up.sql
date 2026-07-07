-- 000044：OA 嵌入流程总结

CREATE TABLE process_summary_configs (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    process_type       VARCHAR(200) NOT NULL,
    process_type_label VARCHAR(200) NOT NULL DEFAULT '',
    main_table_name    VARCHAR(200) NOT NULL DEFAULT '',
    main_fields        JSONB NOT NULL DEFAULT '[]',
    detail_tables      JSONB NOT NULL DEFAULT '[]',
    summary_blocks     JSONB NOT NULL DEFAULT '[]',
    embed_enabled      BOOLEAN NOT NULL DEFAULT false,
    embed_config       JSONB NOT NULL DEFAULT '{}',
    status             VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, process_type)
);

CREATE INDEX idx_psc_tenant_status ON process_summary_configs (tenant_id, status);

COMMENT ON TABLE process_summary_configs IS '流程总结配置（租户级，供 OA 嵌入页使用）';
COMMENT ON COLUMN process_summary_configs.summary_blocks IS '总结块配置 JSON 数组：字段选择、用户提示词、排序与启用状态';
COMMENT ON COLUMN process_summary_configs.embed_config IS 'OA 嵌入总结行为配置，如自动总结、过期自动刷新';

CREATE TABLE process_summary_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    process_id        VARCHAR(100) NOT NULL,
    title             VARCHAR(500) NOT NULL DEFAULT '',
    process_type      VARCHAR(200) NOT NULL DEFAULT '',
    status            VARCHAR(20) NOT NULL DEFAULT 'completed',
    summary_result    JSONB NOT NULL DEFAULT '{}',
    process_snapshot  JSONB NOT NULL DEFAULT '{}',
    duration_ms       INTEGER NOT NULL DEFAULT 0,
    raw_content       TEXT NOT NULL DEFAULT '',
    parse_error       TEXT NOT NULL DEFAULT '',
    error_message     TEXT NOT NULL DEFAULT '',
    trigger_source    VARCHAR(30) NOT NULL DEFAULT 'summary_embed_manual',
    oa_context_anchor JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_psl_tenant_process_created ON process_summary_logs (tenant_id, process_id, created_at DESC);
CREATE INDEX idx_psl_tenant_status_created ON process_summary_logs (tenant_id, status, created_at DESC);

COMMENT ON TABLE process_summary_logs IS '流程总结执行日志';
COMMENT ON COLUMN process_summary_logs.summary_result IS '结构化总结结果，包含 blocks 数组';
COMMENT ON COLUMN process_summary_logs.process_snapshot IS '总结时使用的流程字段、附件与审批流快照';
COMMENT ON COLUMN process_summary_logs.parse_error IS '模型返回结构无法完整解析时的错误；已兜底包装为可展示结果';

CREATE TABLE process_summary_snapshots (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    process_id          VARCHAR(100) NOT NULL,
    valid_log_ids       JSONB NOT NULL DEFAULT '[]',
    latest_valid_log_id UUID NOT NULL REFERENCES process_summary_logs(id) ON DELETE CASCADE,
    title               VARCHAR(500) NOT NULL DEFAULT '',
    process_type        VARCHAR(200) NOT NULL DEFAULT '',
    block_count         INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, process_id)
);

CREATE INDEX idx_pss_tenant_updated ON process_summary_snapshots (tenant_id, updated_at DESC);
CREATE INDEX idx_pss_tenant_process ON process_summary_snapshots (tenant_id, process_id);

COMMENT ON TABLE process_summary_snapshots IS '流程总结有效结果快照';
COMMENT ON COLUMN process_summary_snapshots.valid_log_ids IS '有效 process_summary_logs.id 的 JSON 数组（字符串 UUID，按时间顺序追加）';
