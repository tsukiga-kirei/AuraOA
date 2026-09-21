-- 000081：将审核、归档复盘与流程总结日志的展示版本号与租户基础配置版本号对齐。
-- 解决后台已发布基础版本（如 v10）与底层执行快照自增序号（如 v8）在前端展示不一致的问题。

UPDATE audit_logs al
SET config_version_no = tcv.version_no
FROM execution_config_versions ecv
JOIN tenant_config_versions tcv ON ecv.base_config_version_id = tcv.id
WHERE al.config_version_id = ecv.id
  AND tcv.version_no IS NOT NULL;

UPDATE archive_logs arl
SET config_version_no = tcv.version_no
FROM execution_config_versions ecv
JOIN tenant_config_versions tcv ON ecv.base_config_version_id = tcv.id
WHERE arl.config_version_id = ecv.id
  AND tcv.version_no IS NOT NULL;

UPDATE process_summary_logs psl
SET config_version_no = tcv.version_no
FROM execution_config_versions ecv
JOIN tenant_config_versions tcv ON ecv.base_config_version_id = tcv.id
WHERE psl.config_version_id = ecv.id
  AND tcv.version_no IS NOT NULL;
