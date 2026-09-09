-- 000071_agent_optimizations_and_feedback.up.sql

-- 1. 扩展 agent_definitions 表，支持自定义快捷输入问题
ALTER TABLE agent_definitions
    ADD COLUMN IF NOT EXISTS quick_questions JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN agent_definitions.quick_questions IS '智能体专属推荐/快捷提问列表 (JSONB 数组: [{icon, title, prompt, detail}])';

-- 为种子智能体注入差异化的高质量默认快捷问题
UPDATE agent_definitions
SET quick_questions = '[
    {
        "icon": "clipboard",
        "title": "查我的待办",
        "prompt": "请帮我查一下最近我有哪些待办流程需要处理？",
        "detail": "梳理待办，找到下一步"
    },
    {
        "icon": "search",
        "title": "查指定流程进展",
        "prompt": "请帮我查下我最近发起的借款或报销流程状态",
        "detail": "追踪关键事项流转进度"
    },
    {
        "icon": "hourglass",
        "title": "查超时滞留审批",
        "prompt": "请帮我检查当前是否有滞留超过3天的待办审批流程？",
        "detail": "发现流程阻塞，提醒催办"
    }
]'::jsonb
WHERE agent_code = 'oa_query' AND (quick_questions IS NULL OR quick_questions = '[]'::jsonb);

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
WHERE agent_code = 'oa_assist' AND (quick_questions IS NULL OR quick_questions = '[]'::jsonb);

-- 2. 扩展 chat_messages 表，支持 AI 回答的点赞/点踩反馈
ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS feedback VARCHAR(16) DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS feedback_at TIMESTAMPTZ DEFAULT NULL;

COMMENT ON COLUMN chat_messages.feedback IS '用户对 AI 回复的评价反馈：like (赞) / dislike (踩) / NULL (未评价)';
COMMENT ON COLUMN chat_messages.feedback_at IS '用户最后一次提交评价反馈的时间';

-- 3. 创建索引加速评价统计查询
CREATE INDEX IF NOT EXISTS idx_chat_messages_tenant_feedback
    ON chat_messages (tenant_id, feedback)
    WHERE feedback IS NOT NULL;
