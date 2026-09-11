-- 000070: 规范审核快照与日志的三级渠道分类定义与触发来源（通用多租户定义，无硬编码）
COMMENT ON COLUMN audit_process_snapshots.channel IS '结论渠道：workbench=系统内 embed_standard=嵌入通用 embed_personal=嵌入个性化 embed=历史兼容';
COMMENT ON COLUMN audit_logs.trigger_source IS '触发源：workbench_manual/workbench_batch/embed_auto/embed_manual/cron/chat';
COMMENT ON COLUMN audit_logs.trigger_detail IS '触发细节扩展：如 personal_embed_manual (OA 嵌入个人视角定制审核)';
