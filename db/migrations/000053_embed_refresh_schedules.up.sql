-- 000053：OA 嵌入审核/总结的流程级持久化调度记录

CREATE TABLE IF NOT EXISTS embed_refresh_schedules (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module            VARCHAR(20) NOT NULL,
    config_id         UUID NOT NULL,
    process_type      VARCHAR(200) NOT NULL,
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    lookback_days     INTEGER NOT NULL DEFAULT 3,
    interval_minutes  INTEGER NOT NULL DEFAULT 5,
    cron_expression   VARCHAR(100) NOT NULL,
    last_run_at       TIMESTAMPTZ,
    next_run_at       TIMESTAMPTZ,
    last_status       VARCHAR(20) NOT NULL DEFAULT '',
    last_error        TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_ers_module CHECK (module IN ('audit', 'summary')),
    CONSTRAINT chk_ers_lookback_days CHECK (lookback_days BETWEEN 1 AND 30),
    CONSTRAINT chk_ers_interval_minutes CHECK (interval_minutes IN (5, 10, 15, 30, 60)),
    CONSTRAINT uq_ers_module_config UNIQUE (module, config_id)
);

CREATE INDEX IF NOT EXISTS idx_ers_active
    ON embed_refresh_schedules(is_active, next_run_at);
CREATE INDEX IF NOT EXISTS idx_ers_tenant_module
    ON embed_refresh_schedules(tenant_id, module);

COMMENT ON TABLE embed_refresh_schedules IS
    'OA 嵌入审核/总结的流程级持久化调度记录，由流程配置保存时同步';
COMMENT ON COLUMN embed_refresh_schedules.module IS
    '调度模块：audit=审核，summary=总结';
COMMENT ON COLUMN embed_refresh_schedules.config_id IS
    '对应 process_audit_configs 或 process_summary_configs 的配置 ID';

WITH audit_source AS (
    SELECT
        tenant_id,
        id AS config_id,
        process_type,
        status = 'active'
            AND embed_enabled
            AND COALESCE(embed_config ->> 'scheduled_refresh_enabled', 'false') = 'true' AS is_active,
        CASE
            WHEN COALESCE(embed_config ->> 'scheduled_refresh_lookback_days', '') ~ '^[0-9]+$'
                 AND (embed_config ->> 'scheduled_refresh_lookback_days')::INTEGER BETWEEN 1 AND 30
                THEN (embed_config ->> 'scheduled_refresh_lookback_days')::INTEGER
            ELSE 3
        END AS lookback_days,
        CASE
            WHEN COALESCE(embed_config ->> 'scheduled_refresh_interval_minutes', '') ~ '^[0-9]+$'
                 AND (embed_config ->> 'scheduled_refresh_interval_minutes')::INTEGER IN (5, 10, 15, 30, 60)
                THEN (embed_config ->> 'scheduled_refresh_interval_minutes')::INTEGER
            ELSE 5
        END AS interval_minutes
    FROM process_audit_configs
),
summary_source AS (
    SELECT
        tenant_id,
        id AS config_id,
        process_type,
        status = 'active'
            AND embed_enabled
            AND COALESCE(embed_config ->> 'scheduled_refresh_enabled', 'false') = 'true' AS is_active,
        CASE
            WHEN COALESCE(embed_config ->> 'scheduled_refresh_lookback_days', '') ~ '^[0-9]+$'
                 AND (embed_config ->> 'scheduled_refresh_lookback_days')::INTEGER BETWEEN 1 AND 30
                THEN (embed_config ->> 'scheduled_refresh_lookback_days')::INTEGER
            ELSE 3
        END AS lookback_days,
        CASE
            WHEN COALESCE(embed_config ->> 'scheduled_refresh_interval_minutes', '') ~ '^[0-9]+$'
                 AND (embed_config ->> 'scheduled_refresh_interval_minutes')::INTEGER IN (5, 10, 15, 30, 60)
                THEN (embed_config ->> 'scheduled_refresh_interval_minutes')::INTEGER
            ELSE 5
        END AS interval_minutes
    FROM process_summary_configs
),
all_source AS (
    SELECT 'audit'::VARCHAR(20) AS module, * FROM audit_source
    UNION ALL
    SELECT 'summary'::VARCHAR(20) AS module, * FROM summary_source
)
INSERT INTO embed_refresh_schedules (
    tenant_id,
    module,
    config_id,
    process_type,
    is_active,
    lookback_days,
    interval_minutes,
    cron_expression
)
SELECT
    tenant_id,
    module,
    config_id,
    process_type,
    is_active,
    lookback_days,
    interval_minutes,
    '0 */' || interval_minutes::TEXT || ' * * * *'
FROM all_source
ON CONFLICT (module, config_id) DO NOTHING;
