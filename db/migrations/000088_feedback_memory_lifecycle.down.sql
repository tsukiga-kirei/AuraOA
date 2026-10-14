DROP TABLE feedback_memory_comment_exclusions;
ALTER TABLE feedback_memory_versions DROP COLUMN generated_items, DROP COLUMN edit_revision, DROP COLUMN edited_by, DROP COLUMN edited_at;
-- 回退到旧模型时移除逻辑删除的素材，历史版本快照不受影响。
DELETE FROM feedback_memory_entries WHERE deleted_at IS NOT NULL;
DROP INDEX feedback_memory_entries_live_source;
ALTER TABLE feedback_memory_entries ADD CONSTRAINT feedback_memory_entries_memory_id_source_id_key UNIQUE(memory_id,source_id);
ALTER TABLE feedback_memory_entries DROP COLUMN implanted_version_id, DROP COLUMN deleted_at;
UPDATE feedback_memories SET published_version_id=NULL WHERE NOT enabled;
ALTER TABLE feedback_memories DROP COLUMN enabled;
