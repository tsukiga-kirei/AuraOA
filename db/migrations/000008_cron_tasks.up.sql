-- 000008_cron_tasks.up.sql
-- 创建定时任务实例表、任务类型系统预设表、任务类型租户覆盖配置表
-- 设计：双层结构
--   cron_task_type_presets  — 系统内置 6 条预设（全局，不绑定租户）
--   cron_task_type_configs  — 租户启用后的自定义覆盖配置（有记录=已启用）
--   cron_tasks              — 实际执行的任务实例

-- ============================================================
-- cron_tasks — 定时任务实例表
-- ============================================================
CREATE TABLE cron_tasks (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    tenant_id       UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- 所属租户ID
    task_type       VARCHAR(50)  NOT NULL,                                           -- 任务类型编码（关联 cron_task_type_presets.task_type）
    task_label      VARCHAR(200) NOT NULL DEFAULT '',                                -- 任务显示名称
    cron_expression VARCHAR(100) NOT NULL,                                           -- Cron 表达式（标准五段式，如 "0 8 * * 1"）
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,                              -- 是否启用（禁用时调度器跳过该任务）
    is_builtin      BOOLEAN      NOT NULL DEFAULT FALSE,                             -- 是否为系统内置任务（内置任务不可删除）
    push_email      VARCHAR(255) DEFAULT '',                                         -- 执行结果推送邮箱（为空则不推送）
    last_run_at     TIMESTAMPTZ,                                                     -- 上次执行时间（NULL 表示从未执行）
    next_run_at     TIMESTAMPTZ,                                                     -- 下次计划执行时间
    success_count   INT          NOT NULL DEFAULT 0,                                 -- 历史成功执行次数
    fail_count      INT          NOT NULL DEFAULT 0,                                 -- 历史失败执行次数
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),                             -- 创建时间
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()                              -- 最后更新时间
);

CREATE INDEX idx_ct_tenant_id ON cron_tasks(tenant_id);

-- ============================================================
-- cron_task_type_presets — 系统内置任务类型预设表（全局，不绑定租户）
-- 共 6 条：审核工作台×3（批量/日报/周报）+ 归档复盘×3（批量/日报/周报）
-- 这些预设是"恢复默认"功能的数据源，租户可覆盖但不可删除
-- ============================================================
CREATE TABLE cron_task_type_presets (
    task_type        VARCHAR(50)  PRIMARY KEY,                                -- 任务类型唯一键（如 audit_batch / archive_daily）
    module           VARCHAR(20)  NOT NULL DEFAULT 'audit',                   -- 所属模块：audit=审核工作台，archive=归档复盘
    label_zh         VARCHAR(200) NOT NULL,                                   -- 中文显示名称
    label_en         VARCHAR(200) NOT NULL DEFAULT '',                        -- 英文显示名称
    description_zh   VARCHAR(500) NOT NULL DEFAULT '',                        -- 中文描述
    description_en   VARCHAR(500) NOT NULL DEFAULT '',                        -- 英文描述
    default_cron     VARCHAR(100) NOT NULL DEFAULT '',                        -- 建议的默认 Cron 表达式（仅供参考）
    push_format      VARCHAR(20)  NOT NULL DEFAULT 'html',                    -- 默认推送格式：html/markdown/plain
    content_template JSONB        NOT NULL DEFAULT '{}'::jsonb,               -- 默认内容模板（subject/header/body_template/footer）
    sort_order       INT          NOT NULL DEFAULT 0,                         -- 页面展示排序序号
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),                     -- 创建时间
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()                      -- 最后更新时间
);

-- ============================================================
-- 初始化 6 条系统内置任务类型预设数据
-- ============================================================
INSERT INTO cron_task_type_presets
    (task_type, module, label_zh, label_en, description_zh, description_en,
     default_cron, push_format, content_template, sort_order)
VALUES

-- 审核工作台 - 批量处理
('audit_batch', 'audit',
 '审核-批量处理', 'Audit - Batch Processing',
 '按计划批量对待审核流程执行 AI 智能审核', 'Batch AI audit for pending workflows on schedule',
 '0 9 * * 1-5', 'html',
 '{"subject":"","header":"","body_template":"","footer":"","batch_limit":50}'::jsonb,
 1),

