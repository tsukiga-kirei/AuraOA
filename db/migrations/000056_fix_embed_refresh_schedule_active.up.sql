-- 000056：修复关闭的流程级定时检查被 GORM 默认值误写为启用

ALTER TABLE embed_refresh_schedules
    ALTER COLUMN is_active SET DEFAULT FALSE;

WITH source_configs AS (
    SELECT
        'audit'::VARCHAR(20) AS module,
        id AS config_id,
        status = 'active'
            AND embed_enabled
            AND COALESCE(embed_config ->> 'scheduled_refresh_enabled', 'false') = 'true' AS is_active
    FROM process_audit_configs

    UNION ALL

    SELECT
        'summary'::VARCHAR(20) AS module,
        id AS config_id,
        status = 'active'
            AND embed_enabled
            AND COALESCE(embed_config ->> 'scheduled_refresh_enabled', 'false') = 'true' AS is_active
    FROM process_summary_configs
)
UPDATE embed_refresh_schedules AS schedule
SET
    is_active = source.is_active,
    next_run_at = CASE WHEN source.is_active THEN schedule.next_run_at ELSE NULL END,
    updated_at = NOW()
FROM source_configs AS source
WHERE schedule.module = source.module
  AND schedule.config_id = source.config_id;

-- 源配置已经删除的孤立调度先停用，服务启动后会继续清理记录。
UPDATE embed_refresh_schedules AS schedule
SET
    is_active = FALSE,
    next_run_at = NULL,
    updated_at = NOW()
WHERE schedule.module = 'audit'
  AND NOT EXISTS (
    SELECT 1
    FROM process_audit_configs AS config
    WHERE config.id = schedule.config_id
);

UPDATE embed_refresh_schedules AS schedule
SET
    is_active = FALSE,
    next_run_at = NULL,
    updated_at = NOW()
WHERE schedule.module = 'summary'
  AND NOT EXISTS (
    SELECT 1
    FROM process_summary_configs AS config
    WHERE config.id = schedule.config_id
);
