-- 000008_cron_tasks.up.sql
-- 创建定时任务表、定时任务类型配置表

-- ============================================================
-- cron_tasks — 定时任务表
-- ============================================================
CREATE TABLE cron_tasks (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    task_type       VARCHAR(50)  NOT NULL,
    task_label      VARCHAR(200) NOT NULL DEFAULT '',
    cron_expression VARCHAR(100) NOT NULL,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    is_builtin      BOOLEAN      NOT NULL DEFAULT FALSE,
    push_email      VARCHAR(255) DEFAULT '',
    last_run_at     TIMESTAMPTZ,
    next_run_at     TIMESTAMPTZ,
    success_count   INT          NOT NULL DEFAULT 0,
    fail_count      INT          NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_ct_tenant_id ON cron_tasks(tenant_id);

-- ============================================================
-- cron_task_type_configs — 定时任务类型配置表
-- ============================================================
CREATE TABLE cron_task_type_configs (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    task_type        VARCHAR(50)  NOT NULL,
    label            VARCHAR(200) NOT NULL,
    enabled          BOOLEAN      NOT NULL DEFAULT TRUE,
    batch_limit      INT          DEFAULT NULL,
    push_format      VARCHAR(20)  NOT NULL DEFAULT 'html',
    content_template JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, task_type)
);

CREATE INDEX idx_cttc_tenant_id ON cron_task_type_configs(tenant_id);
