-- 000064_attachment_document_parser_types.down.sql
-- 回退前把新选择同步回旧开关，避免降级后静默丢失旧版 Office / OFD 路由。

UPDATE system_configs
SET value = CASE
    WHEN EXISTS (
        SELECT 1
        FROM system_configs selected
        WHERE selected.key = 'attachment.document_parser_types'
          AND string_to_array(selected.value, ',') && ARRAY['doc', 'xls', 'ppt']
    ) THEN 'true'
    ELSE 'false'
END
WHERE key = 'attachment.legacy_office_enabled';

UPDATE system_configs
SET value = CASE
    WHEN EXISTS (
        SELECT 1
        FROM system_configs selected
        WHERE selected.key = 'attachment.document_parser_types'
          AND 'ofd' = ANY(string_to_array(selected.value, ','))
    ) THEN 'true'
    ELSE 'false'
END
WHERE key = 'attachment.ofd_enabled';

DELETE FROM system_configs WHERE key = 'attachment.document_parser_types';
