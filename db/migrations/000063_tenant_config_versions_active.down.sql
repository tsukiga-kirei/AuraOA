DROP INDEX IF EXISTS idx_tenant_config_versions_active;

ALTER TABLE tenant_config_versions
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS updated_at;
