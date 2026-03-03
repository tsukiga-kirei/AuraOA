-- 004_org_roles.sql
-- Seed data: 3 system org roles with page_permissions
-- Run after 001_tenants.sql

-- Fixed UUIDs for referential integrity:
-- OrgRoles (all under DEMO_HQ tenant a0000000-0000-0000-0000-000000000001):
--   d0000000-0000-0000-0000-000000000001  审核员
--   d0000000-0000-0000-0000-000000000002  审核主管
--   d0000000-0000-0000-0000-000000000003  租户管理员

INSERT INTO org_roles (id, tenant_id, name, description, page_permissions, is_system)
VALUES
    (
        'd0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000001',
        '审核员',
        '负责日常流程审核工作',
        '[
            {"key": "review-workbench", "label": "审核工作台", "path": "/review/workbench"},
            {"key": "review-history", "label": "审核历史", "path": "/review/history"}
        ]',
        TRUE
    ),
    (
        'd0000000-0000-0000-0000-000000000002',
        'a0000000-0000-0000-0000-000000000001',
        '审核主管',
        '负责审核监督和团队管理',
        '[
            {"key": "review-workbench", "label": "审核工作台", "path": "/review/workbench"},
            {"key": "review-history", "label": "审核历史", "path": "/review/history"},
            {"key": "review-dashboard", "label": "审核看板", "path": "/review/dashboard"},
            {"key": "review-rules", "label": "规则管理", "path": "/review/rules"}
        ]',
        TRUE
    ),
    (
        'd0000000-0000-0000-0000-000000000003',
        'a0000000-0000-0000-0000-000000000001',
        '租户管理员',
        '负责租户内部的组织和配置管理',
        '[
            {"key": "review-workbench", "label": "审核工作台", "path": "/review/workbench"},
            {"key": "review-history", "label": "审核历史", "path": "/review/history"},
            {"key": "review-dashboard", "label": "审核看板", "path": "/review/dashboard"},
            {"key": "review-rules", "label": "规则管理", "path": "/review/rules"},
            {"key": "tenant-org", "label": "组织管理", "path": "/admin/tenant/org"},
            {"key": "tenant-settings", "label": "租户设置", "path": "/admin/tenant/settings"}
        ]',
        TRUE
    );
