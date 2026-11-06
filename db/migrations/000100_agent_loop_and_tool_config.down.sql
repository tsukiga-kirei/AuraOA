UPDATE agent_definitions
SET system_prompt = REPLACE(
        system_prompt,
        '4. 调用 execute_safe_query 工具执行查询。需要多组数据时，在一次调用的 queries 数组里提交多条 SELECT，不要拆成多次循环。系统按该智能体配置限制每条 SQL 的行数，并做五层安全拦截。',
        '4. 调用 execute_safe_query 工具执行查询，系统会自动限制条数并进行五层安全拦截。'
    ),
    updated_at = NOW()
WHERE agent_code = 'oa_analyst'
  AND system_prompt LIKE '%在一次调用的 queries 数组里提交多条 SELECT%';

UPDATE agent_skills
SET content = REPLACE(
        content,
        '- 行数由智能体上的查数配置截断，SQL 里不必再手写 LIMIT。需要多组数据时，在一次 execute_safe_query 调用的 queries 数组里提交多条 SELECT。',
        '- 必须带限制条数（如 LIMIT 20）。'
    ),
    updated_at = NOW()
WHERE skill_code = 'oa_workflow_stats'
  AND content LIKE '%行数由智能体上的查数配置截断%';

ALTER TABLE agent_tool_bindings DROP COLUMN IF EXISTS config;
ALTER TABLE agent_definitions DROP COLUMN IF EXISTS max_loop_steps;
