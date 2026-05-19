DROP INDEX IF EXISTS idx_al_trigger_source;

ALTER TABLE process_audit_configs
    DROP COLUMN IF EXISTS embed_config,
    DROP COLUMN IF EXISTS embed_enabled;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS oa_context_anchor,
    DROP COLUMN IF EXISTS trigger_source;
