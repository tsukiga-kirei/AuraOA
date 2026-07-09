-- 000048_rule_context_mounts.down.sql

ALTER TABLE audit_rules
    DROP COLUMN IF EXISTS context_mounts,
    DROP COLUMN IF EXISTS context_enabled;

ALTER TABLE archive_rules
    DROP COLUMN IF EXISTS context_mounts,
    DROP COLUMN IF EXISTS context_enabled;
