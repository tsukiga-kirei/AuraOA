-- 000009_audit_cron_archive_logs.up.sql
-- 创建审核日志表、定时任务日志表、归档复盘日志表

-- ============================================================
-- audit_logs — AI审核执行日志表
-- ============================================================
CREATE TABLE audit_logs (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    tenant_id      UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- 所属租户ID
    user_id        UUID         NOT NULL REFERENCES users(id),                     -- 发起审核的用户ID
    process_id     VARCHAR(100) NOT NULL,                                           -- OA流程单号/请求ID
    title          VARCHAR(500) NOT NULL,                                           -- 流程标题（冗余存储便于列表展示）
    process_type   VARCHAR(200) NOT NULL,                                           -- 流程类型（冗余存储便于筛选统计）
    recommendation VARCHAR(20)  NOT NULL,                                           -- AI审核建议：approve=通过/return=退回/review=人工复核
    score          INT          NOT NULL DEFAULT 0,                                 -- AI综合评分（0-100）
    audit_result   JSONB        NOT NULL DEFAULT '{}'::jsonb,                       -- 完整审核结果（含rule_results/risk_points/suggestions/confidence）
    duration_ms    INT          NOT NULL DEFAULT 0,                                 -- 审核耗时（毫秒）
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()                              -- 审核完成时间
);

CREATE INDEX idx_al_tenant_id  ON audit_logs(tenant_id);
CREATE INDEX idx_al_user_id    ON audit_logs(user_id);
CREATE INDEX idx_al_created_at ON audit_logs(tenant_id, created_at DESC);

-- ============================================================
-- cron_logs — 定时任务执行日志表
-- ============================================================
CREATE TABLE cron_logs (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    tenant_id   UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- 所属租户ID
    task_id     UUID         NOT NULL REFERENCES cron_tasks(id),                -- 关联的定时任务ID
    task_type   VARCHAR(50)  NOT NULL,                                           -- 任务类型（冗余存储）
    status      VARCHAR(20)  NOT NULL DEFAULT 'running',                        -- 执行状态：running=执行中/success=成功/failed=失败
    message     TEXT         DEFAULT '',                                         -- 执行结果消息或错误详情
    started_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),                             -- 任务开始执行时间
    finished_at TIMESTAMPTZ                                                      -- 任务完成时间（NULL表示仍在执行中）
);

CREATE INDEX idx_cl_tenant_id ON cron_logs(tenant_id);
CREATE INDEX idx_cl_task_id   ON cron_logs(task_id);

-- ============================================================
-- archive_logs — 归档复盘日志表
-- ============================================================
CREATE TABLE archive_logs (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    tenant_id        UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- 所属租户ID
    user_id          UUID         NOT NULL REFERENCES users(id),                     -- 发起归档复盘的用户ID
    process_id       VARCHAR(100) NOT NULL,                                           -- OA流程单号
    title            VARCHAR(500) NOT NULL,                                           -- 流程标题
    process_type     VARCHAR(200) NOT NULL,                                           -- 流程类型
    compliance       VARCHAR(30)  NOT NULL,                                           -- 合规结论：compliant=合规/non_compliant=不合规/partial=部分合规
    compliance_score INT          NOT NULL DEFAULT 0,                                 -- 合规评分（0-100）
    archive_result   JSONB        NOT NULL DEFAULT '{}'::jsonb,                       -- 完整归档复盘结果
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now()                              -- 归档时间
);

CREATE INDEX idx_arcl_tenant_id ON archive_logs(tenant_id);
CREATE INDEX idx_arcl_user_id   ON archive_logs(user_id);

-- ============================================================
-- 数据库注释（中文）
-- ============================================================
COMMENT ON TABLE audit_logs IS 'AI审核执行日志表';
COMMENT ON COLUMN audit_logs.id IS '主键UUID';
COMMENT ON COLUMN audit_logs.tenant_id IS '所属租户ID';
COMMENT ON COLUMN audit_logs.user_id IS '发起审核的用户ID';
COMMENT ON COLUMN audit_logs.process_id IS 'OA流程单号/请求ID';
COMMENT ON COLUMN audit_logs.title IS '流程标题（冗余存储便于列表展示）';
COMMENT ON COLUMN audit_logs.process_type IS '流程类型（冗余存储便于筛选统计）';
COMMENT ON COLUMN audit_logs.recommendation IS 'AI审核建议：approve=通过/return=退回/review=人工复核';
COMMENT ON COLUMN audit_logs.score IS 'AI综合评分（0-100）';
COMMENT ON COLUMN audit_logs.audit_result IS '完整审核结果（含rule_results/risk_points/suggestions/confidence）';
COMMENT ON COLUMN audit_logs.duration_ms IS '审核耗时（毫秒）';
COMMENT ON COLUMN audit_logs.created_at IS '审核完成时间';

COMMENT ON TABLE cron_logs IS '定时任务执行日志表';
COMMENT ON COLUMN cron_logs.id IS '主键UUID';
COMMENT ON COLUMN cron_logs.tenant_id IS '所属租户ID';
COMMENT ON COLUMN cron_logs.task_id IS '关联的定时任务ID';
COMMENT ON COLUMN cron_logs.task_type IS '任务类型（冗余存储）';
COMMENT ON COLUMN cron_logs.status IS '执行状态：running=执行中/success=成功/failed=失败';
COMMENT ON COLUMN cron_logs.message IS '执行结果消息或错误详情';
COMMENT ON COLUMN cron_logs.started_at IS '任务开始执行时间';
COMMENT ON COLUMN cron_logs.finished_at IS '任务完成时间（NULL表示仍在执行中）';

COMMENT ON TABLE archive_logs IS '归档复盘日志表';
COMMENT ON COLUMN archive_logs.id IS '主键UUID';
COMMENT ON COLUMN archive_logs.tenant_id IS '所属租户ID';
COMMENT ON COLUMN archive_logs.user_id IS '发起归档复盘的用户ID';
COMMENT ON COLUMN archive_logs.process_id IS 'OA流程单号';
COMMENT ON COLUMN archive_logs.title IS '流程标题';
COMMENT ON COLUMN archive_logs.process_type IS '流程类型';
COMMENT ON COLUMN archive_logs.compliance IS '合规结论：compliant=合规/non_compliant=不合规/partial=部分合规';
COMMENT ON COLUMN archive_logs.compliance_score IS '合规评分（0-100）';
COMMENT ON COLUMN archive_logs.archive_result IS '完整归档复盘结果';
COMMENT ON COLUMN archive_logs.created_at IS '归档时间';
