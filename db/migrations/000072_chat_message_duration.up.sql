-- 增加 chat_messages 表 duration_ms 字段，记录 Assistant 回复耗时（毫秒）
ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS duration_ms INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN chat_messages.duration_ms IS 'Assistant 生成单轮回复总耗时（毫秒）';
