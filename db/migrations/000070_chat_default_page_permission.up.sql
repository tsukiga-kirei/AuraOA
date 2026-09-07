-- 已有工作台权限的系统角色补齐对话入口，与新建租户默认值对齐。
UPDATE org_roles
SET page_permissions = page_permissions || '["/chat"]'::jsonb
WHERE is_system = true
  AND page_permissions @> '["/dashboard"]'::jsonb
  AND NOT page_permissions @> '["/chat"]'::jsonb;
