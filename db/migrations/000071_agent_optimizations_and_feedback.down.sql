-- 000071_agent_optimizations_and_feedback.down.sql

DROP INDEX IF EXISTS idx_chat_messages_tenant_feedback;

ALTER TABLE chat_messages
    DROP COLUMN IF EXISTS feedback,
    DROP COLUMN IF EXISTS feedback_at;

ALTER TABLE agent_definitions
    DROP COLUMN IF EXISTS quick_questions;
