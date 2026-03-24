-- 000013_audit_async_status.up.sql
-- 审核日志表增加异步审核状态支持

-- 新增 status 列：pending=排队中/reasoning=推理中/extracting=提取中/completed=已完成/failed=失败
ALTER TABLE audit_logs ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'completed';

-- 新增 error_message 列：失败时存储错误详情
ALTER TABLE audit_logs ADD COLUMN error_message TEXT DEFAULT '';

-- 新增 updated_at 列：异步流程中各阶段的最后更新时间
ALTER TABLE audit_logs ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- 存量记录全部视为已完成
UPDATE audit_logs SET updated_at = created_at WHERE status = 'completed';

-- 索引：按租户+状态查询活跃任务
CREATE INDEX idx_al_tenant_status ON audit_logs(tenant_id, status);

-- 索引：按流程+状态查询（前端轮询用）
CREATE INDEX idx_al_process_status ON audit_logs(process_id, status);

-- 索引：今日已完成统计（按 updated_at 范围 + status）
CREATE INDEX idx_al_completed_today ON audit_logs(tenant_id, status, updated_at DESC);

COMMENT ON COLUMN audit_logs.status IS '审核状态：pending=排队中/reasoning=推理中/extracting=提取中/completed=已完成/failed=失败';
COMMENT ON COLUMN audit_logs.error_message IS '失败时的错误信息';
COMMENT ON COLUMN audit_logs.updated_at IS '最后更新时间（各阶段流转时更新）';
