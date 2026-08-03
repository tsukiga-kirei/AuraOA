-- 回滚 000058：恢复保存完成事件名称并删除事件解析记录

UPDATE audit_logs
SET trigger_detail = 'save_complete'
WHERE trigger_detail IN ('save_requested', 'submit_requested', 'legacy_operation');

UPDATE process_summary_logs
SET trigger_detail = 'save_complete'
WHERE trigger_detail IN ('save_requested', 'submit_requested', 'legacy_operation');

COMMENT ON COLUMN audit_logs.trigger_detail IS
    '嵌入审核详细触发来源：manual、visible_open、save_complete、scheduled_scan、legacy_auto';
COMMENT ON COLUMN process_summary_logs.trigger_detail IS
    '详细触发来源：manual、visible_open、save_complete、scheduled_scan、legacy';

DROP TABLE IF EXISTS embed_refresh_events;
