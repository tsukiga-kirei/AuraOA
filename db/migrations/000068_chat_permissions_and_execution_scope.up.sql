-- 个人视角拥有独立绑定；标准视角继续使用空 scope。
ALTER TABLE process_execution_config_bindings ADD COLUMN scope VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE process_execution_config_bindings DROP CONSTRAINT uq_process_execution_config_bindings_process;
ALTER TABLE process_execution_config_bindings ADD CONSTRAINT uq_process_execution_config_bindings_process UNIQUE (tenant_id, module, process_id, scope);
-- 已被个人配置覆盖的旧绑定归到原使用者，避免继续污染标准视角。
UPDATE process_execution_config_bindings b SET scope = 'user:' || b.bound_by::text
FROM execution_config_versions v
WHERE b.config_version_id = v.id AND b.module IN ('audit', 'archive') AND b.bound_by IS NOT NULL
AND COALESCE((v.config_snapshot->>'personal_config_version_no')::integer, 0) > 0;
-- 只为迁移前已经获得聊天页面权限的角色显式初始化现有配额；后续清空授权即拒绝。
INSERT INTO org_role_agent_grants (tenant_id, role_id, agent_code)
SELECT r.tenant_id, r.id, code FROM org_roles r JOIN tenant_chat_allocations a ON a.tenant_id = r.tenant_id,
LATERAL jsonb_array_elements_text(a.agent_codes) code
WHERE r.page_permissions ? '/chat' AND NOT EXISTS (SELECT 1 FROM org_role_agent_grants g WHERE g.role_id = r.id)
ON CONFLICT DO NOTHING;
INSERT INTO org_role_tool_grants (tenant_id, role_id, tool_code)
SELECT r.tenant_id, r.id, code FROM org_roles r JOIN tenant_chat_allocations a ON a.tenant_id = r.tenant_id,
LATERAL jsonb_array_elements_text(a.tool_codes) code
WHERE r.page_permissions ? '/chat' AND NOT EXISTS (SELECT 1 FROM org_role_tool_grants g WHERE g.role_id = r.id)
ON CONFLICT DO NOTHING;
