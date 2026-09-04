-- 000065：为流程总结补充个人展示偏好与前台工作台页面权限。

ALTER TABLE user_personal_configs
    ADD COLUMN summary_details JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN user_personal_configs.summary_details IS
    '用户在流程总结工作台中的分块展示偏好';

-- 现有具备审核工作台或归档复盘权限的角色默认获得流程总结入口。
UPDATE org_roles
SET page_permissions = page_permissions || '["/summary"]'::jsonb
WHERE (page_permissions @> '["/dashboard"]'::jsonb OR page_permissions @> '["/archive"]'::jsonb)
  AND NOT page_permissions @> '["/summary"]'::jsonb;
