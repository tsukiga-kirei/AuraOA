-- 000055：审核与总结任务按明确队列类型分流

ALTER TABLE audit_logs
    ADD COLUMN queue_kind VARCHAR(20) NOT NULL DEFAULT 'workbench';

ALTER TABLE audit_logs
    ADD CONSTRAINT chk_audit_logs_queue_kind
    CHECK (queue_kind IN ('interactive', 'background', 'scheduled', 'workbench'));

UPDATE audit_logs
SET queue_kind = CASE
        WHEN trigger_detail IN ('manual', 'visible_open')
            OR trigger_source = 'embed_manual'
            THEN 'interactive'
        WHEN trigger_detail = 'scheduled_scan'
            THEN 'scheduled'
        WHEN trigger_source = 'embed_auto'
            THEN 'background'
        ELSE 'workbench'
    END;

ALTER TABLE process_summary_logs
    ADD COLUMN queue_kind VARCHAR(20) NOT NULL DEFAULT 'background';

ALTER TABLE process_summary_logs
    ADD CONSTRAINT chk_process_summary_logs_queue_kind
    CHECK (queue_kind IN ('interactive', 'background', 'scheduled'));

UPDATE process_summary_logs
SET queue_kind = CASE
        WHEN trigger_detail IN ('manual', 'visible_open')
            OR trigger_source = 'summary_embed_manual'
            THEN 'interactive'
        WHEN trigger_detail = 'scheduled_scan'
            THEN 'scheduled'
        ELSE 'background'
    END,
    trigger_detail = CASE
        WHEN trigger_detail = 'workbench' THEN 'legacy'
        ELSE trigger_detail
    END;

-- 000054 的数字 priority 同时表达队列类型与优先级，本迁移改用 queue_kind 后移除。
DROP INDEX IF EXISTS idx_psl_pending_priority;

ALTER TABLE process_summary_logs
    DROP COLUMN priority;

CREATE INDEX idx_audit_logs_pending_queue_kind
    ON audit_logs (tenant_id, status, queue_kind, created_at ASC)
    WHERE status = 'pending';

CREATE INDEX idx_psl_pending_queue_kind
    ON process_summary_logs (tenant_id, status, queue_kind, created_at ASC)
    WHERE status = 'pending';

COMMENT ON COLUMN audit_logs.queue_kind IS
    '任务队列类型：interactive=OA手动/可见页，background=OA保存提交，scheduled=OA流程定时扫描，workbench=系统内审核工作台';
COMMENT ON COLUMN process_summary_logs.queue_kind IS
    '任务队列类型：interactive=手动/可见页，background=保存提交，scheduled=流程定时扫描';
