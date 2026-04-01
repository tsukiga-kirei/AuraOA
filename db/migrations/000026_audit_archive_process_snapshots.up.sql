-- 流程级「有效审核/归档复盘」快照：仅含解析成功、语义有效的结论；供工作台/归档列表与审核链查询，避免依赖全量日志表扫描。

CREATE TABLE audit_process_snapshots (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    process_id           VARCHAR(100) NOT NULL,
    valid_log_ids        JSONB        NOT NULL DEFAULT '[]'::jsonb,
    latest_valid_log_id  UUID         NOT NULL REFERENCES audit_logs(id) ON DELETE CASCADE,
    title                VARCHAR(500) NOT NULL DEFAULT '',
    process_type         VARCHAR(200) NOT NULL DEFAULT '',
    recommendation       VARCHAR(20)  NOT NULL,
    score                INT          NOT NULL DEFAULT 0,
    confidence           INT          NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, process_id)
);

CREATE INDEX idx_aps_tenant_updated ON audit_process_snapshots (tenant_id, updated_at DESC);
CREATE INDEX idx_aps_tenant_process ON audit_process_snapshots (tenant_id, process_id);

COMMENT ON TABLE audit_process_snapshots IS '审核有效结论快照（仅成功解析的日志 id 链）';
COMMENT ON COLUMN audit_process_snapshots.valid_log_ids IS '有效 audit_logs.id 的 JSON 数组（字符串 UUID，按时间顺序追加）';
COMMENT ON COLUMN audit_process_snapshots.latest_valid_log_id IS '当前有效结论对应的最新一条成功审核日志';

CREATE TABLE archive_process_snapshots (
    id                        UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    process_id                VARCHAR(100) NOT NULL,
    valid_archive_log_ids     JSONB        NOT NULL DEFAULT '[]'::jsonb,
    latest_valid_archive_log_id UUID       NOT NULL REFERENCES archive_logs(id) ON DELETE CASCADE,
    title                     VARCHAR(500) NOT NULL DEFAULT '',
    process_type              VARCHAR(200) NOT NULL DEFAULT '',
    compliance                VARCHAR(30)  NOT NULL,
    compliance_score        INT          NOT NULL DEFAULT 0,
    confidence                INT          NOT NULL DEFAULT 0,
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, process_id)
);

CREATE INDEX idx_arps_tenant_updated ON archive_process_snapshots (tenant_id, updated_at DESC);
CREATE INDEX idx_arps_tenant_process ON archive_process_snapshots (tenant_id, process_id);

COMMENT ON TABLE archive_process_snapshots IS '归档复盘有效结论快照（仅成功解析的 archive_logs id 链）';

-- 从历史数据回填（仅含「已完成且无解析错误」的有效记录）
INSERT INTO audit_process_snapshots (
    tenant_id, process_id, valid_log_ids, latest_valid_log_id,
    title, process_type, recommendation, score, confidence, created_at, updated_at
)
SELECT
    a.tenant_id,
    a.process_id,
    COALESCE(
        (
            SELECT jsonb_agg(x.id::text ORDER BY x.created_at)
            FROM audit_logs x
            WHERE x.tenant_id = a.tenant_id
              AND x.process_id = a.process_id
              AND x.status = 'completed'
              AND (x.parse_error IS NULL OR x.parse_error = '')
              AND x.recommendation IN ('approve', 'return', 'review')
        ),
        '[]'::jsonb
    ),
    a.id,
    a.title,
    a.process_type,
    a.recommendation,
    a.score,
    a.confidence,
    a.created_at,
    a.updated_at
FROM audit_logs a
INNER JOIN (
    SELECT tenant_id, process_id, MAX(created_at) AS max_created
    FROM audit_logs
    WHERE status = 'completed'
      AND (parse_error IS NULL OR parse_error = '')
      AND recommendation IN ('approve', 'return', 'review')
    GROUP BY tenant_id, process_id
) m ON a.tenant_id = m.tenant_id AND a.process_id = m.process_id AND a.created_at = m.max_created
WHERE a.status = 'completed'
  AND (a.parse_error IS NULL OR a.parse_error = '')
  AND a.recommendation IN ('approve', 'return', 'review');

INSERT INTO archive_process_snapshots (
    tenant_id, process_id, valid_archive_log_ids, latest_valid_archive_log_id,
    title, process_type, compliance, compliance_score, confidence, created_at, updated_at
)
SELECT
    a.tenant_id,
    a.process_id,
    COALESCE(
        (
            SELECT jsonb_agg(x.id::text ORDER BY x.created_at)
            FROM archive_logs x
            WHERE x.tenant_id = a.tenant_id
              AND x.process_id = a.process_id
              AND x.status = 'completed'
              AND (x.parse_error IS NULL OR x.parse_error = '')
              AND x.compliance IN ('compliant', 'partially_compliant', 'non_compliant')
        ),
        '[]'::jsonb
    ),
    a.id,
    a.title,
    a.process_type,
    a.compliance,
    a.compliance_score,
    a.confidence,
    a.created_at,
    a.updated_at
FROM archive_logs a
INNER JOIN (
    SELECT tenant_id, process_id, MAX(created_at) AS max_created
    FROM archive_logs
    WHERE status = 'completed'
      AND (parse_error IS NULL OR parse_error = '')
      AND compliance IN ('compliant', 'partially_compliant', 'non_compliant')
    GROUP BY tenant_id, process_id
) m ON a.tenant_id = m.tenant_id AND a.process_id = m.process_id AND a.created_at = m.max_created
WHERE a.status = 'completed'
  AND (a.parse_error IS NULL OR a.parse_error = '')
  AND a.compliance IN ('compliant', 'partially_compliant', 'non_compliant');
