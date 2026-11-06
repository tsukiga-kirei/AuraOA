UPDATE agent_skills
SET content = REPLACE(
        content,
        '先用 list_creatable_workflows 确认当前用户能创建目标流程。名称明确时传流程名，含糊时查看全部可创建流程。没有权限就停止，不要读取表单。确认有权限后，再利用 get_workflow_schema 获取表单下拉选项：',
        '用户通常不清楚具体的下拉框分类。利用 get_workflow_schema 获取表单下拉选项：'
    ),
    updated_at = NOW()
WHERE skill_code = 'create_supplies_request'
  AND content LIKE '%先用 list_creatable_workflows 确认当前用户能创建目标流程%';

UPDATE agent_definitions
SET system_prompt = REPLACE(
        system_prompt,
        '1. 先查创建权限，再看表单样式：根据用户陈述调用 list_creatable_workflows。流程名称明确时传 workflow_name；含糊时不传条件，查当前用户全部可创建流程再匹配。没有命中就说明无权发起并停止。确认有权限后，才调用 get_workflow_schema 读取该流程的表单字段与下拉选项。',
        '1. 意图理解与表单自查：根据用户陈述的事项，调用 get_workflow_schema 自查目标流程的表单字段与下拉字典选项（如物品领用、请假等）。'
    ),
    updated_at = NOW()
WHERE agent_code = 'oa_creator'
  AND system_prompt LIKE '%先查创建权限，再看表单样式%';

DELETE FROM agent_tool_bindings
WHERE tool_code = 'list_creatable_workflows';
