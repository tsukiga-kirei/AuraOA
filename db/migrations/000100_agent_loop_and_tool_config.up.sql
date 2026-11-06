-- 智能体可配置对话循环步数；工具绑定可保存分页、行数等运行参数。

ALTER TABLE agent_definitions
    ADD COLUMN IF NOT EXISTS max_loop_steps INT NOT NULL DEFAULT 8;

COMMENT ON COLUMN agent_definitions.max_loop_steps IS '单轮对话中模型调用工具的最大步数，默认 8，范围 1-20';

ALTER TABLE agent_tool_bindings
    ADD COLUMN IF NOT EXISTS config JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN agent_tool_bindings.config IS '该智能体上此工具的运行配置，如分页条数、SQL 行数上限';

UPDATE agent_definitions
SET system_prompt = REPLACE(
        system_prompt,
        '4. 调用 execute_safe_query 工具执行查询，系统会自动限制条数并进行五层安全拦截。',
        '4. 调用 execute_safe_query 工具执行查询。需要多组数据时，在一次调用的 queries 数组里提交多条 SELECT，不要拆成多次循环。系统按该智能体配置限制每条 SQL 的行数，并做五层安全拦截。'
    ),
    updated_at = NOW()
WHERE agent_code = 'oa_analyst'
  AND system_prompt LIKE '%调用 execute_safe_query 工具执行查询，系统会自动限制条数并进行五层安全拦截。%';

UPDATE agent_skills
SET content = REPLACE(
        content,
        '- 必须带限制条数（如 LIMIT 20）。',
        '- 行数由智能体上的查数配置截断，SQL 里不必再手写 LIMIT。需要多组数据时，在一次 execute_safe_query 调用的 queries 数组里提交多条 SELECT。'
    ),
    updated_at = NOW()
WHERE skill_code = 'oa_workflow_stats'
  AND content LIKE '%必须带限制条数（如 LIMIT 20）。%';
