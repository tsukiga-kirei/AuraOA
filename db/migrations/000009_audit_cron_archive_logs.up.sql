-- 000009_audit_cron_archive_logs.up.sql
-- 创建审核日志表、定时任务日志表、归档复盘日志表

-- ============================================================
-- audit_logs — 审核日志表
-- ============================================================
CREATE TABLE audit_logs (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id        UUID         NOT NULL REFERENCES users(id),
    process_id     VARCHAR(100) NOT NULL,
    title          VARCHAR(500) NOT NULL,
    process_type   VARCHAR(200) NOT NULL,
    recommendation VARCHAR(20)  NOT NULL,
    score          INT          NOT NULL DEFAULT 0,
    audit_result   JSONB        NOT NULL DEFAULT '{}'::jsonb,
    duration_ms    INT          NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_al_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_al_user_id ON audit_logs(user_id);
CREATE INDEX idx_al_created_at ON audit_logs(tenant_id, created_at DESC);

-- ============================================================
-- cron_logs — 定时任务日志表
-- ============================================================
CREATE TABLE cron_logs (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    task_id     UUID         NOT NULL REFERENCES cron_tasks(id),
    task_type   VARCHAR(50)  NOT NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'running',
    message     TEXT         DEFAULT '',
    started_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX idx_cl_tenant_id ON cron_logs(tenant_id);
CREATE INDEX idx_cl_task_id ON cron_logs(task_id);

-- ============================================================
-- archive_logs — 归档复盘日志表
-- ============================================================
CREATE TABLE archive_logs (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id          UUID         NOT NULL REFERENCES users(id),
    process_id       VARCHAR(100) NOT NULL,
    title            VARCHAR(500) NOT NULL,
    process_type     VARCHAR(200) NOT NULL,
    compliance       VARCHAR(30)  NOT NULL,
    compliance_score INT          NOT NULL DEFAULT 0,
    archive_result   JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_arcl_tenant_id ON archive_logs(tenant_id);
CREATE INDEX idx_arcl_user_id ON archive_logs(user_id);
