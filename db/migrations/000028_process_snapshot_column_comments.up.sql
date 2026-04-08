-- 000028：补全 audit_process_snapshots / archive_process_snapshots 遗漏的列 COMMENT

COMMENT ON COLUMN audit_process_snapshots.id IS '主键UUID';
COMMENT ON COLUMN audit_process_snapshots.tenant_id IS '所属租户ID';
COMMENT ON COLUMN audit_process_snapshots.process_id IS 'OA流程单号';
COMMENT ON COLUMN audit_process_snapshots.title IS '流程标题（快照展示用）';
COMMENT ON COLUMN audit_process_snapshots.process_type IS '流程类型';
COMMENT ON COLUMN audit_process_snapshots.recommendation IS '当前有效结论：approve=通过/return=退回/review=人工复核';
COMMENT ON COLUMN audit_process_snapshots.score IS 'AI综合评分（0-100）';
COMMENT ON COLUMN audit_process_snapshots.confidence IS '结论置信度（0-100）';
COMMENT ON COLUMN audit_process_snapshots.created_at IS '快照首次创建时间';
COMMENT ON COLUMN audit_process_snapshots.updated_at IS '快照最后更新时间（随有效结论链追加而更新）';

COMMENT ON COLUMN archive_process_snapshots.id IS '主键UUID';
COMMENT ON COLUMN archive_process_snapshots.tenant_id IS '所属租户ID';
COMMENT ON COLUMN archive_process_snapshots.process_id IS 'OA流程单号';
COMMENT ON COLUMN archive_process_snapshots.valid_archive_log_ids IS '有效 archive_logs.id 的 JSON 数组（字符串 UUID，按时间顺序追加）';
COMMENT ON COLUMN archive_process_snapshots.latest_valid_archive_log_id IS '当前有效结论对应的最新一条成功归档复盘日志';
COMMENT ON COLUMN archive_process_snapshots.title IS '流程标题（快照展示用）';
COMMENT ON COLUMN archive_process_snapshots.process_type IS '流程类型';
COMMENT ON COLUMN archive_process_snapshots.compliance IS '当前有效合规结论：compliant=合规/non_compliant=不合规/partial=部分合规';
COMMENT ON COLUMN archive_process_snapshots.compliance_score IS '合规评分（0-100）';
COMMENT ON COLUMN archive_process_snapshots.confidence IS '结论置信度（0-100）';
COMMENT ON COLUMN archive_process_snapshots.created_at IS '快照首次创建时间';
COMMENT ON COLUMN archive_process_snapshots.updated_at IS '快照最后更新时间（随有效结论链追加而更新）';
