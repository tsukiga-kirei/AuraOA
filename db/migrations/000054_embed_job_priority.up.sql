-- 000054：嵌入任务来源、总结优先级、失败指纹和定时任务归属

ALTER TABLE process_summary_logs
    ADD COLUMN trigger_detail VARCHAR(30) NOT NULL DEFAULT '',
    ADD COLUMN priority INTEGER NOT NULL DEFAULT 50,
    ADD COLUMN attempt_fingerprint VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN schedule_config_id UUID;

UPDATE process_summary_logs
SET trigger_detail = CASE
        WHEN trigger_source = 'summary_embed_manual' THEN 'manual'
        WHEN trigger_source = 'summary_embed_auto' THEN 'legacy_auto'
        ELSE 'workbench'
    END,
    priority = CASE
        WHEN trigger_source = 'summary_embed_manual' THEN 100
        WHEN trigger_source = 'summary_embed_auto' THEN 20
        ELSE 80
    END
WHERE trigger_detail = '';

CREATE INDEX idx_psl_pending_priority
    ON process_summary_logs (tenant_id, status, priority DESC, created_at ASC)
    WHERE status = 'pending';

CREATE INDEX idx_psl_schedule_pending
    ON process_summary_logs (tenant_id, schedule_config_id, status)
    WHERE schedule_config_id IS NOT NULL;

COMMENT ON COLUMN process_summary_logs.trigger_detail IS
    '详细触发来源：manual、visible_open、save_or_submit、scheduled_scan、workbench';
COMMENT ON COLUMN process_summary_logs.priority IS
    '任务优先级，数值越大越优先；交互任务使用独立 Redis Stream';
COMMENT ON COLUMN process_summary_logs.attempt_fingerprint IS
    '本次执行依赖指纹；相同指纹失败后自动来源不重复执行';
COMMENT ON COLUMN process_summary_logs.schedule_config_id IS
    'scheduled_scan 对应的流程配置 ID，用于关闭定时任务时取消未执行任务';

ALTER TABLE audit_logs
    ADD COLUMN trigger_detail VARCHAR(30) NOT NULL DEFAULT '',
    ADD COLUMN attempt_fingerprint VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN schedule_config_id UUID;

UPDATE audit_logs
SET trigger_detail = CASE
        WHEN trigger_source = 'embed_manual' THEN 'manual'
        WHEN trigger_source = 'embed_auto' THEN 'legacy_auto'
        ELSE 'workbench'
    END
WHERE trigger_detail = '';

CREATE INDEX idx_audit_logs_schedule_pending
    ON audit_logs (tenant_id, schedule_config_id, status)
    WHERE schedule_config_id IS NOT NULL;

COMMENT ON COLUMN audit_logs.trigger_detail IS
    '嵌入审核详细触发来源：manual、save_or_submit、scheduled_scan、legacy_auto';
COMMENT ON COLUMN audit_logs.attempt_fingerprint IS
    '本次嵌入审核依赖指纹；相同指纹失败后自动来源不重复执行';
COMMENT ON COLUMN audit_logs.schedule_config_id IS
    'scheduled_scan 对应的流程配置 ID，用于关闭定时任务时取消未执行任务';
