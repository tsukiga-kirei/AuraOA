-- 增加 chat_messages 表 feedback_comment 字段，记录用户点踩时的改进建议与反馈意见
ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS feedback_comment TEXT;

COMMENT ON COLUMN chat_messages.feedback_comment IS '用户针对 Assistant 单轮回复的反馈意见与改进建议';