-- 审核工作台 - 日报推送
('audit_daily', 'audit',
 '审核-日报推送', 'Audit - Daily Report',
 '每日推送审核工作台的审核统计日报', 'Daily audit statistics report for the audit workbench',
 '0 18 * * 1-5', 'html',
 '{"subject":"【OA智审】审核日报 - {{date}}","header":"今日审核工作概览：","body_template":"今日共处理 {{total}} 条审核，通过 {{approved}} 条，退回 {{rejected}} 条，改签 {{revised}} 条。\n通过率 {{pass_rate}}%。\n\n{{detail_list}}\n\n统计数据：\n{{statistics}}\n\n以上数据截至 {{time}}。","footer":"详情请登录系统查看。此邮件由系统自动发送，请勿直接回复。"}'::jsonb,
 2),

-- 审核工作台 - 周报推送
('audit_weekly', 'audit',
 '审核-周报推送', 'Audit - Weekly Report',
 '每周推送审核工作台的审核汇总周报', 'Weekly audit summary report for the audit workbench',
 '0 10 * * 1', 'markdown',
 '{"subject":"【OA智审】审核周报 - 第{{week}}周（{{date_range}}）","header":"本周审核工作总结：","body_template":"本周共处理 {{total}} 条审核，较上周{{trend}}。\n合规率 {{compliance_rate}}%，环比{{compliance_trend}}。\n\n{{statistics}}\n\n{{detail_list}}","footer":"报告生成时间：{{time}}。如需详细数据请导出归档记录。"}'::jsonb,
 3),

-- 归档复盘 - 批量处理
('archive_batch', 'archive',
 '归档复盘-批量处理', 'Archive - Batch Processing',
 '按计划批量对已归档流程执行 AI 合规复盘', 'Batch AI compliance review for archived workflows on schedule',
 '0 2 * * *', 'html',
 '{"subject":"","header":"","body_template":"","footer":"","batch_limit":50}'::jsonb,
 4),

-- 归档复盘 - 日报推送
('archive_daily', 'archive',
 '归档复盘-日报推送', 'Archive - Daily Report',
 '每日推送归档复盘的合规分析日报', 'Daily compliance analysis report for archive review',
 '0 19 * * 1-5', 'html',
 '{"subject":"【OA智审】归档复盘日报 - {{date}}","header":"今日归档复盘概览：","body_template":"今日共复盘 {{total}} 条归档流程，合规 {{approved}} 条，不合规 {{rejected}} 条，部分合规 {{revised}} 条。\n合规率 {{pass_rate}}%。\n\n{{detail_list}}\n\n统计数据：\n{{statistics}}\n\n以上数据截至 {{time}}。","footer":"详情请登录系统查看。此邮件由系统自动发送，请勿直接回复。"}'::jsonb,
 5),

-- 归档复盘 - 周报推送
('archive_weekly', 'archive',
 '归档复盘-周报推送', 'Archive - Weekly Report',
 '每周推送归档复盘的合规汇总周报', 'Weekly compliance summary report for archive review',
 '0 9 * * 1', 'markdown',
 '{"subject":"【OA智审】归档复盘周报 - 第{{week}}周（{{date_range}}）","header":"本周归档复盘总结：","body_template":"本周共复盘 {{total}} 条归档流程，较上周{{trend}}。\n整体合规率 {{compliance_rate}}%，环比{{compliance_trend}}。\n\n{{statistics}}\n\n{{detail_list}}","footer":"报告生成时间：{{time}}。如需详细数据请导出归档记录。"}'::jsonb,
 6);

-- ============================================================
-- cron_task_type_configs — 定时任务类型配置表（租户覆盖层）
-- 租户启用某任务类型后，在此表创建一条记录保存自定义配置
-- 若租户未启用，则不存在对应记录（表示未启用）
-- 删除记录即表示关闭该任务类型（如需"恢复默认"则同时清空自定义内容）
-- ============================================================
CREATE TABLE cron_task_type_configs (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),                   -- 主键UUID
    tenant_id        UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,      -- 所属租户ID
    task_type        VARCHAR(50)  NOT NULL REFERENCES cron_task_type_presets(task_type),  -- 关联预设类型编码
    batch_limit      INT          DEFAULT NULL,                                             -- 单次批处理数量上限（NULL 表示使用预设默认值）
    push_format      VARCHAR(20)  NOT NULL DEFAULT 'html',                                 -- 租户自定义推送格式：html/markdown/plain
    content_template JSONB        NOT NULL DEFAULT '{}'::jsonb,                            -- 租户自定义内容模板（subject/header/body_template/footer）
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),                                  -- 创建时间
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),                                  -- 最后更新时间
    UNIQUE(tenant_id, task_type)
);

CREATE INDEX idx_cttc_tenant_id ON cron_task_type_configs(tenant_id);
