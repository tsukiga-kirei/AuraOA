-- 回滚触发来源名称；无法还原历史 save 与 submit 的原始细分

UPDATE audit_logs
SET trigger_detail = 'save_or_submit'
WHERE trigger_detail = 'save_complete';

UPDATE process_summary_logs
SET trigger_detail = 'save_or_submit'
WHERE trigger_detail = 'save_complete';

COMMENT ON COLUMN audit_logs.trigger_detail IS
    '嵌入审核详细触发来源：manual、save_or_submit、scheduled_scan、legacy_auto';
COMMENT ON COLUMN process_summary_logs.trigger_detail IS
    '详细触发来源：manual、visible_open、save_or_submit、scheduled_scan、workbench';
