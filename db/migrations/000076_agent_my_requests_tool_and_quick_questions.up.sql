-- 000076_agent_my_requests_tool_and_quick_questions.up.sql

-- 1. 更新 OA 辅助办理助手（oa_assist）的灵感提问列表：将第三项替换为“我的申请进度”
UPDATE agent_definitions
SET quick_questions = '[
    {
        "icon": "edit",
        "title": "起草审批意见",
        "prompt": "针对最新的待办流程，请帮我起草一份规范专业的审批同意意见",
        "detail": "规范措辞，合规高效批复"
    },
    {
        "icon": "barchart",
        "title": "流程深度总结",
        "prompt": "请帮我梳理并总结最近重点流程的流转节点与处理耗时",
        "detail": "提取要点，掌握宏观进展"
    },
    {
        "icon": "clock",
        "title": "我的申请进度",
        "prompt": "请帮我查一下我最近发起的流程有哪些，以及它们当前的审批状态",
        "detail": "追踪发起单据，掌握流转进展"
    }
]'::jsonb
WHERE agent_code = 'oa_assist';

-- 2. 为系统种子智能体（oa_query 和 oa_assist）以及租户覆盖的智能体绑定新工具 list_my_requests
INSERT INTO agent_tool_bindings (tenant_id, agent_id, tool_type, tool_code)
SELECT a.tenant_id, a.id, 'system', 'list_my_requests'
FROM agent_definitions a
WHERE a.agent_code IN ('oa_query', 'oa_assist')
  AND NOT EXISTS (
      SELECT 1 FROM agent_tool_bindings b
      WHERE b.agent_id = a.id AND b.tool_code = 'list_my_requests'
  );
