-- 000005_system_options_oa_ai.down.sql
-- Rollback: restore tenants columns, drop new tables

-- 恢复租户表列
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS fk_tenants_oa_db;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS fk_tenants_primary_model;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS fk_tenants_fallback_model;

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS oa_type VARCHAR(50) NOT NULL,
    ADD COLUMN IF NOT EXISTS allow_custom_model BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS ai_config JSONB NOT NULL DEFAULT '{}';

ALTER TABLE tenants
    DROP COLUMN IF EXISTS primary_model_id,
    DROP COLUMN IF EXISTS fallback_model_id,
    DROP COLUMN IF EXISTS max_tokens_per_request,
    DROP COLUMN IF EXISTS temperature,
    DROP COLUMN IF EXISTS timeout_seconds,
    DROP COLUMN IF EXISTS retry_count;

DROP TABLE IF EXISTS ai_model_configs;
DROP TABLE IF EXISTS oa_database_connections;
DROP TABLE IF EXISTS ai_provider_options;
DROP TABLE IF EXISTS ai_deploy_type_options;
DROP TABLE IF EXISTS db_driver_options;
DROP TABLE IF EXISTS oa_type_options;
