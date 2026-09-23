-- 000082：OA 嵌入审核与总结页支持配置是否允许以最新版本重新执行

UPDATE process_audit_configs
SET embed_config = COALESCE(embed_config, '{}'::jsonb) || jsonb_build_object(
    'allow_latest_config_rerun', false
)
WHERE NOT (COALESCE(embed_config, '{}'::jsonb) ? 'allow_latest_config_rerun');

UPDATE process_summary_configs
SET embed_config = COALESCE(embed_config, '{}'::jsonb) || jsonb_build_object(
    'allow_latest_config_rerun', false
)
WHERE NOT (COALESCE(embed_config, '{}'::jsonb) ? 'allow_latest_config_rerun');

COMMENT ON COLUMN process_audit_configs.embed_config IS
    'OA 嵌入审核策略：首次打开、业务数据变化、退回重提、普通审批流变化、是否允许最新配置重新审核';
COMMENT ON COLUMN process_summary_configs.embed_config IS
    'OA 嵌入总结策略：首次打开、业务数据变化、退回重提、普通审批流变化、是否允许最新配置重新总结';
