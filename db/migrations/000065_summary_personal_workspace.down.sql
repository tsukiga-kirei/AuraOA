-- 回滚流程总结个人展示偏好与前台入口。

UPDATE org_roles
SET page_permissions = page_permissions - '/summary'
WHERE page_permissions @> '["/summary"]'::jsonb;

ALTER TABLE user_personal_configs
    DROP COLUMN IF EXISTS summary_details;
