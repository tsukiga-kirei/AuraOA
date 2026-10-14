-- 智能体多轮对话中保留最近几轮工具执行明细，默认 3 轮
ALTER TABLE agent_definitions ADD COLUMN tool_history_turns INT NOT NULL DEFAULT 3;
COMMENT ON COLUMN agent_definitions.tool_history_turns IS '智能体多轮对话中保留最近几轮工具执行明细，默认3轮（0为不保留明细）';
