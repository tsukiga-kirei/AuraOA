UPDATE org_roles
SET page_permissions = page_permissions - '/admin/tenant/agents'
WHERE page_permissions @> '["/admin/tenant/agents"]'::jsonb;
