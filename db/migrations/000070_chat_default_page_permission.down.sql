UPDATE org_roles
SET page_permissions = page_permissions - '/chat'
WHERE is_system = true
  AND page_permissions @> '["/dashboard"]'::jsonb
  AND page_permissions @> '["/chat"]'::jsonb;
