-- 000059：记录 OA 点击保存/提交的客户端时间，用于识别异常延迟事件

ALTER TABLE embed_refresh_events
    ADD COLUMN IF NOT EXISTS occurred_at_ms BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN embed_refresh_events.occurred_at_ms IS
    'OA 点击保存/提交时的浏览器 Unix 毫秒时间戳；仅用于时序排错，不作为数据库时间依据';
