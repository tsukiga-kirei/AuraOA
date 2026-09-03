-- 000064_attachment_document_parser_types.up.sql
-- 允许管理员按扩展名选择代码文档解析；未选择的 PDF / 新版 Office 保持 MinerU 路由。

INSERT INTO system_configs (key, value, remark)
SELECT
    'attachment.document_parser_types',
    CONCAT_WS(',',
        CASE WHEN COALESCE((SELECT value FROM system_configs WHERE key = 'attachment.legacy_office_enabled'), 'false') IN ('true', '1')
             THEN 'doc,xls,ppt' END,
        CASE WHEN COALESCE((SELECT value FROM system_configs WHERE key = 'attachment.ofd_enabled'), 'false') IN ('true', '1')
             THEN 'ofd' END
    ),
    '使用代码提取正文的文件类型（pdf、Office、OFD，逗号分隔）'
ON CONFLICT (key) DO NOTHING;
