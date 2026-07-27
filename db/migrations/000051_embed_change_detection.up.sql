-- 000051：细分 OA 嵌入页自动刷新策略

UPDATE process_audit_configs
SET embed_config = COALESCE(embed_config, '{}'::jsonb) || jsonb_build_object(
    'auto_audit_on_data_change', true,
    'auto_audit_on_return_resubmit', true,
    'auto_audit_on_flow_change', false
);

UPDATE process_summary_configs
SET embed_config = COALESCE(embed_config, '{}'::jsonb) || jsonb_build_object(
    'auto_summary_on_data_change', true,
    'auto_summary_on_return_resubmit', true,
    'auto_summary_on_flow_change', false
);

COMMENT ON COLUMN process_audit_configs.embed_config IS
    'OA 嵌入审核策略：首次打开、业务数据变化、退回重提、普通审批流变化';
COMMENT ON COLUMN process_summary_configs.embed_config IS
    'OA 嵌入总结策略：首次打开、业务数据变化、退回重提、普通审批流变化';
