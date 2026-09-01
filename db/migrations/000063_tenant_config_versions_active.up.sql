-- 000063：为租户基础配置版本添加 is_active 标识，支持多版本并存与自由切换当前可用版本。

ALTER TABLE tenant_config_versions
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 将每个配置的最新版本默认标记为 active
WITH latest_versions AS (
    SELECT DISTINCT ON (tenant_id, module, source_config_id) id
    FROM tenant_config_versions
    ORDER BY tenant_id, module, source_config_id, version_no DESC
)
UPDATE tenant_config_versions
SET is_active = true
WHERE id IN (SELECT id FROM latest_versions);

CREATE INDEX idx_tenant_config_versions_active
    ON tenant_config_versions (tenant_id, module, source_config_id)
    WHERE is_active = true;

COMMENT ON COLUMN tenant_config_versions.is_active IS '是否为当前租户启用/生效的版本';
COMMENT ON COLUMN tenant_config_versions.updated_at IS '最后修改时间，支持对历史版本进行编辑重存';
