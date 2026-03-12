-- 000008_cron_tasks.up.sql
-- 创建定时任务表、定时任务类型配置表

-- ============================================================
-- cron_tasks — 定时任务实例表
-- ============================================================
CREATE TABLE cron_tasks (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    tenant_id       UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- 所属租户ID
    task_type       VARCHAR(50)  NOT NULL,                                           -- 任务类型编码（关联cron_task_type_configs.task_type）
    task_label      VARCHAR(200) NOT NULL DEFAULT '',                                -- 任务显示名称
    cron_expression VARCHAR(100) NOT NULL,                                           -- Cron表达式（标准五段式，如"0 8 * * 1"）
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,                              -- 是否启用（禁用时调度器跳过该任务）
    is_builtin      BOOLEAN      NOT NULL DEFAULT FALSE,                             -- 是否为系统内置任务（内置任务不可删除）
    push_email      VARCHAR(255) DEFAULT '',                                         -- 任务执行结果推送邮箱（为空则不推送）
    last_run_at     TIMESTAMPTZ,                                                     -- 上次执行时间（NULL表示从未执行）
    next_run_at     TIMESTAMPTZ,                                                     -- 下次计划执行时间
    success_count   INT          NOT NULL DEFAULT 0,                                 -- 历史成功执行次数
    fail_count      INT          NOT NULL DEFAULT 0,                                 -- 历史失败执行次数
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),                             -- 创建时间
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()                              -- 最后更新时间
);

CREATE INDEX idx_ct_tenant_id ON cron_tasks(tenant_id);

-- ============================================================
-- cron_task_type_configs — 定时任务类型配置表
-- ============================================================
CREATE TABLE cron_task_type_configs (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    tenant_id        UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- 所属租户ID
    task_type        VARCHAR(50)  NOT NULL,                                           -- 任务类型编码（租户内唯一）
    label            VARCHAR(200) NOT NULL,                                           -- 任务类型显示名称
    enabled          BOOLEAN      NOT NULL DEFAULT TRUE,                              -- 是否启用该任务类型
    batch_limit      INT          DEFAULT NULL,                                       -- 单次批处理数量上限（NULL表示不限制）
    push_format      VARCHAR(20)  NOT NULL DEFAULT 'html',                            -- 推送内容格式：html/text/markdown
    content_template JSONB        NOT NULL DEFAULT '{}'::jsonb,                       -- 推送内容模板配置（含主题/正文模板等）
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),                             -- 创建时间
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),                             -- 最后更新时间
    UNIQUE(tenant_id, task_type)
);

CREATE INDEX idx_cttc_tenant_id ON cron_task_type_configs(tenant_id);
