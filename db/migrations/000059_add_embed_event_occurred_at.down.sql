-- 回滚 000059：删除 OA 操作客户端时间

ALTER TABLE embed_refresh_events
    DROP COLUMN IF EXISTS occurred_at_ms;
