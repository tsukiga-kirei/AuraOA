ALTER TABLE archive_logs
    DROP COLUMN IF EXISTS config_version_no,
    DROP COLUMN IF EXISTS config_version_id;

ALTER TABLE process_summary_logs
    DROP COLUMN IF EXISTS config_version_no,
    DROP COLUMN IF EXISTS config_version_id;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS config_version_no,
    DROP COLUMN IF EXISTS config_version_id;

DROP TABLE IF EXISTS process_execution_config_bindings;
DROP TABLE IF EXISTS execution_config_versions;
