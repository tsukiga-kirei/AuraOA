DROP INDEX IF EXISTS idx_audit_logs_pending_queue_kind;
DROP INDEX IF EXISTS idx_psl_pending_queue_kind;

ALTER TABLE process_summary_logs
    ADD COLUMN priority INTEGER NOT NULL DEFAULT 50;

UPDATE process_summary_logs
SET priority = CASE
        WHEN trigger_detail = 'manual' THEN 100
        WHEN trigger_detail = 'visible_open' THEN 90
        WHEN trigger_detail = 'scheduled_scan' THEN 10
        WHEN trigger_detail = 'save_or_submit' OR trigger_source = 'summary_embed_auto' THEN 20
        ELSE 80
    END,
    trigger_detail = CASE
        WHEN trigger_detail = 'legacy' THEN 'workbench'
        ELSE trigger_detail
    END;

CREATE INDEX idx_psl_pending_priority
    ON process_summary_logs (tenant_id, status, priority DESC, created_at ASC)
    WHERE status = 'pending';

COMMENT ON COLUMN process_summary_logs.priority IS
    '任务优先级，数值越大越优先；交互任务使用独立 Redis Stream';

ALTER TABLE audit_logs
    DROP COLUMN queue_kind;

ALTER TABLE process_summary_logs
    DROP COLUMN queue_kind;
