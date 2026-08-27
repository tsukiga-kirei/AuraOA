-- 000060：为审核、流程总结和归档复盘引入不可变执行配置版本。
-- 配置版本保存最终生效快照；流程绑定决定后续自动或手动执行继续使用哪一版。

CREATE TABLE execution_config_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module VARCHAR(20) NOT NULL,
    source_config_id UUID NOT NULL,
    version_no INTEGER NOT NULL,
    fingerprint VARCHAR(80) NOT NULL,
    config_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_execution_config_versions_module
        CHECK (module IN ('audit', 'summary', 'archive')),
    CONSTRAINT uq_execution_config_versions_number
        UNIQUE (tenant_id, module, source_config_id, version_no),
    CONSTRAINT uq_execution_config_versions_fingerprint
        UNIQUE (tenant_id, module, source_config_id, fingerprint)
);

CREATE TABLE process_execution_config_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module VARCHAR(20) NOT NULL,
    process_id VARCHAR(100) NOT NULL,
    process_type VARCHAR(200) NOT NULL DEFAULT '',
    config_version_id UUID NOT NULL REFERENCES execution_config_versions(id) ON DELETE RESTRICT,
    bound_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_process_execution_config_bindings_module
        CHECK (module IN ('audit', 'summary', 'archive')),
    CONSTRAINT uq_process_execution_config_bindings_process
        UNIQUE (tenant_id, module, process_id)
);

CREATE INDEX idx_execution_config_versions_source
    ON execution_config_versions (tenant_id, module, source_config_id, version_no DESC);
CREATE INDEX idx_process_execution_config_bindings_version
    ON process_execution_config_bindings (tenant_id, config_version_id);

ALTER TABLE audit_logs
    ADD COLUMN config_version_id UUID REFERENCES execution_config_versions(id) ON DELETE RESTRICT,
    ADD COLUMN config_version_no INTEGER;

ALTER TABLE process_summary_logs
    ADD COLUMN config_version_id UUID REFERENCES execution_config_versions(id) ON DELETE RESTRICT,
    ADD COLUMN config_version_no INTEGER;

ALTER TABLE archive_logs
    ADD COLUMN config_version_id UUID REFERENCES execution_config_versions(id) ON DELETE RESTRICT,
    ADD COLUMN config_version_no INTEGER;

COMMENT ON TABLE execution_config_versions IS '审核、总结、归档复盘共用的不可变最终生效配置版本';
COMMENT ON COLUMN execution_config_versions.config_snapshot IS '执行所需的最终生效字段、规则、尺度和提示词快照';
COMMENT ON TABLE process_execution_config_bindings IS '流程实例与执行配置版本的稳定绑定';
COMMENT ON COLUMN process_execution_config_bindings.config_version_id IS '默认重审继续使用的不可变配置版本';
COMMENT ON COLUMN audit_logs.config_version_id IS '本次审核实际使用的执行配置版本';
COMMENT ON COLUMN process_summary_logs.config_version_id IS '本次总结实际使用的执行配置版本';
COMMENT ON COLUMN archive_logs.config_version_id IS '本次归档复盘实际使用的执行配置版本';
