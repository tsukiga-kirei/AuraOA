-- 011_user_personal_configs.sql
-- Seed data: 用户个人配置 + 仪表板偏好
-- Run after 010_process_audit_configs.sql (audit_details 引用 process_type 和 rule_id)
-- Run after 002_users.sql, 003_tenants.sql
--
-- 外键依赖：
--   user_personal_configs.tenant_id → tenants(id)
--   user_personal_configs.user_id → users(id)
--   user_dashboard_prefs.tenant_id → tenants(id)
--   user_dashboard_prefs.user_id → users(id)
--   user_dashboard_prefs.pref_scope：business | tenant_admin | platform（需在迁移 000021 之后执行本 seed）
--
-- UUID 约定：
--   user_personal_configs: d4000000-0000-0000-0000-00000000000x
--   user_dashboard_prefs:  d5000000-0000-0000-0000-00000000000x

-- ============================================================
-- DEMO_HQ 用户个人配置
-- ============================================================

-- reviewer01 (张三) 在 DEMO_HQ 的个人配置
INSERT INTO user_personal_configs (id, tenant_id, user_id, audit_details, cron_details, archive_details)
VALUES (
    'd4000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000003',
    '[{
        "process_type": "采购审批",
        "custom_rules": [
            {"id": "cr-001", "content": "采购金额超过 30,000 元需附三家比价单", "enabled": true},
            {"id": "cr-002", "content": "IT 设备采购须经信息部确认", "enabled": true}
        ],
        "field_overrides": ["main:cgje", "main:gys", "main:htbh"],
        "field_mode": "selected",
        "strictness_override": "strict",
        "rule_toggle_overrides": [
            {"rule_id": "d2000000-0000-0000-0000-000000000005", "enabled": true}
        ]
    }]'::jsonb,
    '[]'::jsonb,
    '[{
        "process_type": "采购审批",
        "custom_rules": [
            {"id": "arc-001", "content": "归档文件必须包含供应商合规承诺书", "enabled": true}
        ],
        "field_overrides": [],
        "field_mode": "all",
        "strictness_override": "standard",
        "rule_toggle_overrides": []
    }]'::jsonb
);

-- reviewer02 (李四) 在 DEMO_HQ 的个人配置
INSERT INTO user_personal_configs (id, tenant_id, user_id, audit_details, cron_details, archive_details)
VALUES (
    'd4000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000004',
    '[{
        "process_type": "费用报销",
        "custom_rules": [
            {"id": "cr-003", "content": "出租车费单次超过 200 元需说明原因", "enabled": true}
        ],
        "field_overrides": [],
        "field_mode": "all",
        "strictness_override": "",
        "rule_toggle_overrides": [
            {"rule_id": "d2000000-0000-0000-0000-000000000012", "enabled": false}
        ]
    }]'::jsonb,
    '[]'::jsonb,
    '[]'::jsonb
);

-- supervisor01 (王五) 在 DEMO_HQ 的个人配置（多流程）
INSERT INTO user_personal_configs (id, tenant_id, user_id, audit_details, cron_details, archive_details)
VALUES (
    'd4000000-0000-0000-0000-000000000003',
    'a0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000005',
    '[
        {
            "process_type": "采购审批",
            "custom_rules": [],
            "field_overrides": [],
            "field_mode": "all",
            "strictness_override": "strict",
            "rule_toggle_overrides": []
        },
        {
            "process_type": "合同审批",
            "custom_rules": [
                {"id": "cr-004", "content": "合同期限超过 3 年须总经理审批", "enabled": true}
            ],
            "field_overrides": ["main:htje", "main:qsrq", "main:dfjg"],
            "field_mode": "selected",
            "strictness_override": "",
            "rule_toggle_overrides": []
        }
    ]'::jsonb,
    '[]'::jsonb,
    '[
        {
            "process_type": "合同审批",
            "custom_rules": [
                {"id": "arc-002", "content": "归档文件须检查印章清晰度", "enabled": true}
            ],
            "field_overrides": [],
            "field_mode": "all",
            "strictness_override": "strict",
            "rule_toggle_overrides": []
        }
    ]'::jsonb
);

-- ============================================================
-- DEMO_BR1 用户个人配置
-- ============================================================

-- reviewer01 (张三) 在 DEMO_BR1 的个人配置（跨租户，独立配置）
INSERT INTO user_personal_configs (id, tenant_id, user_id, audit_details, cron_details, archive_details)
VALUES (
    'd4000000-0000-0000-0000-000000000004',
    'a0000000-0000-0000-0000-000000000002',
    'b0000000-0000-0000-0000-000000000003',
    '[{
        "process_type": "采购审批",
        "custom_rules": [
            {"id": "cr-005", "content": "分公司采购须注明项目编号", "enabled": true}
        ],
        "field_overrides": [],
        "field_mode": "all",
        "strictness_override": "",
        "rule_toggle_overrides": []
    }]'::jsonb,
    '[]'::jsonb,
    '[]'::jsonb
);

-- br1_reviewer (赵六) 在 DEMO_BR1 的个人配置
INSERT INTO user_personal_configs (id, tenant_id, user_id, audit_details, cron_details, archive_details)
VALUES (
    'd4000000-0000-0000-0000-000000000005',
    'a0000000-0000-0000-0000-000000000002',
    'b0000000-0000-0000-0000-000000000007',
    '[{
        "process_type": "采购审批",
        "custom_rules": [],
        "field_overrides": ["main:cgje"],
        "field_mode": "selected",
        "strictness_override": "loose",
        "rule_toggle_overrides": [
            {"rule_id": "d2000000-0000-0000-0000-000000000014", "enabled": false}
        ]
    }]'::jsonb,
    '[]'::jsonb,
    '[{
        "process_type": "采购审批",
        "custom_rules": [],
        "field_overrides": [],
        "field_mode": "all",
        "strictness_override": "loose",
        "rule_toggle_overrides": []
    }]'::jsonb
);

-- ============================================================
-- 仪表板偏好
-- ============================================================

-- reviewer01 在 DEMO_HQ（业务身份布局）
INSERT INTO user_dashboard_prefs (id, tenant_id, user_id, pref_scope, enabled_widgets, widget_sizes)
VALUES (
    'd5000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000003',
    'business',
    '["pending_tasks", "recent_audits", "token_usage", "rule_stats"]'::jsonb,
    '{"pending_tasks":"large","recent_audits":"medium","token_usage":"small","rule_stats":"small"}'::jsonb
);

-- supervisor01 在 DEMO_HQ（租户管理员身份布局）
INSERT INTO user_dashboard_prefs (id, tenant_id, user_id, pref_scope, enabled_widgets, widget_sizes)
VALUES (
    'd5000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000005',
    'tenant_admin',
    '["pending_tasks", "recent_audits", "team_overview", "token_usage"]'::jsonb,
    '{"pending_tasks":"medium","recent_audits":"large","team_overview":"medium","token_usage":"small"}'::jsonb
);

-- br1_reviewer 在 DEMO_BR1（业务身份布局）
INSERT INTO user_dashboard_prefs (id, tenant_id, user_id, pref_scope, enabled_widgets, widget_sizes)
VALUES (
    'd5000000-0000-0000-0000-000000000003',
    'a0000000-0000-0000-0000-000000000002',
    'b0000000-0000-0000-0000-000000000007',
    'business',
    '["pending_tasks", "recent_audits"]'::jsonb,
    '{"pending_tasks":"large","recent_audits":"large"}'::jsonb
);
