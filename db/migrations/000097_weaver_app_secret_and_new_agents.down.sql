-- 000097_weaver_app_secret_and_new_agents.down.sql

-- 1. 删除工具与技能绑定
DELETE FROM agent_tool_bindings 
WHERE agent_id IN (
    SELECT id FROM agent_definitions 
    WHERE agent_code IN ('oa_creator', 'oa_analyst') AND tenant_id IS NULL
);

-- 2. 删除预置技能
DELETE FROM agent_skills 
WHERE skill_code IN ('create_supplies_request', 'oa_workflow_stats') AND tenant_id IS NULL;

-- 3. 删除预置智能体
DELETE FROM agent_definitions 
WHERE agent_code IN ('oa_creator', 'oa_analyst') AND tenant_id IS NULL;

-- 4. 删除字段
ALTER TABLE oa_database_connections 
    DROP COLUMN IF EXISTS weaver_app_secret;
