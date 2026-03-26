-- 000015_archive_review_runtime.up.sql
-- 为 archive_logs 增加异步归档复盘运行时字段

ALTER TABLE archive_logs ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'completed';
ALTER TABLE archive_logs ADD COLUMN process_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE archive_logs ADD COLUMN duration_ms INT NOT NULL DEFAULT 0;
ALTER TABLE archive_logs ADD COLUMN ai_reasoning TEXT DEFAULT '';
ALTER TABLE archive_logs ADD COLUMN confidence INT NOT NULL DEFAULT 0;
ALTER TABLE archive_logs ADD COLUMN raw_content TEXT DEFAULT '';
ALTER TABLE archive_logs ADD COLUMN parse_error TEXT DEFAULT '';
ALTER TABLE archive_logs ADD COLUMN error_message TEXT DEFAULT '';
ALTER TABLE archive_logs ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE archive_logs SET updated_at = created_at WHERE status = 'completed';

CREATE INDEX idx_arcl_tenant_status ON archive_logs(tenant_id, status);
CREATE INDEX idx_arcl_process_status ON archive_logs(process_id, status);
CREATE INDEX idx_arcl_process_created_at ON archive_logs(tenant_id, process_id, created_at DESC);

COMMENT ON COLUMN archive_logs.status IS '归档复盘状态：pending=排队中/assembling=组装提示词/reasoning=推理中/extracting=提取中/completed=已完成/failed=失败';
COMMENT ON COLUMN archive_logs.process_snapshot IS '归档复盘时使用的流程快照（申请人/部门/时间/审批流等）';
COMMENT ON COLUMN archive_logs.duration_ms IS '归档复盘耗时（毫秒）';
COMMENT ON COLUMN archive_logs.ai_reasoning IS 'AI 推理阶段输出';
COMMENT ON COLUMN archive_logs.confidence IS 'AI 结果置信度（0-100）';
COMMENT ON COLUMN archive_logs.raw_content IS '提取阶段模型原始输出';
COMMENT ON COLUMN archive_logs.parse_error IS '结构化提取解析错误';
COMMENT ON COLUMN archive_logs.error_message IS '任务失败时的错误信息';
COMMENT ON COLUMN archive_logs.updated_at IS '最后更新时间（各阶段流转时更新）';
