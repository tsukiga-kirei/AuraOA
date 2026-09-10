-- 000076_agent_my_requests_tool_and_quick_questions.down.sql

DELETE FROM agent_tool_bindings WHERE tool_code = 'list_my_requests';

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
        "icon": "lightbulb",
        "title": "流程规范咨询",
        "prompt": "公司的财务报销与差旅审批有哪些最新的合规要求？",
        "detail": "结合企业规范，给出合规建议"
    }
]'::jsonb
WHERE agent_code = 'oa_assist';
