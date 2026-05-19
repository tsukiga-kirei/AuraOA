-- 审核快照按渠道拆分：workbench（审核工作台）与 embed（OA 嵌入）互不干扰

ALTER TABLE audit_process_snapshots
    ADD COLUMN IF NOT EXISTS channel VARCHAR(20) NOT NULL DEFAULT 'workbench';

COMMENT ON COLUMN audit_process_snapshots.channel IS '结论渠道：workbench=审核工作台 embed=OA嵌入';

ALTER TABLE audit_process_snapshots
    DROP CONSTRAINT IF EXISTS audit_process_snapshots_tenant_id_process_id_key;

-- 按 trigger_source 重建快照（有效 completed 且无 parse_error）
DELETE FROM audit_process_snapshots;

INSERT INTO audit_process_snapshots (
    tenant_id, process_id, channel, valid_log_ids, latest_valid_log_id,
    title, process_type, recommendation, score, confidence, created_at, updated_at
)
SELECT
    g.tenant_id,
    g.process_id,
    g.channel,
    COALESCE(
        (
            SELECT jsonb_agg(x.id::text ORDER BY x.created_at)
            FROM audit_logs x
            WHERE x.tenant_id = g.tenant_id
              AND x.process_id = g.process_id
              AND (
                  (g.channel = 'embed' AND x.trigger_source IN ('embed_auto', 'embed_manual'))
                  OR (g.channel = 'workbench' AND x.trigger_source NOT IN ('embed_auto', 'embed_manual'))
              )
              AND x.status = 'completed'
              AND (x.parse_error IS NULL OR x.parse_error = '')
              AND x.recommendation IN ('approve', 'return', 'review')
        ),
        '[]'::jsonb
    ),
    g.latest_id,
    g.title,
    g.process_type,
    g.recommendation,
    g.score,
    g.confidence,
    g.created_at,
    g.updated_at
FROM (
    SELECT DISTINCT ON (a.tenant_id, a.process_id, ch.channel)
        a.tenant_id,
        a.process_id,
        ch.channel,
        a.id AS latest_id,
        a.title,
        a.process_type,
        a.recommendation,
        a.score,
        a.confidence,
        a.created_at,
        a.updated_at
    FROM audit_logs a
    CROSS JOIN LATERAL (
        VALUES
            ('embed'),
            ('workbench')
    ) AS ch(channel)
    WHERE a.status = 'completed'
      AND (a.parse_error IS NULL OR a.parse_error = '')
      AND a.recommendation IN ('approve', 'return', 'review')
      AND (
          (ch.channel = 'embed' AND a.trigger_source IN ('embed_auto', 'embed_manual'))
          OR (ch.channel = 'workbench' AND a.trigger_source NOT IN ('embed_auto', 'embed_manual'))
      )
    ORDER BY a.tenant_id, a.process_id, ch.channel, a.created_at DESC
) g;

ALTER TABLE audit_process_snapshots
    ADD CONSTRAINT audit_process_snapshots_tenant_process_channel_key
    UNIQUE (tenant_id, process_id, channel);

CREATE INDEX IF NOT EXISTS idx_aps_tenant_channel_updated
    ON audit_process_snapshots (tenant_id, channel, updated_at DESC);
