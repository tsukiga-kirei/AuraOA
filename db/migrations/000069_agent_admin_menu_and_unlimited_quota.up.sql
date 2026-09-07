-- 已有租户管理员角色补齐智能体管理菜单，避免新建租户之外看不到入口。
UPDATE org_roles
SET page_permissions = page_permissions || '["/admin/tenant/agents"]'::jsonb
WHERE page_permissions @> '["/admin/tenant/org"]'::jsonb
  AND NOT page_permissions @> '["/admin/tenant/agents"]'::jsonb;

COMMENT ON COLUMN tenants.token_quota IS 'Token 配额；小于 0 表示不限制';
