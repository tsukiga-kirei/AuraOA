-- 000061：将租户基础配置版本与最终执行版本分层。
-- 租户版本只记录管理员保存的字段、规则和提示词；执行版本再叠加个人配置，二者不能共用版本号。

CREATE TABLE tenant_config_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module VARCHAR(20) NOT NULL,
    source_config_id UUID NOT NULL,
    version_no INTEGER NOT NULL,
    fingerprint VARCHAR(80) NOT NULL,
    config_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_tenant_config_versions_module
        CHECK (module IN ('audit', 'summary', 'archive')),
    CONSTRAINT uq_tenant_config_versions_number
        UNIQUE (tenant_id, module, source_config_id, version_no),
    CONSTRAINT uq_tenant_config_versions_fingerprint
        UNIQUE (tenant_id, module, source_config_id, fingerprint)
);

ALTER TABLE execution_config_versions
    ADD COLUMN base_config_version_id UUID REFERENCES tenant_config_versions(id) ON DELETE RESTRICT;

CREATE INDEX idx_tenant_config_versions_source
    ON tenant_config_versions (tenant_id, module, source_config_id, version_no DESC);
CREATE INDEX idx_execution_config_versions_base
    ON execution_config_versions (tenant_id, base_config_version_id);

COMMENT ON TABLE tenant_config_versions IS '管理员租户配置的不可变基础版本，不包含用户个人覆盖';
COMMENT ON COLUMN tenant_config_versions.config_snapshot IS '租户字段、规则、尺度和提示词等基础配置快照';
COMMENT ON COLUMN execution_config_versions.base_config_version_id IS
    '最终执行快照所基于的租户基础配置版本；历史空值表示创建时尚未分层记录';
