UPDATE process_audit_configs
SET embed_config = embed_config
    - 'auto_audit_on_data_change'
    - 'auto_audit_on_return_resubmit'
    - 'auto_audit_on_flow_change';

UPDATE process_summary_configs
SET embed_config = embed_config
    - 'auto_summary_on_data_change'
    - 'auto_summary_on_return_resubmit'
    - 'auto_summary_on_flow_change';

COMMENT ON COLUMN process_audit_configs.embed_config IS
    '嵌入页行为配置（auto_audit_on_open/auto_audit_on_stale 等）';
COMMENT ON COLUMN process_summary_configs.embed_config IS
    'OA 嵌入总结行为配置，如自动总结、过期自动刷新';
