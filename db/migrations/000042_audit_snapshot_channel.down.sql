DROP INDEX IF EXISTS idx_aps_tenant_channel_updated;

ALTER TABLE audit_process_snapshots
    DROP CONSTRAINT IF EXISTS audit_process_snapshots_tenant_process_channel_key;

DELETE FROM audit_process_snapshots WHERE channel = 'embed';

ALTER TABLE audit_process_snapshots DROP COLUMN IF EXISTS channel;

ALTER TABLE audit_process_snapshots
    ADD CONSTRAINT audit_process_snapshots_tenant_id_process_id_key UNIQUE (tenant_id, process_id);
