-- 000057：OA 仅在 OPER_SAVECOMPLETE 后触发，统一历史详细来源名称

UPDATE audit_logs
SET trigger_detail = 'save_complete'
WHERE trigger_detail IN ('save', 'submit', 'save_or_submit');

UPDATE process_summary_logs
SET trigger_detail = 'save_complete'
WHERE trigger_detail IN ('save', 'submit', 'save_or_submit');

COMMENT ON COLUMN audit_logs.trigger_detail IS
    '嵌入审核详细触发来源：manual、visible_open、save_complete、scheduled_scan、legacy_auto';
COMMENT ON COLUMN process_summary_logs.trigger_detail IS
    '详细触发来源：manual、visible_open、save_complete、scheduled_scan、legacy';
