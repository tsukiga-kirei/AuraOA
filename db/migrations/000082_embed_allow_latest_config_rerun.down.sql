-- 000082 回滚：移除 allow_latest_config_rerun

UPDATE process_audit_configs
SET embed_config = embed_config - 'allow_latest_config_rerun'
WHERE embed_config ? 'allow_latest_config_rerun';

UPDATE process_summary_configs
SET embed_config = embed_config - 'allow_latest_config_rerun'
WHERE embed_config ? 'allow_latest_config_rerun';

COMMENT ON COLUMN process_audit_configs.embed_config IS
    'OA 嵌入审核策略：首次打开、业务数据变化、退回重提、普通审批流变化';
COMMENT ON COLUMN process_summary_configs.embed_config IS
    'OA 嵌入总结策略：首次打开、业务数据变化、退回重提、普通审批流变化';
