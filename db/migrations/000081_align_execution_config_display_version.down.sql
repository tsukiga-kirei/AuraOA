-- 回滚：将展示版本号还原为执行配置快照自身的 version_no

UPDATE audit_logs al
SET config_version_no = ecv.version_no
FROM execution_config_versions ecv
WHERE al.config_version_id = ecv.id;

UPDATE archive_logs arl
SET config_version_no = ecv.version_no
FROM execution_config_versions ecv
WHERE arl.config_version_id = ecv.id;

UPDATE process_summary_logs psl
SET config_version_no = ecv.version_no
FROM execution_config_versions ecv
WHERE psl.config_version_id = ecv.id;
