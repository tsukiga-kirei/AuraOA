DROP INDEX IF EXISTS idx_execution_config_versions_base;

ALTER TABLE execution_config_versions
    DROP COLUMN IF EXISTS base_config_version_id;

DROP INDEX IF EXISTS idx_tenant_config_versions_source;
DROP TABLE IF EXISTS tenant_config_versions;
